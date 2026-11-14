package device

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/netutil"
	"github.com/zyeyefyass/CubeX/internal/ros"
)

// DevicePolicy defines the routing strategy for a device.
type DevicePolicy struct {
	Mode      string `json:"mode"`       // "rule" | "bypass" | "node"
	BoundNode string `json:"bound_node"` // specific node or proxy group
}

// Device represents an identified LAN endpoint.
type Device struct {
	IP        string `json:"ip"`
	MAC       string `json:"mac"`
	HostName  string `json:"hostname"`
	Name      string `json:"name"` // User custom alias or fallback
	IsCustom  bool   `json:"is_custom"`
	Interface string `json:"interface"`
	Source    string `json:"source"`
	Online    bool   `json:"online"`
	LastSeen  int64  `json:"last_seen"`
	Mode      string `json:"mode"`       // "proxy" (走旁路加速) | "direct" (主路由直连) | "rule" | "bypass" | "node"
	IsProxied bool   `json:"is_proxied"`  // true: in cubex-clients or proxy active; false: direct
	BoundNode string `json:"bound_node"` // specific node or proxy group when mode == "node"
}

// DeviceStore stores the persistent aliases and routing policies mapping.
type DeviceStore struct {
	Aliases  map[string]string       `json:"aliases"`  // key can be IP or UPPER(MAC)
	Policies map[string]DevicePolicy `json:"policies"` // key can be IP or UPPER(MAC)
}

// Manager manages LAN device discovery, alias persistence, and RouterOS sync.
type Manager struct {
	rootDir    string
	storeFile  string
	rosMgr     *ros.Manager
	clashURL   string
	mu         sync.RWMutex
	store      DeviceStore
	httpClient *http.Client
}

// NewManager creates a new device Manager.
func NewManager(rootDir string, rosMgr *ros.Manager, clashURL string) *Manager {
	if clashURL == "" {
		clashURL = "http://127.0.0.1:9090"
	}
	m := &Manager{
		rootDir:    rootDir,
		storeFile:  filepath.Join(rootDir, "data", "devices.json"),
		rosMgr:     rosMgr,
		clashURL:   clashURL,
		httpClient: &http.Client{Timeout: 3 * time.Second},
		store:      DeviceStore{Aliases: make(map[string]string), Policies: make(map[string]DevicePolicy)},
	}
	m.loadStore()
	return m
}

func (m *Manager) loadStore() {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := os.ReadFile(m.storeFile)
	if err == nil {
		_ = json.Unmarshal(data, &m.store)
	}
	if m.store.Aliases == nil {
		m.store.Aliases = make(map[string]string)
	}
	if m.store.Policies == nil {
		m.store.Policies = make(map[string]DevicePolicy)
	}
}

func (m *Manager) saveStoreLocked() error {
	_ = os.MkdirAll(filepath.Dir(m.storeFile), 0755)
	data, err := json.MarshalIndent(m.store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.storeFile, data, 0644)
}

// SetAlias updates or deletes a custom name for an IP or MAC address.
func (m *Manager) SetAlias(ip, mac, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name = strings.TrimSpace(name)
	ip = strings.TrimSpace(ip)
	mac = strings.ToUpper(strings.TrimSpace(mac))

	if name == "" {
		if mac != "" {
			delete(m.store.Aliases, mac)
		}
		if ip != "" {
			delete(m.store.Aliases, ip)
		}
	} else {
		if mac != "" {
			// MAC is authoritative and permanent: prevents alias mismatch when DHCP IP changes
			m.store.Aliases[mac] = name
			if ip != "" {
				delete(m.store.Aliases, ip) // Purge stale IP key to prevent collision
			}
		} else if ip != "" {
			m.store.Aliases[ip] = name
		}
	}

	if err := m.saveStoreLocked(); err != nil {
		return err
	}

	// Try to sync alias back to RouterOS DHCP / ARP comment if connected
	if m.rosMgr != nil && mac != "" {
		go func(targetMAC, aliasName string) {
			// Find lease in RouterOS and patch comment
			if data, err := m.rosMgr.QueryROS("/ip/dhcp-server/lease"); err == nil {
				var leases []map[string]any
				if json.Unmarshal(data, &leases) == nil {
					for _, l := range leases {
						lMac, _ := l["mac-address"].(string)
						lId, _ := l[".id"].(string)
						if strings.EqualFold(lMac, targetMAC) && lId != "" {
							_ = m.rosMgr.PatchROS("/ip/dhcp-server/lease/"+lId, map[string]string{
								"comment": aliasName,
							})
						}
					}
				}
			}
		}(mac, name)
	}

	return nil
}

// ResolveName returns the user custom alias for an IP or MAC, or empty string.
func (m *Manager) ResolveName(ip, mac string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if mac != "" {
		if name, ok := m.store.Aliases[strings.ToUpper(mac)]; ok && name != "" {
			return name
		}
	}
	if ip != "" {
		if name, ok := m.store.Aliases[ip]; ok && name != "" {
			return name
		}
	}
	return ""
}

// SetPolicy updates or creates a routing policy for an IP or MAC address.
func (m *Manager) SetPolicy(ip, mac, mode, boundNode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ip = strings.TrimSpace(ip)
	mac = strings.ToUpper(strings.TrimSpace(mac))
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "rule"
	}

	if m.store.Policies == nil {
		m.store.Policies = make(map[string]DevicePolicy)
	}

	policy := DevicePolicy{
		Mode:      mode,
		BoundNode: boundNode,
	}

	if ip != "" {
		m.store.Policies[ip] = policy
	}
	if mac != "" {
		m.store.Policies[mac] = policy
	}

	if err := m.saveStoreLocked(); err != nil {
		return err
	}

	// Sync to RouterOS cubex-clients if connected
	if m.rosMgr != nil && ip != "" && ip != "-" {
		enable := (mode == "proxy" || mode == "rule" || mode == "node")
		aliasName := ""
		if mac != "" {
			aliasName = m.store.Aliases[mac]
		}
		if aliasName == "" {
			aliasName = m.store.Aliases[ip]
		}
		_ = m.rosMgr.SetClientProxy(ip, enable, aliasName)
	}

	return nil
}

// ToggleDeviceProxy switches a single device between CubeX proxy acceleration and router direct.
func (m *Manager) ToggleDeviceProxy(ip, mac string, enable bool) error {
	targetMode := "direct"
	if enable {
		targetMode = "proxy"
	}
	return m.SetPolicy(ip, mac, targetMode, "")
}

// BatchSetDeviceProxy enables or disables proxy for all discovered devices or given IPs.
func (m *Manager) BatchSetDeviceProxy(enable bool) error {
	devices := m.GetDevices()
	for _, d := range devices {
		if d.IP != "" && d.IP != "-" {
			_ = m.ToggleDeviceProxy(d.IP, d.MAC, enable)
		}
	}
	return nil
}

// GetPolicies returns all stored routing policies.
func (m *Manager) GetPolicies() map[string]DevicePolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := make(map[string]DevicePolicy)
	for k, v := range m.store.Policies {
		out[k] = v
	}
	return out
}

// GetDevices returns a comprehensive list of LAN devices discovered across all sources.
func (m *Manager) GetDevices() []Device {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deviceMap := make(map[string]*Device) // key: IP
	macToIP := make(map[string]string)

	now := time.Now().Unix()

	var rosProxiedClients map[string]bool
	if m.rosMgr != nil {
		if proxiedIPs, err := m.rosMgr.GetProxiedClients(); err == nil {
			rosProxiedClients = make(map[string]bool)
			for _, pIP := range proxiedIPs {
				rosProxiedClients[pIP] = true
			}
		}
	}

	// 1. Fetch from RouterOS DHCP server leases (richest info)
	if m.rosMgr != nil {
		if data, err := m.rosMgr.QueryROS("/ip/dhcp-server/lease"); err == nil {
			var leases []map[string]any
			if json.Unmarshal(data, &leases) == nil {
				for _, l := range leases {
					ip, _ := l["address"].(string)
					mac, _ := l["mac-address"].(string)
					host, _ := l["host-name"].(string)
					comment, _ := l["comment"].(string)
					status, _ := l["status"].(string)
					if ip == "" {
						continue
					}
					macUpper := strings.ToUpper(mac)
					d := &Device{
						IP:        ip,
						MAC:       macUpper,
						HostName:  host,
						Source:    "RouterOS DHCP",
						Online:    status == "bound",
						LastSeen:  now,
						Interface: "bridge",
					}
					if comment != "" {
						d.Name = comment
						d.IsCustom = true
					}
					deviceMap[ip] = d
					if macUpper != "" {
						macToIP[macUpper] = ip
					}
				}
			}
		}

		// 2. Fetch from RouterOS ARP table
		if data, err := m.rosMgr.QueryROS("/ip/arp"); err == nil {
			var arps []map[string]any
			if json.Unmarshal(data, &arps) == nil {
				for _, a := range arps {
					ip, _ := a["address"].(string)
					mac, _ := a["mac-address"].(string)
					comment, _ := a["comment"].(string)
					iface, _ := a["interface"].(string)
					if ip == "" {
						continue
					}
					macUpper := strings.ToUpper(mac)
					if existing, ok := deviceMap[ip]; ok {
						if existing.MAC == "" {
							existing.MAC = macUpper
						}
						if existing.Interface == "" {
							existing.Interface = iface
						}
						if existing.Name == "" && comment != "" {
							existing.Name = comment
							existing.IsCustom = true
						}
					} else {
						d := &Device{
							IP:        ip,
							MAC:       macUpper,
							Interface: iface,
							Source:    "RouterOS ARP",
							Online:    true,
							LastSeen:  now,
						}
						if comment != "" {
							d.Name = comment
							d.IsCustom = true
						}
						deviceMap[ip] = d
						if macUpper != "" {
							macToIP[macUpper] = ip
						}
					}
				}
			}
		}
	}

	// 3. Read local Linux /proc/net/arp
	if f, err := os.Open("/proc/net/arp"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		isHeader := true
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if isHeader {
				isHeader = false
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 6 {
				ip := parts[0]
				macUpper := strings.ToUpper(parts[3])
				iface := parts[5]
				if macUpper == "00:00:00:00:00:00" || ip == "" {
					continue
				}
				if existing, ok := deviceMap[ip]; ok {
					if existing.MAC == "" {
						existing.MAC = macUpper
					}
					if existing.Interface == "" {
						existing.Interface = iface
					}
				} else {
					deviceMap[ip] = &Device{
						IP:        ip,
						MAC:       macUpper,
						Interface: iface,
						Source:    "Linux ARP",
						Online:    true,
						LastSeen:  now,
					}
					macToIP[macUpper] = ip
				}
			}
		}
	}

	// 4. Inject active connections client IPs from Clash
	req, err := http.NewRequest(http.MethodGet, m.clashURL+"/connections", nil)
	if err == nil {
		if resp, err := m.httpClient.Do(req); err == nil {
			var connData struct {
				Connections []struct {
					Metadata struct {
						SourceIP string `json:"sourceIP"`
					} `json:"metadata"`
				} `json:"connections"`
			}
			if json.NewDecoder(resp.Body).Decode(&connData) == nil {
				for _, c := range connData.Connections {
					srcIP := c.Metadata.SourceIP
					if srcIP != "" && srcIP != "127.0.0.1" && srcIP != "::1" &&
						!strings.HasPrefix(srcIP, "172.17.") &&
						!strings.HasPrefix(srcIP, "172.18.") &&
						!strings.HasPrefix(srcIP, "172.19.") &&
						!strings.HasPrefix(srcIP, "198.18.") {
						if existing, ok := deviceMap[srcIP]; ok {
							existing.Online = true
							existing.LastSeen = now
						} else {
							deviceMap[srcIP] = &Device{
								IP:       srcIP,
								Source:   "活跃链路",
								Online:   true,
								LastSeen: now,
							}
						}
					}
				}
			}
			netutil.DrainAndCloseBody(resp)
		}
	}

	// 5. Apply user custom aliases from local store
	var result []Device
	for ip, d := range deviceMap {
		// Priority 1: Check alias by MAC (permanent and authoritative)
		if d.MAC != "" && d.MAC != "-" {
			if alias, ok := m.store.Aliases[d.MAC]; ok && alias != "" {
				d.Name = alias
				d.IsCustom = true
			}
		}
		// Priority 2: Check alias by IP only if MAC is unknown/missing
		if !d.IsCustom && (d.MAC == "" || d.MAC == "-") {
			if alias, ok := m.store.Aliases[ip]; ok && alias != "" {
				d.Name = alias
				d.IsCustom = true
			}
		}
		// Fallback name
		if d.Name == "" {
			if d.HostName != "" {
				d.Name = d.HostName
			} else {
				d.Name = d.IP
			}
		}

		// Populate routing policy
		if pol, ok := m.store.Policies[ip]; ok {
			d.Mode = pol.Mode
			d.BoundNode = pol.BoundNode
		} else if d.MAC != "" {
			if pol, ok := m.store.Policies[d.MAC]; ok {
				d.Mode = pol.Mode
				d.BoundNode = pol.BoundNode
			}
		}
		if d.Mode == "" {
			d.Mode = "rule"
		}

		// When RouterOS is connected, cubex-clients address list is authoritative for proxy state
		if rosProxiedClients != nil {
			if rosProxiedClients[d.IP] {
				d.IsProxied = true
				d.Mode = "proxy"
			} else {
				d.IsProxied = false
				if d.Mode == "rule" || d.Mode == "proxy" {
					d.Mode = "direct"
				}
			}
		} else {
			d.IsProxied = (d.Mode == "proxy" || d.Mode == "rule" || d.Mode == "node")
		}

		result = append(result, *d)
	}

	// Also add any stored aliases/policies that might not be in ARP currently (offline devices)
	for key, alias := range m.store.Aliases {
		if strings.Contains(key, ":") { // MAC
			if _, exists := macToIP[key]; !exists {
				// Check if an offline device with the exact same alias was already added by IP
				merged := false
				for i := range result {
					if result[i].Name == alias && (result[i].MAC == "-" || result[i].MAC == "") {
						result[i].MAC = key
						merged = true
						break
					}
				}
				if !merged {
					dev := Device{
						IP:       "-",
						MAC:      key,
						Name:     alias,
						IsCustom: true,
						Source:   "已保存离线设备",
						Online:   false,
						LastSeen: 0,
						Mode:     "rule",
					}
					if pol, ok := m.store.Policies[key]; ok {
						dev.Mode = pol.Mode
						dev.BoundNode = pol.BoundNode
					}
					dev.IsProxied = (dev.Mode == "proxy" || dev.Mode == "rule" || dev.Mode == "node")
					result = append(result, dev)
				}
			}
		} else if strings.Contains(key, ".") { // IP
			if _, exists := deviceMap[key]; !exists {
				// Check if an offline device with the exact same alias was already added by MAC
				merged := false
				for i := range result {
					if result[i].Name == alias && (result[i].IP == "-" || result[i].IP == "") {
						result[i].IP = key
						merged = true
						break
					}
				}
				if !merged {
					dev := Device{
						IP:       key,
						MAC:      "-",
						Name:     alias,
						IsCustom: true,
						Source:   "已保存离线设备",
						Online:   false,
						LastSeen: 0,
						Mode:     "rule",
					}
					if pol, ok := m.store.Policies[key]; ok {
						dev.Mode = pol.Mode
						dev.BoundNode = pol.BoundNode
					}
					dev.IsProxied = (dev.Mode == "proxy" || dev.Mode == "rule" || dev.Mode == "node")
					result = append(result, dev)
				}
			}
		}
	}

	// Surface any explicitly persisted device policies that were not yet seen in ARP/leases
	if len(result) == 0 && len(m.store.Policies) > 0 {
		for ip, pol := range m.store.Policies {
			name := ip
			isCustom := false
			if alias, ok := m.store.Aliases[ip]; ok && alias != "" {
				name = alias
				isCustom = true
			}
			result = append(result, Device{
				IP:        ip,
				Name:      name,
				HostName:  name,
				Mode:      pol.Mode,
				BoundNode: pol.BoundNode,
				Source:    "已保存策略",
				IsCustom:  isCustom,
				Online:    false,
				LastSeen:  now,
				IsProxied: pol.Mode == "proxy" || pol.Mode == "rule" || pol.Mode == "node",
			})
		}
	}

	return result
}
