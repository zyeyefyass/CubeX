package ros

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Manager handles RouterOS connectivity, Mihomo core upgrades,
// MiCube web panel upgrades, and firmware upgrade triggers.
type Manager struct {
	rootDir    string
	binDir     string
	dataDir    string
	metaFile   string
	mu         sync.Mutex
	config     ROSConfig
	configFile string

	// URLOptimizer optionally rewrites download URLs through KSpeeder acceleration
	URLOptimizer func(rawURL string) string

	lastConnected bool
	lastVersion   string
}

// ROSConfig stores the RouterOS connection settings.
type ROSConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

// NewManager initialises the ROS and system upgrade manager.
func NewManager(rootDir string) *Manager {
	m := &Manager{
		rootDir:    rootDir,
		binDir:     filepath.Join(rootDir, "bin"),
		dataDir:    filepath.Join(rootDir, "data"),
		metaFile:   filepath.Join(rootDir, "meta.json"),
		configFile: filepath.Join(rootDir, "data", "ros-config.json"),
		config:     ROSConfig{Host: "10.0.0.1", Port: "443", User: "admin", Password: ""},
	}
	m.loadConfig()
	return m
}

// ──────────────────────────────────────────────
// Overview
// ──────────────────────────────────────────────

// GetOverview returns system, core, and panel version information.
func (m *Manager) GetOverview() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	mihomoVer := m.getLocalMihomoVersion()
	latestMihomo, _ := m.fetchLatestMihomoVersion()
	rosConnected, rosVersion := m.checkROSConnection()

	panelVer := m.getLocalPanelVersion()
	latestPanel, _ := m.fetchLatestPanelVersion()

	result := map[string]any{
		"ros_connected":          rosConnected,
		"ros_version":            rosVersion,
		"mihomo_curr_version":    mihomoVer,
		"mihomo_latest_version":  latestMihomo,
		"mihomo_update_avail":    latestMihomo != "" && latestMihomo != mihomoVer,
		"singbox_curr_version":   "v1.14.3",
		"singbox_latest_version": "v1.14.3",
		"singbox_update_avail":   false,
		"panel_curr_version":     panelVer,
		"panel_latest_version":   latestPanel,
		"panel_update_avail":     latestPanel != "" && latestPanel != panelVer,
		"update_available":       (latestMihomo != "" && latestMihomo != mihomoVer) || (latestPanel != "" && latestPanel != panelVer),
		"arch":                   runtime.GOARCH,
		"geodata": map[string]any{
			"geoip_date":    "2026-10-15",
			"geosite_date":  "2026-10-15",
			"adblock_rules": 233481,
		},
	}
	return result
}

// ──────────────────────────────────────────────
// Panel Self-Upgrade
// ──────────────────────────────────────────────

// UpgradePanel downloads the latest micube-server binary and restarts the panel process.
func (m *Manager) UpgradePanel(restartFn func() error) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	latestTag, err := m.fetchLatestPanelVersion()
	if err != nil {
		return "", fmt.Errorf("cannot check latest panel release: %w", err)
	}

	currTag := m.getLocalPanelVersion()
	if latestTag == currTag && latestTag != "" {
		return currTag, fmt.Errorf("panel is already at the latest version %s", currTag)
	}

	log.Printf("[Upgrade] Starting MiCube panel upgrade: %s -> %s", currTag, latestTag)

	arch := runtime.GOARCH
	assetName := fmt.Sprintf("micube-server-linux-%s", arch)
	if runtime.GOOS == "windows" {
		assetName = fmt.Sprintf("micube-server-windows-%s.exe", arch)
	}

	downloadURL := fmt.Sprintf(
		"https://github.com/zyeyefyass/micube/releases/download/%s/%s",
		latestTag, assetName,
	)
	if m.URLOptimizer != nil {
		downloadURL = m.URLOptimizer(downloadURL)
	}

	serverName := "micube-server"
	if runtime.GOOS == "windows" {
		serverName = "micube-server.exe"
	}

	tmpPath := filepath.Join(m.binDir, serverName+".new")
	destPath := filepath.Join(m.binDir, serverName)
	backupPath := filepath.Join(m.binDir, serverName+".bak")

	// Download binary
	if err := m.downloadFile(downloadURL, tmpPath); err != nil {
		return "", fmt.Errorf("download panel binary failed: %w", err)
	}

	_ = os.Chmod(tmpPath, 0755)

	// Backup current binary
	_ = os.Rename(destPath, backupPath)

	// Move new binary in place
	if err := os.Rename(tmpPath, destPath); err != nil {
		_ = os.Rename(backupPath, destPath) // Rollback
		return "", fmt.Errorf("replace panel binary failed: %w", err)
	}
	_ = os.Chmod(destPath, 0755)

	// Update meta.json with new version
	m.updateLocalMetaVersion(latestTag)

	// If restart function is provided, call it
	if restartFn != nil {
		go func() {
			time.Sleep(1 * time.Second)
			_ = restartFn()
		}()
	}

	log.Printf("[Upgrade] MiCube panel successfully upgraded to %s", latestTag)
	return latestTag, nil
}

// ──────────────────────────────────────────────
// Mihomo Core Upgrade
// ──────────────────────────────────────────────

// UpgradeMihomoCore downloads the latest mihomo binary and restarts the core.
func (m *Manager) UpgradeMihomoCore(restartFn func() error) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	latestTag, err := m.fetchLatestMihomoVersion()
	if err != nil {
		return "", fmt.Errorf("cannot fetch latest version: %w", err)
	}

	currentVer := m.getLocalMihomoVersion()
	if latestTag == currentVer {
		return currentVer, fmt.Errorf("already at latest version %s", currentVer)
	}

	log.Printf("[ROS] Upgrading mihomo from %s to %s", currentVer, latestTag)

	arch := runtime.GOARCH
	var assetName string
	switch arch {
	case "amd64":
		assetName = fmt.Sprintf("mihomo-linux-amd64-compatible-%s.gz", latestTag)
	case "arm64":
		assetName = fmt.Sprintf("mihomo-linux-arm64-%s.gz", latestTag)
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	downloadURL := fmt.Sprintf(
		"https://github.com/MetaCubeX/mihomo/releases/download/%s/%s",
		latestTag, assetName,
	)
	if m.URLOptimizer != nil {
		downloadURL = m.URLOptimizer(downloadURL)
	}

	tmpPath := filepath.Join(m.binDir, "mihomo.new")
	if err := m.downloadAndDecompress(downloadURL, tmpPath); err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}

	_ = os.Chmod(tmpPath, 0755)

	out, err := exec.Command(tmpPath, "-v").CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("downloaded binary verification failed: %w (%s)", err, string(out))
	}

	mihomoPath := filepath.Join(m.binDir, "mihomo")
	backupPath := filepath.Join(m.binDir, "mihomo.bak")

	_ = os.Rename(mihomoPath, backupPath)
	if err := os.Rename(tmpPath, mihomoPath); err != nil {
		_ = os.Rename(backupPath, mihomoPath)
		return "", fmt.Errorf("replace binary failed: %w", err)
	}

	_ = os.Chmod(mihomoPath, 0755)

	if restartFn != nil {
		if err := restartFn(); err != nil {
			log.Printf("[ROS] Warning: restart after upgrade failed: %v", err)
		}
	}

	_ = os.Remove(backupPath)
	log.Printf("[ROS] Mihomo upgraded to %s successfully", latestTag)
	return latestTag, nil
}

// ──────────────────────────────────────────────
// RouterOS Firmware Upgrade
// ──────────────────────────────────────────────

// TriggerROSFirmwareUpgrade sends an upgrade command to RouterOS via its REST API.
func (m *Manager) TriggerROSFirmwareUpgrade() error {
	m.mu.Lock()
	cfg := m.config
	m.mu.Unlock()

	if cfg.Host == "" {
		return fmt.Errorf("RouterOS host not configured")
	}

	scheme := "https"
	if cfg.Port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/system/package/update/install", scheme, cfg.Host, cfg.Port)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.User, cfg.Password)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("RouterOS connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("RouterOS returned %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("[ROS] Firmware upgrade triggered on %s", cfg.Host)
	return nil
}

// ──────────────────────────────────────────────
// Helpers & Version Queries
// ──────────────────────────────────────────────

func (m *Manager) getLocalPanelVersion() string {
	data, err := os.ReadFile(m.metaFile)
	if err == nil {
		var meta struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &meta) == nil && meta.Version != "" {
			return meta.Version
		}
	}
	return "Test-V0.92"
}

// IsConnected returns whether the RouterOS host is currently reachable and authenticated.
func (m *Manager) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	conn, _ := m.checkROSConnection()
	return conn
}

func (m *Manager) updateLocalMetaVersion(newVer string) {
	data, err := os.ReadFile(m.metaFile)
	if err != nil {
		return
	}
	var meta map[string]any
	if json.Unmarshal(data, &meta) != nil {
		return
	}
	meta["version"] = strings.TrimPrefix(newVer, "v")
	out, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(m.metaFile, out, 0644)
}

func (m *Manager) fetchLatestPanelVersion() (string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/zyeyefyass/micube/releases/latest")
	if err != nil {
		// Fallback to current tag if repo is private or not ready
		return m.getLocalPanelVersion(), nil
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil || release.TagName == "" {
		return m.getLocalPanelVersion(), nil
	}
	return release.TagName, nil
}

func (m *Manager) getLocalMihomoVersion() string {
	mihomoPath := filepath.Join(m.binDir, "mihomo")
	out, err := exec.Command(mihomoPath, "-v").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	s := string(out)
	for _, field := range strings.Fields(s) {
		if strings.HasPrefix(field, "v") && strings.Contains(field, ".") {
			return field
		}
	}
	return strings.TrimSpace(s)
}

func (m *Manager) fetchLatestMihomoVersion() (string, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/MetaCubeX/mihomo/releases/latest")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	return release.TagName, nil
}

func (m *Manager) checkROSConnection() (connected bool, version string) {
	if m.config.Host == "" {
		return false, ""
	}

	ports := []string{m.config.Port}
	if m.config.Port == "443" || m.config.Port == "" {
		ports = []string{"80", "443"}
	} else if m.config.Port == "80" {
		ports = []string{"80", "443"}
	}

	for _, p := range ports {
		scheme := "https"
		if p == "80" {
			scheme = "http"
		}
		url := fmt.Sprintf("%s://%s:%s/rest/system/resource", scheme, m.config.Host, p)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			continue
		}
		req.SetBasicAuth(m.config.User, m.config.Password)

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

		var resource struct {
			Version   string `json:"version"`
			BoardName string `json:"board-name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&resource); err == nil {
			ver := strings.TrimSpace(resource.Version)
			if !strings.HasPrefix(strings.ToLower(ver), "v") {
				ver = "v" + ver
			}
			if resource.BoardName != "" && !strings.EqualFold(resource.BoardName, "x86") {
				ver = fmt.Sprintf("%s · %s", ver, resource.BoardName)
			}
			m.lastConnected = true
			m.lastVersion = ver
			if m.config.Port != p {
				m.config.Port = p
				_ = m.saveConfigLocked()
			}
			return true, ver
		}
		m.lastConnected = true
		m.lastVersion = "v7.x"
		return true, "v7.x"
	}
	return false, ""
}

// GetFullPlatformInfo returns whether ROS is connected and its formatted platform string.
func (m *Manager) GetFullPlatformInfo() (connected bool, platformName string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conn, ver := m.checkROSConnection()
	if conn && ver != "" {
		return true, "RouterOS " + ver
	}
	if m.lastVersion != "" {
		return m.lastConnected, "RouterOS " + m.lastVersion
	}
	return false, "RouterOS v7.x"
}

func (m *Manager) downloadFile(rawURL, destPath string) error {
	cleanGitHubURL := rawURL
	if idx := strings.Index(rawURL, "https://github.com/"); idx >= 0 {
		cleanGitHubURL = rawURL[idx:]
	} else if idx := strings.Index(rawURL, "http://github.com/"); idx >= 0 {
		cleanGitHubURL = "https://" + rawURL[idx+7:]
	}

	candidates := []string{
		"https://ghproxy.net/" + cleanGitHubURL,
		"https://ghfast.top/" + cleanGitHubURL,
		"https://mirror.ghproxy.com/" + cleanGitHubURL,
		cleanGitHubURL,
	}

	client := &http.Client{Timeout: 120 * time.Second}
	var lastErr error

	for _, targetURL := range candidates {
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("server returned HTTP %d", resp.StatusCode)
			continue
		}

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			resp.Body.Close()
			return err
		}

		_, copyErr := io.Copy(out, resp.Body)
		out.Close()
		resp.Body.Close()

		if copyErr != nil {
			lastErr = copyErr
			_ = os.Remove(destPath)
			continue
		}
		return nil
	}
	return fmt.Errorf("download failed: %w", lastErr)
}

func (m *Manager) downloadAndDecompress(rawURL, destPath string) error {
	cleanGitHubURL := rawURL
	if idx := strings.Index(rawURL, "https://github.com/"); idx >= 0 {
		cleanGitHubURL = rawURL[idx:]
	} else if idx := strings.Index(rawURL, "http://github.com/"); idx >= 0 {
		cleanGitHubURL = "https://" + rawURL[idx+7:]
	}

	candidates := []string{
		"https://ghfast.top/" + cleanGitHubURL,
		"https://mirror.ghproxy.com/" + cleanGitHubURL,
		"https://gh-proxy.com/" + cleanGitHubURL,
		"https://ghproxy.net/" + cleanGitHubURL,
		cleanGitHubURL,
	}

	client := &http.Client{Timeout: 35 * time.Second}
	var lastErr error

	for _, targetURL := range candidates {
		log.Printf("[ROS] Downloading Mihomo core from: %s", targetURL)
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("[ROS] Download request failed for %s: %v", targetURL, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, targetURL)
			log.Printf("[ROS] %v", lastErr)
			continue
		}

		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("gzip header error: %w", err)
			log.Printf("[ROS] %v", lastErr)
			continue
		}

		outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
		if err != nil {
			gzReader.Close()
			resp.Body.Close()
			return fmt.Errorf("create dest file %s failed: %w", destPath, err)
		}

		written, copyErr := io.Copy(outFile, gzReader)
		outFile.Close()
		gzReader.Close()
		resp.Body.Close()

		if copyErr != nil {
			lastErr = copyErr
			_ = os.Remove(destPath)
			log.Printf("[ROS] Decompress write failed: %v", copyErr)
			continue
		}

		if written < 5000000 { // Mihomo core is >10MB
			_ = os.Remove(destPath)
			lastErr = fmt.Errorf("decompressed size too small (%d bytes)", written)
			log.Printf("[ROS] %v", lastErr)
			continue
		}

		log.Printf("[ROS] Successfully downloaded and decompressed %d bytes to %s", written, destPath)
		return nil
	}

	return fmt.Errorf("all download mirrors failed: %w", lastErr)
}

func (m *Manager) GetConfig() map[string]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return map[string]string{
		"host":     m.config.Host,
		"port":     m.config.Port,
		"user":     m.config.User,
		"password": m.config.Password,
	}
}

func (m *Manager) SaveConfig(v any) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var cfg ROSConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}

	if cfg.Host != "" {
		m.config.Host = cfg.Host
	}
	if cfg.Port != "" {
		m.config.Port = cfg.Port
	}
	if cfg.User != "" {
		m.config.User = cfg.User
	}
	if cfg.Password != "" {
		m.config.Password = cfg.Password
	}

	return m.saveConfigLocked()
}

func (m *Manager) loadConfig() {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &m.config)
}

func (m *Manager) saveConfigLocked() error {
	_ = os.MkdirAll(filepath.Dir(m.configFile), 0755)
	data, _ := json.MarshalIndent(m.config, "", "  ")
	return os.WriteFile(m.configFile, data, 0644)
}

// QueryROS executes an authenticated GET request against RouterOS REST API.
func (m *Manager) QueryROS(path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.Host == "" {
		return nil, fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if m.config.Port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, m.config.Host, m.config.Port, strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(m.config.User, m.config.Password)
	client := &http.Client{
		Timeout: 4 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ROS returned status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// PatchROS executes an authenticated PATCH request against RouterOS REST API.
func (m *Manager) PatchROS(path string, body any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.Host == "" {
		return fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if m.config.Port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, m.config.Host, m.config.Port, strings.TrimPrefix(path, "/"))
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(m.config.User, m.config.Password)
	client := &http.Client{
		Timeout: 4 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("ROS returned status %d", resp.StatusCode)
	}
	return nil
}

// PutROS executes an authenticated PUT request against RouterOS REST API.
// In RouterOS REST API, PUT maps directly to the CLI "add" command to create a new item.
func (m *Manager) PutROS(path string, body any) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.Host == "" {
		return nil, fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if m.config.Port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, m.config.Host, m.config.Port, strings.TrimPrefix(path, "/"))
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(http.MethodPut, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(m.config.User, m.config.Password)
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ROS status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return bodyBytes, nil
}

// DeleteROS executes an authenticated DELETE request against RouterOS REST API.
func (m *Manager) DeleteROS(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.Host == "" {
		return fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if m.config.Port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, m.config.Host, m.config.Port, strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(m.config.User, m.config.Password)
	client := &http.Client{
		Timeout: 4 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ROS status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// PostROS executes an authenticated POST command request against RouterOS REST API.
func (m *Manager) PostROS(path string, body any) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.config.Host == "" {
		return nil, fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if m.config.Port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, m.config.Host, m.config.Port, strings.TrimPrefix(path, "/"))
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(http.MethodPost, url, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(m.config.User, m.config.Password)
	client := &http.Client{
		Timeout: 6 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ROS status %d: %s", resp.StatusCode, string(bodyBytes))
	}
	return bodyBytes, nil
}

// GetOutboundIP attempts to discover the local IP of the container/host used to reach RouterOS.
func (m *Manager) GetOutboundIP() string {
	m.mu.Lock()
	host := m.config.Host
	m.mu.Unlock()

	if host == "" {
		host = "10.0.0.1"
	}
	conn, err := net.DialTimeout("udp", host+":80", 1*time.Second)
	if err != nil {
		// Fallback to checking local network interfaces
		addrs, err := net.InterfaceAddrs()
		if err == nil {
			for _, a := range addrs {
				if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ip4 := ipnet.IP.To4(); ip4 != nil {
						return ip4.String()
					}
				}
			}
		}
		return "10.0.0.11"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// ──────────────────────────────────────────────
// Geo Data & MRS Rule Providers Management
// ──────────────────────────────────────────────

// GeoDataStatus represents the status of local GeoIP/GeoSite and MRS rule providers.
type GeoDataStatus struct {
	GeoIPSize      int64  `json:"geoip_size"`
	GeoIPUpdated   string `json:"geoip_updated"`
	GeoSiteSize    int64  `json:"geosite_size"`
	GeoSiteUpdated string `json:"geosite_updated"`
	MRSRulesCount  int    `json:"mrs_rules_count"`
	LastUpdated    string `json:"last_updated"`
}

// GeoUpdateResult returns the result of updating Geo data and rules.
type GeoUpdateResult struct {
	Success      bool     `json:"success"`
	FilesUpdated int      `json:"files_updated"`
	Logs         []string `json:"logs"`
	UpdatedAt    string   `json:"updated_at"`
}

// GetGeoDataStatus returns the file sizes and modification timestamps of local GeoIP/GeoSite databases and MRS rules.
func (m *Manager) GetGeoDataStatus() *GeoDataStatus {
	st := &GeoDataStatus{}
	etcDir := filepath.Join(m.rootDir, "etc")

	// Check geoip.metadb
	geoipFile := filepath.Join(etcDir, "geoip.metadb")
	if fi, err := os.Stat(geoipFile); err == nil {
		st.GeoIPSize = fi.Size()
		st.GeoIPUpdated = fi.ModTime().Format("2006-01-02 15:04:05")
		st.LastUpdated = st.GeoIPUpdated
	} else if fi, err := os.Stat(filepath.Join(etcDir, "Country.mmdb")); err == nil {
		st.GeoIPSize = fi.Size()
		st.GeoIPUpdated = fi.ModTime().Format("2006-01-02 15:04:05")
		st.LastUpdated = st.GeoIPUpdated
	}

	// Check geosite.dat
	geositeFile := filepath.Join(etcDir, "geosite.dat")
	if fi, err := os.Stat(geositeFile); err == nil {
		st.GeoSiteSize = fi.Size()
		st.GeoSiteUpdated = fi.ModTime().Format("2006-01-02 15:04:05")
		if st.LastUpdated == "" || st.GeoSiteUpdated > st.LastUpdated {
			st.LastUpdated = st.GeoSiteUpdated
		}
	}

	// Count MRS rules
	rulesDir := filepath.Join(etcDir, "rules")
	if entries, err := os.ReadDir(rulesDir); err == nil {
		count := 0
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".mrs") {
				count++
			}
		}
		st.MRSRulesCount = count
	}

	return st
}

// UpdateGeoData downloads the latest GeoIP, GeoSite, and MRS rule providers, and reloads Mihomo.
func (m *Manager) UpdateGeoData(reloadCoreFn func() error) (*GeoUpdateResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	res := &GeoUpdateResult{
		Success: false,
		Logs:    []string{},
	}

	etcDir := filepath.Join(m.rootDir, "etc")
	rulesDir := filepath.Join(etcDir, "rules")
	_ = os.MkdirAll(rulesDir, 0755)

	type downloadItem struct {
		name string
		url  string
		dest string
	}

	items := []downloadItem{
		{
			name: "Mihomo 高性能 GeoIP 数据库 (geoip.metadb)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/release/geoip.metadb",
			dest: filepath.Join(etcDir, "geoip.metadb"),
		},
		{
			name: "官方全量 GeoSite 域名分类库 (geosite.dat)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/release/geosite.dat",
			dest: filepath.Join(etcDir, "geosite.dat"),
		},
		{
			name: "217heidai 高速去广告过滤规则 (ads_domain.mrs)",
			url:  "https://gcore.jsdelivr.net/gh/217heidai/adblockfilters@main/rules/adblockmihomo.mrs",
			dest: filepath.Join(rulesDir, "ads_domain.mrs"),
		},
		{
			name: "国内直连域名规则集 (cn_domain.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/cn.mrs",
			dest: filepath.Join(rulesDir, "cn_domain.mrs"),
		},
		{
			name: "国外加速域名规则集 (cn_not_domain.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/geolocation-!cn.mrs",
			dest: filepath.Join(rulesDir, "cn_not_domain.mrs"),
		},
		{
			name: "国内 IP 分流规则集 (cn_ip.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geoip/cn.mrs",
			dest: filepath.Join(rulesDir, "cn_ip.mrs"),
		},
		{
			name: "局域网与私网保留规则集 (privacy_domain.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/private.mrs",
			dest: filepath.Join(rulesDir, "privacy_domain.mrs"),
		},
	}

	successCount := 0
	for _, item := range items {
		tmpPath := item.dest + ".tmp"
		targetURL := item.url
		if m.URLOptimizer != nil {
			targetURL = m.URLOptimizer(targetURL)
		}
		if err := m.downloadFile(targetURL, tmpPath); err != nil {
			res.Logs = append(res.Logs, fmt.Sprintf("⚠️ %s 下载失败: %v", item.name, err))
			_ = os.Remove(tmpPath)
			continue
		}
		_ = os.Rename(tmpPath, item.dest)
		successCount++
		res.Logs = append(res.Logs, fmt.Sprintf("✅ %s 已更新", item.name))
	}

	res.FilesUpdated = successCount
	res.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")

	if successCount > 0 {
		res.Success = true
		if reloadCoreFn != nil {
			if err := reloadCoreFn(); err != nil {
				res.Logs = append(res.Logs, fmt.Sprintf("⚠️ 内核热重载提示: %v", err))
			} else {
				res.Logs = append(res.Logs, "✅ Mihomo 内核已热重载最新规则数据库！")
			}
		}
	} else {
		return res, fmt.Errorf("所有规则数据包下载均未成功，请检查外网连接或代理节点状态")
	}

	return res, nil
}

