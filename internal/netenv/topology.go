package netenv

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"
)

// DefaultFakeIPCIDR 是协议层标准保留的基准网段（RFC 2544），允许作为唯一协议常量集中定义
const DefaultFakeIPCIDR = "198.18.0.0/15"

// PlaceholderLocalIP 当尚未探测到本机 IP 时用于引导脚本与界面的安全占位符（绝不伪造假 IP）
const PlaceholderLocalIP = "<待配置本机IP>"

// PlaceholderGatewayIP 当尚未探测到网关 IP 时用于引导脚本与界面的安全占位符
const PlaceholderGatewayIP = "<待配置网关IP>"

// HostTopology 描述从宿主机/容器内核动态嗅探到的真实网络拓扑（绝不注入假 IP 兜底）
type HostTopology struct {
	Ready             bool   `json:"ready"`              // 是否已成功探测到完整本机 IP 与上级主路由网关
	Interface         string `json:"interface"`          // 主出站网卡名，如 "eth0"
	LocalIP           string `json:"local_ip"`           // 真实本机 IPv4，未探到时严格为 ""
	GatewayIP         string `json:"gateway_ip"`         // 真实上级主路由 IPv4，未探到时严格为 ""
	FakeIPCIDR        string `json:"fake_ip_cidr"`       // 分流网段 "198.18.0.0/15"
	DiagnosticWarning string `json:"diagnostic_warning"` // 当 Ready == false 时直接展示给前端用户的明确排障提示
}

// DetectHostTopology 直接读取内核 /proc/net/route 与网卡地址表，0 外部命令依赖、0 假数据兜底
func DetectHostTopology(envHostIPOverride, envGatewayOverride string) HostTopology {
	return DetectHostTopologyWithProcRoute("/proc/net/route", envHostIPOverride, envGatewayOverride)
}

// DetectHostTopologyWithProcRoute 允许传入指定 proc 路由文件路径（用于测试与沙箱注入）
func DetectHostTopologyWithProcRoute(procPath, envHostIPOverride, envGatewayOverride string) HostTopology {
	localIP := strings.TrimSpace(envHostIPOverride)
	if localIP == "" {
		localIP = strings.TrimSpace(os.Getenv("CUBEX_HOST_IP"))
	}
	gatewayIP := strings.TrimSpace(envGatewayOverride)
	if gatewayIP == "" {
		gatewayIP = strings.TrimSpace(os.Getenv("CUBEX_GATEWAY_IP"))
	}

	topo := HostTopology{
		LocalIP:    localIP,
		GatewayIP:  gatewayIP,
		FakeIPCIDR: DefaultFakeIPCIDR,
	}

	// 1. 直接解析 Linux 内核 /proc/net/route 获取默认路由网卡与网关 IP
	iface, gwIP := parseProcNetDefaultRoute(procPath)
	topo.Interface = iface
	if topo.GatewayIP == "" {
		topo.GatewayIP = gwIP
	}

	// 2. 通过标准库 net.Interfaces 获取对应网卡或首个非环回网卡的真实 IPv4
	if topo.LocalIP == "" {
		topo.LocalIP = resolveInterfaceIPv4(topo.Interface)
	}

	// 3. 若处于无 /proc 虚拟化环境（如开发宿主或测试），尝试 UDP 出站探针推断本机地址
	if topo.LocalIP == "" {
		if conn, err := net.Dial("udp", "223.5.5.5:80"); err == nil {
			if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok && udpAddr.IP != nil {
				if v4 := udpAddr.IP.To4(); v4 != nil && !v4.IsLoopback() {
					topo.LocalIP = v4.String()
				}
			}
			_ = conn.Close()
		}
	}

	// 4. 严格完备性判定与自环防御
	switch {
	case topo.LocalIP == "" && topo.GatewayIP == "":
		topo.Ready = false
		topo.DiagnosticWarning = "未探测到有效局域网 IP 与默认网关，请检查容器网络模式（建议使用 host 或 macvlan）或通过环境变量 CUBEX_HOST_IP 指定。"
	case topo.LocalIP == "":
		topo.Ready = false
		topo.DiagnosticWarning = fmt.Sprintf("已探测到上级网关 (%s)，但未能获取本机网卡 IPv4 地址，请检查网卡状态。", topo.GatewayIP)
	case topo.GatewayIP == "":
		topo.Ready = false
		topo.DiagnosticWarning = fmt.Sprintf("已探测到本机 IP (%s)，但系统路由表缺少默认网关 (default route)，请检查主路由 DHCP 或静态网关配置。", topo.LocalIP)
	case topo.LocalIP == topo.GatewayIP:
		topo.Ready = false
		topo.DiagnosticWarning = fmt.Sprintf("检测到本机 IP 与上级网关同为 %s，存在路由自环风险，请修正主路由网关地址！", topo.LocalIP)
	default:
		topo.Ready = true
	}

	return topo
}

// parseProcNetDefaultRoute 解析 /proc/net/route 中 Destination == 00000000 的默认路由行
func parseProcNetDefaultRoute(procPath string) (iface string, gatewayIP string) {
	f, err := os.Open(procPath)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// 跳过首行表头: Iface Destination Gateway Flags RefCnt Use Metric Mask MTU Window IRTT
	if !scanner.Scan() {
		return "", ""
	}

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 4 {
			continue
		}
		// Destination 为全 0 代表默认路由 (0.0.0.0/0)
		if fields[1] != "00000000" {
			continue
		}
		ipStr, ok := parseHexLittleEndianIPv4(fields[2])
		if ok && ipStr != "0.0.0.0" {
			return fields[0], ipStr
		}
	}
	return "", ""
}

func parseHexLittleEndianIPv4(hexStr string) (string, bool) {
	if len(hexStr) != 8 {
		return "", false
	}
	raw, err := hex.DecodeString(hexStr)
	if err != nil || len(raw) != 4 {
		return "", false
	}
	// /proc/net/route 在小端架构下按字节逆序存储 IPv4
	val := binary.LittleEndian.Uint32(raw)
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, val)
	return ip.String(), true
}

func resolveInterfaceIPv4(preferredIface string) string {
	if preferredIface != "" {
		if nif, err := net.InterfaceByName(preferredIface); err == nil {
			if ip := firstNonLoopbackIPv4(nif); ip != "" {
				return ip
			}
		}
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for i := range ifaces {
		if (ifaces[i].Flags&net.FlagUp) == 0 || (ifaces[i].Flags&net.FlagLoopback) != 0 {
			continue
		}
		if strings.HasPrefix(ifaces[i].Name, "cubex") || strings.HasPrefix(ifaces[i].Name, "tun") || strings.HasPrefix(ifaces[i].Name, "docker") {
			continue
		}
		if ip := firstNonLoopbackIPv4(&ifaces[i]); ip != "" {
			return ip
		}
	}
	return ""
}

func firstNonLoopbackIPv4(nif *net.Interface) string {
	addrs, err := nif.Addrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if v4 := ipNet.IP.To4(); v4 != nil {
				return v4.String()
			}
		}
	}
	return ""
}
