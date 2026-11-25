package netenv

import (
	"fmt"
	"net"
	"os"
	"strings"
)

// ResolvConfAudit captures host /etc/resolv.conf self-loop inspection results and safe fallback guidance.
type ResolvConfAudit struct {
	Path        string   `json:"path"`
	Readable    bool     `json:"readable"`
	Nameservers []string `json:"nameservers"`
	LoopedAddrs []string `json:"looped_addrs"`
	HasSelfLoop bool     `json:"has_self_loop"`
	FallbackDNS string   `json:"fallback_dns"`
	Warning     string   `json:"warning,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

// ParseResolvConfNameservers extracts all configured nameserver IP addresses from resolv.conf content.
func ParseResolvConfNameservers(content string) []string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, 4)
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if idx := strings.IndexAny(line, "#;"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.EqualFold(fields[0], "nameserver") {
			addr := strings.TrimSpace(fields[1])
			if addr != "" {
				out = append(out, addr)
			}
		}
	}
	return out
}

// CollectLocalInterfaceIPs gathers all unicast IPv4/IPv6 addresses currently assigned to host interfaces.
func CollectLocalInterfaceIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	seen := make(map[string]bool)
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP == nil {
			continue
		}
		norm := normalizeIPString(ipNet.IP.String())
		if norm != "" && !seen[norm] {
			seen[norm] = true
			out = append(out, norm)
		}
	}
	return out
}

func normalizeIPString(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if zoneIdx := strings.Index(s, "%"); zoneIdx >= 0 {
		s = s[:zoneIdx]
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return ""
	}
	return ip.String()
}

// IsLoopbackOrSelfIP reports whether addr points to loopback (127.0.0.0/8, ::1), unspecified (0.0.0.0, ::),
// or any local interface IP of the current host.
func IsLoopbackOrSelfIP(addr string, localIPs []string) bool {
	norm := normalizeIPString(addr)
	if norm == "" {
		return false
	}
	ip := net.ParseIP(norm)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsUnspecified() {
		return true
	}
	for _, local := range localIPs {
		if normalizeIPString(local) == norm {
			return true
		}
	}
	return false
}

// SelectSafeFallbackDNS picks an upstream gateway DNS if it is valid and not loopback/self, otherwise 223.5.5.5.
func SelectSafeFallbackDNS(gatewayIP string, localIPs []string) string {
	gwNorm := normalizeIPString(gatewayIP)
	if gwNorm != "" && !IsLoopbackOrSelfIP(gwNorm, localIPs) {
		return gwNorm
	}
	return "223.5.5.5"
}

// AuditResolvConf reads resolvPath (default /etc/resolv.conf) and checks for self-looping DNS entries.
func AuditResolvConf(resolvPath string, localIPs []string, gatewayIP string) ResolvConfAudit {
	if strings.TrimSpace(resolvPath) == "" {
		resolvPath = "/etc/resolv.conf"
	}
	if localIPs == nil {
		localIPs = CollectLocalInterfaceIPs()
	}
	if strings.TrimSpace(gatewayIP) == "" {
		_, gatewayIP = parseProcNetDefaultRoute("/proc/net/route")
	}

	raw, err := os.ReadFile(resolvPath)
	if err != nil {
		return ResolvConfAudit{
			Path:        resolvPath,
			Readable:    false,
			FallbackDNS: SelectSafeFallbackDNS(gatewayIP, localIPs),
		}
	}
	return AuditResolvConfContent(resolvPath, string(raw), localIPs, gatewayIP)
}

// AuditResolvConfContent evaluates resolv.conf text for loopback or local-IP DNS self-loops.
func AuditResolvConfContent(resolvPath, content string, localIPs []string, gatewayIP string) ResolvConfAudit {
	if strings.TrimSpace(resolvPath) == "" {
		resolvPath = "/etc/resolv.conf"
	}
	if localIPs == nil {
		localIPs = CollectLocalInterfaceIPs()
	}
	nameservers := ParseResolvConfNameservers(content)
	looped := make([]string, 0)
	seenLoop := make(map[string]bool)
	for _, ns := range nameservers {
		if IsLoopbackOrSelfIP(ns, localIPs) {
			if !seenLoop[ns] {
				seenLoop[ns] = true
				looped = append(looped, ns)
			}
		}
	}

	fallbackDNS := SelectSafeFallbackDNS(gatewayIP, localIPs)
	audit := ResolvConfAudit{
		Path:        resolvPath,
		Readable:    true,
		Nameservers: nameservers,
		LoopedAddrs: looped,
		HasSelfLoop: len(looped) > 0,
		FallbackDNS: fallbackDNS,
	}

	if audit.HasSelfLoop {
		loopStr := strings.Join(looped, ", ")
		audit.Warning = fmt.Sprintf(
			"检测到宿主机 %s 包含指向本机/回环的 nameserver (%s)，若主路由 DHCP 将 DNS 下发给本机，极易在 DHCP 续租后形成 DNS 自环死锁；sing-box 已自动弃用 dns-local 并安全回退至 %s。",
			resolvPath, loopStr, fallbackDNS,
		)
		audit.Remediation = fmt.Sprintf(
			"请温和固定本机静态 IP 与独立上游 DNS（切勿使用 chattr +i 锁死 %s，以免破坏 systemd-resolved / NetworkManager）：\n"+
				"1) NetworkManager: nmcli con mod <连接名> ipv4.dns \"%s\" ipv4.ignore-auto-dns yes && nmcli con up <连接名>\n"+
				"2) systemd-resolved: 在 /etc/systemd/resolved.conf 设置 DNS=%s 后执行 systemctl restart systemd-resolved\n"+
				"3) /etc/network/interfaces: 改为 static 静态地址并显式指定 dns-nameservers %s",
			resolvPath, fallbackDNS, fallbackDNS, fallbackDNS,
		)
	}
	return audit
}
