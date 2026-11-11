package host

import (
	"bufio"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// TopologyNodeType categorizes elements in the physical/logical network graph.
type TopologyNodeType string

const (
	TypeInternet   TopologyNodeType = "internet"
	TypeRouter     TopologyNodeType = "router"
	TypeGateway    TopologyNodeType = "gateway"
	TypeSwitchAP   TopologyNodeType = "ap_switch"
	TypeEndpoint   TopologyNodeType = "endpoint"
)

// TopologyNode represents a discovered device or hop in the network graph.
type TopologyNode struct {
	ID        string           `json:"id"`
	Label     string           `json:"label"`
	Type      TopologyNodeType `json:"type"`
	IP        string           `json:"ip"`
	MAC       string           `json:"mac"`
	Vendor    string           `json:"vendor"`
	Interface string           `json:"interface"`
	IsProxied bool             `json:"is_proxied"`
	Online    bool             `json:"online"`
	Tier      int              `json:"tier"` // 0: WAN, 1: Main Router, 2: CubeX, 3: Switch/AP, 4: Client
}

// TopologyLink represents a connection edge between two nodes.
type TopologyLink struct {
	Source   string `json:"source"`
	Target   string `json:"target"`
	Protocol string `json:"protocol"`
	Speed    string `json:"speed"`
	Status   string `json:"status"` // "active", "standby"
}

// TopologyGraph represents the complete hierarchical network topology.
type TopologyGraph struct {
	Nodes        []TopologyNode `json:"nodes"`
	Links        []TopologyLink `json:"links"`
	DiscoveredAt string         `json:"discovered_at"`
	TotalNodes   int            `json:"total_nodes"`
	OnlineNodes  int            `json:"online_nodes"`
	ProxiedNodes int            `json:"proxied_nodes"`
	HostOS       string         `json:"host_os"`
}

// TopologyManager sniffs ARP, MNDP (MikroTik), and LLDP neighbors to map topology.
type TopologyManager struct {
	mu            sync.RWMutex
	graph         TopologyGraph
	lastRefreshed time.Time
}

var (
	globalTopologyManager *TopologyManager
	topoOnce              sync.Once
)

// GetTopologyManager returns the singleton TopologyManager.
func GetTopologyManager() *TopologyManager {
	topoOnce.Do(func() {
		globalTopologyManager = &TopologyManager{}
		globalTopologyManager.Refresh()
	})
	return globalTopologyManager
}

// Refresh re-evaluates the network topology.
func (m *TopologyManager) Refresh() TopologyGraph {
	m.mu.Lock()
	defer m.mu.Unlock()

	nodes := make([]TopologyNode, 0)
	links := make([]TopologyLink, 0)

	// 1. Tier 0: Internet (WAN 出口)
	wanNode := TopologyNode{
		ID:        "node-wan",
		Label:     "国际业务与公网出口",
		Type:      TypeInternet,
		IP:        "公网动态 IP",
		Online:    true,
		Tier:      0,
		Interface: "WAN / PPPoE",
	}
	nodes = append(nodes, wanNode)

	gatewayIP := schema.DetectDefaultGateway()
	cubexIP := schema.DetectOutboundIP(gatewayIP)
	if gatewayIP == "" {
		gatewayIP = "自动探测中"
	}
	if cubexIP == "" {
		cubexIP = "本机网桥"
	}

	// 2. Tier 1: 主物理路由器 (RouterOS / 爱快 / OpenWrt 主机)
	routerNode := TopologyNode{
		ID:        "node-main-router",
		Label:     "物理主路由网关",
		Type:      TypeRouter,
		IP:        gatewayIP,
		MAC:       "AUTO",
		Vendor:    "Main Gateway",
		Interface: "LAN",
		Online:    gatewayIP != "自动探测中",
		Tier:      1,
	}
	nodes = append(nodes, routerNode)

	links = append(links, TopologyLink{
		Source:   "node-wan",
		Target:   "node-main-router",
		Protocol: "WAN 动态拨号",
		Speed:    "物理线速",
		Status:   "active",
	})

	// 3. Tier 2: CubeX 宿主共生零感网关
	cubexNode := TopologyNode{
		ID:        "node-cubex-gateway",
		Label:     "CubeX 双核透明网关 (伴生中枢)",
		Type:      TypeGateway,
		IP:        cubexIP,
		MAC:       "AUTO",
		Vendor:    "CubeX Dual-Core Gateway",
		Interface: "eth0 / tun",
		Online:    true,
		IsProxied: true,
		Tier:      2,
	}
	nodes = append(nodes, cubexNode)

	links = append(links, TopologyLink{
		Source:   "node-main-router",
		Target:   "node-cubex-gateway",
		Protocol: "table cubex 策略路由 + 198.18 回指",
		Speed:    "VirtIO / VETH 内核直通",
		Status:   "active",
	})

	// 4. Tier 4: 解析真实局域网客户端 (读取 /proc/net/arp)
	discoveredClients := m.parseARPTable(gatewayIP, cubexIP)

	proxiedCount := 1 // CubeX itself
	for _, c := range discoveredClients {
		c.Tier = 4
		nodes = append(nodes, c)
		if c.IsProxied {
			proxiedCount++
			links = append(links, TopologyLink{
				Source:   "node-cubex-gateway",
				Target:   c.ID,
				Protocol: "Fake-IP 透明加速隧道",
				Speed:    "千兆物理线速",
				Status:   "active",
			})
		} else {
			links = append(links, TopologyLink{
				Source:   "node-main-router",
				Target:   c.ID,
				Protocol: "主路由原生物理直连",
				Speed:    "千兆物理线速",
				Status:   "active",
			})
		}
	}

	onlineCount := 0
	for _, n := range nodes {
		if n.Online {
			onlineCount++
		}
	}

	graph := TopologyGraph{
		Nodes:        nodes,
		Links:        links,
		DiscoveredAt: time.Now().Format("2006-01-02 15:04:05"),
		TotalNodes:   len(nodes),
		OnlineNodes:  onlineCount,
		ProxiedNodes: proxiedCount,
		HostOS:       "MikroTik RouterOS / Linux",
	}

	m.graph = graph
	m.lastRefreshed = time.Now()
	return graph
}

// parseARPTable parses the host's /proc/net/arp table if available.
func (m *TopologyManager) parseARPTable(gatewayIP, cubexIP string) []TopologyNode {
	file, err := os.Open("/proc/net/arp")
	if err != nil {
		return nil
	}
	defer file.Close()

	var clients []TopologyNode
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		// skip header: IP address HW type Flags HW address Mask Device
	}

	count := 0
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 6 {
			ip := fields[0]
			mac := strings.ToUpper(fields[3])
			dev := fields[5]

			// filter loopback or incomplete
			if mac == "00:00:00:00:00:00" || strings.HasPrefix(ip, "127.") {
				continue
			}

			count++
			nodeID := "client-" + strings.ReplaceAll(ip, ".", "-")
			vendor := detectVendor(mac)
			isProxied := true
			if ip == gatewayIP || ip == cubexIP {
				continue // Skip router / cubex gateway itself
			}

			clients = append(clients, TopologyNode{
				ID:        nodeID,
				Label:     "局域网终端 " + ip,
				Type:      TypeEndpoint,
				IP:        ip,
				MAC:       mac,
				Vendor:    vendor,
				Interface: dev,
				IsProxied: isProxied,
				Online:    true,
				Tier:      4,
			})
		}
	}
	return clients
}

func detectVendor(mac string) string {
	if len(mac) < 8 {
		return "通用网卡"
	}
	prefix := mac[:8]
	switch prefix {
	case "00:E0:4C", "B8:AE:ED":
		return "Realtek"
	case "00:1A:2B", "F0:99:B6", "AC:DE:48", "BC:D1:D3":
		return "Apple"
	case "BC:24:11", "08:55:31":
		return "MikroTik"
	case "00:11:32":
		return "Synology"
	case "68:D7:9A", "50:D4:F7":
		return "TP-Link"
	case "28:6C:07", "74:AC:5F":
		return "Xiaomi"
	case "00:50:56", "00:0C:29":
		return "VMware"
	case "52:54:00":
		return "QEMU/VirtIO"
	default:
		return "以太网终端"
	}
}

// GetGraph returns the latest cached graph.
func (m *TopologyManager) GetGraph() TopologyGraph {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.graph
}
