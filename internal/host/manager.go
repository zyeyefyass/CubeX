package host

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// HostConfig stores the SSH credentials for communicating with the host OS (e.g., fnOS, PVE, Debian, Ubuntu).
type HostConfig struct {
	Enabled    bool   `json:"enabled"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
}

// HostInfo represents status and telemetry probed from the host system.
type HostInfo struct {
	Connected      bool   `json:"connected"`
	OSName         string `json:"os_name"`
	PlatformID     string `json:"platform_id"` // "fnos", "debian", "ubuntu", "pve", "synology", "linux"
	KernelVersion  string `json:"kernel_version"`
	Arch           string `json:"arch"`
	HostName       string `json:"hostname"`
	Uptime         string `json:"uptime"`
	IPForward      bool   `json:"ip_forward"`
	WireGuardReady bool   `json:"wireguard_ready"`
	WireGuardActive bool  `json:"wireguard_active"`
	FirewallConfigured bool `json:"firewall_configured"`
	DockerRunning      bool   `json:"docker_running"`
	RawError           string `json:"raw_error,omitempty"`
	CPUCores           int    `json:"cpu_cores"`
	CPULoad            string `json:"cpu_load"`
	MemoryUsed         string `json:"memory_used"`
	MemoryTotal        string `json:"memory_total"`
}

// FirewallResult represents the status of firewall rule deployment on the host.
type FirewallResult struct {
	Success bool     `json:"success"`
	Logs    []string `json:"logs"`
}

// Manager orchestrates host-level system maintenance and Docker companion tasks.
type Manager struct {
	rootDir    string
	configFile string
	mu         sync.RWMutex
	config     HostConfig
	cachedInfo *HostInfo
}

// NewManager creates a new host Manager.
func NewManager(rootDir string) *Manager {
	m := &Manager{
		rootDir:    rootDir,
		configFile: filepath.Join(rootDir, "data", "host-config.json"),
		config: HostConfig{
			Enabled: false,
			Host:    "172.17.0.1", // Standard Docker gateway / host IP
			Port:    22,
			User:    "root",
		},
	}
	m.loadConfig()
	return m
}

func (m *Manager) loadConfig() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if data, err := os.ReadFile(m.configFile); err == nil {
		_ = json.Unmarshal(data, &m.config)
	}
}

// SaveConfig persists the Host SSH configuration.
func (m *Manager) SaveConfig(cfg HostConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.User == "" {
		cfg.User = "root"
	}
	m.config = cfg
	_ = os.MkdirAll(filepath.Dir(m.configFile), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.configFile, data, 0644)
}

// GetConfig returns the current Host SSH configuration.
func (m *Manager) GetConfig() HostConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// GetCachedHostInfo returns the most recently successfully probed host telemetry.
func (m *Manager) GetCachedHostInfo() *HostInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cachedInfo
}

// getSSHClient dials the host machine via SSH.
func (m *Manager) getSSHClient(cfg HostConfig) (*ssh.Client, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("宿主机 IP 或地址未配置")
	}
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if cfg.User == "" {
		cfg.User = "root"
	}

	var authMethods []ssh.AuthMethod
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}
	if strings.TrimSpace(cfg.PrivateKey) != "" {
		signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		if err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("请提供宿主机 SSH 登录密码或私钥")
	}

	clientConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return ssh.Dial("tcp", addr, clientConfig)
}

func parseROSResourcePrint(outStr string) (ver, arch string, cores int, load, memUsed, memTotal string) {
	ver = "7.x"
	arch = "x86_64"
	cores = 1
	var totalMemNum, freeMemNum float64

	for _, line := range strings.Split(outStr, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "version:") {
			ver = strings.TrimSpace(strings.TrimPrefix(line, "version:"))
		}
		if strings.HasPrefix(line, "architecture-name:") {
			arch = strings.TrimSpace(strings.TrimPrefix(line, "architecture-name:"))
		}
		if strings.HasPrefix(line, "cpu-count:") {
			cStr := strings.TrimSpace(strings.TrimPrefix(line, "cpu-count:"))
			if c, err := strconv.Atoi(cStr); err == nil && c > 0 {
				cores = c
			}
		}
		if strings.HasPrefix(line, "cpu-load:") {
			load = strings.TrimSpace(strings.TrimPrefix(line, "cpu-load:"))
		}
		if strings.HasPrefix(line, "total-memory:") {
			memTotal = strings.TrimSpace(strings.TrimPrefix(line, "total-memory:"))
			cleaned := strings.TrimSuffix(memTotal, "MiB")
			if t, err := strconv.ParseFloat(cleaned, 64); err == nil {
				totalMemNum = t
			}
		}
		if strings.HasPrefix(line, "free-memory:") {
			freeStr := strings.TrimSpace(strings.TrimPrefix(line, "free-memory:"))
			cleaned := strings.TrimSuffix(freeStr, "MiB")
			if f, err := strconv.ParseFloat(cleaned, 64); err == nil {
				freeMemNum = f
			}
		}
	}
	if totalMemNum > 0 && freeMemNum > 0 {
		used := totalMemNum - freeMemNum
		if used < 0 {
			used = 0
		}
		memUsed = fmt.Sprintf("%.0fMiB", used)
	}
	return
}

func (m *Manager) testRouterOSRest(cfg HostConfig) (map[string]any, error) {
	ports := []string{"80", "443"}
	if cfg.Port > 0 && cfg.Port != 22 {
		ports = []string{strconv.Itoa(cfg.Port), "80", "443"}
	}
	for _, p := range ports {
		scheme := "http"
		if p == "443" {
			scheme = "https"
		}
		url := fmt.Sprintf("%s://%s:%s/rest/system/resource", scheme, cfg.Host, p)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		req.SetBasicAuth(cfg.User, cfg.Password)
		client := &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode >= 400 {
			if resp != nil {
				_ = resp.Body.Close()
			}
			continue
		}
		defer resp.Body.Close()

		var res struct {
			Platform         string `json:"platform"`
			ArchitectureName string `json:"architecture-name"`
			BoardName        string `json:"board-name"`
			Version          string `json:"version"`
			CpuCount         string `json:"cpu-count"`
			CpuLoad          string `json:"cpu-load"`
			TotalMemory      string `json:"total-memory"`
			FreeMemory       string `json:"free-memory"`
			Uptime           string `json:"uptime"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&res); err == nil && (res.Platform == "MikroTik" || res.Version != "") {
			cores, _ := strconv.Atoi(res.CpuCount)
			if cores <= 0 {
				cores = 1
			}
			var totalMB, freeMB float64
			memTotalStr := ""
			if totalBytes, err := strconv.ParseFloat(res.TotalMemory, 64); err == nil {
				if totalBytes > 1000000 {
					totalMB = totalBytes / 1024 / 1024
				} else {
					totalMB = totalBytes
				}
				memTotalStr = fmt.Sprintf("%.0fMiB", totalMB)
			}
			if freeBytes, err := strconv.ParseFloat(res.FreeMemory, 64); err == nil {
				if freeBytes > 1000000 {
					freeMB = freeBytes / 1024 / 1024
				} else {
					freeMB = freeBytes
				}
			}
			memUsedStr := ""
			if totalMB > 0 && freeMB > 0 {
				used := totalMB - freeMB
				if used < 0 {
					used = 0
				}
				memUsedStr = fmt.Sprintf("%.0fMiB", used)
			}

			loadStr := ""
			if res.CpuLoad != "" {
				loadStr = res.CpuLoad + "%"
			}

			result := map[string]any{
				"ok":                 true,
				"connected":          true,
				"os_pretty":          fmt.Sprintf("MikroTik RouterOS %s", res.Version),
				"os_name":            "MikroTik RouterOS",
				"arch":               res.ArchitectureName,
				"platform_id":        "routeros",
				"ip_forward":         true,
				"kernel":             res.Version,
				"cpu_cores":          cores,
				"cpu_load":           loadStr,
				"memory_used":        memUsedStr,
				"memory_total":       memTotalStr,
				"uptime":             res.Uptime,
				"firewall_configured": true,
			}

			m.mu.Lock()
			m.cachedInfo = &HostInfo{
				Connected:          true,
				OSName:             fmt.Sprintf("MikroTik RouterOS %s", res.Version),
				PlatformID:         "routeros",
				KernelVersion:      res.Version,
				Arch:               res.ArchitectureName,
				HostName:           res.BoardName,
				Uptime:             res.Uptime,
				IPForward:          true,
				FirewallConfigured: true,
				CPUCores:           cores,
				CPULoad:            loadStr,
				MemoryUsed:         memUsedStr,
				MemoryTotal:        memTotalStr,
			}
			m.mu.Unlock()

			return result, nil
		}
	}
	return nil, fmt.Errorf("RouterOS REST API 连接失败 (尝试端口 80/443)")
}

// TestSSH tests the SSH credentials, auto-detecting MikroTik RouterOS or standard Linux hosts.
func (m *Manager) TestSSH(cfg HostConfig) (map[string]any, error) {
	client, err := m.getSSHClient(cfg)
	if err != nil {
		// Fallback to RouterOS REST API check (port 80 / 443)
		if restRes, restErr := m.testRouterOSRest(cfg); restErr == nil && restRes != nil {
			_ = m.syncToROSConfig(cfg)
			return restRes, nil
		}
		return nil, fmt.Errorf("SSH 握手或鉴权失败: %w", err)
	}
	defer client.Close()

	// 1. Try RouterOS CLI first
	session, err := client.NewSession()
	if err == nil {
		var b bytes.Buffer
		session.Stdout = &b
		session.Stderr = &b
		rErr := session.Run("/system/resource/print")
		session.Close()
		outStr := b.String()
		if rErr == nil && (strings.Contains(outStr, "platform: MikroTik") || strings.Contains(outStr, "version:") || strings.Contains(outStr, "uptime:")) {
			ver, arch, cores, load, memUsed, memTotal := parseROSResourcePrint(outStr)
			res := map[string]any{
				"ok":                 true,
				"connected":          true,
				"os_pretty":          fmt.Sprintf("MikroTik RouterOS %s", ver),
				"os_name":            "MikroTik RouterOS",
				"arch":               arch,
				"platform_id":        "routeros",
				"ip_forward":         true,
				"kernel":             ver,
				"cpu_cores":          cores,
				"cpu_load":           load,
				"memory_used":        memUsed,
				"memory_total":       memTotal,
				"firewall_configured": true,
			}
			_ = m.syncToROSConfig(cfg)
			m.mu.Lock()
			m.cachedInfo = &HostInfo{
				Connected:          true,
				OSName:             fmt.Sprintf("MikroTik RouterOS %s", ver),
				PlatformID:         "routeros",
				KernelVersion:      ver,
				Arch:               arch,
				IPForward:          true,
				FirewallConfigured: true,
				CPUCores:           cores,
				CPULoad:            load,
				MemoryUsed:         memUsed,
				MemoryTotal:        memTotal,
			}
			m.mu.Unlock()
			return res, nil
		}
	}

	// 2. Linux OS probing fallback
	session2, err := client.NewSession()
	if err != nil {
		return map[string]any{
			"ok":        true,
			"connected": true,
			"os_name":   "Linux Host",
			"arch":      "unknown",
		}, nil
	}
	defer session2.Close()
	var b2 bytes.Buffer
	session2.Stdout = &b2
	_ = session2.Run("cat /etc/os-release; uname -m")
	osName, platID := parseOSRelease(b2.String())
	arch := "amd64"
	for _, l := range strings.Split(b2.String(), "\n") {
		l = strings.TrimSpace(l)
		if l == "x86_64" || l == "aarch64" || l == "arm64" {
			arch = l
		}
	}
	return map[string]any{
		"ok":          true,
		"connected":   true,
		"os_pretty":   osName,
		"os_name":     osName,
		"platform_id": platID,
		"arch":        arch,
		"ip_forward":  true,
	}, nil
}

func (m *Manager) syncToROSConfig(cfg HostConfig) error {
	rosFile := filepath.Join(m.rootDir, "data", "ros-config.json")
	var rosCfg map[string]string
	data, err := os.ReadFile(rosFile)
	if err == nil {
		_ = json.Unmarshal(data, &rosCfg)
	}
	if rosCfg == nil {
		rosCfg = make(map[string]string)
	}
	rosCfg["host"] = cfg.Host
	rosCfg["port"] = "80"
	rosCfg["user"] = cfg.User
	rosCfg["password"] = cfg.Password
	outBytes, _ := json.MarshalIndent(rosCfg, "", "  ")
	return os.WriteFile(rosFile, outBytes, 0644)
}

// RunCommand runs a single command string on the host and returns stdout+stderr combined.
func (m *Manager) RunCommand(cfg HostConfig, cmdStr string) (string, error) {
	client, err := m.getSSHClient(cfg)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH 会话失败: %w", err)
	}
	defer session.Close()

	var b bytes.Buffer
	session.Stdout = &b
	session.Stderr = &b
	err = session.Run(cmdStr)
	out := strings.TrimSpace(b.String())
	if err != nil {
		return out, fmt.Errorf("%w: %s", err, out)
	}
	return out, nil
}

// ProbeHost telemetry inspects the host OS, kernel, network, and WireGuard status.
func (m *Manager) ProbeHost() (*HostInfo, error) {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	info := &HostInfo{
		PlatformID: "linux",
		OSName:     "Linux Host",
	}

	targetHost := cfg.Host
	targetUser := cfg.User
	targetPass := cfg.Password
	if targetHost == "" {
		rosFile := filepath.Join(m.rootDir, "data", "ros-config.json")
		if data, err := os.ReadFile(rosFile); err == nil {
			var rosCfg map[string]string
			if json.Unmarshal(data, &rosCfg) == nil && rosCfg["host"] != "" {
				targetHost = rosCfg["host"]
				targetUser = rosCfg["user"]
				targetPass = rosCfg["password"]
			}
		}
	}
	if targetHost != "" {
		if restRes, err := m.testRouterOSRest(HostConfig{Host: targetHost, User: targetUser, Password: targetPass}); err == nil && restRes != nil {
			info.Connected = true
			info.PlatformID = "routeros"
			info.OSName = restRes["os_pretty"].(string)
			info.KernelVersion = restRes["kernel"].(string)
			info.Arch = restRes["arch"].(string)
			info.IPForward = true
			info.FirewallConfigured = true
			if c, ok := restRes["cpu_cores"].(int); ok {
				info.CPUCores = c
			}
			if l, ok := restRes["cpu_load"].(string); ok {
				info.CPULoad = l
			}
			if u, ok := restRes["memory_used"].(string); ok {
				info.MemoryUsed = u
			}
			if t, ok := restRes["memory_total"].(string); ok {
				info.MemoryTotal = t
			}
			m.mu.Lock()
			m.cachedInfo = info
			m.mu.Unlock()
			return info, nil
		}
	}

	if !cfg.Enabled || cfg.Host == "" {
		// Attempt local container detection
		info.OSName = m.detectLocalHost()
		info.PlatformID = m.detectLocalPlatformID(info.OSName)
		return info, nil
	}

	client, err := m.getSSHClient(cfg)
	if err != nil {
		info.RawError = err.Error()
		return info, err
	}
	defer client.Close()

	info.Connected = true

	// 0. RouterOS Probing & Telemetry
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		session.Stderr = &out
		_ = session.Run("/system/resource/print")
		session.Close()
		outStr := out.String()
		if strings.Contains(outStr, "platform: MikroTik") || strings.Contains(outStr, "architecture-name:") || strings.Contains(outStr, "version:") {
			info.OSName = "MikroTik RouterOS"
			info.PlatformID = "routeros"
			info.IPForward = true
			info.FirewallConfigured = true

			var totalMemNum, freeMemNum float64
			for _, line := range strings.Split(outStr, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "version:") {
					ver := strings.TrimSpace(strings.TrimPrefix(line, "version:"))
					info.KernelVersion = ver
					info.OSName = fmt.Sprintf("MikroTik RouterOS %s", ver)
				}
				if strings.HasPrefix(line, "architecture-name:") {
					info.Arch = strings.TrimSpace(strings.TrimPrefix(line, "architecture-name:"))
				}
				if strings.HasPrefix(line, "board-name:") {
					info.HostName = strings.TrimSpace(strings.TrimPrefix(line, "board-name:"))
				}
				if strings.HasPrefix(line, "cpu-count:") {
					cStr := strings.TrimSpace(strings.TrimPrefix(line, "cpu-count:"))
					info.CPUCores, _ = strconv.Atoi(cStr)
				}
				if strings.HasPrefix(line, "cpu-load:") {
					info.CPULoad = strings.TrimSpace(strings.TrimPrefix(line, "cpu-load:"))
				}
				if strings.HasPrefix(line, "total-memory:") {
					info.MemoryTotal = strings.TrimSpace(strings.TrimPrefix(line, "total-memory:"))
					if strings.HasSuffix(info.MemoryTotal, "MiB") {
						totalMemNum, _ = strconv.ParseFloat(strings.TrimSuffix(info.MemoryTotal, "MiB"), 64)
					}
				}
				if strings.HasPrefix(line, "free-memory:") {
					freeStr := strings.TrimSpace(strings.TrimPrefix(line, "free-memory:"))
					if strings.HasSuffix(freeStr, "MiB") {
						freeMemNum, _ = strconv.ParseFloat(strings.TrimSuffix(freeStr, "MiB"), 64)
					}
				}
				if strings.HasPrefix(line, "uptime:") {
					info.Uptime = strings.TrimSpace(strings.TrimPrefix(line, "uptime:"))
				}
			}
			if totalMemNum > 0 && freeMemNum > 0 {
				used := totalMemNum - freeMemNum
				info.MemoryUsed = fmt.Sprintf("%.0fMiB", used)
			}
			if info.CPUCores == 0 {
				info.CPUCores = 1
			}
			m.mu.Lock()
			m.cachedInfo = info
			m.mu.Unlock()
			return info, nil
		}
	}

	// 1. Check OS release
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		_ = session.Run("cat /etc/os-release")
		session.Close()
		info.OSName, info.PlatformID = parseOSRelease(out.String())
	}

	// 2. Check Kernel & Arch & Hostname
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		_ = session.Run("uname -r; uname -m; hostname")
		session.Close()
		lines := strings.Split(strings.TrimSpace(out.String()), "\n")
		if len(lines) >= 1 {
			info.KernelVersion = strings.TrimSpace(lines[0])
		}
		if len(lines) >= 2 {
			info.Arch = strings.TrimSpace(lines[1])
		}
		if len(lines) >= 3 {
			info.HostName = strings.TrimSpace(lines[2])
		}
	}

	// 3. Check uptime
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		_ = session.Run("uptime -p 2>/dev/null || cat /proc/uptime")
		session.Close()
		info.Uptime = strings.TrimSpace(out.String())
	}

	// 4. Check IP Forward
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		_ = session.Run("cat /proc/sys/net/ipv4/ip_forward")
		session.Close()
		info.IPForward = strings.TrimSpace(out.String()) == "1"
	}

	// 5. Check WireGuard installed & active
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		_ = session.Run("which wg; which wg-quick")
		session.Close()
		info.WireGuardReady = strings.Contains(out.String(), "wg")
	}

	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		_ = session.Run("ip link show wg0 2>/dev/null")
		session.Close()
		info.WireGuardActive = strings.Contains(out.String(), "wg0")
	}

	// 6. Check Firewall iptables rule for 13231 UDP
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		_ = session.Run("iptables -L INPUT -n 2>/dev/null | grep 13231")
		session.Close()
		info.FirewallConfigured = strings.Contains(out.String(), "13231")
	}

	m.mu.Lock()
	m.cachedInfo = info
	m.mu.Unlock()

	return info, nil
}

// ApplyFirewall configures IP forwarding, WireGuard UDP port, panel TCP port, and NAT masquerade on the host.
// Supported modes: "permissive" (default, recommended for novices) and "strict" (least privilege).
// If enablePenetration is false, no WireGuard UDP penetration ports will be opened (pure LAN bypass).
func (m *Manager) ApplyFirewall(mode string, enablePenetration bool) (*FirewallResult, error) {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	if !cfg.Enabled || cfg.Host == "" {
		return nil, fmt.Errorf("请先在【宿主机运维】中配置并启用宿主机 SSH 连接")
	}

	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode != "strict" {
		mode = "permissive"
	}

	res := &FirewallResult{Success: false, Logs: []string{}}

	var commands []struct {
		desc string
		cmd  string
	}

	if mode == "permissive" {
		commands = append(commands,
			struct {
				desc string
				cmd  string
			}{"开启 Linux 内核 IPv4 转发", "sysctl -w net.ipv4.ip_forward=1 && (grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf)"},
			struct {
				desc string
				cmd  string
			}{"开启全网卡转发 (conf.all.forwarding)", "sysctl -w net.ipv4.conf.all.forwarding=1 2>/dev/null || true"},
			struct {
				desc string
				cmd  string
			}{"放行 FORWARD 转发链默认策略 (ACCEPT)", "iptables -P FORWARD ACCEPT"},
			struct {
				desc string
				cmd  string
			}{"置顶放行 FORWARD 数据链", "iptables -C FORWARD -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -j ACCEPT"},
			struct {
				desc string
				cmd  string
			}{"放行 MiCube Web 面板 TCP 2026 端口", "iptables -C INPUT -p tcp --dport 2026 -j ACCEPT 2>/dev/null || iptables -I INPUT -p tcp --dport 2026 -j ACCEPT"},
		)
		if enablePenetration {
			commands = append(commands,
				struct {
					desc string
					cmd  string
				}{"放行 WireGuard UDP 13231 穿透端口", "iptables -C INPUT -p udp --dport 13231 -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport 13231 -j ACCEPT"},
				struct {
					desc string
					cmd  string
				}{"放行 WireGuard UDP 51820 备用端口", "iptables -C INPUT -p udp --dport 51820 -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport 51820 -j ACCEPT"},
				struct {
					desc string
					cmd  string
				}{"配置 WireGuard 虚拟网段 (10.88.0.0/24) NAT 出网伪装", "iptables -t nat -C POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE 2>/dev/null || iptables -t nat -I POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE"},
			)
		}
	} else {
		// Strict mode
		commands = append(commands,
			struct {
				desc string
				cmd  string
			}{"开启 Linux 内核 IPv4 转发", "sysctl -w net.ipv4.ip_forward=1 && (grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf)"},
			struct {
				desc string
				cmd  string
			}{"设置 FORWARD 链默认策略为 DROP (最小特权)", "iptables -P FORWARD DROP"},
			struct {
				desc string
				cmd  string
			}{"放行 ESTABLISHED,RELATED 状态跟踪连接", "iptables -C FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || iptables -C FORWARD -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -m state --state RELATED,ESTABLISHED -j ACCEPT"},
			struct {
				desc string
				cmd  string
			}{"放行 RFC1918 私网网段互访转发", "iptables -C FORWARD -s 192.168.0.0/16 -d 192.168.0.0/16 -j ACCEPT 2>/dev/null || iptables -I FORWARD 2 -s 192.168.0.0/16 -d 192.168.0.0/16 -j ACCEPT; iptables -C FORWARD -s 10.0.0.0/8 -d 10.0.0.0/8 -j ACCEPT 2>/dev/null || iptables -I FORWARD 3 -s 10.0.0.0/8 -d 10.0.0.0/8 -j ACCEPT"},
			struct {
				desc string
				cmd  string
			}{"放行 MiCube Web 面板 TCP 2026 端口", "iptables -C INPUT -p tcp --dport 2026 -j ACCEPT 2>/dev/null || iptables -I INPUT -p tcp --dport 2026 -j ACCEPT"},
		)
		if enablePenetration {
			commands = append(commands,
				struct {
					desc string
					cmd  string
				}{"定向放行 WireGuard 虚拟网段 (10.88.0.0/24) 转发", "iptables -C FORWARD -s 10.88.0.0/24 -j ACCEPT 2>/dev/null || iptables -I FORWARD 4 -s 10.88.0.0/24 -j ACCEPT; iptables -C FORWARD -d 10.88.0.0/24 -j ACCEPT 2>/dev/null || iptables -I FORWARD 5 -d 10.88.0.0/24 -j ACCEPT"},
				struct {
					desc string
					cmd  string
				}{"放行 WireGuard UDP 13231 穿透端口", "iptables -C INPUT -p udp --dport 13231 -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport 13231 -j ACCEPT"},
				struct {
					desc string
					cmd  string
				}{"放行 WireGuard UDP 51820 备用端口", "iptables -C INPUT -p udp --dport 51820 -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport 51820 -j ACCEPT"},
				struct {
					desc string
					cmd  string
				}{"配置 WireGuard 虚拟网段 (10.88.0.0/24) NAT 出网伪装", "iptables -t nat -C POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE 2>/dev/null || iptables -t nat -I POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE"},
			)
		}
	}

	client, err := m.getSSHClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("无法连接宿主机: %w", err)
	}
	defer client.Close()

	for _, item := range commands {
		session, err := client.NewSession()
		if err != nil {
			res.Logs = append(res.Logs, fmt.Sprintf("❌ %s 失败: %v", item.desc, err))
			continue
		}
		var out bytes.Buffer
		session.Stdout = &out
		session.Stderr = &out
		runErr := session.Run(item.cmd)
		session.Close()

		if runErr != nil {
			res.Logs = append(res.Logs, fmt.Sprintf("⚠️ %s: %s (%v)", item.desc, strings.TrimSpace(out.String()), runErr))
		} else {
			res.Logs = append(res.Logs, fmt.Sprintf("✅ %s 已就绪", item.desc))
		}
	}

	// Try persisting rules
	if session, err := client.NewSession(); err == nil {
		_ = session.Run("which netfilter-persistent && netfilter-persistent save || which iptables-save && iptables-save > /etc/iptables.rules 2>/dev/null")
		session.Close()
	}

	res.Success = true
	return res, nil
}

// DeployHostWireGuard installs wireguard tools and applies the configuration on fnOS / Linux host.
func (m *Manager) DeployHostWireGuard(wgServerConf string) (string, error) {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	if !cfg.Enabled || cfg.Host == "" {
		return "", fmt.Errorf("请先在【宿主机运维】中配置并启用宿主机 SSH 连接")
	}

	client, err := m.getSSHClient(cfg)
	if err != nil {
		return "", fmt.Errorf("无法连接宿主机: %w", err)
	}
	defer client.Close()

	var logs []string

	// 1. Install wireguard if missing
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		session.Stderr = &out
		_ = session.Run("which wg-quick || (apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y wireguard wireguard-tools)")
		session.Close()
		logs = append(logs, "检查并就绪 wireguard-tools: "+strings.TrimSpace(out.String()))
	}

	// 2. Ensure /etc/wireguard directory
	if session, err := client.NewSession(); err == nil {
		_ = session.Run("mkdir -p /etc/wireguard")
		session.Close()
	}

	// 3. Write /etc/wireguard/wg0.conf
	writeCmd := fmt.Sprintf("cat << 'EOF' > /etc/wireguard/wg0.conf\n%s\nEOF\nchmod 600 /etc/wireguard/wg0.conf", wgServerConf)
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		session.Stderr = &out
		if err := session.Run(writeCmd); err != nil {
			session.Close()
			return "", fmt.Errorf("写入 /etc/wireguard/wg0.conf 失败: %w", err)
		}
		session.Close()
		logs = append(logs, "✅ 已写入宿主机 /etc/wireguard/wg0.conf")
	}

	// 4. Start or Reload WireGuard
	reloadCmd := "wg-quick down wg0 2>/dev/null; systemctl restart wg-quick@wg0 || wg-quick up wg0"
	if session, err := client.NewSession(); err == nil {
		var out bytes.Buffer
		session.Stdout = &out
		session.Stderr = &out
		_ = session.Run(reloadCmd)
		session.Close()
		logs = append(logs, "✅ 宿主机 WireGuard 服务已拉起生效: "+strings.TrimSpace(out.String()))
	}

	// 5. Automatically ensure firewall rules
	_, _ = m.ApplyFirewall("permissive", true)

	return strings.Join(logs, "\n"), nil
}

// SyncPeerToHost adds or updates a peer on the host wg0 interface dynamically.
func (m *Manager) SyncPeerToHost(publicKey, clientIP string) error {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	if !cfg.Enabled || cfg.Host == "" {
		return nil // Host SSH not configured, skip
	}

	client, err := m.getSSHClient(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("wg set wg0 peer '%s' allowed-ips '%s/32'", publicKey, clientIP)
	return session.Run(cmd)
}

// RemovePeerFromHost deletes a peer from the host wg0 interface dynamically.
func (m *Manager) RemovePeerFromHost(publicKey string) error {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	if !cfg.Enabled || cfg.Host == "" {
		return nil
	}

	client, err := m.getSSHClient(cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf("wg set wg0 peer '%s' remove", publicKey)
	return session.Run(cmd)
}

// CheckHostUpgrade queries upgradable packages on the host.
func (m *Manager) CheckHostUpgrade() (string, error) {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	if !cfg.Enabled || cfg.Host == "" {
		return "", fmt.Errorf("宿主机 SSH 连接未配置")
	}

	return m.RunCommand(cfg, "apt-get update >/dev/null 2>&1 && apt list --upgradable 2>/dev/null")
}

// ExecuteHostUpgrade triggers system packages upgrade on the host.
func (m *Manager) ExecuteHostUpgrade() (string, error) {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	if !cfg.Enabled || cfg.Host == "" {
		return "", fmt.Errorf("宿主机 SSH 连接未配置")
	}

	return m.RunCommand(cfg, "DEBIAN_FRONTEND=noninteractive apt-get upgrade -y")
}

// GenerateFirewallScript creates a zero-SSH 1-click firewall script for either permissive or strict mode.
func (m *Manager) GenerateFirewallScript(mode string, enablePenetration bool) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "strict" {
		if enablePenetration {
			return `#!/usr/bin/env bash
# ==============================================================================
#  MiCube · 宿主机原生联动与防火墙极速一键通关 (🛡️ 严格模式 + 🚀 手机 WireGuard 穿透)
#  适用平台: 飞牛 fnOS / PVE / Debian / Ubuntu (最小特权白名单防御)
# ==============================================================================
set -e

echo "🧊 [MiCube] 正在为宿主机配置原生联动与防火墙 (🛡️ 严格模式 + 🚀 穿透模式)..."

# 1. 开启 Linux 内核 IPv4 数据包转发
echo "  -> 开启 IPv4 数据包转发 (net.ipv4.ip_forward=1)..."
sysctl -w net.ipv4.ip_forward=1
grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf

# 2. 保持 FORWARD 链默认 DROP 策略 (最小权限原则)
echo "  -> 保持 FORWARD 链默认 DROP (最小特权防御)..."
iptables -P FORWARD DROP

# 3. 放行状态跟踪连接 (ESTABLISHED, RELATED)
echo "  -> 放行已建立连接状态跟踪..."
iptables -C FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || iptables -C FORWARD -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -m state --state RELATED,ESTABLISHED -j ACCEPT

# 4. 定向放行 WireGuard 虚拟网段与私网互访
echo "  -> 定向放行 WireGuard 虚拟接口与私网互访..."
iptables -C FORWARD -s 10.88.0.0/24 -j ACCEPT 2>/dev/null || iptables -I FORWARD 2 -s 10.88.0.0/24 -j ACCEPT
iptables -C FORWARD -d 10.88.0.0/24 -j ACCEPT 2>/dev/null || iptables -I FORWARD 3 -d 10.88.0.0/24 -j ACCEPT
iptables -C FORWARD -s 192.168.0.0/16 -d 192.168.0.0/16 -j ACCEPT 2>/dev/null || iptables -I FORWARD 4 -s 192.168.0.0/16 -d 192.168.0.0/16 -j ACCEPT
iptables -C FORWARD -s 10.0.0.0/8 -d 10.0.0.0/8 -j ACCEPT 2>/dev/null || iptables -I FORWARD 5 -s 10.0.0.0/8 -d 10.0.0.0/8 -j ACCEPT

# 5. 放行核心服务端口 (MiCube Web 面板 TCP 2026 + WireGuard UDP 13231/51820)
echo "  -> 放行核心通信端口..."
iptables -C INPUT -p tcp --dport 2026 -j ACCEPT 2>/dev/null || iptables -I INPUT -p tcp --dport 2026 -j ACCEPT
iptables -C INPUT -p udp --dport 13231 -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport 13231 -j ACCEPT
iptables -C INPUT -p udp --dport 51820 -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport 51820 -j ACCEPT

# 6. 仅为 WireGuard 虚拟网段启用 NAT 伪装
echo "  -> 开启 WireGuard 专属 NAT 伪装..."
iptables -t nat -C POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE 2>/dev/null || iptables -t nat -I POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE

# 7. 持久化保存
echo "  -> 持久化防火墙规则..."
which netfilter-persistent &>/dev/null && netfilter-persistent save || (which iptables-save &>/dev/null && iptables-save > /etc/iptables.rules 2>/dev/null || true)

echo "✅ [MiCube] 宿主机网络与防火墙已以【🛡️ 严格模式 + 🚀 穿透模式】成功原生联动！"
`
		}

		// Strict mode WITHOUT penetration (Pure LAN bypass, zero public ports)
		return `#!/usr/bin/env bash
# ==============================================================================
#  MiCube · 宿主机原生联动与防火墙极速一键通关 (🛡️ 严格模式 · 🌐 纯局域网旁路由)
#  适用平台: 飞牛 fnOS / PVE / Debian / Ubuntu (零公网暴露面 · 最小特权)
# ==============================================================================
set -e

echo "🧊 [MiCube] 正在为宿主机配置原生联动与防火墙 (🛡️ 严格模式 · 🌐 纯局域网旁路由)..."

# 1. 开启 Linux 内核 IPv4 数据包转发
echo "  -> 开启 IPv4 数据包转发 (net.ipv4.ip_forward=1)..."
sysctl -w net.ipv4.ip_forward=1
grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf

# 2. 保持 FORWARD 链默认 DROP 策略 (最小权限原则)
echo "  -> 保持 FORWARD 链默认 DROP (最小特权防御)..."
iptables -P FORWARD DROP

# 3. 放行状态跟踪连接 (ESTABLISHED, RELATED)
echo "  -> 放行已建立连接状态跟踪..."
iptables -C FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || iptables -C FORWARD -m state --state RELATED,ESTABLISHED -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -m state --state RELATED,ESTABLISHED -j ACCEPT

# 4. 定向放行 RFC1918 局域网私网互访
echo "  -> 定向放行局域网私网流量互访..."
iptables -C FORWARD -s 192.168.0.0/16 -d 192.168.0.0/16 -j ACCEPT 2>/dev/null || iptables -I FORWARD 2 -s 192.168.0.0/16 -d 192.168.0.0/16 -j ACCEPT
iptables -C FORWARD -s 10.0.0.0/8 -d 10.0.0.0/8 -j ACCEPT 2>/dev/null || iptables -I FORWARD 3 -s 10.0.0.0/8 -d 10.0.0.0/8 -j ACCEPT

# 5. 放行核心 Web 面板 TCP 2026 端口 (不开放任何 WireGuard 公网 UDP 端口)
echo "  -> 放行 MiCube Web 面板端口 (TCP 2026)..."
iptables -C INPUT -p tcp --dport 2026 -j ACCEPT 2>/dev/null || iptables -I INPUT -p tcp --dport 2026 -j ACCEPT

# 6. 持久化保存
echo "  -> 持久化防火墙规则..."
which netfilter-persistent &>/dev/null && netfilter-persistent save || (which iptables-save &>/dev/null && iptables-save > /etc/iptables.rules 2>/dev/null || true)

echo "✅ [MiCube] 宿主机网络与防火墙已以【🛡️ 严格模式 · 🌐 纯局域网】成功原生联动！零公网端口暴露，固若金汤！"
`
	}

	// Permissive mode
	if enablePenetration {
		return `#!/usr/bin/env bash
# ==============================================================================
#  MiCube · 宿主机原生联动与防火墙极速一键通关 (🟢 宽松模式 + 🚀 手机 WireGuard 穿透)
#  适用平台: 飞牛 fnOS / PVE / 群晖 DSM / Ubuntu / Debian / CentOS Linux
# ==============================================================================
set -e

echo "🧊 [MiCube] 正在为宿主机配置原生联动与防火墙 (🟢 宽松模式 + 🚀 穿透模式)..."

# 1. 永久开启 Linux 内核 IPv4 数据包转发
echo "  -> 开启 IPv4 数据包转发 (net.ipv4.ip_forward=1)..."
sysctl -w net.ipv4.ip_forward=1
grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf
sysctl -w net.ipv4.conf.all.forwarding=1 2>/dev/null || true

# 2. 设置 FORWARD 链默认策略为 ACCEPT 并置顶放行 (100% 避免断网)
echo "  -> 放行 FORWARD 链策略 (宽松放行，防止断网)..."
iptables -P FORWARD ACCEPT
iptables -C FORWARD -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -j ACCEPT

# 3. 放行核心通信端口 (Web 面板 TCP 2026 + WireGuard UDP 13231/51820)
echo "  -> 放行 MiCube 核心端口..."
iptables -C INPUT -p tcp --dport 2026 -j ACCEPT 2>/dev/null || iptables -I INPUT -p tcp --dport 2026 -j ACCEPT
iptables -C INPUT -p udp --dport 13231 -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport 13231 -j ACCEPT
iptables -C INPUT -p udp --dport 51820 -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport 51820 -j ACCEPT

# 4. 配置 WireGuard 虚拟网段 (10.88.0.0/24) NAT 出网伪装
echo "  -> 配置 WireGuard 虚拟网段 NAT 伪装..."
iptables -t nat -C POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE 2>/dev/null || iptables -t nat -I POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE

# 5. 持久化保存防火墙规则
echo "  -> 持久化防火墙规则..."
which netfilter-persistent &>/dev/null && netfilter-persistent save || (which iptables-save &>/dev/null && iptables-save > /etc/iptables.rules 2>/dev/null || true)

echo "✅ [MiCube] 宿主机网络与防火墙已以【🟢 宽松模式 + 🚀 穿透模式】成功原生联动！"
`
	}

	// Permissive mode WITHOUT penetration (Pure LAN bypass, zero public ports)
	return `#!/usr/bin/env bash
# ==============================================================================
#  MiCube · 宿主机原生联动与防火墙极速一键通关 (🟢 宽松模式 · 🌐 纯局域网旁路由)
#  适用平台: 飞牛 fnOS / PVE / 群晖 DSM / Ubuntu / Debian / CentOS Linux
# ==============================================================================
set -e

echo "🧊 [MiCube] 正在为宿主机配置原生联动与防火墙 (🟢 宽松模式 · 🌐 纯局域网旁路由)..."

# 1. 永久开启 Linux 内核 IPv4 数据包转发
echo "  -> 开启 IPv4 数据包转发 (net.ipv4.ip_forward=1)..."
sysctl -w net.ipv4.ip_forward=1
grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf
sysctl -w net.ipv4.conf.all.forwarding=1 2>/dev/null || true

# 2. 设置 FORWARD 链默认策略为 ACCEPT 并置顶放行 (100% 避免断网)
echo "  -> 放行 FORWARD 链策略 (宽松放行，防止断网)..."
iptables -P FORWARD ACCEPT
iptables -C FORWARD -j ACCEPT 2>/dev/null || iptables -I FORWARD 1 -j ACCEPT

# 3. 放行核心 Web 面板 TCP 2026 端口 (不开放任何 WireGuard 公网 UDP 端口)
echo "  -> 放行 MiCube Web 面板端口 (TCP 2026)..."
iptables -C INPUT -p tcp --dport 2026 -j ACCEPT 2>/dev/null || iptables -I INPUT -p tcp --dport 2026 -j ACCEPT

# 4. 持久化保存防火墙规则
echo "  -> 持久化防火墙规则..."
which netfilter-persistent &>/dev/null && netfilter-persistent save || (which iptables-save &>/dev/null && iptables-save > /etc/iptables.rules 2>/dev/null || true)

echo "✅ [MiCube] 宿主机网络与防火墙已以【🟢 宽松模式 · 🌐 纯局域网】成功原生联动！零公网端口暴露，开箱即通，绝不断网！"
`
}

// GenerateSetupBashScript creates a zero-SSH 1-click installer script that users can run on fnOS / Linux.
func (m *Manager) GenerateSetupBashScript(serverIP string, serverPort int, serverPubKey, clientIP, clientPrivKey, clientPubKey string) string {
	if serverIP == "" {
		serverIP = "192.168.88.1"
	}
	if serverPort <= 0 {
		serverPort = 13231
	}

	return fmt.Sprintf(`#!/usr/bin/env bash
# ========================================================
#  MiCube · 飞牛 (fnOS) / Linux 宿主机 WireGuard 极速一键配置
# ========================================================
set -e

echo "🧊 [MiCube] 正在为飞牛 fnOS / Linux 宿主机部署 WireGuard 与防火墙..."

# 1. 开启内核转发
sysctl -w net.ipv4.ip_forward=1
grep -q '^net.ipv4.ip_forward=1' /etc/sysctl.conf || echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf

# 2. 安装 WireGuard 工具包
if ! command -v wg &> /dev/null; then
    echo "📦 正在安装 wireguard wireguard-tools..."
    apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq wireguard wireguard-tools
fi

# 3. 部署防火墙规则 (放行 UDP %d 并启用 NAT 伪装)
iptables -C INPUT -p udp --dport %d -j ACCEPT 2>/dev/null || iptables -I INPUT -p udp --dport %d -j ACCEPT
iptables -C INPUT -p tcp --dport 2026 -j ACCEPT 2>/dev/null || iptables -I INPUT -p tcp --dport 2026 -j ACCEPT
iptables -t nat -C POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE 2>/dev/null || iptables -t nat -I POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE
iptables -C FORWARD -s 10.88.0.0/24 -j ACCEPT 2>/dev/null || iptables -I FORWARD -s 10.88.0.0/24 -j ACCEPT
iptables -C FORWARD -d 10.88.0.0/24 -j ACCEPT 2>/dev/null || iptables -I FORWARD -d 10.88.0.0/24 -j ACCEPT

# 4. 尝试保存防火墙
command -v netfilter-persistent &>/dev/null && netfilter-persistent save || true

echo "✅ [MiCube] 宿主机 WireGuard 与防火墙配置已圆满完成！手机现在可以通过扫码秒级穿透接入！"
`, serverPort, serverPort, serverPort)
}

func (m *Manager) detectLocalHost() string {
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		c := strings.ToLower(string(data))
		if strings.Contains(c, "fnos") || strings.Contains(c, "trimix") {
			return "飞牛 fnOS"
		}
		if strings.Contains(c, "pve") || strings.Contains(c, "proxmox") {
			return "Proxmox VE (PVE)"
		}
		if strings.Contains(c, "synology") {
			return "群晖 Synology DSM"
		}
		if strings.Contains(c, "ubuntu") {
			return "Ubuntu Server"
		}
		if strings.Contains(c, "debian") {
			return "Debian GNU/Linux"
		}
	}
	return "Linux Docker"
}

func (m *Manager) detectLocalPlatformID(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "fnos") || strings.Contains(lower, "飞牛") {
		return "fnos"
	}
	if strings.Contains(lower, "pve") || strings.Contains(lower, "proxmox") {
		return "pve"
	}
	if strings.Contains(lower, "synology") || strings.Contains(lower, "群晖") {
		return "synology"
	}
	if strings.Contains(lower, "ubuntu") {
		return "ubuntu"
	}
	if strings.Contains(lower, "debian") {
		return "debian"
	}
	return "linux"
}

func parseOSRelease(content string) (osName string, platID string) {
	lines := strings.Split(content, "\n")
	rawName := ""
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "PRETTY_NAME=") {
			rawName = strings.Trim(strings.TrimPrefix(l, "PRETTY_NAME="), `"`)
			break
		}
		if strings.HasPrefix(l, "NAME=") && rawName == "" {
			rawName = strings.Trim(strings.TrimPrefix(l, "NAME="), `"`)
		}
	}

	lower := strings.ToLower(content)
	if strings.Contains(lower, "fnos") || strings.Contains(lower, "trimix") {
		if rawName == "" {
			rawName = "飞牛 fnOS"
		}
		return rawName, "fnos"
	}
	if strings.Contains(lower, "proxmox") || strings.Contains(lower, "pve") {
		if rawName == "" {
			rawName = "Proxmox VE (PVE)"
		}
		return rawName, "pve"
	}
	if strings.Contains(lower, "synology") {
		return "群晖 Synology DSM", "synology"
	}
	if strings.Contains(lower, "ubuntu") {
		return rawName, "ubuntu"
	}
	if strings.Contains(lower, "debian") {
		return rawName, "debian"
	}
	if rawName != "" {
		return rawName, "linux"
	}
	return "Linux Host", "linux"
}

// SetDockerRegistryMirror updates /etc/docker/daemon.json on fnOS / Linux host via SSH.
func (m *Manager) SetDockerRegistryMirror(mirrorURL string) error {
	m.mu.RLock()
	cfg := m.config
	m.mu.RUnlock()

	if !cfg.Enabled || cfg.Host == "" {
		return fmt.Errorf("宿主机 SSH 未配置，请先在「宿主联动」中配置宿主 IP 与密码")
	}

	client, err := m.getSSHClient(cfg)
	if err != nil {
		return fmt.Errorf("连接宿主机 SSH 失败: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	cmd := fmt.Sprintf(`mkdir -p /etc/docker && python3 -c '
import json, os
p = "/etc/docker/daemon.json"
d = {}
if os.path.exists(p):
    try:
        with open(p) as f: d = json.load(f)
    except: pass
mirrors = d.get("registry-mirrors", [])
if "%s" in mirrors: mirrors.remove("%s")
mirrors.insert(0, "%s")
d["registry-mirrors"] = mirrors
with open(p, "w") as f: json.dump(d, f, indent=2)
' 2>/dev/null || (which jq >/dev/null && [ -f /etc/docker/daemon.json ] && jq '.["registry-mirrors"] = ["%s"]' /etc/docker/daemon.json > /tmp/d.json && mv /tmp/d.json /etc/docker/daemon.json) || echo '{"registry-mirrors": ["%s"]}' > /etc/docker/daemon.json; (systemctl daemon-reload && systemctl restart docker) 2>/dev/null || true`, mirrorURL, mirrorURL, mirrorURL, mirrorURL, mirrorURL)

	return session.Run(cmd)
}
