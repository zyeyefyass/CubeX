package onboarding

import (
	"fmt"
	"strings"

	"github.com/zyeyefyass/CubeX/internal/netenv"
)

// NonDNSCIDREntry 定义不走 DNS、硬编码 IP 直连的核心海外业务网段
type NonDNSCIDREntry struct {
	CIDR    string `json:"cidr"`
	Netmask string `json:"netmask"`
	Network string `json:"network"`
	Comment string `json:"comment"`
}

// EssentialNonDNSCIDRs 收录不走 DNS 解析、直接硬编码 IP 连接的核心海外服务网段（共 10 条，极简零负担）
var EssentialNonDNSCIDRs = []NonDNSCIDREntry{
	// 1. CubeX 核心 Fake-IP 回程网段
	{CIDR: "198.18.0.0/15", Netmask: "255.254.0.0", Network: "198.18.0.0", Comment: "CubeX-FakeIP-Core"},
	// 2. Telegram 全球 DC1~DC5 硬编码直连 IP 段（不经过 DNS，必须静态路由接管）
	{CIDR: "149.154.160.0/20", Netmask: "255.255.240.0", Network: "149.154.160.0", Comment: "CubeX-Telegram-DC"},
	{CIDR: "91.108.4.0/22", Netmask: "255.255.252.0", Network: "91.108.4.0", Comment: "CubeX-Telegram-DC"},
	{CIDR: "91.108.8.0/22", Netmask: "255.255.252.0", Network: "91.108.8.0", Comment: "CubeX-Telegram-DC"},
	{CIDR: "91.108.12.0/22", Netmask: "255.255.252.0", Network: "91.108.12.0", Comment: "CubeX-Telegram-DC"},
	{CIDR: "91.108.16.0/22", Netmask: "255.255.252.0", Network: "91.108.16.0", Comment: "CubeX-Telegram-DC"},
	{CIDR: "91.108.20.0/22", Netmask: "255.255.252.0", Network: "91.108.20.0", Comment: "CubeX-Telegram-DC"},
	{CIDR: "91.108.56.0/22", Netmask: "255.255.252.0", Network: "91.108.56.0", Comment: "CubeX-Telegram-DC"},
	{CIDR: "91.105.192.0/23", Netmask: "255.255.254.0", Network: "91.105.192.0", Comment: "CubeX-Telegram-Media"},
	{CIDR: "185.76.151.0/24", Netmask: "255.255.255.0", Network: "185.76.151.0", Comment: "CubeX-Telegram-Auth"},
}

// BuildRouterOSFullRoutes 生成 RouterOS v7 含 Fake-IP + Telegram 纯 IP 补盲的一键路由命令
func BuildRouterOSFullRoutes(cubexIP string) string {
	cubexIP = strings.TrimSpace(cubexIP)
	if cubexIP == "" {
		topo := netenv.DetectHostTopology("", "")
		if topo.LocalIP != "" {
			cubexIP = topo.LocalIP
		} else {
			cubexIP = netenv.PlaceholderLocalIP
		}
	}
	var b strings.Builder
	b.WriteString(`/ip route remove [find where comment~"^CubeX-"]` + "\n")
	for _, item := range EssentialNonDNSCIDRs {
		b.WriteString(fmt.Sprintf(`/ip route add dst-address=%s gateway=%s distance=1 comment=%q`+"\n",
			item.CIDR, cubexIP, item.Comment))
	}
	return strings.TrimSpace(b.String())
}

// BuildOpenWrtFullRoutes 生成 OpenWrt UCI 批量静态路由注入命令
func BuildOpenWrtFullRoutes(cubexIP string) string {
	cubexIP = strings.TrimSpace(cubexIP)
	if cubexIP == "" {
		topo := netenv.DetectHostTopology("", "")
		if topo.LocalIP != "" {
			cubexIP = topo.LocalIP
		} else {
			cubexIP = netenv.PlaceholderLocalIP
		}
	}
	var b strings.Builder
	for i, item := range EssentialNonDNSCIDRs {
		sec := fmt.Sprintf("cubex_rt_%d", i)
		b.WriteString(fmt.Sprintf("uci -q delete network.%s\n", sec))
		b.WriteString(fmt.Sprintf("uci set network.%s='route'\n", sec))
		b.WriteString(fmt.Sprintf("uci set network.%s.interface='lan'\n", sec))
		b.WriteString(fmt.Sprintf("uci set network.%s.target='%s'\n", sec, item.Network))
		b.WriteString(fmt.Sprintf("uci set network.%s.netmask='%s'\n", sec, item.Netmask))
		b.WriteString(fmt.Sprintf("uci set network.%s.gateway='%s'\n", sec, cubexIP))
	}
	b.WriteString("uci commit network && /etc/init.d/network reload")
	return b.String()
}
