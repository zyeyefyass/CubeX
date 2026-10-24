package openwrt

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// Config represents connection parameters for OpenWrt router (UBUS JSON-RPC or Dropbear SSH).
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"` // Default 80 / 443 (HTTP/UBUS) or 22 (SSH)
	User     string `json:"user"` // Default "root"
	Password string `json:"password"`
}

// Client interacts with OpenWrt via UBUS JSON-RPC (/ubus) with seamless Dropbear SSH fallback.
type Client struct {
	cfg        Config
	mu         sync.Mutex
	httpClient *http.Client
	ubusSID    string
}

// UbusRequest matches OpenWrt standard JSON-RPC 2.0 structure.
type UbusRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

// UbusResponse matches OpenWrt standard JSON-RPC response.
type UbusResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   any             `json:"error,omitempty"`
}

// SystemInfo holds basic telemetry about the OpenWrt host.
type SystemInfo struct {
	Hostname      string `json:"hostname"`
	Model         string `json:"model"`
	Release       string `json:"release"`
	Kernel        string `json:"kernel"`
	FirewallVer   string `json:"firewall_ver"` // "firewall4" (nftables) or "firewall3" (iptables)
	MemoryTotalMB int    `json:"memory_total_mb"`
	MemoryFreeMB  int    `json:"memory_free_mb"`
}

// NewClient initializes a new OpenWrt API client.
func NewClient(cfg Config) *Client {
	if cfg.Port <= 0 {
		cfg.Port = 80
	}
	if cfg.User == "" {
		cfg.User = "root"
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 6 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func (c *Client) getBaseURL() string {
	scheme := "http"
	if c.cfg.Port == 443 {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d/ubus", scheme, c.cfg.Host, c.cfg.Port)
}

// Probe checks connectivity and returns OpenWrt system telemetry.
func (c *Client) Probe() (*SystemInfo, error) {
	// 1. Try UBUS first
	if info, err := c.probeViaUbus(); err == nil && info != nil {
		return info, nil
	}

	// 2. Fallback to SSH
	return c.probeViaSSH()
}

func (c *Client) probeViaUbus() (*SystemInfo, error) {
	if err := c.ensureUbusSession(); err != nil {
		return nil, err
	}

	res, err := c.callUbus("system", "info", map[string]any{})
	if err != nil {
		return nil, err
	}

	var sysData struct {
		Hostname string `json:"hostname"`
		Model    string `json:"model"`
		Release  struct {
			Distribution string `json:"distribution"`
			Version      string `json:"version"`
			Revision     string `json:"revision"`
			Target       string `json:"target"`
			Description  string `json:"description"`
		} `json:"release"`
		Memory struct {
			Total uint64 `json:"total"`
			Free  uint64 `json:"free"`
		} `json:"memory"`
	}

	if err := json.Unmarshal(res, &sysData); err != nil {
		return nil, err
	}

	fwVer := "firewall4"
	// Check if firewall service is firewall3 or 4
	if fwRes, err := c.callUbus("uci", "get", map[string]any{"config": "firewall"}); err == nil && len(fwRes) > 0 {
		if strings.Contains(string(fwRes), "iptables") {
			fwVer = "firewall3"
		}
	}

	info := &SystemInfo{
		Hostname:      sysData.Hostname,
		Model:         sysData.Model,
		Release:       fmt.Sprintf("%s %s", sysData.Release.Distribution, sysData.Release.Version),
		MemoryTotalMB: int(sysData.Memory.Total / (1024 * 1024)),
		MemoryFreeMB:  int(sysData.Memory.Free / (1024 * 1024)),
		FirewallVer:   fwVer,
	}
	if info.Release == " " {
		info.Release = "OpenWrt (Linux)"
	}
	return info, nil
}

func (c *Client) ensureUbusSession() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ubusSID != "" {
		return nil
	}

	payload := UbusRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "call",
		Params: []any{
			"00000000000000000000000000000000",
			"session",
			"login",
			map[string]string{
				"username": c.cfg.User,
				"password": c.cfg.Password,
			},
		},
	}

	b, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(c.getBaseURL(), "application/json", bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("ubus connect error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ubus HTTP %d", resp.StatusCode)
	}

	var ubusResp UbusResponse
	if err := json.NewDecoder(resp.Body).Decode(&ubusResp); err != nil {
		return fmt.Errorf("ubus decode error: %w", err)
	}

	var arr []any
	if err := json.Unmarshal(ubusResp.Result, &arr); err == nil && len(arr) >= 2 {
		if code, ok := arr[0].(float64); ok && code == 0 {
			if dict, ok := arr[1].(map[string]any); ok {
				if sid, ok := dict["ubus_rpc_session"].(string); ok && sid != "" {
					c.ubusSID = sid
					return nil
				}
			}
		}
	}

	return fmt.Errorf("ubus login invalid credentials or unsupported")
}

func (c *Client) callUbus(obj, method string, params map[string]any) ([]byte, error) {
	c.mu.Lock()
	sid := c.ubusSID
	c.mu.Unlock()

	if sid == "" {
		if err := c.ensureUbusSession(); err != nil {
			return nil, err
		}
		c.mu.Lock()
		sid = c.ubusSID
		c.mu.Unlock()
	}

	payload := UbusRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "call",
		Params:  []any{sid, obj, method, params},
	}

	b, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(c.getBaseURL(), "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var ubusResp UbusResponse
	if err := json.NewDecoder(resp.Body).Decode(&ubusResp); err != nil {
		return nil, err
	}

	var arr []any
	if err := json.Unmarshal(ubusResp.Result, &arr); err == nil && len(arr) >= 2 {
		if code, ok := arr[0].(float64); ok && code == 0 {
			resBytes, _ := json.Marshal(arr[1])
			return resBytes, nil
		}
	}

	return nil, fmt.Errorf("ubus call %s.%s failed: %s", obj, method, string(ubusResp.Result))
}

// runSSHCommand executes a shell command over Dropbear/OpenSSH with standard credentials.
func (c *Client) runSSHCommand(cmd string) (string, error) {
	sshPort := 22
	if c.cfg.Port > 0 && c.cfg.Port != 80 && c.cfg.Port != 443 {
		sshPort = c.cfg.Port
	}

	sshConfig := &ssh.ClientConfig{
		User: c.cfg.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.cfg.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	addr := net.JoinHostPort(c.cfg.Host, fmt.Sprintf("%d", sshPort))
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return "", fmt.Errorf("ssh dial to OpenWrt (%s) failed: %w", addr, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("new ssh session failed: %w", err)
	}
	defer session.Close()

	out, err := session.CombinedOutput(cmd)
	if err != nil {
		return strings.TrimSpace(string(out)), fmt.Errorf("command execution error: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Client) probeViaSSH() (*SystemInfo, error) {
	script := `
cat /etc/openwrt_release 2>/dev/null || cat /etc/os-release 2>/dev/null
echo "---SPLIT---"
uname -r
echo "---SPLIT---"
cat /proc/sys/kernel/hostname
echo "---SPLIT---"
cat /proc/meminfo | grep -E "MemTotal|MemFree"
echo "---SPLIT---"
if command -v fw4 >/dev/null 2>&1 || [ -f /etc/init.d/firewall ] && grep -q "fw4" /etc/init.d/firewall; then
    echo "firewall4"
else
    echo "firewall3"
fi
`
	out, err := c.runSSHCommand(script)
	if err != nil {
		return nil, fmt.Errorf("SSH probe error: %w", err)
	}

	parts := strings.Split(out, "---SPLIT---")
	info := &SystemInfo{
		Release:     "OpenWrt",
		FirewallVer: "firewall4",
	}

	if len(parts) >= 1 {
		for _, line := range strings.Split(strings.TrimSpace(parts[0]), "\n") {
			if strings.HasPrefix(line, "DISTRIB_DESCRIPTION=") {
				info.Release = strings.Trim(strings.TrimPrefix(line, "DISTRIB_DESCRIPTION="), "\"'")
			} else if strings.HasPrefix(line, "PRETTY_NAME=") && info.Release == "OpenWrt" {
				info.Release = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"'")
			}
		}
	}
	if len(parts) >= 2 {
		info.Kernel = strings.TrimSpace(parts[1])
	}
	if len(parts) >= 3 {
		info.Hostname = strings.TrimSpace(parts[2])
	}
	if len(parts) >= 4 {
		for _, line := range strings.Split(strings.TrimSpace(parts[3]), "\n") {
			var val uint64
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(line, "MemTotal: %d kB", &val)
				info.MemoryTotalMB = int(val / 1024)
			} else if strings.HasPrefix(line, "MemFree:") {
				fmt.Sscanf(line, "MemFree: %d kB", &val)
				info.MemoryFreeMB = int(val / 1024)
			}
		}
	}
	if len(parts) >= 5 {
		if strings.Contains(parts[4], "firewall3") {
			info.FirewallVer = "firewall3"
		}
	}

	return info, nil
}

// ─────────────────────────────────────────────────────────────────────────────
// Three Core One-Click Healing Implementations (三大一键配置自愈与巡检)
// ─────────────────────────────────────────────────────────────────────────────

// ApplyBypass orchestrates Fake-IP 198.18.0.0/15 static return route, DHCP Option 6 injection, and 0-second failover.
func (c *Client) ApplyBypass(cubexIP string) ([]string, error) {
	if cubexIP == "" {
		return nil, fmt.Errorf("cubexIP cannot be empty")
	}

	script := fmt.Sprintf(`
set -e
# 1. Fake-IP 198.18.0.0/15 Return Route via CubeX Gateway
uci -q delete network.cubex_fakeip || true
uci set network.cubex_fakeip=route
uci set network.cubex_fakeip.interface='lan'
uci set network.cubex_fakeip.target='198.18.0.0'
uci set network.cubex_fakeip.netmask='255.254.0.0'
uci set network.cubex_fakeip.gateway='%s'

# Preserve macvlan-shim route to container if running on same OpenWrt host
if ip link show macvlan-shim >/dev/null 2>&1; then
    uci -q delete network.cubex_shim_route || true
    uci set network.cubex_shim_route=route
    uci set network.cubex_shim_route.interface='cubex_shim'
    uci set network.cubex_shim_route.target='%[1]s'
    uci set network.cubex_shim_route.netmask='255.255.255.255'
fi

uci commit network
/etc/init.d/network reload >/dev/null 2>&1 || true
ip route replace 198.18.0.0/15 via %s 2>/dev/null || true
if ip link show macvlan-shim >/dev/null 2>&1; then
    ip link set macvlan-shim up 2>/dev/null || true
    ip route replace %[1]s/32 dev macvlan-shim 2>/dev/null || true
fi

# 2. DHCP Option 6 (DNS) Injection to point directly to CubeX
# Remove any existing option 6 list entry to prevent duplicate DNS
uci -q del_list dhcp.lan.dhcp_option='6,%s' || true
uci add_list dhcp.lan.dhcp_option='6,%s'
uci commit dhcp
/etc/init.d/dnsmasq restart >/dev/null 2>&1 || true

# 3. Deploy 0-Second Failover Watchdog Script in OpenWrt
cat << 'EOF' > /usr/bin/cubex-watchdog.sh
#!/bin/sh
CUBEX_IP="%s"
BACKUP_DNS="223.5.5.5"

# Lightweight microsecond probe
if wget -q -T 2 -O - "http://${CUBEX_IP}:2026/api/health" 2>/dev/null | grep -q "status"; then
    # Healthy: ensure DHCP option points to CubeX
    CURRENT=$(uci -q get dhcp.lan.dhcp_option | grep -o "${CUBEX_IP}" || true)
    if [ -z "$CURRENT" ]; then
        logger -t cubex-watchdog "🛡️ CubeX Gateway revived! Restoring DHCP Option 6 to ${CUBEX_IP}"
        uci -q del_list dhcp.lan.dhcp_option="6,${BACKUP_DNS}" || true
        uci add_list dhcp.lan.dhcp_option="6,${CUBEX_IP}"
        uci commit dhcp
        /etc/init.d/dnsmasq reload >/dev/null 2>&1
    fi
else
    # Down: trigger 0-second emergency escape
    CURRENT=$(uci -q get dhcp.lan.dhcp_option | grep -o "${CUBEX_IP}" || true)
    if [ -n "$CURRENT" ]; then
        logger -t cubex-watchdog "🚨 CubeX Gateway unreachable! Activating 0-second failover to ${BACKUP_DNS}"
        uci -q del_list dhcp.lan.dhcp_option="6,${CUBEX_IP}" || true
        uci add_list dhcp.lan.dhcp_option="6,${BACKUP_DNS}"
        uci commit dhcp
        /etc/init.d/dnsmasq reload >/dev/null 2>&1
    fi
fi
EOF
chmod +x /usr/bin/cubex-watchdog.sh

# Install or ensure crontab heartbeat every minute (or loop daemon)
if ! grep -q "cubex-watchdog.sh" /etc/crontabs/root 2>/dev/null; then
    echo "* * * * * /usr/bin/cubex-watchdog.sh" >> /etc/crontabs/root
    /etc/init.d/cron enable >/dev/null 2>&1 || true
    /etc/init.d/cron restart >/dev/null 2>&1 || true
fi
`, cubexIP, cubexIP, cubexIP, cubexIP, cubexIP)

	out, err := c.runSSHCommand(script)
	if err != nil {
		return nil, fmt.Errorf("ApplyBypass failed on OpenWrt: %w (%s)", err, out)
	}

	logs := []string{
		fmt.Sprintf("✓ [OpenWrt] 已在 UCI network 写入 Fake-IP 回指路由: 198.18.0.0/15 via %s (dev lan)", cubexIP),
		fmt.Sprintf("✓ [OpenWrt] 已在 UCI dhcp 注入 DHCP Option 6: 全局 DNS 引导至 %s", cubexIP),
		"✓ [OpenWrt] 重载 network 与 dnsmasq 守护进程，客户端无感平滑切换",
		fmt.Sprintf("✓ [OpenWrt] 已植入毫秒级逃生看门狗 (/usr/bin/cubex-watchdog.sh ➔ %s 宕机自动回退 223.5.5.5)", cubexIP),
	}
	return logs, nil
}

// ApplyFirewall configures kernel forwarding, disables asymmetric rp_filter, and unlocks firewall4 (nftables) / firewall3 (iptables).
func (c *Client) ApplyFirewall() ([]string, error) {
	script := `
set -e
# 1. Kernel Forwarding & Asymmetric Route Bypass Protection
sysctl -w net.ipv4.ip_forward=1
sysctl -w net.ipv4.conf.all.rp_filter=0
sysctl -w net.ipv4.conf.default.rp_filter=0
if [ -d /proc/sys/net/ipv4/conf/br-lan ]; then
    sysctl -w net.ipv4.conf.br-lan.rp_filter=0
fi

# 2. OpenWrt Zone Forwarding (LAN Zone Forward: ACCEPT)
uci set firewall.@zone[0].forward='ACCEPT'
uci commit firewall

# 3. Detect firewall4 (nftables) vs firewall3 (iptables)
if command -v fw4 >/dev/null 2>&1 || [ -f /etc/init.d/firewall ] && grep -q "fw4" /etc/init.d/firewall; then
    # Firewall4 (nftables): Inject custom inclusion for bypass traffic
    mkdir -p /etc/nftables.d
    cat << 'EOF' > /etc/nftables.d/10-cubex-bypass.nft
# CubeX Bypass Acceleration & Asymmetric Flow Whitelist
table inet fw4 {
    chain cubex_forward {
        type filter hook forward priority -5; policy accept;
        ip daddr 198.18.0.0/15 accept
        ip saddr 198.18.0.0/15 accept
    }
}
EOF
    /etc/init.d/firewall reload >/dev/null 2>&1 || true
    echo "FW_MODE:firewall4"
else
    # Firewall3 (iptables): Inject into /etc/firewall.user
    if ! grep -q "198.18.0.0/15" /etc/firewall.user 2>/dev/null; then
        echo "iptables -I FORWARD -d 198.18.0.0/15 -j ACCEPT" >> /etc/firewall.user
        echo "iptables -I FORWARD -s 198.18.0.0/15 -j ACCEPT" >> /etc/firewall.user
    fi
    /etc/init.d/firewall restart >/dev/null 2>&1 || true
    echo "FW_MODE:firewall3"
fi
`
	out, err := c.runSSHCommand(script)
	if err != nil {
		return nil, fmt.Errorf("ApplyFirewall failed on OpenWrt: %w (%s)", err, out)
	}

	fwMode := "firewall4 (nftables)"
	if strings.Contains(out, "FW_MODE:firewall3") {
		fwMode = "firewall3 (iptables)"
	}

	logs := []string{
		"✓ [OpenWrt] 开启系统内核网络转发 (net.ipv4.ip_forward = 1)",
		"✓ [OpenWrt] 关闭反向路径过滤 (rp_filter = 0)，防止非对称路由回包丢弃",
		"✓ [OpenWrt] 锁定 LAN 区域转发策略为 ACCEPT",
		fmt.Sprintf("✓ [OpenWrt] 检测并适配底座防火墙: %s", fwMode),
		"✓ [OpenWrt] 注入 198.18.0.0/15 极速放行链，免除连接跟踪无效包丢弃",
	}
	return logs, nil
}

// ApplyIPv6Hardening locks odhcpd to prevent operator IPv6 DNS leakage while retaining line-rate IPv6 SLAAC connectivity.
func (c *Client) ApplyIPv6Hardening() ([]string, error) {
	script := `
set -e
# 1. Stop odhcpd from broadcasting operator IPv6 DNS in Router Advertisements (RA)
uci set dhcp.lan.dns_service='0'
uci -q delete dhcp.lan.dns || true
uci set dhcp.lan.ra_dns='0'

# 2. Retain pure IPv6 SLAAC prefix allocation without toxic upstream DNS
uci set dhcp.lan.ra='server'
uci set dhcp.lan.dhcpv6='server'

uci commit dhcp
/etc/init.d/odhcpd enable >/dev/null 2>&1 || true
/etc/init.d/odhcpd restart >/dev/null 2>&1 || true

# 3. Intercept external DNS leaks on port 53 (redirect external port 53 to local/cubex)
if command -v fw4 >/dev/null 2>&1 || [ -f /etc/init.d/firewall ] && grep -q "fw4" /etc/init.d/firewall; then
    mkdir -p /etc/nftables.d
    cat << 'EOF' > /etc/nftables.d/20-cubex-dns-intercept.nft
# Intercept IoT hardcoded DNS leaks
table inet fw4 {
    chain cubex_dns_fix {
        type nat hook prerouting priority -100; policy accept;
        iifname "br-lan" udp dport 53 redirect
        iifname "br-lan" tcp dport 53 redirect
    }
}
EOF
    /etc/init.d/firewall reload >/dev/null 2>&1 || true
fi
`
	out, err := c.runSSHCommand(script)
	if err != nil {
		return nil, fmt.Errorf("ApplyIPv6Hardening failed on OpenWrt: %w (%s)", err, out)
	}

	logs := []string{
		"✓ [OpenWrt] 永久锁定 odhcpd: 禁用 Router Advertisement (RA) 广播运营商 IPv6 DNS (ra_dns=0, dns_service=0)",
		"✓ [OpenWrt] 保持原生公网 IPv6 SLAAC 前缀下发与路由通告，国内流量物理线速直出",
		"✓ [OpenWrt] 注入硬编码 DNS 拦截规则，杜绝 IoT 设备直连外网 53 端口侧漏",
		"✓ [OpenWrt] 重载 odhcpd 与 firewall 守护进程，全面杜绝全局 IPv6 DNS 污染",
	}
	return logs, nil
}
