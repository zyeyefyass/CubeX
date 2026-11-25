package ikuai

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
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

	// 🛡️ 师父防雷补丁 4: 熔断器与失败计数 (防止高并发雪崩与挂起)
	failCount        int
	circuitOpenUntil time.Time
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

func (c *Client) getBaseURLLocked() string {
	scheme := "http"
	if c.cfg.Port == 443 || c.cfg.Port == 8443 {
		scheme = "https"
	}
	port := c.cfg.Port
	if port <= 0 {
		port = 80
	}
	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(c.cfg.Host, strconv.Itoa(port)))
}

func (c *Client) getBaseURL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.getBaseURLLocked()
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

func (c *Client) recordSuccess() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failCount = 0
}

func (c *Client) recordFailure() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failCount++
	if c.failCount >= 3 {
		c.circuitOpenUntil = time.Now().Add(30 * time.Second)
	}
}

// Login authenticates with iKuai Web interface and obtains sess_key.
// Seamlessly adapts to iKuai 4.0+ (flat JSON with salt_11), 3.x (RPC envelope), and HTTP->HTTPS redirects.
func (c *Client) Login() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loginLocked()
}

func (c *Client) loginLocked() error {
	// Candidate URLs (auto-resolving port 80 HTTP to 443 HTTPS redirect)
	candidateURLs := []string{c.getBaseURLLocked() + "/Action/login"}
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
			c.failCount = 0
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
			c.failCount = 0
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

// 🛡️ 师父防雷补丁 4: 双检锁 (Double-Checked Locking) 保证并发下只有 1 个协程刷新 Token，杜绝并发重登雪崩
func (c *Client) ensureValidSession() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.sessKey != "" && time.Since(c.lastLogin) < 15*time.Minute {
		return c.sessKey, nil
	}

	if err := c.loginLocked(); err != nil {
		return "", err
	}
	return c.sessKey, nil
}

// 🛡️ 师父防雷补丁 4: 会话失效时的重登双检，如果已被同行协程刷新则复用新 Session
func (c *Client) forceRelogin(failedSess string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	//同行协程若已换取了更新的 Token，直接复用
	if c.sessKey != "" && c.sessKey != failedSess && time.Since(c.lastLogin) < 15*time.Minute {
		return c.sessKey, nil
	}

	if err := c.loginLocked(); err != nil {
		return "", err
	}
	return c.sessKey, nil
}

// Call sends a JSON-RPC request to iKuai with session cookie.
func (c *Client) Call(funcName, action string, param any) (*RPCResponse, error) {
	// 1. 熔断保护检查 (Circuit Breaker)
	c.mu.Lock()
	if time.Now().Before(c.circuitOpenUntil) {
		rem := time.Until(c.circuitOpenUntil).Round(time.Second)
		c.mu.Unlock()
		return nil, fmt.Errorf("爱快管理中枢服务熔断保护中 (将在 %v 后重试)", rem)
	}
	c.mu.Unlock()

	// 2. 双检锁获取有效 Session
	sess, err := c.ensureValidSession()
	if err != nil {
		c.recordFailure()
		return nil, err
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
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
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
		c.recordFailure()
		return nil, fmt.Errorf("爱快接口请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.recordFailure()
		return nil, err
	}

	var rpcResp RPCResponse
	if err := json.Unmarshal(respBytes, &rpcResp); err != nil {
		c.recordFailure()
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 3. 🛡️ 师父防雷补丁 4: 会话失效自动单飞重登重试 (HTTP 401/403 或 10001/10003)
	if rpcResp.Result == 10001 || rpcResp.Result == 10003 || rpcResp.Code == 10001 || rpcResp.Code == 10003 ||
		strings.Contains(rpcResp.ErrorMessage(), "登录") || strings.Contains(rpcResp.ErrorMessage(), "authentication") {
		newSess, errRelogin := c.forceRelogin(sess)
		if errRelogin == nil && newSess != "" {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel2()
			req2, req2Err := http.NewRequestWithContext(ctx2, http.MethodPost, url, bytes.NewReader(bodyBytes))
			if req2Err == nil {
				req2.Header.Set("Content-Type", "application/json;charset=utf-8")
				req2.Header.Set("Accept", "application/json, text/javascript, */*; q=0.01")
				req2.Header.Set("User-Agent", "CubeX-Host-Symbiosis/1.0")
				req2.AddCookie(&http.Cookie{Name: "sess_key", Value: newSess})
				resp2, err2 := c.httpClient.Do(req2)
				if err2 == nil {
					defer resp2.Body.Close()
					var rpcResp2 RPCResponse
					if errDecode := json.NewDecoder(resp2.Body).Decode(&rpcResp2); errDecode == nil && rpcResp2.IsSuccess() {
						c.recordSuccess()
						return &rpcResp2, nil
					}
				}
			}
		}
	}

	if rpcResp.IsSuccess() {
		c.recordSuccess()
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

// InspectDHCPOption6 queries custom DHCP options to check if Option 6 (Domain Name Server) is configured.
func (c *Client) InspectDHCPOption6(cubexIP string) (hasOpt6 bool, opt6Val string, err error) {
	funcs := []string{"dhcp_option", "dhcp_opts"}
	for _, fn := range funcs {
		resp, callErr := c.Call(fn, "show", map[string]string{
			"TYPE":  "data,total",
			"limit": "0,50",
		})
		if callErr == nil && resp.IsSuccess() && len(resp.Payload()) > 0 {
			var optData struct {
				Data []map[string]any `json:"data"`
			}
			if jsonErr := json.Unmarshal(resp.Payload(), &optData); jsonErr == nil && len(optData.Data) > 0 {
				for _, item := range optData.Data {
					optCode := fmt.Sprintf("%v", item["opt"])
					if optCode == "<nil>" || optCode == "" {
						optCode = fmt.Sprintf("%v", item["code"])
					}
					if optCode == "6" {
						val := fmt.Sprintf("%v", item["ip"])
						if val == "" || val == "<nil>" {
							val = fmt.Sprintf("%v", item["content"])
						}
						if val == "" || val == "<nil>" {
							val = fmt.Sprintf("%v", item["val"])
						}
						if cubexIP == "" || val == cubexIP {
							return true, val, nil
						}
						return false, val, nil
					}
				}
			}
		}
	}
	return false, "", nil
}

// ApplyDHCPOption6 ensures custom DHCP Option 6 is configured to point to cubexIP.
func (c *Client) ApplyDHCPOption6(cubexIP, lanIf string) ([]string, error) {
	logs := []string{}
	funcs := []string{"dhcp_option", "dhcp_opts", "dhcp_custom"}
	for _, fn := range funcs {
		resp, err := c.Call(fn, "show", map[string]string{
			"TYPE":  "data,total",
			"limit": "0,50",
		})
		if err == nil && resp.IsSuccess() && len(resp.Payload()) > 0 {
			var optData struct {
				Data []map[string]any `json:"data"`
			}
			if uErr := json.Unmarshal(resp.Payload(), &optData); uErr != nil {
				return logs, fmt.Errorf("解析爱快 %s 响应失败: %w", fn, uErr)
			}
			found := false
			for _, item := range optData.Data {
				optCode := fmt.Sprintf("%v", item["opt"])
				if optCode == "<nil>" || optCode == "" {
					optCode = fmt.Sprintf("%v", item["code"])
				}
				if optCode == "6" {
					found = true
					var val string
					for _, k := range []string{"ip", "content", "val"} {
						if s, ok := item[k].(string); ok && strings.TrimSpace(s) != "" {
							val = strings.TrimSpace(s)
							break
						}
					}
					if val == cubexIP {
						logs = append(logs, fmt.Sprintf("爱快自定义 DHCP Option 6 已正确指向 CubeX (%s)", cubexIP))
					} else {
						item["ip"] = cubexIP
						item["content"] = cubexIP
						item["val"] = cubexIP
						editResp, editErr := c.Call(fn, "edit", item)
						if editErr == nil && editResp.IsSuccess() {
							logs = append(logs, fmt.Sprintf("成功修正爱快 DHCP Option 6 指向: %s (原值: %s)", cubexIP, val))
						} else {
							return logs, fmt.Errorf("修正爱快 DHCP Option 6 (%s) 失败: err=%v resp=%v", fn, editErr, editResp)
						}
					}
					break
				}
			}
			if !found {
				addPayload := map[string]any{
					"enabled":   "yes",
					"opt":       "6",
					"code":      6,
					"ip":        cubexIP,
					"content":   cubexIP,
					"val":       cubexIP,
					"interface": lanIf,
					"comment":   "CubeX-DNS-Option6",
				}
				addResp, addErr := c.Call(fn, "add", addPayload)
				if addErr == nil && addResp.IsSuccess() {
					logs = append(logs, fmt.Sprintf("成功下发爱快自定义 DHCP Option 6 -> %s (接口: %s)", cubexIP, lanIf))
				} else {
					return logs, fmt.Errorf("下发爱快 DHCP Option 6 (%s) 失败: err=%v resp=%v", fn, addErr, addResp)
				}
			}
			return logs, nil
		}
	}
	return logs, nil
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

	c.mu.Lock()
	isV4 := c.isV4
	c.mu.Unlock()

	if isV4 {
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
	if hasOpt6, opt6Val, _ := c.InspectDHCPOption6(cubexIP); hasOpt6 {
		msgParts = append(msgParts, fmt.Sprintf("DHCP Option 6 联动就绪 (%s)", opt6Val))
	}
	if mwInfo, errMW := c.InspectMultiWANPolicy(cubexIP); errMW == nil && mwInfo.HasMultiWAN {
		msgParts = append(msgParts, mwInfo.Details)
	}
	details = strings.Join(msgParts, " · ")

	return hasRoute, hasDNS, details, nil
}

// parseIPv4Mask parses a dotted-decimal mask ("255.255.255.0") or CIDR length ("24").
func parseIPv4Mask(maskStr string) net.IPMask {
	maskIP := net.ParseIP(maskStr)
	if maskIP != nil {
		if ip4 := maskIP.To4(); ip4 != nil {
			return net.IPv4Mask(ip4[0], ip4[1], ip4[2], ip4[3])
		}
	}
	var bits int
	if _, err := fmt.Sscanf(maskStr, "%d", &bits); err == nil && bits >= 0 && bits <= 32 {
		return net.CIDRMask(bits, 32)
	}
	return net.CIDRMask(24, 32)
}

// DetectLANInterface dynamically finds which iKuai LAN interface contains targetIP.
// If multiple LANs exist (e.g., lan1, lan2, vlan1), it performs CIDR subnet containment check.
// Falls back to "lan1" if dynamic detection cannot find a matching subnet.
func (c *Client) DetectLANInterface(targetIP string) string {
	if targetIP == "" {
		return "lan1"
	}
	parsedTarget := net.ParseIP(targetIP)
	if parsedTarget == nil {
		return "lan1"
	}

	// 1. Try querying "lan" show (standard iKuai 3.x/4.x LAN interface list)
	resp, err := c.Call("lan", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,50",
	})
	if err == nil && resp.IsSuccess() && len(resp.Payload()) > 0 {
		var lanData struct {
			Data []map[string]any `json:"data"`
		}
		if jsonErr := json.Unmarshal(resp.Payload(), &lanData); jsonErr == nil && len(lanData.Data) > 0 {
			for _, item := range lanData.Data {
				iface := fmt.Sprintf("%v", item["interface"])
				if iface == "" || iface == "<nil>" {
					iface = fmt.Sprintf("%v", item["name"])
				}
				ipStr := fmt.Sprintf("%v", item["ip_addr"])
				maskStr := fmt.Sprintf("%v", item["netmask"])
				if (ipStr == "" || ipStr == "<nil>") && item["ip_mask"] != nil {
					if parts := strings.Split(fmt.Sprintf("%v", item["ip_mask"]), "/"); len(parts) == 2 {
						ipStr = parts[0]
						maskStr = parts[1]
					}
				}
				if iface != "" && iface != "<nil>" && ipStr != "" && ipStr != "<nil>" && maskStr != "" && maskStr != "<nil>" {
					ip := net.ParseIP(ipStr)
					mask := parseIPv4Mask(maskStr)
					if ip != nil && mask != nil {
						ipNet := &net.IPNet{IP: ip.Mask(mask), Mask: mask}
						if ipNet.Contains(parsedTarget) {
							return iface
						}
					}
				}
			}
		}
	}

	// 2. Fallback to DHCP server configs
	_, dhcpList, errDHCP := c.getDHCPConfig()
	if errDHCP == nil && len(dhcpList) > 0 {
		for _, d := range dhcpList {
			iface, _ := d["interface"].(string)
			gw, _ := d["gateway"].(string)
			maskStr, _ := d["netmask"].(string)
			if maskStr == "" {
				maskStr = "255.255.255.0"
			}
			if iface != "" && gw != "" {
				gwIP := net.ParseIP(gw)
				mask := parseIPv4Mask(maskStr)
				if gwIP != nil && mask != nil {
					ipNet := &net.IPNet{IP: gwIP.Mask(mask), Mask: mask}
					if ipNet.Contains(parsedTarget) {
						return iface
					}
				}
			}
		}
	}

	// 3. Fallback to VLAN interface configs (vlan show)
	respVlan, errVlan := c.Call("vlan", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,50",
	})
	if errVlan == nil && respVlan.IsSuccess() && len(respVlan.Payload()) > 0 {
		var vlanData struct {
			Data []map[string]any `json:"data"`
		}
		if jsonErr := json.Unmarshal(respVlan.Payload(), &vlanData); jsonErr == nil && len(vlanData.Data) > 0 {
			for _, item := range vlanData.Data {
				iface := fmt.Sprintf("%v", item["interface"])
				if iface == "" || iface == "<nil>" {
					iface = fmt.Sprintf("%v", item["vlan_name"])
				}
				ipStr := fmt.Sprintf("%v", item["ip_addr"])
				maskStr := fmt.Sprintf("%v", item["netmask"])
				if iface != "" && iface != "<nil>" && ipStr != "" && maskStr != "" {
					ip := net.ParseIP(ipStr)
					mask := parseIPv4Mask(maskStr)
					if ip != nil && mask != nil {
						ipNet := &net.IPNet{IP: ip.Mask(mask), Mask: mask}
						if ipNet.Contains(parsedTarget) {
							return iface
						}
					}
				}
			}
		}
	}

	return "lan1"
}

// ApplyBypass automatically injects 198.18.0.0/15 static route and updates DHCP DNS in iKuai.
// Fully compatible with both iKuai 4.0+ (static_rt) and 3.x (route_static).
func (c *Client) ApplyBypass(cubexIP string) ([]string, error) {
	cubexIP = strings.TrimSpace(cubexIP)
	parsedIP := net.ParseIP(cubexIP)
	hostIsLoopback := net.ParseIP(strings.TrimSpace(c.cfg.Host)) != nil && net.ParseIP(strings.TrimSpace(c.cfg.Host)).IsLoopback()
	if cubexIP == "" || parsedIP == nil || parsedIP.To4() == nil || (parsedIP.IsLoopback() && !hostIsLoopback) {
		return nil, fmt.Errorf("CubeX IP 无效 (%q)，无法配置爱快静态路由下一跳", cubexIP)
	}

	logs := []string{}
	// Dynamically discover which LAN interface contains CubeX IP (prevents multi-LAN / VLAN packet loss)
	routeLanIf := c.DetectLANInterface(cubexIP)
	logs = append(logs, fmt.Sprintf("动态匹配 CubeX (%s) 所属爱快 LAN 接口: %s", cubexIP, routeLanIf))

	// 1. Inspect DHCP server: query existing settings and safely preserve them
	optLanIf := routeLanIf
	fnDHCP, dhcpList, errDHCP := c.getDHCPConfig()
	if errDHCP == nil && len(dhcpList) > 0 {
		for _, d := range dhcpList {
			entryLanIf := routeLanIf
			if iface, ok := d["interface"].(string); ok && iface != "" {
				entryLanIf = iface
				if len(dhcpList) == 1 {
					optLanIf = iface
				}
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
					logs = append(logs, fmt.Sprintf("成功更新爱快 DHCP 首选 DNS 为 CubeX 独立 IP: %s (备用DNS保持: %v, 网关保持: %s, 接口: %s)", cubexIP, d["dns2"], gw, entryLanIf))
				} else {
					errMsg := "未知错误"
					if editErr != nil {
						errMsg = editErr.Error()
					} else if editResp != nil {
						errMsg = editResp.ErrorMessage()
					}
					logs = append(logs, fmt.Sprintf("更新爱快 DHCP (%s) 失败: %s", entryLanIf, errMsg))
				}
			} else if currDNS1 == cubexIP {
				logs = append(logs, fmt.Sprintf("爱快 DHCP 首选 DNS 已是 CubeX IP (%s)，备用DNS: %s，接口: %s，状态健康", cubexIP, currDNS2, entryLanIf))
			}
		}
	}

	// 1.1 Custom DHCP Option 6 synchronization
	opt6Logs, opt6Err := c.ApplyDHCPOption6(cubexIP, optLanIf)
	if len(opt6Logs) > 0 {
		logs = append(logs, opt6Logs...)
	}
	if opt6Err != nil {
		logs = append(logs, fmt.Sprintf("⚠️ 爱快 DHCP Option 6 同步告警: %v", opt6Err))
	}

	// 2. Static route injection: Try 4.0 static_rt first
	route4Resp, err4 := c.Call("static_rt", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,100",
	})

	routeHandledV4 := false
	if err4 == nil && route4Resp.IsSuccess() && len(route4Resp.Payload()) > 0 {
		routeHandledV4 = true
		// 爱快 4.0+ 静态路由处理
		var rData struct {
			Data []struct {
				ID      any    `json:"id"`
				RowID   any    `json:"rowid"`
				DstAddr string `json:"dst_addr"`
				Gateway string `json:"gateway"`
			} `json:"data"`
		}
		if uErr := json.Unmarshal(route4Resp.Payload(), &rData); uErr != nil {
			return logs, fmt.Errorf("解析爱快 4.0 静态路由响应失败: %w", uErr)
		}

		hasRoute := false
		for _, r := range rData.Data {
			if strings.HasPrefix(r.DstAddr, "198.18.0.0") || strings.HasPrefix(r.DstAddr, "198.19.0.0") {
				if r.Gateway == cubexIP {
					hasRoute = true
					logs = append(logs, fmt.Sprintf("爱快 4.0 静态路由 %s -> %s 已存在，无需重复添加", r.DstAddr, cubexIP))
				} else {
					// Gateway mismatched, delete old (4.0.317 uses "id", older builds use "rowid")
					if r.ID != nil {
						_, _ = c.Call("static_rt", "del", map[string]any{"id": r.ID})
					} else if r.RowID != nil {
						_, _ = c.Call("static_rt", "del", map[string]any{"rowid": r.RowID})
					}
					logs = append(logs, fmt.Sprintf("清理指向错误网关 (%s) 的旧爱快 4.0 静态路由", r.Gateway))
				}
			}
		}

		if !hasRoute {
			// Note: iKuai 4.0.317 strictly validates tagname length (<= 15 chars)
			addResp, addErr := c.Call("static_rt", "add", map[string]any{
				"enabled":   "yes",
				"ip_type":   "4",
				"tagname":   "CubeX-Bypass",
				"comment":   "CubeX-FakeIP",
				"prio":      1,
				"dst_addr":  "198.18.0.0",
				"netmask":   "255.254.0.0",
				"gateway":   cubexIP,
				"interface": routeLanIf,
			})
			if addErr == nil && addResp.IsSuccess() {
				logs = append(logs, fmt.Sprintf("成功在爱快 4.0 中创建静态回指路由: 198.18.0.0/15 -> %s (接口 %s)", cubexIP, routeLanIf))
			} else {
				// Fallback: dual /16 routes in case /15 mask validation fails on certain 4.0 builds
				r1, e1 := c.Call("static_rt", "add", map[string]any{
					"enabled": "yes", "ip_type": "4", "tagname": "CubeX-Bypass-1", "comment": "CubeX-FakeIP-1",
					"prio": 1, "dst_addr": "198.18.0.0", "netmask": "255.255.0.0",
					"gateway": cubexIP, "interface": routeLanIf,
				})
				r2, e2 := c.Call("static_rt", "add", map[string]any{
					"enabled": "yes", "ip_type": "4", "tagname": "CubeX-Bypass-2", "comment": "CubeX-FakeIP-2",
					"prio": 1, "dst_addr": "198.19.0.0", "netmask": "255.255.0.0",
					"gateway": cubexIP, "interface": routeLanIf,
				})
				if e1 == nil && r1.IsSuccess() && e2 == nil && r2.IsSuccess() {
					logs = append(logs, fmt.Sprintf("成功在爱快 4.0 中创建双 /16 静态回指路由: 198.18.0.0/16 & 198.19.0.0/16 -> %s (接口 %s)", cubexIP, routeLanIf))
				} else {
					errMsg := "未知错误"
					if addErr != nil {
						errMsg = addErr.Error()
					} else if addResp != nil {
						errMsg = addResp.ErrorMessage()
					}
					logs = append(logs, fmt.Sprintf("爱快 4.0 添加静态路由失败: %s", errMsg))
					return logs, fmt.Errorf("爱快 4.0 添加静态路由失败: %s", errMsg)
				}
			}
		}
	}

	if !routeHandledV4 {
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
		if len(routeResp.Payload()) > 0 {
			if uErr := json.Unmarshal(routeResp.Payload(), &rData); uErr != nil {
				return logs, fmt.Errorf("解析爱快 3.x 静态路由响应失败: %w", uErr)
			}
		}

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
				"interface": routeLanIf,
				"comment":   "CubeX-FakeIP-Bypass",
				"enabled":   "yes",
			})
			if err == nil && addResp.IsSuccess() {
				logs = append(logs, fmt.Sprintf("成功在爱快 3.x 中创建静态回指路由: 198.18.0.0/15 -> %s (接口 %s)", cubexIP, routeLanIf))
			} else {
				r1, e1 := c.Call("route_static", "add", map[string]string{
					"ip":        "198.18.0.0",
					"netmask":   "255.255.0.0",
					"gateway":   cubexIP,
					"interface": routeLanIf,
					"comment":   "CubeX-FakeIP-Bypass-1",
					"enabled":   "yes",
				})
				r2, e2 := c.Call("route_static", "add", map[string]string{
					"ip":        "198.19.0.0",
					"netmask":   "255.255.0.0",
					"gateway":   cubexIP,
					"interface": routeLanIf,
					"comment":   "CubeX-FakeIP-Bypass-2",
					"enabled":   "yes",
				})
				if e1 == nil && r1.IsSuccess() && e2 == nil && r2.IsSuccess() {
					logs = append(logs, fmt.Sprintf("成功在爱快 3.x 中创建双 /16 静态回指路由: 198.18.0.0/16 & 198.19.0.0/16 -> %s (接口 %s)", cubexIP, routeLanIf))
				} else {
					errMsg := "未知错误"
					if err != nil {
						errMsg = err.Error()
					} else if addResp != nil {
						errMsg = addResp.ErrorMessage()
					}
					return logs, fmt.Errorf("爱快 3.x 添加静态路由失败: %s", errMsg)
				}
			}
		}
	}

	// 3. Multi-WAN policy anti-hijack exemption (both iKuai 4.0+ and 3.x)
	mwLogs, mwErr := c.ApplyMultiWANExemption(cubexIP, routeLanIf)
	if len(mwLogs) > 0 {
		logs = append(logs, mwLogs...)
	}
	if mwErr != nil {
		logs = append(logs, fmt.Sprintf("⚠️ 多线流控豁免同步告警: %v", mwErr))
	}

	return logs, nil
}

// MultiWANPolicyInfo represents iKuai Multi-WAN stream and load balancing status.
type MultiWANPolicyInfo struct {
	HasMultiWAN     bool     `json:"has_multi_wan"`
	RuleCount       int      `json:"rule_count"`
	FakeIPProtected bool     `json:"fake_ip_protected"`
	WANInterfaces   []string `json:"wan_interfaces"`
	Details         string   `json:"details"`
}

// InspectMultiWANPolicy queries iKuai multi-WAN stream rules (stream_ip, stream_port, stream_protocol).
func (c *Client) InspectMultiWANPolicy(cubexIP string) (*MultiWANPolicyInfo, error) {
	info := &MultiWANPolicyInfo{
		WANInterfaces: make([]string, 0),
	}

	// 1. Query stream_ip (多线负载与IP分流)
	resp, err := c.Call("stream_ip", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,100",
	})
	if err == nil && resp.IsSuccess() && len(resp.Payload()) > 0 {
		var sData struct {
			Data []map[string]any `json:"data"`
		}
		if jsonErr := json.Unmarshal(resp.Payload(), &sData); jsonErr == nil && len(sData.Data) > 0 {
			info.HasMultiWAN = true
			info.RuleCount += len(sData.Data)
			for _, r := range sData.Data {
				dstIP := fmt.Sprintf("%v", r["dst_ip"])
				comment := fmt.Sprintf("%v", r["comment"])
				iface := fmt.Sprintf("%v", r["interface"])
				if iface != "" && iface != "<nil>" && !strSliceContains(info.WANInterfaces, iface) {
					info.WANInterfaces = append(info.WANInterfaces, iface)
				}
				if strings.Contains(dstIP, "198.18.0.0") || strings.Contains(comment, "CubeX-MultiWAN") {
					info.FakeIPProtected = true
				}
			}
		}
	}

	// 2. Query stream_port if stream_ip is empty
	if !info.HasMultiWAN {
		respPort, errPort := c.Call("stream_port", "show", map[string]string{
			"TYPE":  "data,total",
			"limit": "0,50",
		})
		if errPort == nil && respPort.IsSuccess() && len(respPort.Payload()) > 0 {
			var pData struct {
				Data []map[string]any `json:"data"`
			}
			if jsonErr := json.Unmarshal(respPort.Payload(), &pData); jsonErr == nil && len(pData.Data) > 0 {
				info.HasMultiWAN = true
				info.RuleCount += len(pData.Data)
			}
		}
	}

	if info.HasMultiWAN {
		if info.FakeIPProtected {
			info.Details = fmt.Sprintf("已检测到多线负载 (%d条分流规则)，Fake-IP 豁免保护已就绪", info.RuleCount)
		} else {
			info.Details = fmt.Sprintf("检测到多线负载 (%d条分流规则)，尚未配置 Fake-IP 豁免规则", info.RuleCount)
		}
	} else {
		info.Details = "单线或未配置多线分流规则"
	}

	return info, nil
}

// ApplyMultiWANExemption ensures iKuai's multi-WAN policy routing does not hijack Fake-IP packets to public WAN.
// 🛡️ 师父防雷补丁 3: 严格去重与幂等处理，防止重复条目堆积
func (c *Client) ApplyMultiWANExemption(cubexIP, lanIf string) ([]string, error) {
	logs := []string{}
	mwInfo, err := c.InspectMultiWANPolicy(cubexIP)
	if err != nil || !mwInfo.HasMultiWAN {
		return logs, nil
	}

	if lanIf == "" {
		lanIf = c.DetectLANInterface(cubexIP)
	}

	// 1. 深度扫描 stream_ip 去重与冲突清理
	respShow, errShow := c.Call("stream_ip", "show", map[string]string{
		"TYPE":  "data,total",
		"limit": "0,100",
	})

	alreadyConfigured := false
	if errShow == nil && respShow.IsSuccess() && len(respShow.Payload()) > 0 {
		var sData struct {
			Data []map[string]any `json:"data"`
		}
		if json.Unmarshal(respShow.Payload(), &sData) == nil && len(sData.Data) > 0 {
			for _, r := range sData.Data {
				dstIP := fmt.Sprintf("%v", r["dst_ip"])
				comment := fmt.Sprintf("%v", r["comment"])
				currIf := fmt.Sprintf("%v", r["interface"])
				idVal := r["id"]

				if strings.Contains(dstIP, "198.18.0.0") || strings.Contains(comment, "CubeX-MultiWAN") {
					if currIf == lanIf && !alreadyConfigured {
						alreadyConfigured = true
						logs = append(logs, fmt.Sprintf("爱快多线分流 Fake-IP 豁免策略已就绪 (接口: %s)，无需重复配置", lanIf))
					} else if idVal != nil {
						// 清理重复条目或接口漂移旧条目
						_, _ = c.Call("stream_ip", "del", map[string]any{"id": idVal})
						logs = append(logs, fmt.Sprintf("已自动清理爱快多线流控中的冗余/冲突规则 (ID: %v, 原接口: %s)", idVal, currIf))
					}
				}
			}
		}
	}

	if alreadyConfigured {
		return logs, nil
	}

	// In iKuai stream_ip: set dst_ip=198.18.0.0/15 and interface=lanIf
	payload := map[string]any{
		"enabled":   "yes",
		"comment":   "CubeX-MultiWAN-Exempt",
		"dst_ip":    "198.18.0.0/15",
		"interface": lanIf,
	}
	addResp, addErr := c.Call("stream_ip", "add", payload)
	if addErr == nil && addResp.IsSuccess() {
		logs = append(logs, fmt.Sprintf("成功在爱快多线流控中注入 Fake-IP 豁免规则: 198.18.0.0/15 锁定回指 %s (避免多 WAN 丢包)", lanIf))
	}

	return logs, nil
}

func strSliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// SyncSelfHealing verifies and auto-repairs iKuai's static route 198.18.0.0/15, DHCP DNS, and Option 6.
func (c *Client) SyncSelfHealing(cubexIP string) ([]string, error) {
	hasRoute, hasDNS, details, err := c.InspectRouting(cubexIP)
	if err != nil {
		return nil, fmt.Errorf("爱快状态自愈巡检失败: %w", err)
	}
	if hasRoute && hasDNS {
		return []string{fmt.Sprintf("✓ 爱快 Web-RPC 联动巡检正常: %s", details)}, nil
	}
	// Anomaly detected, auto-reheal
	logs, applyErr := c.ApplyBypass(cubexIP)
	if applyErr != nil {
		return nil, fmt.Errorf("爱快自愈修复失败: %w", applyErr)
	}
	return append([]string{fmt.Sprintf("⚠️ 检测到爱快旁路配置漂移 (%s)，已触发自动自愈加固：", details)}, logs...), nil
}

// RemoveBypass cleanly withdraws CubeX static routes, Multi-WAN exemptions, DHCP DNS, and Option 6 from iKuai.
func (c *Client) RemoveBypass(cubexIP, backupDNS string) ([]string, error) {
	if strings.TrimSpace(backupDNS) == "" {
		backupDNS = "223.5.5.5"
	}
	logs := []string{}
	var firstErr error
	recordErr := func(e error) {
		if e != nil && firstErr == nil {
			firstErr = e
		}
	}

	// 1. Restore DHCP server dns1 if pointing to cubexIP
	fnDHCP, dhcpList, errDHCP := c.getDHCPConfig()
	if errDHCP != nil {
		recordErr(errDHCP)
	} else if len(dhcpList) > 0 {
		for _, d := range dhcpList {
			currDNS1, _ := d["dns1"].(string)
			currDNS2, _ := d["dns2"].(string)
			if currDNS1 != "" && (cubexIP == "" || currDNS1 == cubexIP) {
				targetDNS := backupDNS
				if currDNS2 != "" && currDNS2 != cubexIP {
					targetDNS = currDNS2
				}
				d["dns1"] = targetDNS
				if editResp, err := c.Call(fnDHCP, "edit", d); err == nil && editResp.IsSuccess() {
					logs = append(logs, fmt.Sprintf("✓ [爱快] 已撤回 DHCP 首选 DNS 并恢复为: %s", targetDNS))
				} else {
					recordErr(fmt.Errorf("恢复爱快 DHCP DNS 失败: err=%v resp=%v", err, editResp))
				}
			}
		}
	}

	// 2. Remove CubeX DHCP Option 6 (across all iKuai option tables)
	for _, fn := range []string{"dhcp_option", "dhcp_opts", "dhcp_custom"} {
		resp, err := c.Call(fn, "show", map[string]string{"TYPE": "data,total", "limit": "0,50"})
		if err == nil && resp.IsSuccess() && len(resp.Payload()) > 0 {
			var optData struct {
				Data []map[string]any `json:"data"`
			}
			if json.Unmarshal(resp.Payload(), &optData) == nil {
				for _, item := range optData.Data {
					comment := fmt.Sprintf("%v", item["comment"])
					var val string
					for _, k := range []string{"ip", "content", "val"} {
						if s, ok := item[k].(string); ok && strings.TrimSpace(s) != "" {
							val = strings.TrimSpace(s)
							break
						}
					}
					if strings.Contains(comment, "CubeX") || (cubexIP != "" && val == cubexIP) {
						idVal := item["id"]
						if idVal == nil {
							idVal = item["rowid"]
						}
						if idVal != nil {
							if delResp, delErr := c.Call(fn, "del", map[string]any{"id": idVal}); delErr == nil && delResp.IsSuccess() {
								logs = append(logs, fmt.Sprintf("✓ [爱快] 已清理自定义 DHCP Option 6 (表: %s, ID: %v)", fn, idVal))
							} else {
								recordErr(fmt.Errorf("清理爱快 Option 6 (%s, id=%v) 失败: err=%v resp=%v", fn, idVal, delErr, delResp))
							}
						}
					}
				}
			}
			break
		}
	}

	// 3. Remove 4.0 static_rt CubeX routes
	if route4Resp, err4 := c.Call("static_rt", "show", map[string]string{"TYPE": "data,total", "limit": "0,100"}); err4 == nil && route4Resp.IsSuccess() {
		var rData struct {
			Data []struct {
				ID      any    `json:"id"`
				RowID   any    `json:"rowid"`
				DstAddr string `json:"dst_addr"`
				TagName string `json:"tagname"`
			} `json:"data"`
		}
		if json.Unmarshal(route4Resp.Payload(), &rData) == nil {
			for _, r := range rData.Data {
				if strings.HasPrefix(r.DstAddr, "198.18.0.0") || strings.HasPrefix(r.DstAddr, "198.19.0.0") || strings.Contains(r.TagName, "CubeX") {
					delParam := map[string]any{"rowid": fmt.Sprintf("%v", r.RowID)}
					keyID := r.RowID
					if r.ID != nil {
						delParam = map[string]any{"id": r.ID}
						keyID = r.ID
					}
					if delResp, delErr := c.Call("static_rt", "del", delParam); delErr == nil && delResp.IsSuccess() {
						logs = append(logs, fmt.Sprintf("✓ [爱快 4.0] 已撤回静态回指路由 %s (id=%v)", r.DstAddr, keyID))
					} else {
						recordErr(fmt.Errorf("撤回爱快 4.0 静态路由 (%v) 失败: err=%v resp=%v", keyID, delErr, delResp))
					}
				}
			}
		}
	}

	// 4. Remove 3.x route_static CubeX routes
	if route3Resp, err3 := c.Call("route_static", "show", map[string]string{"TYPE": "data,total", "limit": "0,100"}); err3 == nil && route3Resp.IsSuccess() {
		var rData struct {
			Data []struct {
				ID      any    `json:"id"`
				IP      string `json:"ip"`
				Comment string `json:"comment"`
			} `json:"data"`
		}
		if json.Unmarshal(route3Resp.Payload(), &rData) == nil {
			for _, r := range rData.Data {
				if r.IP == "198.18.0.0" || r.IP == "198.19.0.0" || strings.Contains(r.Comment, "CubeX") {
					if delResp, delErr := c.Call("route_static", "del", map[string]string{"id": fmt.Sprintf("%v", r.ID)}); delErr == nil && delResp.IsSuccess() {
						logs = append(logs, fmt.Sprintf("✓ [爱快 3.x] 已撤回静态回指路由 %s (id=%v)", r.IP, r.ID))
					} else {
						recordErr(fmt.Errorf("撤回爱快 3.x 静态路由 (%v) 失败: err=%v resp=%v", r.ID, delErr, delResp))
					}
				}
			}
		}
	}

	// 5. Remove Multi-WAN exemptions
	if mwResp, mwErr := c.Call("stream_ip", "show", map[string]string{"TYPE": "data,total", "limit": "0,100"}); mwErr == nil && mwResp.IsSuccess() {
		var sData struct {
			Data []map[string]any `json:"data"`
		}
		if json.Unmarshal(mwResp.Payload(), &sData) == nil {
			for _, r := range sData.Data {
				dstIP := fmt.Sprintf("%v", r["dst_ip"])
				comment := fmt.Sprintf("%v", r["comment"])
				if (strings.Contains(dstIP, "198.18.0.0") || strings.Contains(comment, "CubeX")) && r["id"] != nil {
					if delResp, delErr := c.Call("stream_ip", "del", map[string]any{"id": r["id"]}); delErr == nil && delResp.IsSuccess() {
						logs = append(logs, fmt.Sprintf("✓ [爱快] 已撤回多线豁免策略 (ID: %v)", r["id"]))
					} else {
						recordErr(fmt.Errorf("撤回爱快多线豁免策略 (%v) 失败: err=%v resp=%v", r["id"], delErr, delResp))
					}
				}
			}
		}
	}

	if len(logs) == 0 && firstErr == nil {
		logs = append(logs, "✓ [爱快] 未发现残留 CubeX 旁路配置，状态已纯净")
	}
	return logs, firstErr
}
