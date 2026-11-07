package ros

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// InterfaceStatus represents an Ethernet or physical port on RouterOS.
type InterfaceStatus struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Running    bool   `json:"running"`
	Disabled   bool   `json:"disabled"`
	Speed      string `json:"speed"`
	FullDuplex bool   `json:"full_duplex"`
	MacAddress string `json:"mac_address"`
	Comment    string `json:"comment,omitempty"`
}

// RouterHardwareInfo holds physical host hardware health and telemetry for RouterOS.
type RouterHardwareInfo struct {
	Connected     bool              `json:"connected"`
	Model         string            `json:"model"`
	Architecture  string            `json:"architecture"`
	ROSVersion    string            `json:"ros_version"`
	Uptime        string            `json:"uptime"`
	CPU           string            `json:"cpu"`
	CPUCount      int               `json:"cpu_count"`
	CPUFrequency  int               `json:"cpu_frequency"`
	CPULoad       int               `json:"cpu_load"`
	FreeMemory    int64             `json:"free_memory"`
	TotalMemory   int64             `json:"total_memory"`
	FreeMemoryMB  float64           `json:"free_memory_mb"`
	TotalMemoryMB float64           `json:"total_memory_mb"`
	Version       string            `json:"version"`
	Arch          string            `json:"arch"`
	MemoryUsedPct float64           `json:"memory_used_pct"`
	FreeHDDSpace  int64             `json:"free_hdd_space"`
	TotalHDDSpace int64             `json:"total_hdd_space"`
	Temperature   float64           `json:"temperature"`
	Voltage       float64           `json:"voltage"`
	WANInterface  string            `json:"wan_interface"`
	WANIP         string            `json:"wan_ip"`
	LANInterface  string            `json:"lan_interface"`
	LANIP         string            `json:"lan_ip"`
	Interfaces    []InterfaceStatus `json:"interfaces"`
	LastUpdated   int64             `json:"last_updated"`
}

var (
	hwCacheLock sync.RWMutex
	cachedHW    *RouterHardwareInfo
	lastHWFetch time.Time
)

// GetRouterHardwareInfo collects live hardware telemetry, CPU load, memory, and interface status.
func (m *Manager) GetRouterHardwareInfo() (*RouterHardwareInfo, error) {
	hwCacheLock.RLock()
	if cachedHW != nil && time.Since(lastHWFetch) < 2*time.Second {
		defer hwCacheLock.RUnlock()
		return cachedHW, nil
	}
	hwCacheLock.RUnlock()

	info := &RouterHardwareInfo{
		Connected:   false,
		Interfaces:  make([]InterfaceStatus, 0),
		LastUpdated: time.Now().Unix(),
	}

	m.mu.RLock()
	host := m.config.Host
	m.mu.RUnlock()

	if host == "" {
		return info, nil
	}

	// 1. Fetch /system/resource
	resData, err := m.QueryROS("/system/resource")
	if err != nil {
		return info, err
	}
	info.Connected = true

	var res map[string]any
	if err := json.Unmarshal(resData, &res); err == nil {
		if b, ok := res["board-name"].(string); ok {
			info.Model = b
		}
		if a, ok := res["architecture-name"].(string); ok {
			info.Architecture = a
		}
		if v, ok := res["version"].(string); ok {
			info.ROSVersion = v
		}
		if u, ok := res["uptime"].(string); ok {
			info.Uptime = u
		}
		if c, ok := res["cpu"].(string); ok {
			info.CPU = c
		}
		if cc, ok := res["cpu-count"].(string); ok {
			info.CPUCount, _ = strconv.Atoi(cc)
		}
		if cf, ok := res["cpu-frequency"].(string); ok {
			info.CPUFrequency, _ = strconv.Atoi(cf)
		}
		if cl, ok := res["cpu-load"].(string); ok {
			info.CPULoad, _ = strconv.Atoi(cl)
		}
		if fm, ok := res["free-memory"].(string); ok {
			info.FreeMemory, _ = strconv.ParseInt(fm, 10, 64)
		}
		if tm, ok := res["total-memory"].(string); ok {
			info.TotalMemory, _ = strconv.ParseInt(tm, 10, 64)
		}
		if fh, ok := res["free-hdd-space"].(string); ok {
			info.FreeHDDSpace, _ = strconv.ParseInt(fh, 10, 64)
		}
		if th, ok := res["total-hdd-space"].(string); ok {
			info.TotalHDDSpace, _ = strconv.ParseInt(th, 10, 64)
		}

		if info.TotalMemory > 0 {
			used := info.TotalMemory - info.FreeMemory
			pct := float64(used) / float64(info.TotalMemory) * 100
			info.MemoryUsedPct = float64(int(pct*10)) / 10.0
		}
		info.FreeMemoryMB = float64(info.FreeMemory) / (1024 * 1024)
		info.TotalMemoryMB = float64(info.TotalMemory) / (1024 * 1024)
		info.Version = info.ROSVersion
		info.Arch = info.Architecture
	}

	// 2. Fetch /system/health (Optional on CHR/VM)
	if hData, err := m.QueryROS("/system/health"); err == nil {
		var hMap map[string]any
		if json.Unmarshal(hData, &hMap) == nil {
			if t, ok := hMap["temperature"].(string); ok {
				info.Temperature, _ = strconv.ParseFloat(t, 64)
			} else if t, ok := hMap["cpu-temperature"].(string); ok {
				info.Temperature, _ = strconv.ParseFloat(t, 64)
			}
			if v, ok := hMap["voltage"].(string); ok {
				info.Voltage, _ = strconv.ParseFloat(v, 64)
			}
		}
	}

	// 3. Fetch /ip/address to identify LAN and WAN IP
	if addrData, err := m.QueryROS("/ip/address"); err == nil {
		var addrs []map[string]any
		if json.Unmarshal(addrData, &addrs) == nil {
			for _, a := range addrs {
				addrStr, _ := a["address"].(string)
				iface, _ := a["interface"].(string)
				disabled, _ := a["disabled"].(string)
				if disabled == "true" {
					continue
				}

				ipPart := strings.Split(addrStr, "/")[0]
				parsedIP := net.ParseIP(ipPart)
				if parsedIP == nil {
					continue
				}

				// Check for WAN: pppoe, or public/CGNAT IP, or interface named WAN
				lowerIface := strings.ToLower(iface)
				if strings.Contains(lowerIface, "pppoe") || strings.Contains(lowerIface, "wan") ||
					(!parsedIP.IsPrivate() && !parsedIP.IsLoopback() && !parsedIP.IsLinkLocalUnicast()) {
					if info.WANIP == "" {
						info.WANIP = ipPart
						info.WANInterface = iface
					}
				}

				// Check for LAN: bridge or matching host IP
				if strings.Contains(lowerIface, "bridge") || strings.Contains(lowerIface, "lan") || ipPart == host {
					if info.LANIP == "" {
						info.LANIP = ipPart
						info.LANInterface = iface
					}
				}
			}
		}
	}

	// Fallback LAN IP to host config if not resolved
	if info.LANIP == "" {
		info.LANIP = host
		info.LANInterface = "bridge1"
	}

	// 4. Fetch /interface/ethernet for port status matrix
	if ethData, err := m.QueryROS("/interface/ethernet"); err == nil {
		var ethList []map[string]any
		if json.Unmarshal(ethData, &ethList) == nil {
			for _, eth := range ethList {
				name, _ := eth["name"].(string)
				running := eth["running"] == "true"
				disabled := eth["disabled"] == "true"
				mac, _ := eth["mac-address"].(string)
				speed, _ := eth["speed"].(string)
				comment, _ := eth["comment"].(string)
				fullDuplex := eth["full-duplex"] == "true"

				if speed == "" || speed == "None" {
					if running {
						speed = "2.5Gbps / 10G VirtIO"
						fullDuplex = true
					} else {
						speed = "未连接"
					}
				}

				info.Interfaces = append(info.Interfaces, InterfaceStatus{
					Name:       name,
					Type:       "ether",
					Running:    running,
					Disabled:   disabled,
					Speed:      speed,
					FullDuplex: fullDuplex,
					MacAddress: mac,
					Comment:    comment,
				})
			}
		}
	}

	hwCacheLock.Lock()
	cachedHW = info
	lastHWFetch = time.Now()
	hwCacheLock.Unlock()

	return info, nil
}

// EnsureDevicePolicyMangleRule ensures the RouterOS prerouting mangle rule for per-device policy routing exists.
func (m *Manager) EnsureDevicePolicyMangleRule() error {
	m.mu.RLock()
	host := m.config.Host
	m.mu.RUnlock()
	if host == "" {
		return fmt.Errorf("RouterOS not configured")
	}

	data, err := m.QueryROS("/ip/firewall/mangle")
	if err != nil {
		return err
	}

	var mangles []map[string]any
	_ = json.Unmarshal(data, &mangles)

	ruleComment := "CubeX Per-Device Policy Routing"
	for _, mg := range mangles {
		comm, _ := mg["comment"].(string)
		if comm == ruleComment {
			return nil // Already present and active
		}
	}

	// Create Mangle Rule
	payload := map[string]any{
		"chain":            "prerouting",
		"action":           "mark-routing",
		"new-routing-mark": "cubex",
		"passthrough":      "yes",
		"src-address-list": "cubex-clients",
		"dst-address-list": "!LAN_SUBNET",
		"comment":          ruleComment,
	}

	_, err = m.PutROS("/ip/firewall/mangle", payload)
	return err
}

// GetProxiedClients returns all IP addresses currently present in cubex-clients address list.
func (m *Manager) GetProxiedClients() ([]string, error) {
	m.mu.RLock()
	host := m.config.Host
	m.mu.RUnlock()
	if host == "" {
		return nil, fmt.Errorf("RouterOS not configured")
	}

	data, err := m.QueryROS("/ip/firewall/address-list")
	if err != nil {
		return nil, err
	}

	var list []map[string]any
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}

	var ips []string
	for _, item := range list {
		lst, _ := item["list"].(string)
		disabled, _ := item["disabled"].(string)
		if lst == "cubex-clients" && disabled != "true" {
			if addr, ok := item["address"].(string); ok && addr != "" {
				ips = append(ips, strings.TrimSpace(addr))
			}
		}
	}

	return ips, nil
}

// SetClientProxy adds or removes a specific IP from cubex-clients address list.
func (m *Manager) SetClientProxy(ip string, enable bool, comment string) error {
	ip = strings.TrimSpace(ip)
	if ip == "" || net.ParseIP(ip) == nil {
		return fmt.Errorf("invalid IP: %s", ip)
	}

	m.mu.RLock()
	host := m.config.Host
	m.mu.RUnlock()
	if host == "" {
		return fmt.Errorf("RouterOS not configured")
	}

	// Always ensure the Mangle rule exists before adding clients
	_ = m.EnsureDevicePolicyMangleRule()

	data, err := m.QueryROS("/ip/firewall/address-list")
	if err != nil {
		return err
	}

	var list []map[string]any
	_ = json.Unmarshal(data, &list)

	var existingID string
	for _, item := range list {
		lst, _ := item["list"].(string)
		addr, _ := item["address"].(string)
		id, _ := item[".id"].(string)
		if lst == "cubex-clients" && addr == ip {
			existingID = id
			break
		}
	}

	if enable {
		if existingID != "" {
			return nil // Already active
		}
		if comment == "" {
			comment = "CubeX Accelerated Client"
		}
		payload := map[string]any{
			"list":    "cubex-clients",
			"address": ip,
			"comment": comment,
		}
		_, err = m.PutROS("/ip/firewall/address-list", payload)
		return err
	}

	// Disable: Delete from address-list
	if existingID != "" {
		return m.DeleteROS("/ip/firewall/address-list/" + existingID)
	}

	return nil
}

// BatchSetClientProxy enables or disables proxy for a batch of IPs.
func (m *Manager) BatchSetClientProxy(ips []string, enable bool) error {
	for _, ip := range ips {
		_ = m.SetClientProxy(ip, enable, "")
	}
	return nil
}
