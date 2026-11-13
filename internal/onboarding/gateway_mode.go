package onboarding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// GatewayTopologyMode 定义用户可理解的两大部署模式
type GatewayTopologyMode string

const (
	// ModeSingleNICBypass 单网口旁路由模式（适合爱快/小米/TP-Link/OpenWrt 主路由 + Docker/LXC 旁路部署）
	ModeSingleNICBypass GatewayTopologyMode = "bypass_single_nic"
	// ModeSymbioticRoute 宿主共生分流模式（适合 RouterOS v7 / 软路由主路由通过 Fake-IP 静态路由精准分流）
	ModeSymbioticRoute GatewayTopologyMode = "symbiotic_fakeip"
)

// GatewayModeSpec 定义主旁模式配置参数
type GatewayModeSpec struct {
	Mode              GatewayTopologyMode `json:"mode"`
	PrimaryInterface  string              `json:"primary_interface"`   // 自动探测，如 "eth0"
	CubeXLocalIP      string              `json:"cubex_local_ip"`      // 自动探测，如 "192.168.1.7"
	UpstreamGatewayIP string              `json:"upstream_gateway_ip"` // 主路由 IP，如 "192.168.1.1"
	UpstreamDNS       string              `json:"upstream_dns"`        // 默认 "223.5.5.5"
	EnableMasquerade  bool                `json:"enable_masquerade"`   // 旁路由防不对称路由丢包开关（默认开启）
}

// GatewayModeStatus 返回给前端向导卡片的完整诊断与操作指引
type GatewayModeStatus struct {
	CurrentSpec      GatewayModeSpec `json:"current_spec"`
	IPForwardEnabled bool            `json:"ip_forward_enabled"`
	RedirectsMuted   bool            `json:"redirects_muted"`
	MasqueradeActive bool            `json:"masquerade_active"`
	Healthy          bool            `json:"healthy"`
	WarningMessage   string          `json:"warning_message,omitempty"`
	// PlainUserSteps 针对当前模式生成的 3 步大白话主路由设置指引（无任何黑话）
	PlainUserSteps   []string        `json:"plain_user_steps"`
}

// GatewayModeController 管控主旁双模探测与内核网络栈一键自愈
type GatewayModeController struct {
	mu        sync.RWMutex
	current   GatewayModeSpec
	stateFile string
}

func NewGatewayModeController() *GatewayModeController {
	iface, localIP, gwIP := autoDetectNetworkTopology()
	return &GatewayModeController{
		current: GatewayModeSpec{
			Mode:              ModeSingleNICBypass,
			PrimaryInterface:  iface,
			CubeXLocalIP:      localIP,
			UpstreamGatewayIP: gwIP,
			UpstreamDNS:       "223.5.5.5",
			EnableMasquerade:  true,
		},
	}
}

// BindPersistenceFile 绑定主旁模式配置文件路径（如 /var/lib/cubex/gateway_mode.json）
func (c *GatewayModeController) BindPersistenceFile(stateFilePath string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stateFile = stateFilePath
}

// saveSpecToDisk 将当前模式原子持久化落盘
func (c *GatewayModeController) saveSpecToDisk(spec GatewayModeSpec) error {
	c.mu.RLock()
	filePath := c.stateFile
	c.mu.RUnlock()

	if filePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return err
	}
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, filePath)
}

// RestoreOnBoot 在 cubexd 守护进程启动时调用：
// 若存在历史保存的主旁模式配置，立即重放 sysctl 与 nftables MASQUERADE 规则，防止容器重启后断网
func (c *GatewayModeController) RestoreOnBoot(ctx context.Context) error {
	c.mu.RLock()
	path := c.stateFile
	c.mu.RUnlock()

	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // 首次启动尚无持久化文件，跳过
		}
		return err
	}

	var saved GatewayModeSpec
	if err := json.Unmarshal(raw, &saved); err != nil {
		return err
	}

	_, err = c.ApplyMode(ctx, saved)
	return err
}

// ApplyMode 一键应用用户选择的网络模式（含防自环死锁校验 + sysctl 调优 + nftables 伪装自愈）
func (c *GatewayModeController) ApplyMode(ctx context.Context, spec GatewayModeSpec) (*GatewayModeStatus, error) {
	if spec.PrimaryInterface == "" || spec.CubeXLocalIP == "" || spec.UpstreamGatewayIP == "" {
		iface, localIP, gwIP := autoDetectNetworkTopology()
		if spec.PrimaryInterface == "" {
			spec.PrimaryInterface = iface
		}
		if spec.CubeXLocalIP == "" {
			spec.CubeXLocalIP = localIP
		}
		if spec.UpstreamGatewayIP == "" {
			spec.UpstreamGatewayIP = gwIP
		}
	}
	if spec.UpstreamDNS == "" {
		spec.UpstreamDNS = "223.5.5.5"
	}

	// [铁律校验] 严防新手将上级主路由网关填成 CubeX 自己的 IP 导致全网自环死锁
	if strings.TrimSpace(spec.CubeXLocalIP) != "" &&
		strings.TrimSpace(spec.CubeXLocalIP) == strings.TrimSpace(spec.UpstreamGatewayIP) {
		return nil, errors.New("上级主路由 IP 不能与 CubeX 本机 IP 相同，否则会导致网络自环断网")
	}

	if spec.UpstreamGatewayIP != "" && net.ParseIP(spec.UpstreamGatewayIP) == nil {
		return nil, fmt.Errorf("无效的上级主路由 IP 地址: %s", spec.UpstreamGatewayIP)
	}

	// Step 1: 无论哪种模式，作为网关必须开启 IPv4 转发并关闭 ICMP 重定向干扰
	_ = writeSysctl("/proc/sys/net/ipv4/ip_forward", "1")
	_ = writeSysctl("/proc/sys/net/ipv4/conf/all/send_redirects", "0")
	_ = writeSysctl("/proc/sys/net/ipv4/conf/default/send_redirects", "0")
	if spec.PrimaryInterface != "" {
		_ = writeSysctl(fmt.Sprintf("/proc/sys/net/ipv4/conf/%s/send_redirects", spec.PrimaryInterface), "0")
	}

	// Step 2: 根据模式原子配置或卸载旁路由 SNAT MASQUERADE 伪装表
	if spec.Mode == ModeSingleNICBypass && spec.EnableMasquerade {
		if err := applyBypassMasqueradeNft(ctx, spec.PrimaryInterface); err != nil {
			// 容器无 nft 权限或非 Linux 时记录告警但不崩溃
			c.mu.Lock()
			c.current = spec
			c.mu.Unlock()
			st := c.InspectStatus(ctx)
			st.WarningMessage = "已启用内核转发，但容器缺少自动配置 NAT 伪装权限，建议在主路由关闭『无效数据包丢弃』或给容器添加 NET_ADMIN 权限"
			return st, nil
		}
	} else {
		_ = removeBypassMasqueradeNft(ctx)
	}

	c.mu.Lock()
	c.current = spec
	c.mu.Unlock()

	_ = c.saveSpecToDisk(spec)

	return c.InspectStatus(ctx), nil
}

// InspectStatus 获取当前网络模式健康度与大白话操作步骤
func (c *GatewayModeController) InspectStatus(ctx context.Context) *GatewayModeStatus {
	c.mu.RLock()
	spec := c.current
	c.mu.RUnlock()

	ipFwd := readSysctl("/proc/sys/net/ipv4/ip_forward") == "1"
	redirMuted := readSysctl("/proc/sys/net/ipv4/conf/all/send_redirects") == "0"
	masqActive := checkMasqueradeTableExists(ctx)

	// 在非 Linux 环境开发测试时保证健康判断不产生假阳性报错
	healthy := (runtime.GOOS != "linux") || (ipFwd && (spec.Mode != ModeSingleNICBypass || !spec.EnableMasquerade || masqActive))

	return &GatewayModeStatus{
		CurrentSpec:      spec,
		IPForwardEnabled: ipFwd,
		RedirectsMuted:   redirMuted,
		MasqueradeActive: masqActive,
		Healthy:          healthy,
		PlainUserSteps:   buildPlainChineseSteps(spec),
	}
}

// buildPlainChineseSteps 生成零黑话、小白一眼看懂的 3 步设置指引
func buildPlainChineseSteps(spec GatewayModeSpec) []string {
	localIP := spec.CubeXLocalIP
	if localIP == "" {
		localIP = "<CubeX本机IP>"
	}
	gwIP := spec.UpstreamGatewayIP
	if gwIP == "" {
		gwIP = "<主路由IP>"
	}

	if spec.Mode == ModeSingleNICBypass {
		return []string{
			fmt.Sprintf("1. 【单机测试（最安全）】：在手机或电脑 Wi-Fi 设置中，将 IP 改为静态，『网关/路由器』和『DNS』都填 %s，即可单机享受加速（不影响家人网络）。", localIP),
			fmt.Sprintf("2. 【全屋自动加速】：登录你的主路由后台（%s），进入『DHCP 服务器 / 局域网设置』，将『默认网关』和『首选 DNS』改为 %s 并保存。", gwIP, localIP),
			"3. 【防断流保护已自动开启】：CubeX 已在后台自动开启 IP 转发与旁路 NAT 伪装，无需你在主路由手动编写任何防火墙规则。",
		}
	}

	return []string{
		fmt.Sprintf("1. 【保持主路由网关不变】：家里所有手机、电脑的默认网关仍然保持为主路由（%s），哪怕拔掉 CubeX 电源也绝不断网。", gwIP),
		fmt.Sprintf("2. 【主路由添加一条静态路由】：在主路由（RouterOS / OpenWrt / 爱快）添加静态路由：目标网段填 198.18.0.0/15（子网掩码 255.254.0.0），下一跳网关填 %s。", localIP),
		fmt.Sprintf("3. 【主路由修改上游 DNS】：将主路由的『上游 DNS 服务器』设为 %s。国内网站直接走主路由满速直出，只有境外流量才会进入 CubeX。", localIP),
	}
}

func applyBypassMasqueradeNft(ctx context.Context, outIface string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if _, err := exec.LookPath("nft"); err != nil {
		return err
	}
	ifaceRule := "masquerade"
	if outIface != "" {
		ifaceRule = fmt.Sprintf("oifname %q masquerade", outIface)
	}
	script := fmt.Sprintf(`table inet cubex_bypass_nat
delete table inet cubex_bypass_nat

table inet cubex_bypass_nat {
    chain postrouting {
        type nat hook postrouting priority srcnat; policy accept;
        %s
    }
}
`, ifaceRule)

	cmd := exec.CommandContext(ctx, "nft", "-f", "-")
	cmd.Stdin = bytes.NewBufferString(script)
	return cmd.Run()
}

func removeBypassMasqueradeNft(ctx context.Context) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	if _, err := exec.LookPath("nft"); err != nil {
		return nil
	}
	script := "table inet cubex_bypass_nat\ndelete table inet cubex_bypass_nat\n"
	cmd := exec.CommandContext(ctx, "nft", "-f", "-")
	cmd.Stdin = bytes.NewBufferString(script)
	_ = cmd.Run()
	return nil
}

func checkMasqueradeTableExists(ctx context.Context) bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if _, err := exec.LookPath("nft"); err != nil {
		return false
	}
	cmd := exec.CommandContext(ctx, "nft", "list", "table", "inet", "cubex_bypass_nat")
	return cmd.Run() == nil
}

func writeSysctl(path, val string) error {
	if _, err := os.Stat("/proc/sys"); err != nil {
		return nil
	}
	return os.WriteFile(path, []byte(val+"\n"), 0644)
}

func readSysctl(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// autoDetectNetworkTopology 自动从 Linux 路由表探测主网卡、本机 IP 与上级默认网关
func autoDetectNetworkTopology() (iface, localIP, gatewayIP string) {
	if runtime.GOOS == "linux" {
		out, err := exec.Command("ip", "-4", "route", "show", "default").Output()
		if err == nil {
			// 格式示例: default via 192.168.1.1 dev eth0 proto static src 192.168.1.7
			fields := strings.Fields(string(out))
			for i := 0; i < len(fields)-1; i++ {
				switch fields[i] {
				case "via":
					gatewayIP = fields[i+1]
				case "dev":
					iface = fields[i+1]
				case "src":
					localIP = fields[i+1]
				}
			}
		}
	}

	if localIP == "" {
		if ifaces, err := net.Interfaces(); err == nil {
			for _, i := range ifaces {
				if (i.Flags&net.FlagUp) == 0 || (i.Flags&net.FlagLoopback) != 0 {
					continue
				}
				addrs, err := i.Addrs()
				if err != nil {
					continue
				}
				for _, a := range addrs {
					if ipNet, ok := a.(*net.IPNet); ok && ipNet.IP.To4() != nil && !ipNet.IP.IsLoopback() {
						localIP = ipNet.IP.String()
						if iface == "" {
							iface = i.Name
						}
						break
					}
				}
				if localIP != "" {
					break
				}
			}
		}
	}

	return iface, localIP, gatewayIP
}
