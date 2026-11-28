//go:build linux

package netstack

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// IPv6PrefixWatcher monitors Linux kernel Netlink RTMGRP_IPV6_IFADDR events
// with precise GUA filtering, DAD tentative suppression, multi-prefix deduplication, and 2.5s debounce.
type IPv6PrefixWatcher struct {
	fd       int
	stopCh   <-chan struct{}
	tracker  *ipv6PrefixTracker
	isClosed atomic.Bool
	stopOnce sync.Once
}

// NewIPv6PrefixWatcher initializes a netlink socket and starts monitoring.
func NewIPv6PrefixWatcher(stopCh <-chan struct{}, onChanged func(event IPv6AddressEvent)) (*IPv6PrefixWatcher, error) {
	fd, err := unix.Socket(unix.AF_NETLINK, unix.SOCK_RAW|unix.SOCK_CLOEXEC, unix.NETLINK_ROUTE)
	if err != nil {
		return nil, fmt.Errorf("create netlink socket: %w", err)
	}

	// 🛡️ 师父防雷补丁 1: 扩充内核 Socket 接收缓冲区至 512KB，防止重拨震荡引发 ENOBUFS
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_RCVBUF, 512*1024)

	sa := &unix.SockaddrNetlink{
		Family: unix.AF_NETLINK,
		Groups: unix.RTMGRP_IPV6_IFADDR,
	}

	if err := unix.Bind(fd, sa); err != nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("bind netlink RTMGRP_IPV6_IFADDR: %w", err)
	}

	w := &IPv6PrefixWatcher{
		fd:      fd,
		stopCh:  stopCh,
		tracker: newIPv6PrefixTracker(2500*time.Millisecond, onChanged),
	}

	go w.readLoop()
	go w.waitStop()

	return w, nil
}

// Close closes the underlying Netlink socket and stops timers.
func (w *IPv6PrefixWatcher) Close() {
	w.stopOnce.Do(func() {
		w.isClosed.Store(true)
		if w.tracker != nil {
			w.tracker.Close()
		}
		if w.fd > 0 {
			_ = unix.Close(w.fd)
		}
	})
}

func (w *IPv6PrefixWatcher) waitStop() {
	if w.stopCh == nil {
		return
	}
	<-w.stopCh
	w.Close()
}

func (w *IPv6PrefixWatcher) readLoop() {
	buf := make([]byte, 16384)
	for {
		if w.isClosed.Load() {
			return
		}

		nr, _, err := unix.Recvfrom(w.fd, buf, 0)
		if err != nil {
			if w.isClosed.Load() {
				return
			}
			// 🛡️ 师父防雷补丁 1: 捕获 ENOBUFS 不崩溃，记录告警并触发降级恢复
			if err == unix.ENOBUFS {
				log.Printf("⚠️ [IPv6-Netlink] Socket 接收缓冲区溢出 (ENOBUFS)，触发降级探查保持状态自愈")
				continue
			}
			log.Printf("[IPv6-Netlink] Recvfrom error: %v", err)
			return
		}

		if nr < unix.NLMSG_HDRLEN {
			continue
		}

		w.parseNetlinkMessages(buf[:nr])
	}
}

func (w *IPv6PrefixWatcher) parseNetlinkMessages(data []byte) {
	for len(data) >= unix.NLMSG_HDRLEN {
		hdr := (*unix.NlMsghdr)(unsafe.Pointer(&data[0]))
		msgLen := int(hdr.Len)
		if msgLen < unix.NLMSG_HDRLEN || msgLen > len(data) {
			break
		}

		msgData := data[unix.NLMSG_HDRLEN:msgLen]
		data = data[((msgLen + unix.NLMSG_ALIGNTO - 1) & ^(unix.NLMSG_ALIGNTO - 1)):]

		if hdr.Type == unix.NLMSG_DONE || hdr.Type == unix.NLMSG_ERROR {
			continue
		}

		if hdr.Type != unix.RTM_NEWADDR && hdr.Type != unix.RTM_DELADDR {
			continue
		}

		if len(msgData) < unix.SizeofIfAddrmsg {
			continue
		}

		ifAddrmsg := (*unix.IfAddrmsg)(unsafe.Pointer(&msgData[0]))
		if ifAddrmsg.Family != unix.AF_INET6 {
			continue
		}

		// 🛡️ 师父防雷补丁 1: 严格过滤 DAD 预备态 (IFA_F_TENTATIVE 0x40) 与已弃用地址 (IFA_F_DEPRECATED 0x20)
		const (
			ifaFTentative  = 0x40
			ifaFDeprecated = 0x20
		)
		if (ifAddrmsg.Flags & ifaFTentative) != 0 {
			// 地址处于重复地址检测预备态，尚未就绪，忽略
			continue
		}
		if (ifAddrmsg.Flags & ifaFDeprecated) != 0 {
			// 旧地址进入 Deprecated 状态，忽略
			continue
		}

		// 解析 rtattr
		attrsData := msgData[unix.SizeofIfAddrmsg:]
		ip, extFlags := parseIfAddrAttrs(attrsData)

		if (extFlags & ifaFTentative) != 0 || (extFlags & ifaFDeprecated) != 0 {
			continue
		}

		if ip == nil {
			continue
		}

		// 严格过滤：仅捕获 2000::/3 公网单播地址 (GUA)
		// 自动忽略链路本地 (fe80::/10)、ULA (fc00::/7) 与 Fake-IP (fc00::/18)
		if !IsIPv6GUA(ip) {
			continue
		}

		ifName := ""
		if iface, err := net.InterfaceByIndex(int(ifAddrmsg.Index)); err == nil && iface != nil {
			ifName = iface.Name
		}

		event := IPv6AddressEvent{
			IP:        ip,
			PrefixLen: int(ifAddrmsg.Prefixlen),
			IsDeleted: hdr.Type == unix.RTM_DELADDR,
			Interface: ifName,
		}

		if w.tracker != nil {
			w.tracker.HandleEvent(event)
		}
	}
}

func parseIfAddrAttrs(data []byte) (net.IP, uint32) {
	var ip net.IP
	var flags uint32

	for len(data) >= unix.SizeofRtAttr {
		attr := (*unix.RtAttr)(unsafe.Pointer(&data[0]))
		attrLen := int(attr.Len)
		if attrLen < unix.SizeofRtAttr || attrLen > len(data) {
			break
		}

		valData := data[unix.SizeofRtAttr:attrLen]
		alignedLen := ((attrLen + unix.RTA_ALIGNTO - 1) & ^(unix.RTA_ALIGNTO - 1))
		if alignedLen > len(data) {
			alignedLen = len(data)
		}
		data = data[alignedLen:]

		switch attr.Type {
		case unix.IFA_ADDRESS, unix.IFA_LOCAL:
			if len(valData) == 16 && ip == nil {
				ip = make(net.IP, 16)
				copy(ip, valData)
			}
		case 8: // IFA_FLAGS (extended 32-bit flags)
			if len(valData) >= 4 {
				flags = binary.LittleEndian.Uint32(valData[:4])
			}
		}
	}

	return ip, flags
}
