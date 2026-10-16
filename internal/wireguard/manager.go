package wireguard

import (
	"bytes"
	"crypto/ecdh"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/host"
	"github.com/zyeyefyass/CubeX/internal/ros"
)

// ServerConfig stores the WireGuard server configuration on RouterOS.
type ServerConfig struct {
	InterfaceName string `json:"interface_name"` // default "micube-wg"
	ListenPort    int    `json:"listen_port"`    // default 13231
	ServerAddress string `json:"server_address"` // default "10.88.0.1/24"
	PrivateKey    string `json:"private_key"`    // Curve25519 base64
	PublicKey     string `json:"public_key"`     // Curve25519 base64
	EndpointHost  string `json:"endpoint_host"`  // DDNS or WAN IP
	DNS           string `json:"dns"`            // default "10.88.0.1"
	IntranetCIDR  string `json:"intranet_cidr"`  // default "192.168.88.0/24, 10.88.0.0/24"
	CloudDDNS     string `json:"cloud_ddns"`     // auto-detected RouterOS IP Cloud DDNS
	DirectDNSMode string `json:"direct_dns_mode"` // "tunnel_strict" or "local_direct"
}

// Peer represents an authorized client device.
type Peer struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	ClientIP       string `json:"client_ip"` // e.g. "10.88.0.2"
	PrivateKey     string `json:"private_key"`
	PublicKey      string `json:"public_key"`
	AllowedIPsMode string `json:"allowed_ips_mode"` // "intranet" or "full"
	AllowedIPs     string `json:"allowed_ips"`
	CreatedAt      string `json:"created_at"`
	SyncedToROS    bool   `json:"synced_to_ros"`
	ROSError       string `json:"ros_error,omitempty"`
}

// WGCheckItem represents one check item in WireGuard alignment.
type WGCheckItem struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "missing", "warning"
	Value   string `json:"value"`
	Details string `json:"details"`
	Advice  string `json:"advice"`
}

// WGAlignmentReport represents the result of checking WireGuard alignment between MiCube and ROS.
type WGAlignmentReport struct {
	Connected   bool          `json:"connected"`
	ROSHost     string        `json:"ros_host"`
	AllAligned  bool          `json:"all_aligned"`
	Items       []WGCheckItem `json:"items"`
	PeerCount   int           `json:"peer_count"`
	SyncedPeers int           `json:"synced_peers"`
}

// WGAlignResult represents the result of 1-click alignment execution.
type WGAlignResult struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Actions []string `json:"actions"`
	Errors  []string `json:"errors"`
}

// State holds the server configuration and all peers.
type State struct {
	Config ServerConfig `json:"config"`
	Peers  []Peer       `json:"peers"`
}

// Manager manages WireGuard state, key generation, RouterOS sync, and client config profiles.
type Manager struct {
	mu         sync.Mutex
	rootDir     string
	stateFile   string
	state       State
	rosManager  *ros.Manager
	hostManager *host.Manager
	httpClient  *http.Client
}

// SetHostManager sets the host SSH manager for Linux/fnOS synchronization.
func (m *Manager) SetHostManager(h *host.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hostManager = h
}

// GenerateHostWireGuardConfig generates a complete wg0.conf for Linux/fnOS host.
func (m *Manager) GenerateHostWireGuardConfig() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	var sb strings.Builder
	sb.WriteString("# ========================================================\n")
	sb.WriteString("#  MiCube · 飞牛 (fnOS) / Linux 宿主机 WireGuard 配置文件 (/etc/wireguard/wg0.conf)\n")
	sb.WriteString("# ========================================================\n\n")
	sb.WriteString("[Interface]\n")
	sb.WriteString(fmt.Sprintf("Address = %s\n", m.state.Config.ServerAddress))
	sb.WriteString(fmt.Sprintf("ListenPort = %d\n", m.state.Config.ListenPort))
	sb.WriteString(fmt.Sprintf("PrivateKey = %s\n", m.state.Config.PrivateKey))
	sb.WriteString("PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -s 10.88.0.0/24 -j MASQUERADE; sysctl -w net.ipv4.ip_forward=1\n")
	sb.WriteString("PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -s 10.88.0.0/24 -j MASQUERADE\n\n")

	for _, p := range m.state.Peers {
		sb.WriteString("[Peer]\n")
		sb.WriteString(fmt.Sprintf("# 设备备注: %s\n", p.Name))
		sb.WriteString(fmt.Sprintf("PublicKey = %s\n", p.PublicKey))
		sb.WriteString(fmt.Sprintf("AllowedIPs = %s/32\n\n", p.ClientIP))
	}
	return sb.String()
}

// NewManager creates a WireGuard manager instance.
func NewManager(rootDir string, rosMgr *ros.Manager) *Manager {
	m := &Manager{
		rootDir:    rootDir,
		stateFile:  filepath.Join(rootDir, "data", "wireguard.json"),
		rosManager: rosMgr,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
	m.loadState()
	m.ensureServerKeys()
	go m.refreshCloudDDNS()
	return m
}

// GenerateKeyPair generates a Curve25519 (X25519) keypair in standard Base64 encoding.
func GenerateKeyPair() (string, string, error) {
	key, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	privB64 := base64.StdEncoding.EncodeToString(key.Bytes())
	pubB64 := base64.StdEncoding.EncodeToString(key.PublicKey().Bytes())
	return privB64, pubB64, nil
}

// ensureServerKeys generates the server keypair if not present.
func (m *Manager) ensureServerKeys() {
	m.mu.Lock()
	defer m.mu.Unlock()

	changed := false
	if m.state.Config.InterfaceName == "" {
		m.state.Config.InterfaceName = "micube-wg"
		changed = true
	}
	if m.state.Config.ListenPort <= 0 {
		m.state.Config.ListenPort = 13231
		changed = true
	}
	if m.state.Config.ServerAddress == "" {
		m.state.Config.ServerAddress = "10.88.0.1/24"
		changed = true
	}
	if m.state.Config.DNS == "" {
		m.state.Config.DNS = "10.88.0.1"
		changed = true
	}
	if m.state.Config.IntranetCIDR == "" {
		m.state.Config.IntranetCIDR = "192.168.88.0/24, 10.88.0.0/24"
		changed = true
	}
	if m.state.Config.PrivateKey == "" || m.state.Config.PublicKey == "" {
		priv, pub, err := GenerateKeyPair()
		if err == nil {
			m.state.Config.PrivateKey = priv
			m.state.Config.PublicKey = pub
			changed = true
		}
	}
	if changed {
		_ = m.saveStateLocked()
	}
}

// GetInfo returns the current WireGuard server status, detected DDNS, and peer list.
func (m *Manager) GetInfo() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	endpoint := m.state.Config.EndpointHost
	if endpoint == "" && m.state.Config.CloudDDNS != "" {
		endpoint = fmt.Sprintf("%s:%d", m.state.Config.CloudDDNS, m.state.Config.ListenPort)
	} else if endpoint != "" && !strings.Contains(endpoint, ":") {
		endpoint = fmt.Sprintf("%s:%d", endpoint, m.state.Config.ListenPort)
	}

	return map[string]any{
		"config":          m.state.Config,
		"peers":           m.state.Peers,
		"total_peers":     len(m.state.Peers),
		"effective_endpoint": endpoint,
		"cloud_ddns":      m.state.Config.CloudDDNS,
	}
}

// UpdateConfig updates server parameters (ListenPort, EndpointHost, DNS, IntranetCIDR).
func (m *Manager) UpdateConfig(cfg ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cfg.ListenPort > 0 {
		m.state.Config.ListenPort = cfg.ListenPort
	}
	if cfg.EndpointHost != "" {
		m.state.Config.EndpointHost = strings.TrimSpace(cfg.EndpointHost)
	}
	if cfg.DNS != "" {
		m.state.Config.DNS = strings.TrimSpace(cfg.DNS)
	}
	if cfg.IntranetCIDR != "" {
		m.state.Config.IntranetCIDR = strings.TrimSpace(cfg.IntranetCIDR)
	}
	if cfg.ServerAddress != "" {
		m.state.Config.ServerAddress = strings.TrimSpace(cfg.ServerAddress)
	}
	if cfg.DirectDNSMode != "" {
		m.state.Config.DirectDNSMode = strings.TrimSpace(cfg.DirectDNSMode)
	}

	return m.saveStateLocked()
}

// CreatePeer registers a new peer, allocates an IP, creates keys, and attempts to sync to RouterOS.
func (m *Manager) CreatePeer(name string, mode string) (*Peer, string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	name = strings.TrimSpace(name)
	if name == "" {
		name = fmt.Sprintf("设备 #%d", len(m.state.Peers)+1)
	}
	if mode != "full" {
		mode = "intranet"
	}

	privKey, pubKey, err := GenerateKeyPair()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate WireGuard keypair: %w", err)
	}

	clientIP := m.allocateNextClientIPLocked()
	allowedIPs := m.state.Config.IntranetCIDR
	if mode == "full" {
		allowedIPs = "0.0.0.0/0, ::/0"
	}

	peer := Peer{
		ID:             fmt.Sprintf("peer-%d", time.Now().UnixNano()),
		Name:           name,
		ClientIP:       clientIP,
		PrivateKey:     privKey,
		PublicKey:      pubKey,
		AllowedIPsMode: mode,
		AllowedIPs:     allowedIPs,
		CreatedAt:      time.Now().Format("2006-01-02 15:04:05"),
		SyncedToROS:    false,
	}

	// Try syncing to RouterOS
	rosErr := m.syncPeerToROSLocked(peer)
	if rosErr != nil {
		peer.ROSError = rosErr.Error()
		log.Printf("[WireGuard] Peer sync to RouterOS skipped/failed: %v", rosErr)
	} else {
		peer.SyncedToROS = true
	}

	m.state.Peers = append(m.state.Peers, peer)
	_ = m.saveStateLocked()

	confText := m.generateClientConfigLocked(peer)
	return &peer, confText, nil
}

// DeletePeer removes a peer by ID from local state and from RouterOS.
func (m *Manager) DeletePeer(peerID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var target *Peer
	idx := -1
	for i, p := range m.state.Peers {
		if p.ID == peerID {
			target = &p
			idx = i
			break
		}
	}
	if target == nil {
		return fmt.Errorf("peer with ID %s not found", peerID)
	}

	// Remove from ROS
	_ = m.removePeerFromROSLocked(target.PublicKey)

	m.state.Peers = append(m.state.Peers[:idx], m.state.Peers[idx+1:]...)
	return m.saveStateLocked()
}

// GetClientConfig returns the raw WireGuard .conf content for a peer.
func (m *Manager) GetClientConfig(peerID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.state.Peers {
		if p.ID == peerID {
			return m.generateClientConfigLocked(p), nil
		}
	}
	return "", fmt.Errorf("peer %s not found", peerID)
}

// GenerateROSScript returns copy-pasteable RouterOS terminal commands for complete setup.
func (m *Manager) GenerateROSScript() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	cfg := m.state.Config
	var b strings.Builder
	b.WriteString("# 🧊 MiCube RouterOS WireGuard 内网穿透一键配置脚本\n")
	b.WriteString("# 请登录 WinBox 打开 Terminal 或通过 SSH 粘贴执行：\n\n")

	b.WriteString("# 1. 创建 WireGuard 穿透服务端接口\n")
	b.WriteString(fmt.Sprintf(
		"/interface wireguard add name=%s listen-port=%d private-key=\"%s\" comment=\"MiCube WireGuard Server\"\n\n",
		cfg.InterfaceName, cfg.ListenPort, cfg.PrivateKey,
	))

	b.WriteString("# 2. 为 WireGuard 接口绑定穿透虚拟网段 IP\n")
	b.WriteString(fmt.Sprintf(
		"/ip address add address=%s interface=%s comment=\"MiCube WireGuard Subnet\"\n\n",
		cfg.ServerAddress, cfg.InterfaceName,
	))

	b.WriteString("# 3. 防火墙放行 WireGuard 监听端口 (UDP)\n")
	b.WriteString(fmt.Sprintf(
		"/ip firewall filter add chain=input protocol=udp dst-port=%d action=accept place-before=0 comment=\"MiCube WireGuard In\"\n\n",
		cfg.ListenPort,
	))

	b.WriteString("# 4. 启用 RouterOS 自带 Cloud DDNS 动态域名 (内网穿透推荐)\n")
	b.WriteString("/ip cloud set ddns-enabled=yes update-time=yes\n/ip cloud force-update\n\n")

	if len(m.state.Peers) > 0 {
		b.WriteString("# 5. 已授权客户端 Peer 列表\n")
		for _, p := range m.state.Peers {
			b.WriteString(fmt.Sprintf(
				"/interface wireguard peers add interface=%s public-key=\"%s\" allowed-address=%s/32 comment=\"%s\"\n",
				cfg.InterfaceName, p.PublicKey, p.ClientIP, p.Name,
			))
		}
	}

	return b.String()
}

func (m *Manager) getBaseURL(rosCfg map[string]string) string {
	scheme := "https"
	if rosCfg["port"] == "80" || rosCfg["scheme"] == "http" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s:%s/rest", scheme, rosCfg["host"], rosCfg["port"])
}

// InitRouterOSServer invokes RouterOS REST API to configure the interface and IP address automatically.
func (m *Manager) InitRouterOSServer() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.initRouterOSServerLocked()
}

func (m *Manager) initRouterOSServerLocked() error {
	rosCfg := m.rosManager.GetConfig()
	if rosCfg["host"] == "" {
		return fmt.Errorf("RouterOS host not configured")
	}

	cfg := m.state.Config
	baseURL := m.getBaseURL(rosCfg)

	// 1. Check if interface exists
	req, _ := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard", nil)
	req.SetBasicAuth(rosCfg["user"], rosCfg["password"])
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("RouterOS connection failed: %w", err)
	}
	defer resp.Body.Close()

	var wgList []map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&wgList)

	hasIface := false
	for _, item := range wgList {
		if item["name"] == cfg.InterfaceName {
			hasIface = true
			break
		}
	}

	if !hasIface {
		// Create interface using PUT
		createPayload := map[string]any{
			"name":        cfg.InterfaceName,
			"listen-port": cfg.ListenPort,
			"private-key": cfg.PrivateKey,
			"comment":     "MiCube WireGuard Server",
		}
		data, _ := json.Marshal(createPayload)
		cReq, _ := http.NewRequest(http.MethodPut, baseURL+"/interface/wireguard", bytes.NewReader(data))
		cReq.Header.Set("Content-Type", "application/json")
		cReq.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		cResp, cErr := m.httpClient.Do(cReq)
		if cErr != nil {
			return fmt.Errorf("failed to create WG interface on ROS: %w", cErr)
		}
		cResp.Body.Close()
	}

	// 2. Add IP address if not already present
	hasAddr := false
	if aGetReq, err := http.NewRequest(http.MethodGet, baseURL+"/ip/address", nil); err == nil {
		aGetReq.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if aGetResp, err := m.httpClient.Do(aGetReq); err == nil {
			var addrList []map[string]any
			_ = json.NewDecoder(aGetResp.Body).Decode(&addrList)
			aGetResp.Body.Close()
			for _, item := range addrList {
				if item["interface"] == cfg.InterfaceName {
					hasAddr = true
					break
				}
			}
		}
	}
	if !hasAddr {
		addrPayload := map[string]any{
			"address":   cfg.ServerAddress,
			"interface": cfg.InterfaceName,
			"comment":   "MiCube WireGuard Subnet",
		}
		aData, _ := json.Marshal(addrPayload)
		aReq, _ := http.NewRequest(http.MethodPut, baseURL+"/ip/address", bytes.NewReader(aData))
		aReq.Header.Set("Content-Type", "application/json")
		aReq.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if aResp, err := m.httpClient.Do(aReq); err == nil {
			aResp.Body.Close()
		}
	}

	// 2.1 Firewall Filter: Allow UDP listen port
	hasFilter := false
	if fGetReq, err := http.NewRequest(http.MethodGet, baseURL+"/ip/firewall/filter", nil); err == nil {
		fGetReq.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if fGetResp, err := m.httpClient.Do(fGetReq); err == nil {
			var filterList []map[string]any
			_ = json.NewDecoder(fGetResp.Body).Decode(&filterList)
			fGetResp.Body.Close()
			portStr := fmt.Sprintf("%d", cfg.ListenPort)
			for _, item := range filterList {
				if item["chain"] == "input" && item["protocol"] == "udp" && item["dst-port"] == portStr {
					hasFilter = true
					break
				}
			}
		}
	}
	if !hasFilter {
		fPayload := map[string]any{
			"chain":        "input",
			"protocol":     "udp",
			"dst-port":     fmt.Sprintf("%d", cfg.ListenPort),
			"action":       "accept",
			"place-before": "0",
			"comment":      "MiCube WireGuard In",
		}
		fData, _ := json.Marshal(fPayload)
		fReq, _ := http.NewRequest(http.MethodPut, baseURL+"/ip/firewall/filter", bytes.NewReader(fData))
		fReq.Header.Set("Content-Type", "application/json")
		fReq.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if fResp, err := m.httpClient.Do(fReq); err == nil {
			fResp.Body.Close()
		}
	}

	// 3. Sync all peers
	for i := range m.state.Peers {
		if err := m.syncPeerToROSLocked(m.state.Peers[i]); err == nil {
			m.state.Peers[i].SyncedToROS = true
			m.state.Peers[i].ROSError = ""
		} else {
			m.state.Peers[i].SyncedToROS = false
			m.state.Peers[i].ROSError = err.Error()
		}
	}
	_ = m.saveStateLocked()

	// 4. Update Cloud DDNS synchronously so client config has domain immediately
	m.refreshCloudDDNSLocked()

	log.Printf("[WireGuard] RouterOS WireGuard service initialized successfully on %s", rosCfg["host"])
	return nil
}

// ──────────────────────────────────────────────
// Internal Helpers
// ──────────────────────────────────────────────

func (m *Manager) generateClientConfigLocked(peer Peer) string {
	cfg := m.state.Config

	endpoint := cfg.EndpointHost
	if endpoint == "" && cfg.CloudDDNS != "" {
		endpoint = fmt.Sprintf("%s:%d", cfg.CloudDDNS, cfg.ListenPort)
	} else if endpoint != "" && !strings.Contains(endpoint, ":") {
		endpoint = fmt.Sprintf("%s:%d", endpoint, cfg.ListenPort)
	}
	if endpoint == "" {
		endpoint = fmt.Sprintf("your-router-ip-or-ddns:%d", cfg.ListenPort)
	}

	dns := cfg.DNS
	if dns == "" {
		dns = "10.88.0.1"
	}
	if cfg.DirectDNSMode == "local_direct" {
		dns = "10.88.0.1, 223.5.5.5"
	}

	allowedIPs := peer.AllowedIPs
	if allowedIPs == "" {
		allowedIPs = "192.168.88.0/24, 10.88.0.0/24"
	}

	return fmt.Sprintf(`[Interface]
PrivateKey = %s
Address = %s/24
DNS = %s

[Peer]
PublicKey = %s
Endpoint = %s
AllowedIPs = %s
PersistentKeepalive = 25
`, peer.PrivateKey, peer.ClientIP, dns, cfg.PublicKey, endpoint, allowedIPs)
}

func (m *Manager) allocateNextClientIPLocked() string {
	used := make(map[int]bool)
	used[1] = true // 10.88.0.1 is server

	for _, p := range m.state.Peers {
		parts := strings.Split(p.ClientIP, ".")
		if len(parts) == 4 {
			if n, err := strconv.Atoi(parts[3]); err == nil {
				used[n] = true
			}
		}
	}

	// Find lowest unused octet from 2 to 254
	for i := 2; i < 255; i++ {
		if !used[i] {
			return fmt.Sprintf("10.88.0.%d", i)
		}
	}
	return "10.88.0.254"
}

func (m *Manager) syncPeerToROSLocked(peer Peer) error {
	rosCfg := m.rosManager.GetConfig()
	if rosCfg["host"] == "" {
		return fmt.Errorf("ROS host not set")
	}

	baseURL := m.getBaseURL(rosCfg)

	// 1. Check if peer already exists by public key on RouterOS
	getReq, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard/peers", nil)
	if err != nil {
		return err
	}
	getReq.SetBasicAuth(rosCfg["user"], rosCfg["password"])
	getResp, err := m.httpClient.Do(getReq)
	if err != nil {
		return err
	}
	defer getResp.Body.Close()

	var existingPeers []map[string]any
	_ = json.NewDecoder(getResp.Body).Decode(&existingPeers)

	var existingID string
	for _, p := range existingPeers {
		if p["public-key"] == peer.PublicKey {
			if id, ok := p[".id"].(string); ok {
				existingID = id
				break
			}
		}
	}

	if existingID != "" {
		// Update existing peer with PATCH
		patchPayload := map[string]any{
			"interface":       m.state.Config.InterfaceName,
			"allowed-address": peer.ClientIP + "/32",
			"comment":         peer.Name,
		}
		pData, _ := json.Marshal(patchPayload)
		pReq, _ := http.NewRequest(http.MethodPatch, baseURL+"/interface/wireguard/peers/"+existingID, bytes.NewReader(pData))
		pReq.Header.Set("Content-Type", "application/json")
		pReq.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		pResp, err := m.httpClient.Do(pReq)
		if err != nil {
			return err
		}
		defer pResp.Body.Close()
		if pResp.StatusCode >= 400 {
			body, _ := io.ReadAll(pResp.Body)
			return fmt.Errorf("status %d: %s", pResp.StatusCode, string(body))
		}
		return nil
	}

	// 2. Add new peer using PUT (PUT maps to RouterOS "add" CLI command)
	payload := map[string]any{
		"interface":       m.state.Config.InterfaceName,
		"public-key":      peer.PublicKey,
		"allowed-address": peer.ClientIP + "/32",
		"comment":         peer.Name,
	}
	data, _ := json.Marshal(payload)

	req, err := http.NewRequest(http.MethodPut, baseURL+"/interface/wireguard/peers", bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(rosCfg["user"], rosCfg["password"])

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (m *Manager) removePeerFromROSLocked(publicKey string) error {
	rosCfg := m.rosManager.GetConfig()
	if rosCfg["host"] == "" {
		return nil
	}

	baseURL := m.getBaseURL(rosCfg)

	// 1. Find peer ID on ROS
	req, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard/peers", nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(rosCfg["user"], rosCfg["password"])
	resp, err := m.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var peers []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return err
	}

	var targetID string
	for _, p := range peers {
		if p["public-key"] == publicKey {
			if id, ok := p[".id"].(string); ok {
				targetID = id
				break
			}
		}
	}

	if targetID == "" {
		return nil
	}

	// 2. Delete peer by ID
	delReq, _ := http.NewRequest(http.MethodDelete, baseURL+"/interface/wireguard/peers/"+targetID, nil)
	delReq.SetBasicAuth(rosCfg["user"], rosCfg["password"])
	if dResp, err := m.httpClient.Do(delReq); err == nil {
		dResp.Body.Close()
	}
	return nil
}

func (m *Manager) refreshCloudDDNS() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshCloudDDNSLocked()
}

func (m *Manager) refreshCloudDDNSLocked() {
	rosCfg := m.rosManager.GetConfig()
	if rosCfg["host"] == "" {
		return
	}

	baseURL := m.getBaseURL(rosCfg)
	req, err := http.NewRequest(http.MethodGet, baseURL+"/ip/cloud", nil)
	if err != nil {
		return
	}
	req.SetBasicAuth(rosCfg["user"], rosCfg["password"])

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var cloudInfo struct {
		DNSName       string `json:"dns-name"`
		PublicAddress string `json:"public-address"`
		Status        string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cloudInfo); err == nil && cloudInfo.DNSName != "" {
		m.state.Config.CloudDDNS = cloudInfo.DNSName
		_ = m.saveStateLocked()
		log.Printf("[WireGuard] Auto-detected RouterOS Cloud DDNS: %s", cloudInfo.DNSName)
	}
}

// Reload re-reads state from disk.
func (m *Manager) Reload() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.loadState()
}

func (m *Manager) loadState() {
	data, err := os.ReadFile(m.stateFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &m.state)
}

func (m *Manager) saveStateLocked() error {
	_ = os.MkdirAll(filepath.Dir(m.stateFile), 0755)
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.stateFile, data, 0644)
}

// RenamePeer updates the nickname of an authorized peer and syncs to ROS.
func (m *Manager) RenamePeer(publicKey, newName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	newName = strings.TrimSpace(newName)
	if newName == "" {
		return fmt.Errorf("peer name cannot be empty")
	}

	found := false
	var targetPeer Peer
	for i := range m.state.Peers {
		if m.state.Peers[i].PublicKey == publicKey || m.state.Peers[i].ID == publicKey {
			m.state.Peers[i].Name = newName
			targetPeer = m.state.Peers[i]
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("peer not found")
	}

	if err := m.syncPeerToROSLocked(targetPeer); err == nil {
		for i := range m.state.Peers {
			if m.state.Peers[i].PublicKey == targetPeer.PublicKey {
				m.state.Peers[i].SyncedToROS = true
				m.state.Peers[i].ROSError = ""
			}
		}
	}
	return m.saveStateLocked()
}

// InspectWireGuardAlignment checks whether WireGuard interface, IP, firewall, DDNS and peers match RouterOS.
func (m *Manager) InspectWireGuardAlignment() (*WGAlignmentReport, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	rosCfg := m.rosManager.GetConfig()
	report := &WGAlignmentReport{
		Connected:   rosCfg["host"] != "",
		ROSHost:     rosCfg["host"],
		AllAligned:  true,
		Items:       make([]WGCheckItem, 0),
		PeerCount:   len(m.state.Peers),
		SyncedPeers: 0,
	}

	if rosCfg["host"] == "" {
		report.AllAligned = false
		report.Items = append(report.Items, WGCheckItem{
			Key:     "ros_connect",
			Name:    "RouterOS 连接",
			Status:  "missing",
			Value:   "未配置",
			Details: "请在「系统升级」中配置 RouterOS 宿主机 IP 与管理员凭据",
			Advice:  "请配置 RouterOS 连接参数",
		})
		return report, nil
	}

	baseURL := m.getBaseURL(rosCfg)
	cfg := m.state.Config

	// 1. Check Interface (/interface/wireguard)
	ifaceOK := false
	var ifaceVal string
	req, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard", nil)
	if err == nil {
		req.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if resp, err := m.httpClient.Do(req); err == nil {
			var wgList []map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&wgList)
			resp.Body.Close()
			for _, item := range wgList {
				if item["name"] == cfg.InterfaceName {
					listenPort := fmt.Sprintf("%v", item["listen-port"])
					if listenPort == fmt.Sprintf("%d", cfg.ListenPort) {
						ifaceOK = true
						ifaceVal = fmt.Sprintf("name=%s port=%s (正常)", cfg.InterfaceName, listenPort)
					} else {
						ifaceVal = fmt.Sprintf("端口不一致 (ROS: %s, MiCube: %d)", listenPort, cfg.ListenPort)
					}
					break
				}
			}
		}
	}
	if ifaceOK {
		report.Items = append(report.Items, WGCheckItem{
			Key:     "interface",
			Name:    "WireGuard 服务端接口 (/interface/wireguard)",
			Status:  "ok",
			Value:   ifaceVal,
			Details: "RouterOS 上 WireGuard 核心接口已就绪并对齐监听端口。",
			Advice:  "已对齐",
		})
	} else {
		report.AllAligned = false
		report.Items = append(report.Items, WGCheckItem{
			Key:     "interface",
			Name:    "WireGuard 服务端接口 (/interface/wireguard)",
			Status:  "missing",
			Value:   ifaceVal,
			Details: fmt.Sprintf("RouterOS 缺失 %s 接口或端口不一致", cfg.InterfaceName),
			Advice:  "点击「⚡ 一键对齐 WireGuard」自动创建/修复接口",
		})
	}

	// 2. Check IP Address (/ip/address)
	addrOK := false
	var addrVal string
	reqAddr, err := http.NewRequest(http.MethodGet, baseURL+"/ip/address", nil)
	if err == nil {
		reqAddr.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if resp, err := m.httpClient.Do(reqAddr); err == nil {
			var addrList []map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&addrList)
			resp.Body.Close()
			for _, item := range addrList {
				if item["interface"] == cfg.InterfaceName {
					addr := fmt.Sprintf("%v", item["address"])
					if addr == cfg.ServerAddress {
						addrOK = true
						addrVal = addr + " (已绑定)"
					} else {
						addrVal = fmt.Sprintf("网段不一致 (ROS: %s, MiCube: %s)", addr, cfg.ServerAddress)
					}
					break
				}
			}
		}
	}
	if addrOK {
		report.Items = append(report.Items, WGCheckItem{
			Key:     "address",
			Name:    "隧道网关 IP (/ip/address)",
			Status:  "ok",
			Value:   addrVal,
			Details: "穿透网段 IP 已正确绑定到 WireGuard 虚拟网卡。",
			Advice:  "已对齐",
		})
	} else {
		report.AllAligned = false
		report.Items = append(report.Items, WGCheckItem{
			Key:     "address",
			Name:    "隧道网关 IP (/ip/address)",
			Status:  "missing",
			Value:   addrVal,
			Details: fmt.Sprintf("未在 %s 上绑定虚拟网段 %s", cfg.InterfaceName, cfg.ServerAddress),
			Advice:  "点击「⚡ 一键对齐 WireGuard」自动下发绑定 IP",
		})
	}

	// 3. Check Firewall Filter (/ip/firewall/filter)
	filterOK := false
	reqFilter, err := http.NewRequest(http.MethodGet, baseURL+"/ip/firewall/filter", nil)
	if err == nil {
		reqFilter.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if resp, err := m.httpClient.Do(reqFilter); err == nil {
			var filterList []map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&filterList)
			resp.Body.Close()
			portStr := fmt.Sprintf("%d", cfg.ListenPort)
			for _, item := range filterList {
				if item["chain"] == "input" && item["protocol"] == "udp" && item["dst-port"] == portStr && item["action"] == "accept" {
					filterOK = true
					break
				}
			}
		}
	}
	if filterOK {
		report.Items = append(report.Items, WGCheckItem{
			Key:     "firewall",
			Name:    "穿透端口防火墙放行 (/ip/firewall/filter)",
			Status:  "ok",
			Value:   fmt.Sprintf("UDP %d 放行 (Accept)", cfg.ListenPort),
			Details: "移动终端可由外网公网直连接入。",
			Advice:  "已对齐",
		})
	} else {
		report.AllAligned = false
		report.Items = append(report.Items, WGCheckItem{
			Key:     "firewall",
			Name:    "穿透端口防火墙放行 (/ip/firewall/filter)",
			Status:  "warning",
			Value:   "未放行 UDP 端口",
			Details: fmt.Sprintf("未在防火墙 Input 链放行 UDP %d，可能阻断外网扫码接入。", cfg.ListenPort),
			Advice:  "建议一键对齐放行防火墙端口",
		})
	}

	// 4. Check Cloud DDNS
	ddnsOK := cfg.CloudDDNS != ""
	report.Items = append(report.Items, WGCheckItem{
		Key:     "cloud_ddns",
		Name:    "RouterOS 官方 IP Cloud 动态域名",
		Status:  func() string { if ddnsOK { return "ok" }; return "warning" }(),
		Value:   func() string { if ddnsOK { return cfg.CloudDDNS }; return "未探查到" }(),
		Details: func() string { if ddnsOK { return "移动端扫码免配公网 IP，自动由动态域名直连。" }; return "若无公网 IP，可前往 RouterOS 开启 /ip/cloud ddns-enabled=yes" }(),
		Advice:  "自动联动中",
	})

	// 5. Check Peers (/interface/wireguard/peers)
	syncedCount := 0
	reqPeers, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard/peers", nil)
	if err == nil {
		reqPeers.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if resp, err := m.httpClient.Do(reqPeers); err == nil {
			var rosPeers []map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&rosPeers)
			resp.Body.Close()

			rosPubKeyMap := make(map[string]map[string]any)
			for _, rp := range rosPeers {
				if pk, ok := rp["public-key"].(string); ok {
					rosPubKeyMap[pk] = rp
				}
			}

			for i := range m.state.Peers {
				p := m.state.Peers[i]
				if _, exists := rosPubKeyMap[p.PublicKey]; exists {
					syncedCount++
					m.state.Peers[i].SyncedToROS = true
					m.state.Peers[i].ROSError = ""
				} else {
					m.state.Peers[i].SyncedToROS = false
				}
			}
		}
	}
	report.SyncedPeers = syncedCount

	peersAligned := len(m.state.Peers) == 0 || syncedCount == len(m.state.Peers)
	if !peersAligned {
		report.AllAligned = false
	}
	report.Items = append(report.Items, WGCheckItem{
		Key:     "peers",
		Name:    "移动端设备 Peer 同步状态",
		Status:  func() string { if peersAligned { return "ok" }; return "warning" }(),
		Value:   fmt.Sprintf("已同步 %d / %d 台已授权设备", syncedCount, len(m.state.Peers)),
		Details: func() string { if peersAligned { return "所有移动端设备公钥与 /32 IP 均已与 RouterOS 硬件对齐。" }; return "存在未同步到 RouterOS 的移动设备，该设备将暂时无法握手。" }(),
		Advice:  "点击「⚡ 一键对齐 WireGuard」立即推送所有未同步 Peer",
	})

	return report, nil
}

// AlignWireGuardToROS triggers a full 1-click alignment of server interface, IP, firewall, DDNS and all peers.
func (m *Manager) AlignWireGuardToROS() (*WGAlignResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res := &WGAlignResult{
		Success: true,
		Actions: make([]string, 0),
		Errors:  make([]string, 0),
	}

	rosCfg := m.rosManager.GetConfig()
	if rosCfg["host"] == "" {
		return nil, fmt.Errorf("RouterOS host not configured")
	}

	// 1. Initialize server interface, address, and firewall
	if err := m.initRouterOSServerLocked(); err != nil {
		res.Errors = append(res.Errors, "初始化服务端失败: "+err.Error())
	} else {
		res.Actions = append(res.Actions, "WireGuard 接口与虚拟网段已就绪")
	}

	// 2. Sync all peers
	synced := 0
	for i := range m.state.Peers {
		if err := m.syncPeerToROSLocked(m.state.Peers[i]); err == nil {
			m.state.Peers[i].SyncedToROS = true
			m.state.Peers[i].ROSError = ""
			synced++
		} else {
			m.state.Peers[i].SyncedToROS = false
			m.state.Peers[i].ROSError = err.Error()
			res.Errors = append(res.Errors, fmt.Sprintf("同步设备 %s 失败: %v", m.state.Peers[i].Name, err))
		}
	}
	_ = m.saveStateLocked()

	res.Actions = append(res.Actions, fmt.Sprintf("已成功对齐并同步 %d/%d 台移动设备 Peer", synced, len(m.state.Peers)))

	if len(res.Errors) > 0 {
		res.Success = false
		res.Message = fmt.Sprintf("对齐完成，但有 %d 项需注意", len(res.Errors))
	} else {
		res.Message = "🎉 RouterOS WireGuard 服务端与客户端 Peer 全部对齐成功！"
	}

	return res, nil
}

// PeerLiveStatus contains runtime handshake and transfer status of a peer.
type PeerLiveStatus struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	PublicKey       string `json:"public_key"`
	ClientIP        string `json:"client_ip"`
	Connected       bool   `json:"connected"`
	LastHandshake   string `json:"last_handshake"`
	RxBytes         int64  `json:"rx_bytes"`
	TxBytes         int64  `json:"tx_bytes"`
	CurrentEndpoint string `json:"current_endpoint"`
}

// GetPeerLiveStatus inspects real-time handshake and traffic from RouterOS.
func (m *Manager) GetPeerLiveStatus(peerID string) (*PeerLiveStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var targetPeer Peer
	found := false
	for _, p := range m.state.Peers {
		if p.ID == peerID || p.Name == peerID {
			targetPeer = p
			found = true
			break
		}
	}
	if !found {
		if peerID == "" && len(m.state.Peers) > 0 {
			targetPeer = m.state.Peers[len(m.state.Peers)-1]
			found = true
		} else {
			return nil, fmt.Errorf("peer not found")
		}
	}

	status := &PeerLiveStatus{
		ID:        targetPeer.ID,
		Name:      targetPeer.Name,
		PublicKey: targetPeer.PublicKey,
		ClientIP:  targetPeer.ClientIP,
		Connected: false,
	}

	rosCfg := m.rosManager.GetConfig()
	if rosCfg["host"] != "" {
		baseURL := m.getBaseURL(rosCfg)
		req, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard/peers", nil)
		if err == nil {
			req.SetBasicAuth(rosCfg["user"], rosCfg["password"])
			if resp, err := m.httpClient.Do(req); err == nil {
				defer resp.Body.Close()
				var peers []map[string]any
				if err := json.NewDecoder(resp.Body).Decode(&peers); err == nil {
					for _, p := range peers {
						if p["public-key"] == targetPeer.PublicKey {
							if lh, ok := p["last-handshake"].(string); ok && lh != "" {
								status.LastHandshake = lh
								status.Connected = true
							}
							if rx, ok := p["rx"].(float64); ok {
								status.RxBytes = int64(rx)
								if status.RxBytes > 0 {
									status.Connected = true
								}
							}
							if tx, ok := p["tx"].(float64); ok {
								status.TxBytes = int64(tx)
							}
							if ep, ok := p["current-endpoint-address"].(string); ok && ep != "" {
								status.CurrentEndpoint = ep
								if port, ok := p["current-endpoint-port"].(float64); ok {
									status.CurrentEndpoint = fmt.Sprintf("%s:%d", ep, int(port))
								}
							}
							break
						}
					}
				}
			}
		}
	}

	return status, nil
}

// WizardAutoSetup handles 1-click end-to-end setup: aligns ROS, creates/fetches device peer, returns peer & config.
func (m *Manager) WizardAutoSetup(deviceName string) (*Peer, string, error) {
	if deviceName == "" {
		deviceName = "我的手机"
	}
	// 1. Align ROS
	_, _ = m.AlignWireGuardToROS()

	// 2. Find or create peer
	m.mu.Lock()
	var targetPeer Peer
	found := false
	for _, p := range m.state.Peers {
		if p.Name == deviceName {
			targetPeer = p
			found = true
			break
		}
	}
	m.mu.Unlock()

	if !found {
		p, conf, err := m.CreatePeer(deviceName, "full")
		if err != nil {
			return nil, "", err
		}
		return p, conf, nil
	}

	// If peer exists, ensure synced to ROS
	m.mu.Lock()
	if !targetPeer.SyncedToROS {
		_ = m.syncPeerToROSLocked(targetPeer)
		_ = m.saveStateLocked()
	}
	m.mu.Unlock()

	conf, err := m.GetClientConfig(targetPeer.ID)
	if err != nil {
		return &targetPeer, "", err
	}
	return &targetPeer, conf, nil
}

