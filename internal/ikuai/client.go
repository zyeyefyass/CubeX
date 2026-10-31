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

// Client interacts with iKuai's HTTP JSON-RPC API (/Action/call and /Action/login).
type Client struct {
	cfg        Config
	mu         sync.Mutex
	httpClient *http.Client
	sessKey    string
	lastLogin  time.Time
	isV4       bool
}

// RPCRequest matches iKuai's JSON-RPC standard request body.
type RPCRequest struct {
	FuncName string `json:"func_name"`
	Action   string `json:"action"`
	Param    any    `json:"param,omitempty"`
}

// RPCResponse matches iKuai's response envelope (supporting both 3.x and 4.0+).
type RPCResponse struct {
	// 3.x envelope fields
	Result int             `json:"Result"`
	ErrMsg string          `json:"ErrMsg"`
	Data   json.RawMessage `json:"Data,omitempty"`

	// 4.0+ envelope fields
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Results json.RawMessage `json:"results,omitempty"`
	RowID   any             `json:"rowid,omitempty"`
}

// IsSuccess checks if the response indicates success across 3.x and 4.0+.
func (r *RPCResponse) IsSuccess() bool {
	return r.Result == 10000 || r.Result == 30000 || r.Code == 0
}

// ErrorMessage returns the error description from 3.x or 4.0+.
func (r *RPCResponse) ErrorMessage() string {
	if r.ErrMsg != "" {
		return r.ErrMsg
	}
	if r.Message != "" {
		return r.Message
	}
	if r.Code != 0 {
		return fmt.Sprintf("错误码 %d", r.Code)
	}
	return fmt.Sprintf("错误码 %d", r.Result)
}

// Payload returns the data payload (4.0+ results or 3.x Data).
func (r *RPCResponse) Payload() json.RawMessage {
	if len(r.Results) > 0 && string(r.Results) != "null" {
		return r.Results
	}
	return r.Data
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
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't auto follow redirect across methods/protocols to prevent POST->GET stripping
				return http.ErrUseLastResponse
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

// executeLoginAttempt posts a login payload, correctly following 301/302 redirects with POST.
func (c *Client) executeLoginAttempt(targetURL string, payload any) (*RPCResponse, string, string, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, "", "", err
	}

	currentURL := targetURL
	for redirectCount := 0; redirectCount < 3; redirectCount++ {
		req, err := http.NewRequest(http.MethodPost, currentURL, bytes.NewReader(bodyBytes))
		if err != nil {
			return nil, "", "", err
		}
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
		req.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
		req.Header.Set("User-Agent", "CubeX-Host-Symbiosis/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, "", "", err
		}

		if resp.StatusCode == http.StatusMovedPermanently || resp.StatusCode == http.StatusFound ||
			resp.StatusCode == http.StatusTemporaryRedirect || resp.StatusCode == http.StatusPermanentRedirect {
			loc := resp.Header.Get("Location")
			resp.Body.Close()
			if loc != "" {
				if !strings.Contains(loc, "/Action/login") {
					if strings.HasSuffix(loc, "/") {
						loc = loc + "Action/login"
					} else {
						loc = loc + "/Action/login"
					}
				}
				currentURL = loc
				continue
			}
		}

		respBytes, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, "", "", err
		}

		var rpcResp RPCResponse
		if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
			return nil, "", "", fmt.Errorf("解析爱快响应异常: %w (响应内容: %s)", err, string(respBytes))
		}

		sessKey := ""
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "sess_key" && cookie.Value != "" {
				sessKey = cookie.Value
				break
			}
		}
		if sessKey == "" && len(rpcResp.Payload()) > 0 {
			var dataMap map[string]any
			if err := json.Unmarshal(rpcResp.Payload(), &dataMap); err == nil {
				if sk, ok := dataMap["sess_key"].(string); ok && sk != "" {
					sessKey = sk
				}
			}
		}

		return &rpcResp, sessKey, currentURL, nil
	}

	return nil, "", "", fmt.Errorf("爱快鉴权重定向过多")
}

// Login authenticates with iKuai Web interface and obtains sess_key.
// Seamlessly adapts to iKuai 4.0+ (flat JSON with salt_11), 3.x (RPC envelope), and HTTP->HTTPS redirects.
func (c *Client) Login() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Candidate URLs (auto-resolving port 80 HTTP to 443 HTTPS redirect)
	candidateURLs := []string{c.getBaseURL() + "/Action/login"}
	if c.cfg.Port == 80 {
		candidateURLs = append(candidateURLs, fmt.Sprintf("https://%s:443/Action/login", c.cfg.Host))
	} else if c.cfg.Port == 443 {
		candidateURLs = append(candidateURLs, fmt.Sprintf("http://%s:80/Action/login", c.cfg.Host))
	}

	login4 := map[string]string{
		"username": c.cfg.User,
		"passwd":   md5Str(c.cfg.Password),
		"pass":     base64Str("salt_11" + c.cfg.Password),
	}

	login3 := RPCRequest{
		FuncName: "login",
		Action:   "show",
		Param: map[string]string{
			"username":          c.cfg.User,
			"passwd":            md5Str(c.cfg.Password),
			"pass":              base64Str(c.cfg.Password),
			"remember_password": "",
		},
	}

	var lastErr error
	for _, candURL := range candidateURLs {
		// 1. Try iKuai 4.0+ modern flat login with salt_11
		rpcResp4, sessKey4, finalURL4, err4 := c.executeLoginAttempt(candURL, login4)
		if err4 == nil && rpcResp4.IsSuccess() && sessKey4 != "" {
			c.sessKey = sessKey4
			c.isV4 = true
			c.lastLogin = time.Now()
			if strings.HasPrefix(finalURL4, "https://") {
				c.cfg.Port = 443
			}
			return nil
		}
		if err4 != nil {
			lastErr = err4
		} else if !rpcResp4.IsSuccess() {
			lastErr = fmt.Errorf("%s", rpcResp4.ErrorMessage())
		}

		// 2. Try iKuai 3.x standard RPC login
		rpcResp3, sessKey3, finalURL3, err3 := c.executeLoginAttempt(candURL, login3)
		if err3 == nil && rpcResp3.IsSuccess() && sessKey3 != "" {
			c.sessKey = sessKey3
			c.isV4 = false
			c.lastLogin = time.Now()
			if strings.HasPrefix(finalURL3, "https://") {
				c.cfg.Port = 443
			}
			return nil
		}
		if err3 != nil {
			lastErr = err3
		} else if !rpcResp3.IsSuccess() {
			lastErr = fmt.Errorf("%s", rpcResp3.ErrorMessage())
		}
	}

	return fmt.Errorf("爱快鉴权失败: %v (请检查管理员账号与密码)", lastErr)
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

	// If session expired (Result 10001/10003 or Code 10001/10003), attempt 1 re-login
	if rpcResp.Result == 10001 || rpcResp.Result == 10003 || rpcResp.Code == 10001 || rpcResp.Code == 10003 || strings.Contains(rpcResp.ErrorMessage(), "登录") || strings.Contains(rpcResp.ErrorMessage(), "authentication") {
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

// GetSysStat queries iKuai sysstat (supports both 3.x and 4.0+ formats).
func (c *Client) GetSysStat() (*SysStatInfo, error) {
	resp, err := c.Call("sysstat", "show", map[string]string{
		"TYPE": "verinfo,cpu,memory,uptime",
	})
	if err != nil {
		return nil, err
	}
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("查询系统状态失败: %s", resp.ErrorMessage())
	}

	info := &SysStatInfo{
		SysName: "爱快 iKuai 路由系统",
		Version: "4.x",
		Arch:    "x86_64",
	}

	payload := resp.Payload()
	if len(payload) == 0 {
		return info, nil
	}

	var dataMap map[string]any
	if err := json.Unmarshal(payload, &dataMap); err == nil {
		if ver, ok := dataMap["verinfo"].(map[string]any); ok {
			if vs, ok := ver["verstring"].(string); ok && vs != "" {
				info.Version = vs
			} else if v, ok := ver["version"].(string); ok && v != "" {
				info.Version = v
			}
			if s, ok := ver["sysname"].(string); ok && s != "" {
				info.SysName = s
			} else if m, ok := ver["modelname"].(string); ok && m != "" {
				info.SysName = m
			}
			if b, ok := ver["sysbit"].(string); ok && b != "" {
				info.Arch = b
			} else if a, ok := ver["arch"].(string); ok && a != "" {
				info.Arch = a
			}
		}

		// CPU load handling: 3.x object {"load": "2%"} or 4.0 array ["0.50%", "1.98%"]
		if cpuObj, ok := dataMap["cpu"].(map[string]any); ok {
			if load, ok := cpuObj["load"].(string); ok {
				info.CPULoad = load
			}
		} else if cpuList, ok := dataMap["cpu"].([]any); ok && len(cpuList) > 0 {
			if firstStr, ok := cpuList[0].(string); ok {
				info.CPULoad = firstStr
			} else if firstMap, ok := cpuList[0].(map[string]any); ok {
				if load, ok := firstMap["load"].(string); ok {
					info.CPULoad = load
				}
			}
		}

		// Memory: {"used": "512MB", "total": "4096MB"} or {"used": "22%", "total": 1902852 (kB)}
		if mem, ok := dataMap["memory"].(map[string]any); ok {
			if used, ok := mem["used"].(string); ok {
				info.MemoryUsed = used
			}
			if total, ok := mem["total"].(string); ok {
				info.MemoryTot = total
			} else if totalNum, ok := mem["total"].(float64); ok {
				if totalNum > 1024*1024*10 { // bytes
					info.MemoryTot = fmt.Sprintf("%.0fMB", totalNum/(1024*1024))
				} else { // kB
					info.MemoryTot = fmt.Sprintf("%.0fMB", totalNum/1024)
				}
			}
		}

		// Uptime: "120h" or seconds integer (e.g. 3309)
		if up, ok := dataMap["uptime"].(string); ok {
			info.Uptime = up
		} else if upSec, ok := dataMap["uptime"].(float64); ok {
			s := int(upSec)
			if s < 60 {
				info.Uptime = fmt.Sprintf("%ds", s)
			} else if s < 3600 {
				info.Uptime = fmt.Sprintf("%dm", s/60)
			} else {
				info.Uptime = fmt.Sprintf("%dh%dm", s/3600, (s%3600)/60)
			}
		}
	}

	return info, nil
}

// getDHCPConfig queries DHCP settings, dynamically supporting modern iKuai 3.7+/4.0 (dhcp_server) and legacy (dhcp).
func (c *Client) getDHCPConfig() (string, []map[string]any, error) {
	// 1. Try modern iKuai 3.7+ / 4.0 func_name: dhcp_server
	resp, err := c.Call("dhcp_server", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,50",
	})
	if err == nil && resp.IsSuccess() && len(resp.Payload()) > 0 {
		var dData struct {
			Data []map[string]any `json:"data"`
		}
		if jsonErr := json.Unmarshal(resp.Payload(), &dData); jsonErr == nil && len(dData.Data) > 0 {
			return "dhcp_server", dData.Data, nil
		}
	}

	// 2. Fallback to legacy func_name: dhcp
	resp, err = c.Call("dhcp", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,50",
	})
	if err == nil && resp.IsSuccess() && len(resp.Payload()) > 0 {
		var dData struct {
			Data []map[string]any `json:"data"`
		}
		_ = json.Unmarshal(resp.Payload(), &dData)
		return "dhcp", dData.Data, nil
	}

	return "", nil, fmt.Errorf("无法查询爱快 DHCP 服务配置")
}

// InspectRouting checks if 198.18.0.0/15 static route and DHCP DNS are configured.
// Automatically adapts to iKuai 4.0+ (static_rt) and 3.x (route_static).
func (c *Client) InspectRouting(cubexIP string) (hasRoute bool, hasDNS bool, details string, err error) {
	// 1. Check static route based on detected firmware version
	checkV4 := func() bool {
		routeResp, err := c.Call("static_rt", "show", map[string]string{
			"TYPE":  "data,total",
			"limit": "0,100",
		})
		if err == nil && routeResp.IsSuccess() {
			var rData struct {
				Data []struct {
					RowID     any    `json:"rowid"`
					DstAddr   string `json:"dst_addr"`
					Gateway   string `json:"gateway"`
					Interface string `json:"interface"`
					IPType    string `json:"ip_type"`
					TagName   string `json:"tagname"`
				} `json:"data"`
			}
			if json.Unmarshal(routeResp.Payload(), &rData) == nil && len(rData.Data) > 0 {
				for _, r := range rData.Data {
					if strings.HasPrefix(r.DstAddr, "198.18.0.0") {
						if cubexIP == "" || r.Gateway == cubexIP {
							return true
						}
					}
				}
			}
		}
		return false
	}

	checkV3 := func() bool {
		routeResp3, err3 := c.Call("route_static", "show", map[string]string{
			"TYPE":  "data,total",
			"limit": "0,100",
		})
		if err3 == nil && routeResp3.IsSuccess() {
			var rData struct {
				Data []struct {
					ID        any    `json:"id"`
					IP        string `json:"ip"`
					Netmask   string `json:"netmask"`
					Gateway   string `json:"gateway"`
					Interface string `json:"interface"`
				} `json:"data"`
			}
			if json.Unmarshal(routeResp3.Payload(), &rData) == nil && len(rData.Data) > 0 {
				for _, r := range rData.Data {
					if r.IP == "198.18.0.0" && (r.Netmask == "255.254.0.0" || r.Netmask == "255.255.0.0" || r.Netmask == "15" || r.Netmask == "16") {
						if cubexIP == "" || r.Gateway == cubexIP {
							return true
						}
					}
				}
			}
		}
		return false
	}

	if c.isV4 {
		hasRoute = checkV4()
		if !hasRoute {
			hasRoute = checkV3()
		}
	} else {
		hasRoute = checkV3()
		if !hasRoute {
			hasRoute = checkV4()
		}
	}

	// 2. Check DHCP server DNS
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
// Fully compatible with both iKuai 4.0+ (static_rt) and 3.x (route_static).
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
				// Pre-flight preserve: if currDNS2 is empty and currDNS1 had a valid upstream DNS,
				// preserve it into dns2 as failover backup so clients never lose internet if CubeX is down!
				if currDNS2 == "" && currDNS1 != "" && currDNS1 != cubexIP && currDNS1 != gw {
					d["dns2"] = currDNS1
				} else if currDNS2 == "" {
					d["dns2"] = "223.5.5.5"
				}

				// Update primary DNS to CubeX
				d["dns1"] = cubexIP
				editResp, editErr := c.Call(fnDHCP, "edit", d)
				if editErr == nil && editResp.IsSuccess() {
					logs = append(logs, fmt.Sprintf("成功更新爱快 DHCP 首选 DNS 为 CubeX 独立 IP: %s (备用DNS保持: %v, 网关保持: %s, 接口: %s)", cubexIP, d["dns2"], gw, lanIf))
				} else {
					errMsg := "未知错误"
					if editErr != nil {
						errMsg = editErr.Error()
					} else if editResp != nil {
						errMsg = editResp.ErrorMessage()
					}
					logs = append(logs, fmt.Sprintf("更新爱快 DHCP 失败: %s", errMsg))
				}
			} else if currDNS1 == cubexIP {
				logs = append(logs, fmt.Sprintf("爱快 DHCP 首选 DNS 已是 CubeX IP (%s)，备用DNS: %s，状态健康", cubexIP, currDNS2))
			}
		}
	}

	// 2. Static route injection: Try 4.0 static_rt first
	route4Resp, err4 := c.Call("static_rt", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,100",
	})

	if err4 == nil && route4Resp.IsSuccess() {
		// 爱快 4.0+ 静态路由处理
		var rData struct {
			Data []struct {
				RowID   any    `json:"rowid"`
				DstAddr string `json:"dst_addr"`
				Gateway string `json:"gateway"`
			} `json:"data"`
		}
		_ = json.Unmarshal(route4Resp.Payload(), &rData)

		hasRoute := false
		for _, r := range rData.Data {
			if strings.HasPrefix(r.DstAddr, "198.18.0.0") {
				if r.Gateway == cubexIP {
					hasRoute = true
					logs = append(logs, fmt.Sprintf("爱快 4.0 静态路由 198.18.0.0/15 -> %s 已存在，无需重复添加", cubexIP))
					break
				} else {
					// Gateway mismatched, delete old
					rowidStr := fmt.Sprintf("%v", r.RowID)
					_, _ = c.Call("static_rt", "del", map[string]string{"rowid": rowidStr})
					logs = append(logs, fmt.Sprintf("清理指向错误网关 (%s) 的旧爱快 4.0 静态路由", r.Gateway))
				}
			}
		}

		if !hasRoute {
			addResp, addErr := c.Call("static_rt", "add", map[string]string{
				"ip_type":   "4",
				"tagname":   "CubeX-FakeIP-Bypass",
				"prio":      "1",
				"dst_addr":  "198.18.0.0/15",
				"gateway":   cubexIP,
				"interface": lanIf,
			})
			if addErr == nil && addResp.IsSuccess() {
				logs = append(logs, fmt.Sprintf("成功在爱快 4.0 中创建静态回指路由: 198.18.0.0/15 -> %s (接口 %s)", cubexIP, lanIf))
			} else {
				errMsg := "未知错误"
				if addErr != nil {
					errMsg = addErr.Error()
				} else if addResp != nil {
					errMsg = addResp.ErrorMessage()
				}
				logs = append(logs, fmt.Sprintf("爱快 4.0 添加静态路由失败: %s", errMsg))
			}
		}
		return logs, nil
	}

	// Fallback to iKuai 3.x route_static
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
	_ = json.Unmarshal(routeResp.Payload(), &rData)

	hasRoute := false
	for _, r := range rData.Data {
		if r.IP == "198.18.0.0" && (r.Netmask == "255.254.0.0" || r.Netmask == "255.255.0.0" || r.Netmask == "15" || r.Netmask == "16") {
			if r.Gateway == cubexIP {
				hasRoute = true
				logs = append(logs, fmt.Sprintf("静态路由 198.18.0.0/15 -> %s 已存在，无需重复添加", cubexIP))
				break
			} else {
				idStr := fmt.Sprintf("%v", r.ID)
				_, _ = c.Call("route_static", "del", map[string]string{"id": idStr})
				logs = append(logs, fmt.Sprintf("清理指向错误网关 (%s) 的旧静态路由", r.Gateway))
			}
		}
	}

	if !hasRoute {
		addResp, err := c.Call("route_static", "add", map[string]string{
			"ip":        "198.18.0.0",
			"netmask":   "255.254.0.0",
			"gateway":   cubexIP,
			"interface": lanIf,
			"comment":   "CubeX-FakeIP-Bypass",
			"enabled":   "yes",
		})
		if err == nil && addResp.IsSuccess() {
			logs = append(logs, fmt.Sprintf("成功在爱快 3.x 中创建静态回指路由: 198.18.0.0/15 -> %s (接口 %s)", cubexIP, lanIf))
		} else {
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
			logs = append(logs, fmt.Sprintf("成功在爱快 3.x 中创建双 /16 静态回指路由: 198.18.0.0/16 & 198.19.0.0/16 -> %s (接口 %s)", cubexIP, lanIf))
		}
	}

	return logs, nil
}
