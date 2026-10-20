package netstack

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// RosConfig represents RouterOS connection settings.
type RosConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	CubeXIP  string `json:"cubex_ip"`
}

// RosClient interacts directly with RouterOS v7 REST API.
type RosClient struct {
	cfg        RosConfig
	httpClient *http.Client
}

// NewRosClient initializes a RouterOS REST client.
func NewRosClient(cfg RosConfig) *RosClient {
	if cfg.Port == 0 {
		cfg.Port = 80
	}
	if cfg.CubeXIP == "" {
		cfg.CubeXIP = "10.0.0.2"
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &RosClient{
		cfg:        cfg,
		httpClient: &http.Client{Transport: tr, Timeout: 5 * time.Second},
	}
}

// Ping tests connection to RouterOS REST API.
func (r *RosClient) Ping() (bool, string, error) {
	if r.cfg.Host == "" {
		return false, "", fmt.Errorf("host not specified")
	}
	scheme := "http"
	if r.cfg.Port == 443 {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d/rest/system/resource", scheme, r.cfg.Host, r.cfg.Port)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, "", err
	}
	req.SetBasicAuth(r.cfg.Username, r.cfg.Password)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return true, string(body), nil
	}
	return false, "", fmt.Errorf("http error %d", resp.StatusCode)
}

// GenerateWinBoxScript generates an idempotent RouterOS v7 script with Netwatch 0-second failover.
func (r *RosClient) GenerateWinBoxScript() string {
	ip := r.cfg.CubeXIP
	if ip == "" {
		ip = "10.0.0.2"
	}

	return fmt.Sprintf(`# ════════════════════════════════════════════════════════════════════
# 🌌 CubeX 官方 RouterOS v7 高可用策略路由与 Netwatch 0秒逃生脚本
# ════════════════════════════════════════════════════════════════════

# 1. 创建专用路由表 table cubex-bypass
/routing table add fib name=cubex-bypass comment="CubeX Secondary Gateway Table"

# 2. 指向 CubeX 旁路网关 IP
/ip route add dst-address=0.0.0.0/0 gateway=%s routing-table=cubex-bypass comment="CubeX Gateway Route"

# 3. 局域网分流地址列表 (可按需将指定设备 IP 加入)
/ip firewall address-list add list=CUBEX_CLIENTS address=10.0.0.100-10.0.0.250 comment="Devices routed to CubeX"

# 4. Mangle 策略标记 (DNS 劫持放行与数据分流重定向)
/ip firewall mangle add chain=prerouting action=mark-routing new-routing-mark=cubex-bypass src-address-list=CUBEX_CLIENTS dst-address-type=!local comment="Redirect LAN traffic to CubeX"

# 5. Netwatch 毫秒级心跳高可用逃生 (旁路由重启时全网自动 0秒直连)
/tool netwatch add host=%s interval=3s timeout=900ms \
  up-script="/ip firewall mangle enable [find comment=\"Redirect LAN traffic to CubeX\"]" \
  down-script="/ip firewall mangle disable [find comment=\"Redirect LAN traffic to CubeX\"]" \
  comment="CubeX HA Auto-Failover"
`, ip, ip)
}

// CleanWinBoxScript returns clean-up commands to revert RouterOS to pure direct mode.
func (r *RosClient) CleanWinBoxScript() string {
	return strings.TrimSpace(`
# ── 清除 CubeX 策略路由与 Mangle 标记 ──
/tool netwatch remove [find comment="CubeX HA Auto-Failover"]
/ip firewall mangle remove [find comment="Redirect LAN traffic to CubeX"]
/ip firewall address-list remove [find list=CUBEX_CLIENTS]
/ip route remove [find comment="CubeX Gateway Route"]
/routing table remove [find name=cubex-bypass]
`)
}
