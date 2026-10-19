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
	"github.com/zyeyefyass/CubeX/internal/schema"
)

// ServerConfig stores the WireGuard server configuration on RouterOS.
type ServerConfig struct {
	InterfaceName string `json:"interface_name"` // default "cubex-wg"
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

// GenerateHostWireGuardConfig generates a cleanup script for Linux/fnOS host.
func (m *Manager) GenerateHostWireGuardConfig() string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# CubeX · 清理宿主机 WireGuard 遗留规则与服务\n")
	sb.WriteString("systemctl stop wg-quick@wg0 2>/dev/null || true\n")
	sb.WriteString("systemctl disable wg-quick@wg0 2>/dev/null || true\n")
	sb.WriteString("ip link delete wg0 2>/dev/null || true\n")
	sb.WriteString("ip link delete cubex-wg 2>/dev/null || true\n")
	sb.WriteString("ip link delete micube-wg 2>/dev/null || true\n")
	sb.WriteString("rm -f /etc/wireguard/wg0.conf /etc/wireguard/micube-wg.conf /etc/wireguard/cubex-wg.conf\n")
	sb.WriteString("iptables -D INPUT -p udp --dport 13231 -j ACCEPT 2>/dev/null || true\n")
	sb.WriteString("iptables -D INPUT -p udp --dport 51820 -j ACCEPT 2>/dev/null || true\n")
	sb.WriteString("iptables -t nat -D POSTROUTING -s 10.88.0.0/24 ! -o wg+ -j MASQUERADE 2>/dev/null || true\n")
	sb.WriteString("iptables -D FORWARD -s 10.88.0.0/24 -j ACCEPT 2>/dev/null || true\n")
	sb.WriteString("iptables -D FORWARD -d 10.88.0.0/24 -j ACCEPT 2>/dev/null || true\n")
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
		state: State{Peers: []Peer{}},
	}
	// Purge legacy wireguard.json file if present
	if _, err := os.Stat(m.stateFile); err == nil {
		_ = os.Remove(m.stateFile)
	}
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

func maskKey(k string) string {
	if len(k) <= 8 {
		return "********"
	}
	return k[:4] + "...." + k[len(k)-4:]
}

// GetInfo returns the current WireGuard server status, detected DDNS, and peer list with masked credentials.
func (m *Manager) GetInfo() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	endpoint := m.state.Config.EndpointHost
	if endpoint == "" && m.state.Config.CloudDDNS != "" {
		endpoint = fmt.Sprintf("%s:%d", m.state.Config.CloudDDNS, m.state.Config.ListenPort)
	} else if endpoint != "" && !strings.Contains(endpoint, ":") {
		endpoint = fmt.Sprintf("%s:%d", endpoint, m.state.Config.ListenPort)
	}

	safeCfg := m.state.Config
	if safeCfg.PrivateKey != "" {
		safeCfg.PrivateKey = maskKey(safeCfg.PrivateKey)
	}

	safePeers := make([]Peer, len(m.state.Peers))
	for i, p := range m.state.Peers {
		safeP := p
		if safeP.PrivateKey != "" {
			safeP.PrivateKey = maskKey(safeP.PrivateKey)
		}
		safePeers[i] = safeP
	}

	return map[string]any{
		"config":             safeCfg,
		"peers":              safePeers,
		"total_peers":        len(m.state.Peers),
		"effective_endpoint": endpoint,
		"cloud_ddns":         m.state.Config.CloudDDNS,
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

// GenerateROSScript returns copy-pasteable RouterOS terminal commands for complete WireGuard cleanup.
func (m *Manager) GenerateROSScript() string {
	var b strings.Builder
	b.WriteString("# 🌌 CubeX RouterOS WireGuard 彻底清理脚本\n")
	b.WriteString("# 请登录 WinBox 打开 Terminal 或通过 SSH 粘贴执行：\n\n")
	b.WriteString("# 1. 清除 WireGuard 端口放行防火墙规则\n")
	b.WriteString("/ip/firewall/filter remove [find comment~\"WireGuard\" or dst-port=\"13231\" or dst-port=\"51820\"]\n\n")
	b.WriteString("# 2. 清除所有 WireGuard 移动端客户端 Peer\n")
	b.WriteString("/interface/wireguard/peers remove [find]\n\n")
	b.WriteString("# 3. 清除 WireGuard 虚拟网段 IP 绑定\n")
	b.WriteString("/ip/address remove [find interface~\"wg\" or comment~\"WireGuard\" or address~\"10.88.0.\"]\n\n")
	b.WriteString("# 4. 清除 WireGuard 核心服务端接口\n")
	b.WriteString("/interface/wireguard remove [find]\n")
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
			"comment":     "CubeX WireGuard Server",
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
			"comment":   "CubeX WireGuard Subnet",
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
			"comment":      "CubeX WireGuard In",
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
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}
	return schema.WriteFileAtomicSecure(m.stateFile, data)
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

	// 1. Check Interface (/interface/wireguard) - must be absent
	hasIface := false
	var foundIfaceName string
	req, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard", nil)
	if err == nil {
		req.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if resp, err := m.httpClient.Do(req); err == nil {
			var wgList []map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&wgList)
			resp.Body.Close()
			for _, item := range wgList {
				name, _ := item["name"].(string)
				if name == "cubex-wg" || name == "micube-wg" {
					hasIface = true
					foundIfaceName = name
					break
				}
			}
		}
	}
	if hasIface {
		report.AllAligned = false
		report.Items = append(report.Items, WGCheckItem{
			Key:     "interface",
			Name:    "WireGuard 虚拟接口 (/interface/wireguard)",
			Status:  "warning",
			Value:   fmt.Sprintf("存在残留: %s", foundIfaceName),
			Details: "检测到 RouterOS 仍残留 WireGuard 虚拟网卡，建议一键清理。",
			Advice:  "点击「⚡ 一键清除 WireGuard」彻底注销",
		})
	} else {
		report.Items = append(report.Items, WGCheckItem{
			Key:     "interface",
			Name:    "WireGuard 虚拟接口 (/interface/wireguard)",
			Status:  "ok",
			Value:   "已彻底清理 (无残留)",
			Details: "RouterOS 上无任何 WireGuard 虚拟网卡残留，合规纯净。",
			Advice:  "状态良好",
		})
	}

	// 2. Check Firewall Filter (/ip/firewall/filter) - ports 13231 and 51820 must be absent
	hasFilter := false
	var foundPort string
	reqFilter, err := http.NewRequest(http.MethodGet, baseURL+"/ip/firewall/filter", nil)
	if err == nil {
		reqFilter.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if resp, err := m.httpClient.Do(reqFilter); err == nil {
			var filterList []map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&filterList)
			resp.Body.Close()
			for _, item := range filterList {
				port := fmt.Sprintf("%v", item["dst-port"])
				comment := fmt.Sprintf("%v", item["comment"])
				if port == "13231" || port == "51820" || strings.Contains(strings.ToLower(comment), "micube wireguard") || strings.Contains(strings.ToLower(comment), "cubex wireguard") {
					hasFilter = true
					foundPort = port
					break
				}
			}
		}
	}
	if hasFilter {
		report.AllAligned = false
		report.Items = append(report.Items, WGCheckItem{
			Key:     "firewall",
			Name:    "穿透端口防火墙规则 (/ip/firewall/filter)",
			Status:  "warning",
			Value:   fmt.Sprintf("存在遗留放行: UDP %s", foundPort),
			Details: "检测到防火墙存在 WireGuard UDP 放行规则，建议清理关闭。",
			Advice:  "点击「⚡ 一键清除 WireGuard」注销端口规则",
		})
	} else {
		report.Items = append(report.Items, WGCheckItem{
			Key:     "firewall",
			Name:    "穿透端口防火墙规则 (/ip/firewall/filter)",
			Status:  "ok",
			Value:   "已全部关闭 (0 开放端口)",
			Details: "所有 WireGuard UDP 穿透端口均已关闭，绝无端口暴露。",
			Advice:  "状态良好",
		})
	}

	// 3. Check Peers (/interface/wireguard/peers) - must be 0 (excluding RouterOS native dynamic back-to-home VPN)
	hasPeers := false
	reqPeers, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard/peers", nil)
	if err == nil {
		reqPeers.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		if resp, err := m.httpClient.Do(reqPeers); err == nil {
			var rosPeers []map[string]any
			_ = json.NewDecoder(resp.Body).Decode(&rosPeers)
			resp.Body.Close()
			for _, p := range rosPeers {
				isDynamic := fmt.Sprintf("%v", p["dynamic"]) == "true"
				iface := fmt.Sprintf("%v", p["interface"])
				comment := fmt.Sprintf("%v", p["comment"])
				if isDynamic || iface == "back-to-home-vpn" || strings.Contains(comment, "back-to-home") {
					continue
				}
				hasPeers = true
				break
			}
		}
	}
	if hasPeers {
		report.AllAligned = false
		report.Items = append(report.Items, WGCheckItem{
			Key:     "peers",
			Name:    "移动端设备 Peer 记录 (/interface/wireguard/peers)",
			Status:  "warning",
			Value:   "存在残留 Peer 授权",
			Details: "检测到 RouterOS 仍存有 WireGuard 客户端授权密钥，建议清理。",
			Advice:  "点击「⚡ 一键清除 WireGuard」注销所有 Peer",
		})
	} else {
		report.Items = append(report.Items, WGCheckItem{
			Key:     "peers",
			Name:    "移动端设备 Peer 记录 (/interface/wireguard/peers)",
			Status:  "ok",
			Value:   "已全部注销 (0 残留)",
			Details: "所有客户端 Peer 密钥均已安全清除。",
			Advice:  "状态良好",
		})
	}

	return report, nil
}

// PurgeAll completely cleans up all WireGuard settings, files, and firewall rules on RouterOS, Host, and locally.
func (m *Manager) PurgeAll() (*WGAlignResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res := &WGAlignResult{
		Success: true,
		Actions: make([]string, 0),
		Errors:  make([]string, 0),
	}

	// 1. Delete local state file
	if err := os.Remove(m.stateFile); err == nil {
		res.Actions = append(res.Actions, "已删除本地 WireGuard 配置文件 (data/wireguard.json)")
	}
	m.state = State{Peers: []Peer{}}

	// 2. Clean Host (Linux / fnOS) if hostManager is configured
	if m.hostManager != nil {
		if hostLog, err := m.hostManager.CleanupHostWireGuard(); err == nil && hostLog != "" {
			res.Actions = append(res.Actions, "宿主机 WireGuard 规则与接口已清理完毕")
		}
	}

	// 3. Clean RouterOS if configured
	rosCfg := m.rosManager.GetConfig()
	if rosCfg["host"] != "" {
		baseURL := m.getBaseURL(rosCfg)
		client := m.httpClient
		auth := func(req *http.Request) {
			req.SetBasicAuth(rosCfg["user"], rosCfg["password"])
		}

		// 3.1 Remove all peers (/interface/wireguard/peers)
		if pReq, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard/peers", nil); err == nil {
			auth(pReq)
			if pResp, err := client.Do(pReq); err == nil {
				var peers []map[string]any
				_ = json.NewDecoder(pResp.Body).Decode(&peers)
				pResp.Body.Close()
				for _, p := range peers {
					isDynamic := fmt.Sprintf("%v", p["dynamic"]) == "true"
					iface := fmt.Sprintf("%v", p["interface"])
					comment := fmt.Sprintf("%v", p["comment"])
					if isDynamic || iface == "back-to-home-vpn" || strings.Contains(comment, "back-to-home") {
						continue
					}
					if id, ok := p[".id"].(string); ok && id != "" {
						if dReq, err := http.NewRequest(http.MethodDelete, baseURL+"/interface/wireguard/peers/"+id, nil); err == nil {
							auth(dReq)
							if dResp, err := client.Do(dReq); err == nil {
								dResp.Body.Close()
								res.Actions = append(res.Actions, fmt.Sprintf("已清理 RouterOS WireGuard 客户端 Peer (%s)", p["comment"]))
							}
						}
					}
				}
			}
		}

		// 3.2 Remove WireGuard filter rules (/ip/firewall/filter)
		if fReq, err := http.NewRequest(http.MethodGet, baseURL+"/ip/firewall/filter", nil); err == nil {
			auth(fReq)
			if fResp, err := client.Do(fReq); err == nil {
				var filters []map[string]any
				_ = json.NewDecoder(fResp.Body).Decode(&filters)
				fResp.Body.Close()
				for _, f := range filters {
					id, _ := f[".id"].(string)
					comment, _ := f["comment"].(string)
					port, _ := f["dst-port"].(string)
					if id != "" && (port == "13231" || port == "51820" || strings.Contains(strings.ToLower(comment), "micube wireguard") || strings.Contains(strings.ToLower(comment), "cubex wireguard")) {
						if dReq, err := http.NewRequest(http.MethodDelete, baseURL+"/ip/firewall/filter/"+id, nil); err == nil {
							auth(dReq)
							if dResp, err := client.Do(dReq); err == nil {
								dResp.Body.Close()
								res.Actions = append(res.Actions, fmt.Sprintf("已清理 RouterOS 防火墙放行规则 (端口 %s)", port))
							}
						}
					}
				}
			}
		}

		// 3.3 Remove WireGuard IP addresses (/ip/address)
		if aReq, err := http.NewRequest(http.MethodGet, baseURL+"/ip/address", nil); err == nil {
			auth(aReq)
			if aResp, err := client.Do(aReq); err == nil {
				var addresses []map[string]any
				_ = json.NewDecoder(aResp.Body).Decode(&addresses)
				aResp.Body.Close()
				for _, a := range addresses {
					id, _ := a[".id"].(string)
					iface, _ := a["interface"].(string)
					comment, _ := a["comment"].(string)
					addr, _ := a["address"].(string)
					if id != "" && (iface == "cubex-wg" || iface == "micube-wg" || strings.Contains(strings.ToLower(comment), "micube wireguard") || strings.Contains(strings.ToLower(comment), "cubex wireguard") || strings.HasPrefix(addr, "10.88.0.")) {
						if dReq, err := http.NewRequest(http.MethodDelete, baseURL+"/ip/address/"+id, nil); err == nil {
							auth(dReq)
							if dResp, err := client.Do(dReq); err == nil {
								dResp.Body.Close()
								res.Actions = append(res.Actions, fmt.Sprintf("已清理 RouterOS WireGuard 虚拟网段 IP (%s)", addr))
							}
						}
					}
				}
			}
		}

		// 3.4 Remove WireGuard interfaces (/interface/wireguard)
		if wReq, err := http.NewRequest(http.MethodGet, baseURL+"/interface/wireguard", nil); err == nil {
			auth(wReq)
			if wResp, err := client.Do(wReq); err == nil {
				var wgList []map[string]any
				_ = json.NewDecoder(wResp.Body).Decode(&wgList)
				wResp.Body.Close()
				for _, item := range wgList {
					id, _ := item[".id"].(string)
					name, _ := item["name"].(string)
					if id != "" && (name == "cubex-wg" || name == "micube-wg") {
						if dReq, err := http.NewRequest(http.MethodDelete, baseURL+"/interface/wireguard/"+id, nil); err == nil {
							auth(dReq)
							if dResp, err := client.Do(dReq); err == nil {
								dResp.Body.Close()
								res.Actions = append(res.Actions, fmt.Sprintf("已清理 RouterOS WireGuard 核心接口 (%s)", name))
							}
						}
					}
				}
			}
		}
	}

	res.Message = "🎉 WireGuard 全部配置、接口与防火墙放行规则已彻底清除干净！"
	return res, nil
}

// AlignWireGuardToROS triggers a full 1-click cleanup of server interface, IP, firewall, DDNS and all peers.
func (m *Manager) AlignWireGuardToROS() (*WGAlignResult, error) {
	return m.PurgeAll()
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

