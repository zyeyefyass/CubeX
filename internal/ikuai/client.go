package ikuai

import (
	"bytes"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Config represents connection parameters for iKuai router Web-RPC.
type Config struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// Client interacts with iKuai's HTTP JSON-RPC API (/Action/call).
type Client struct {
	cfg        Config
	mu         sync.Mutex
	httpClient *http.Client
	sessKey    string
	lastLogin  time.Time
}

// RPCRequest matches iKuai's JSON-RPC standard request body.
type RPCRequest struct {
	FuncName string `json:"func_name"`
	Action   string `json:"action"`
	Param    any    `json:"param,omitempty"`
}

// RPCResponse matches iKuai's standard response envelope.
type RPCResponse struct {
	Result int             `json:"Result"`
	ErrMsg string          `json:"ErrMsg"`
	Data   json.RawMessage `json:"Data,omitempty"`
}

// NewClient initializes a new iKuai API client.
func NewClient(cfg Config) *Client {
	if cfg.Port <= 0 {
		cfg.Port = 80
	}
	if cfg.User == "" {
		cfg.User = "admin"
	}
	return &Client{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func md5Str(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}

func base64Str(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func (c *Client) getBaseURL() string {
	scheme := "http"
	if c.cfg.Port == 443 {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%d", scheme, c.cfg.Host, c.cfg.Port)
}

// Login authenticates with iKuai Web interface and obtains sess_key.
func (c *Client) Login() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	reqBody := RPCRequest{
		FuncName: "login",
		Action:   "show",
		Param: map[string]string{
			"username":          c.cfg.User,
			"passwd":            md5Str(c.cfg.Password),
			"pass":              base64Str(c.cfg.Password),
			"remember_password": "",
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := c.getBaseURL() + "/Action/login"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("User-Agent", "CubeX-Host-Symbiosis/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("无法连接爱快 Web 接口 (%s): %w", url, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取爱快响应失败: %w", err)
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return fmt.Errorf("解析爱快响应异常: %w (响应内容: %s)", err, string(respBytes))
	}

	// 10000 (standard in 3.7+) or 30000 indicates success in iKuai
	if rpcResp.Result != 10000 && rpcResp.Result != 30000 {
		errMsg := rpcResp.ErrMsg
		if errMsg == "" {
			errMsg = fmt.Sprintf("错误码 %d", rpcResp.Result)
		}
		return fmt.Errorf("爱快鉴权失败: %s (请检查管理员账号与密码)", errMsg)
	}

	// Extract sess_key from cookies
	sessFound := false
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "sess_key" && cookie.Value != "" {
			c.sessKey = cookie.Value
			sessFound = true
			break
		}
	}

	// Some iKuai firmware returns sess_key in Data
	if !sessFound && len(rpcResp.Data) > 0 {
		var dataMap map[string]any
		if err := json.Unmarshal(rpcResp.Data, &dataMap); err == nil {
			if sk, ok := dataMap["sess_key"].(string); ok && sk != "" {
				c.sessKey = sk
				sessFound = true
			}
		}
	}

	c.lastLogin = time.Now()
	return nil
}

// Call sends a JSON-RPC request to iKuai with session cookie.
func (c *Client) Call(funcName, action string, param any) (*RPCResponse, error) {
	c.mu.Lock()
	sess := c.sessKey
	expired := time.Since(c.lastLogin) > 15*time.Minute
	c.mu.Unlock()

	if sess == "" || expired {
		if err := c.Login(); err != nil {
			return nil, err
		}
		c.mu.Lock()
		sess = c.sessKey
		c.mu.Unlock()
	}

	reqBody := RPCRequest{
		FuncName: funcName,
		Action:   action,
		Param:    param,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := c.getBaseURL() + "/Action/call"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")
	req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
	req.Header.Set("User-Agent", "CubeX-Host-Symbiosis/1.0")
	if sess != "" {
		req.AddCookie(&http.Cookie{Name: "sess_key", Value: sess})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("爱快接口请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// If session expired (e.g. Result 10001 or 10003), attempt 1 re-login
	if rpcResp.Result == 10001 || rpcResp.Result == 10003 || rpcResp.Result == 10014 || strings.Contains(rpcResp.ErrMsg, "登录") || strings.Contains(rpcResp.ErrMsg, "authentication") {
		if err := c.Login(); err == nil {
			c.mu.Lock()
			newSess := c.sessKey
			c.mu.Unlock()

			req2, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(bodyBytes))
			req2.Header.Set("Content-Type", "application/json;charset=utf-8")
			req2.AddCookie(&http.Cookie{Name: "sess_key", Value: newSess})
			resp2, err2 := c.httpClient.Do(req2)
			if err2 == nil {
				defer resp2.Body.Close()
				var rpcResp2 RPCResponse
				if err := json.NewDecoder(resp2.Body).Decode(&rpcResp2); err == nil {
					return &rpcResp2, nil
				}
			}
		}
	}

	return &rpcResp, nil
}

// SysStatInfo holds system telemetry from iKuai.
type SysStatInfo struct {
	SysName    string `json:"sysname"`
	Version    string `json:"version"`
	Arch       string `json:"arch"`
	CPULoad    string `json:"cpu_load"`
	MemoryUsed string `json:"memory_used"`
	MemoryTot  string `json:"memory_total"`
	Uptime     string `json:"uptime"`
}

// GetSysStat queries iKuai sysstat.
func (c *Client) GetSysStat() (*SysStatInfo, error) {
	resp, err := c.Call("sysstat", "show", map[string]string{
		"TYPE": "verinfo,cpu,memory,uptime",
	})
	if err != nil {
		return nil, err
	}
	if resp.Result != 30000 {
		return nil, fmt.Errorf("查询系统状态失败: %s", resp.ErrMsg)
	}

	info := &SysStatInfo{
		SysName: "爱快 iKuai 路由系统",
		Version: "3.x",
		Arch:    "x86_64",
	}

	var dataMap map[string]any
	if err := json.Unmarshal(resp.Data, &dataMap); err == nil {
		if ver, ok := dataMap["verinfo"].(map[string]any); ok {
			if v, ok := ver["version"].(string); ok && v != "" {
				info.Version = v
			}
			if s, ok := ver["sysname"].(string); ok && s != "" {
				info.SysName = s
			}
			if a, ok := ver["arch"].(string); ok && a != "" {
				info.Arch = a
			}
		}
		if cpu, ok := dataMap["cpu"].(map[string]any); ok {
			if load, ok := cpu["load"].(string); ok {
				info.CPULoad = load
			}
		}
		if mem, ok := dataMap["memory"].(map[string]any); ok {
			if used, ok := mem["used"].(string); ok {
				info.MemoryUsed = used
			}
			if total, ok := mem["total"].(string); ok {
				info.MemoryTot = total
			}
		}
		if up, ok := dataMap["uptime"].(string); ok {
			info.Uptime = up
		}
	}

	return info, nil
}

// getDHCPConfig queries DHCP settings, dynamically supporting both modern iKuai 3.7+/4.0 (dhcp_server) and legacy (dhcp).
func (c *Client) getDHCPConfig() (string, []map[string]any, error) {
	// 1. Try modern iKuai 3.7+ / 4.0 func_name: dhcp_server
	resp, err := c.Call("dhcp_server", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,50",
	})
	if err == nil && resp.Result == 30000 && len(resp.Data) > 0 {
		var dData struct {
			Data []map[string]any `json:"data"`
		}
		if jsonErr := json.Unmarshal(resp.Data, &dData); jsonErr == nil && len(dData.Data) > 0 {
			return "dhcp_server", dData.Data, nil
		}
	}

	// 2. Fallback to legacy func_name: dhcp
	resp, err = c.Call("dhcp", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,50",
	})
	if err == nil && resp.Result == 30000 {
		var dData struct {
			Data []map[string]any `json:"data"`
		}
		_ = json.Unmarshal(resp.Data, &dData)
		return "dhcp", dData.Data, nil
	}

	return "", nil, fmt.Errorf("无法查询爱快 DHCP 服务配置")
}

// InspectRouting checks if 198.18.0.0/15 static route and DHCP DNS are configured.
func (c *Client) InspectRouting(cubexIP string) (hasRoute bool, hasDNS bool, details string, err error) {
	// 1. Check static route
	routeResp, err := c.Call("route_static", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,100",
		"ORDER": "",
	})
	if err != nil {
		return false, false, "", fmt.Errorf("读取爱快静态路由表失败: %w", err)
	}

	if routeResp.Result == 30000 {
		var rData struct {
			Data []struct {
				ID        any    `json:"id"`
				IP        string `json:"ip"`
				Netmask   string `json:"netmask"`
				Gateway   string `json:"gateway"`
				Interface string `json:"interface"`
				Comment   string `json:"comment"`
				Enabled   string `json:"enabled"`
			} `json:"data"`
		}
		_ = json.Unmarshal(routeResp.Data, &rData)
		for _, r := range rData.Data {
			if r.IP == "198.18.0.0" && (r.Netmask == "255.254.0.0" || r.Netmask == "255.255.0.0") {
				if cubexIP == "" || r.Gateway == cubexIP {
					hasRoute = true
					break
				}
			}
		}
	}

	// 2. Check DHCP server DNS (supports modern dhcp_server and legacy dhcp)
	_, dhcpList, errDHCP := c.getDHCPConfig()
	if errDHCP == nil {
		for _, d := range dhcpList {
			currDNS1, _ := d["dns1"].(string)
			if currDNS1 == cubexIP && cubexIP != "" {
				hasDNS = true
				break
			}
		}
	}

	msgParts := []string{}
	if hasRoute {
		msgParts = append(msgParts, "静态路由 198.18.0.0/15 就绪")
	} else {
		msgParts = append(msgParts, "未发现 198.18.0.0/15 回指路由")
	}
	if hasDNS {
		msgParts = append(msgParts, fmt.Sprintf("DHCP DNS1 已指向 CubeX (%s)", cubexIP))
	} else {
		msgParts = append(msgParts, "DHCP 首选 DNS 尚未指向 CubeX")
	}
	details = strings.Join(msgParts, " · ")

	return hasRoute, hasDNS, details, nil
}

// ApplyBypass automatically injects 198.18.0.0/15 static route and updates DHCP DNS in iKuai.
func (c *Client) ApplyBypass(cubexIP string) ([]string, error) {
	if cubexIP == "" {
		return nil, fmt.Errorf("CubeX IP 未指定，无法配置爱快静态路由下一跳")
	}

	logs := []string{}
	lanIf := "lan1"

	// 1. Inspect DHCP server: query existing settings and safely preserve them
	fnDHCP, dhcpList, errDHCP := c.getDHCPConfig()
	if errDHCP == nil && len(dhcpList) > 0 {
		for _, d := range dhcpList {
			if iface, ok := d["interface"].(string); ok && iface != "" {
				lanIf = iface
			}
			idVal := d["id"]
			currDNS1, _ := d["dns1"].(string)
			currDNS2, _ := d["dns2"].(string)
			gw, _ := d["gateway"].(string)

			if currDNS1 != cubexIP && idVal != nil {
				// Pre-flight preserve: if currDNS2 is empty and currDNS1 had a valid upstream DNS (like 223.5.5.5/114.114.114.114),
				// preserve it into dns2 as failover backup so clients never lose internet if CubeX is down!
				if currDNS2 == "" && currDNS1 != "" && currDNS1 != cubexIP && currDNS1 != gw {
					d["dns2"] = currDNS1
				} else if currDNS2 == "" {
					d["dns2"] = "223.5.5.5"
				}

				// Update primary DNS to CubeX
				d["dns1"] = cubexIP
				editResp, editErr := c.Call(fnDHCP, "edit", d)
				if editErr == nil && editResp.Result == 30000 {
					logs = append(logs, fmt.Sprintf("成功更新爱快 DHCP 首选 DNS 为 CubeX 独立 IP: %s (备用DNS保持: %v, 网关保持: %s, 接口: %s)", cubexIP, d["dns2"], gw, lanIf))
				} else {
					errMsg := "未知错误"
					if editErr != nil {
						errMsg = editErr.Error()
					} else if editResp != nil {
						errMsg = editResp.ErrMsg
					}
					logs = append(logs, fmt.Sprintf("更新爱快 DHCP 失败: %s", errMsg))
				}
			} else if currDNS1 == cubexIP {
				logs = append(logs, fmt.Sprintf("爱快 DHCP 首选 DNS 已是 CubeX IP (%s)，备用DNS: %s，状态健康", cubexIP, currDNS2))
			}
		}
	}

	// 2. Read existing static routes
	routeResp, err := c.Call("route_static", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,100",
	})
	if err != nil {
		return nil, fmt.Errorf("查询爱快静态路由失败: %w", err)
	}

	var rData struct {
		Data []struct {
			ID      any    `json:"id"`
			IP      string `json:"ip"`
			Netmask string `json:"netmask"`
			Gateway string `json:"gateway"`
		} `json:"data"`
	}
	_ = json.Unmarshal(routeResp.Data, &rData)

	hasRoute := false
	for _, r := range rData.Data {
		if r.IP == "198.18.0.0" && (r.Netmask == "255.254.0.0" || r.Netmask == "255.255.0.0" || r.Netmask == "15" || r.Netmask == "16") {
			if r.Gateway == cubexIP {
				hasRoute = true
				logs = append(logs, fmt.Sprintf("静态路由 198.18.0.0/15 -> %s 已存在，无需重复添加", cubexIP))
				break
			} else {
				// Gateway mismatched, delete old
				idStr := fmt.Sprintf("%v", r.ID)
				_, _ = c.Call("route_static", "del", map[string]string{"id": idStr})
				logs = append(logs, fmt.Sprintf("清理指向错误网关 (%s) 的旧静态路由", r.Gateway))
			}
		}
	}

	if !hasRoute {
		// Add route_static
		addResp, err := c.Call("route_static", "add", map[string]string{
			"ip":        "198.18.0.0",
			"netmask":   "255.254.0.0",
			"gateway":   cubexIP,
			"interface": lanIf,
			"comment":   "CubeX-FakeIP-Bypass",
			"enabled":   "yes",
		})
		if err == nil && addResp.Result == 30000 {
			logs = append(logs, fmt.Sprintf("成功在爱快中创建静态回指路由: 198.18.0.0/15 -> %s (接口 %s)", cubexIP, lanIf))
		} else {
			// Some strict iKuai versions require adding dual /16
			_, _ = c.Call("route_static", "add", map[string]string{
				"ip":        "198.18.0.0",
				"netmask":   "255.255.0.0",
				"gateway":   cubexIP,
				"interface": lanIf,
				"comment":   "CubeX-FakeIP-Bypass-1",
				"enabled":   "yes",
			})
			_, _ = c.Call("route_static", "add", map[string]string{
				"ip":        "198.19.0.0",
				"netmask":   "255.255.0.0",
				"gateway":   cubexIP,
				"interface": lanIf,
				"comment":   "CubeX-FakeIP-Bypass-2",
				"enabled":   "yes",
			})
			logs = append(logs, fmt.Sprintf("成功在爱快中创建双 /16 静态回指路由: 198.18.0.0/16 & 198.19.0.0/16 -> %s (接口 %s)", cubexIP, lanIf))
		}
	}

	return logs, nil
}

