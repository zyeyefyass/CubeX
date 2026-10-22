package ros

import (
	"context"
	"math"
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Manager handles RouterOS connectivity, Mihomo core upgrades,
// CubeX web panel upgrades, and firmware upgrade triggers.
type Manager struct {
	isUpdatingGeo     bool
	geoStatusCache    *GeoDataStatus
	geoCacheTime      time.Time
	rootDir    string
	binDir     string
	dataDir    string
	metaFile   string
	mu         sync.RWMutex
	config     ROSConfig
	configFile string

	// URLOptimizer optionally rewrites download URLs through KSpeeder acceleration
	URLOptimizer func(rawURL string) string

	lastConnected     bool
	lastVersion       string
	lastFreeHddSpace  int64
	lastTotalHddSpace int64

	cachedLatestMihomo  string
	cachedLatestSingbox string
	cachedLatestPanel   string
	cachedVerTime       time.Time
	lastContainerMem    float64
	lastContainerMemAt  time.Time
	httpClient          *http.Client
}

func (m *Manager) getHTTPClient() *http.Client {
	if m.httpClient == nil {
		dialer := &net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{Timeout: 3 * time.Second}
					conn, err := d.DialContext(ctx, "udp", "223.5.5.5:53")
					if err != nil {
						conn, err = d.DialContext(ctx, "udp", "119.29.29.29:53")
						if err != nil {
							return d.DialContext(ctx, "udp", "114.114.114.114:53")
						}
					}
					return conn, nil
				},
			},
		}
		tr := &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			DialContext:         dialer.DialContext,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 5,
			IdleConnTimeout:     30 * time.Second,
		}
		m.httpClient = &http.Client{
			Transport: tr,
			Timeout:   15 * time.Second,
		}
	}
	return m.httpClient
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
		config:     ROSConfig{Host: "10.0.0.1", Port: "80", User: "", Password: ""},
	}
	m.loadConfig()

	// Launch background worker for periodic non-blocking version checks
	go m.startAsyncVersionWorker()

	return m
}

func (m *Manager) startAsyncVersionWorker() {
	time.Sleep(1 * time.Second)
	m.refreshVersionsAsync()

	ticker := time.NewTicker(2 * time.Hour)
	for range ticker.C {
		m.refreshVersionsAsync()
	}
}

func (m *Manager) refreshVersionsAsync() {
	var wg sync.WaitGroup
	var latestMihomo, latestPanel, latestSingbox string

	wg.Add(3)
	go func() {
		defer wg.Done()
		latestMihomo, _ = m.fetchLatestMihomoVersion()
	}()
	go func() {
		defer wg.Done()
		latestPanel, _ = m.fetchLatestPanelVersion()
	}()
	go func() {
		defer wg.Done()
		latestSingbox, _ = m.fetchLatestSingboxVersion()
	}()
	wg.Wait()

	m.mu.Lock()
	defer m.mu.Unlock()
	if latestMihomo != "" {
		m.cachedLatestMihomo = latestMihomo
	}
	if latestPanel != "" {
		m.cachedLatestPanel = latestPanel
	}
	if latestSingbox != "" {
		m.cachedLatestSingbox = latestSingbox
	}
	m.cachedVerTime = time.Now()
}

// ──────────────────────────────────────────────
// Overview
// ──────────────────────────────────────────────

// GetOverview returns system, core, and panel version information.
func (m *Manager) GetOverview() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	mihomoVer := m.getLocalMihomoVersion()
	panelVer := m.getLocalPanelVersion()
	singboxVer := m.getLocalSingboxVersion()
	rosConnected, rosVersion := m.checkROSConnection()

	latestMihomo := m.cachedLatestMihomo
	if latestMihomo == "" {
		latestMihomo = "v1.19.33"
	}
	latestPanel := m.cachedLatestPanel
	if latestPanel == "" {
		latestPanel = "Test v2.4"
	}
	latestSingbox := m.cachedLatestSingbox
	if latestSingbox == "" {
		latestSingbox = "v1.14.3"
	}

	panelUpdateAvail := IsVersionNewer(latestPanel, panelVer)
	mihomoUpdateAvail := IsVersionNewer(latestMihomo, mihomoVer)
	singboxUpdateAvail := IsVersionNewer(latestSingbox, singboxVer)

	result := map[string]any{
		"ros_connected":          rosConnected,
		"ros_version":            rosVersion,
		"ros_free_hdd_mb":        m.lastFreeHddSpace / (1024 * 1024),
		"ros_total_hdd_mb":       m.lastTotalHddSpace / (1024 * 1024),
		"ros_hdd_warning":        m.lastFreeHddSpace > 0 && (m.lastFreeHddSpace < 200*1024*1024),
		"mihomo_curr_version":    mihomoVer,
		"mihomo_latest_version":  latestMihomo,
		"mihomo_update_avail":    mihomoUpdateAvail,
		"singbox_curr_version":   singboxVer,
		"singbox_latest_version": latestSingbox,
		"singbox_update_avail":   singboxUpdateAvail,
		"panel_curr_version":     panelVer,
		"panel_latest_version":   latestPanel,
		"panel_update_avail":     panelUpdateAvail,
		"update_available":       mihomoUpdateAvail || singboxUpdateAvail || panelUpdateAvail,
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

// UpgradePanel downloads the latest cubexd binary and restarts the panel process.
func (m *Manager) UpgradePanel(force bool, restartFn func() error) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	latestTag, err := m.fetchLatestPanelVersion()
	if err != nil {
		return "", fmt.Errorf("cannot check latest panel release: %w", err)
	}

	currTag := m.getLocalPanelVersion()
	if !force && !IsVersionNewer(latestTag, currTag) {
		return currTag, fmt.Errorf("当前版本 (%s) 已是最新或高于云端版本 (%s)，无需升级", currTag, latestTag)
	}

	log.Printf("[Upgrade] Starting CubeX panel upgrade: %s -> %s", currTag, latestTag)

	arch := runtime.GOARCH
	destPath, err := os.Executable()
	if err != nil || destPath == "" {
		serverName := "cubexd"
		if runtime.GOOS == "windows" {
			serverName = "cubexd.exe"
		}
		destPath = filepath.Join(m.binDir, serverName)
	}

	tmpPath := destPath + ".new"
	backupPath := destPath + ".bak"

	// Try candidate 1: direct binary asset
	assetName := fmt.Sprintf("cubexd-linux-%s", arch)
	if runtime.GOOS == "windows" {
		assetName = "cubexd.exe"
	}
	binURL := fmt.Sprintf(
		"https://github.com/zyeyefyass/CubeX/releases/download/%s/%s",
		latestTag, assetName,
	)
	if m.URLOptimizer != nil {
		binURL = m.URLOptimizer(binURL)
	}

	downloadErr := m.downloadFile(binURL, tmpPath)
	if downloadErr != nil {
		// Try candidate 2: extract binary from zip release package
		zipPkgName := fmt.Sprintf("CubeX-%s-linux-%s.zip", latestTag, arch)
		if runtime.GOOS == "windows" {
			zipPkgName = fmt.Sprintf("CubeX-%s-windows-%s.zip", latestTag, arch)
		}
		zipURL := fmt.Sprintf("https://github.com/zyeyefyass/CubeX/releases/download/%s/%s", latestTag, zipPkgName)
		if m.URLOptimizer != nil {
			zipURL = m.URLOptimizer(zipURL)
		}
		targetBinName := "cubexd"
		if runtime.GOOS == "windows" {
			targetBinName = "cubexd.exe"
		}
		downloadErr = m.downloadAndExtractZip(zipURL, targetBinName, tmpPath)
	}

	if downloadErr != nil {
		return "", fmt.Errorf("download panel binary failed: %w", downloadErr)
	}

	_ = os.Chmod(tmpPath, 0755)

	// Verify new binary by executing quick version test if possible
	if runtime.GOOS != "windows" {
		if out, err := exec.Command(tmpPath, "-help").CombinedOutput(); err != nil && len(out) == 0 {
			_ = os.Remove(tmpPath)
			return "", fmt.Errorf("downloaded binary verification failed: %w", err)
		}
	}

	// Backup current binary
	_ = os.Remove(backupPath)
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
			time.Sleep(2 * time.Second)
			_ = restartFn()
		}()
	}

	log.Printf("[Upgrade] CubeX panel successfully upgraded to %s", latestTag)
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

// UpgradeSingboxCore downloads the latest sing-box binary and restarts the core.
func (m *Manager) UpgradeSingboxCore(restartFn func() error) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	latestTag, err := m.fetchLatestSingboxVersion()
	if err != nil {
		return "", fmt.Errorf("cannot fetch latest sing-box version: %w", err)
	}

	currentVer := m.getLocalSingboxVersion()
	if latestTag == currentVer {
		return currentVer, fmt.Errorf("already at latest version %s", currentVer)
	}

	log.Printf("[ROS] Upgrading sing-box from %s to %s", currentVer, latestTag)

	arch := runtime.GOARCH
	var targetArch string
	switch arch {
	case "amd64":
		targetArch = "amd64"
	case "arm64":
		targetArch = "arm64"
	default:
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	verNum := strings.TrimPrefix(latestTag, "v")
	// Try musl first (mandatory for Alpine Linux in RouterOS container)
	assetMusl := fmt.Sprintf("sing-box-%s-linux-%s-musl.tar.gz", verNum, targetArch)
	downloadURL := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/%s/%s", latestTag, assetMusl)
	if m.URLOptimizer != nil {
		downloadURL = m.URLOptimizer(downloadURL)
	}

	tmpPath := filepath.Join(m.binDir, "sing-box.new")
	if err := m.downloadAndExtractTarGz(downloadURL, "sing-box", tmpPath); err != nil {
		// Fallback to standard build
		assetStd := fmt.Sprintf("sing-box-%s-linux-%s.tar.gz", verNum, targetArch)
		downloadURL2 := fmt.Sprintf("https://github.com/SagerNet/sing-box/releases/download/%s/%s", latestTag, assetStd)
		if m.URLOptimizer != nil {
			downloadURL2 = m.URLOptimizer(downloadURL2)
		}
		if err2 := m.downloadAndExtractTarGz(downloadURL2, "sing-box", tmpPath); err2 != nil {
			return "", fmt.Errorf("download sing-box failed: %w (fallback: %v)", err, err2)
		}
	}

	_ = os.Chmod(tmpPath, 0755)

	out, err := exec.Command(tmpPath, "version").CombinedOutput()
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", fmt.Errorf("downloaded binary verification failed: %w (%s)", err, string(out))
	}

	singboxPath := filepath.Join(m.binDir, "sing-box")
	backupPath := filepath.Join(m.binDir, "sing-box.bak")

	_ = os.Rename(singboxPath, backupPath)
	if err := os.Rename(tmpPath, singboxPath); err != nil {
		_ = os.Rename(backupPath, singboxPath)
		return "", fmt.Errorf("replace binary failed: %w", err)
	}

	_ = os.Chmod(singboxPath, 0755)

	if restartFn != nil {
		if err := restartFn(); err != nil {
			log.Printf("[ROS] Warning: restart after upgrade failed: %v", err)
		}
	}

	_ = os.Remove(backupPath)
	log.Printf("[ROS] sing-box upgraded to %s successfully", latestTag)
	return latestTag, nil
}

// ──────────────────────────────────────────────
// RouterOS Firmware Upgrade
// ──────────────────────────────────────────────

// ROSFirmwareCheckResult holds parsed RouterOS package update status.
type ROSFirmwareCheckResult struct {
	InstalledVersion string `json:"installed_version"`
	LatestVersion    string `json:"latest_version"`
	Channel          string `json:"channel"`
	Status           string `json:"status"`
	HasUpdate        bool   `json:"has_update"`
	Message          string `json:"message"`
}

func parseROSUpdateJSON(data []byte) (installed, latest, channel, status string) {
	var arr []map[string]any
	if err := json.Unmarshal(data, &arr); err == nil && len(arr) > 0 {
		last := arr[len(arr)-1]
		if v, ok := last["installed-version"].(string); ok {
			installed = v
		}
		if v, ok := last["latest-version"].(string); ok {
			latest = v
		}
		if v, ok := last["channel"].(string); ok {
			channel = v
		}
		if v, ok := last["status"].(string); ok {
			status = v
		}
		return
	}
	var single map[string]any
	if err := json.Unmarshal(data, &single); err == nil {
		if v, ok := single["installed-version"].(string); ok {
			installed = v
		}
		if v, ok := single["latest-version"].(string); ok {
			latest = v
		}
		if v, ok := single["channel"].(string); ok {
			channel = v
		}
		if v, ok := single["status"].(string); ok {
			status = v
		}
	}
	return
}

// CheckROSFirmwareUpgradeDetails queries RouterOS system package updates and returns structured details.
func (m *Manager) CheckROSFirmwareUpgradeDetails() (*ROSFirmwareCheckResult, error) {
	m.mu.Lock()
	cfg := m.config
	m.mu.Unlock()

	if cfg.Host == "" {
		return nil, fmt.Errorf("RouterOS host not configured")
	}

	scheme := "https"
	if cfg.Port == "80" {
		scheme = "http"
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}

	checkURL := fmt.Sprintf("%s://%s:%s/rest/system/package/update/check-for-updates", scheme, cfg.Host, cfg.Port)
	checkReq, _ := http.NewRequest(http.MethodPost, checkURL, bytes.NewBuffer([]byte("{}")))
	checkReq.SetBasicAuth(cfg.User, cfg.Password)
	checkReq.Header.Set("Content-Type", "application/json")

	var installed, latest, channel, status string
	if resp, err := client.Do(checkReq); err == nil {
		postBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		installed, latest, channel, status = parseROSUpdateJSON(postBody)
	}

	getURL := fmt.Sprintf("%s://%s:%s/rest/system/package/update", scheme, cfg.Host, cfg.Port)
	// Poll GET if status is still searching
	for i := 0; i < 5; i++ {
		lowerStatus := strings.ToLower(status)
		if latest != "" && !strings.Contains(lowerStatus, "finding") && !strings.Contains(lowerStatus, "checking") {
			break
		}
		time.Sleep(800 * time.Millisecond)
		getReq, err := http.NewRequest(http.MethodGet, getURL, nil)
		if err != nil {
			break
		}
		getReq.SetBasicAuth(cfg.User, cfg.Password)
		if resp, err := client.Do(getReq); err == nil {
			getBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			inst, lat, ch, st := parseROSUpdateJSON(getBody)
			if inst != "" {
				installed = inst
			}
			if lat != "" {
				latest = lat
			}
			if ch != "" {
				channel = ch
			}
			if st != "" {
				status = st
			}
		}
	}

	if channel == "" {
		channel = "stable"
	}
	if status == "" {
		status = "System is already up to date"
	}
	if latest == "" && installed != "" && (strings.Contains(strings.ToLower(status), "up to date") || strings.Contains(strings.ToLower(status), "already")) {
		latest = installed
	}

	hasUpdate := IsVersionNewer(latest, installed)
	msg := ""
	if hasUpdate {
		msg = fmt.Sprintf("发现 RouterOS 官方新固件: %s (当前版本: %s, 升级通道: %s, 状态: %s)", latest, installed, channel, status)
	} else if installed != "" {
		msg = fmt.Sprintf("RouterOS 固件已是最新稳定版本 (当前版本: %s, 升级通道: %s, 状态: %s)", installed, channel, status)
	} else {
		msg = fmt.Sprintf("RouterOS 升级通道状态: %s (通道: %s)", status, channel)
	}

	return &ROSFirmwareCheckResult{
		InstalledVersion: installed,
		LatestVersion:    latest,
		Channel:          channel,
		Status:           status,
		HasUpdate:        hasUpdate,
		Message:          msg,
	}, nil
}

// CheckROSFirmwareUpgrade queries RouterOS system package updates via REST API.
func (m *Manager) CheckROSFirmwareUpgrade() (string, error) {
	res, err := m.CheckROSFirmwareUpgradeDetails()
	if err != nil {
		return "", err
	}
	return res.Message, nil
}

// TriggerROSFirmwareUpgrade sends an upgrade command to RouterOS via its REST API.
func (m *Manager) TriggerROSFirmwareUpgrade() error {
	// First check whether an update actually exists
	checkRes, err := m.CheckROSFirmwareUpgradeDetails()
	if err == nil && checkRes != nil {
		if !checkRes.HasUpdate && checkRes.InstalledVersion != "" {
			return fmt.Errorf("RouterOS 当前固件已是最新稳定版本 (%s)，无需重复更新", checkRes.InstalledVersion)
		}
	}

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
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte("{}")))
	if err != nil {
		return err
	}
	req.SetBasicAuth(cfg.User, cfg.Password)
	req.Header.Set("Content-Type", "application/json")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
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
	return "Test v2.4"
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

func normalizeVersion(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = strings.TrimPrefix(v, "v")
	v = strings.ReplaceAll(v, " ", "")
	v = strings.ReplaceAll(v, "-", "")
	v = strings.ReplaceAll(v, "_", "")
	return v
}

// extractVersionNumbers parses major, minor, patch, prerelease from strings like "Test v1.7", "Test-v1.4", "v1.19.0".
func extractVersionNumbers(v string) []int {
	v = strings.ToLower(v)
	var b strings.Builder
	for _, r := range v {
		if (r >= '0' && r <= '9') || r == '.' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	parts := strings.Fields(b.String())
	if len(parts) == 0 {
		return nil
	}
	verStr := parts[len(parts)-1]
	segStrs := strings.Split(verStr, ".")
	nums := make([]int, 0, len(segStrs))
	for _, s := range segStrs {
		n, err := strconv.Atoi(s)
		if err == nil {
			nums = append(nums, n)
		}
	}
	return nums
}

// isVersionNewer returns true only if candidate is strictly greater than current.
func IsVersionNewer(candidate, current string) bool {
	candidate = strings.TrimSpace(candidate)
	current = strings.TrimSpace(current)
	if candidate == "" || current == "" {
		return false
	}
	if strings.EqualFold(candidate, current) {
		return false
	}
	cNums := extractVersionNumbers(candidate)
	currNums := extractVersionNumbers(current)
	if len(cNums) == 0 || len(currNums) == 0 {
		return false
	}
	maxLen := len(cNums)
	if len(currNums) > maxLen {
		maxLen = len(currNums)
	}
	for i := 0; i < maxLen; i++ {
		cVal := 0
		if i < len(cNums) {
			cVal = cNums[i]
		}
		currVal := 0
		if i < len(currNums) {
			currVal = currNums[i]
		}
		if cVal > currVal {
			return true
		}
		if cVal < currVal {
			return false
		}
	}
	return false
}

func fetchGitHubLatestTag(repo string) (string, error) {
	// Tier 0: Direct query via local active proxy port (127.0.0.1:7890) if engine is running
	proxyURL, err := url.Parse("http://127.0.0.1:7890")
	if err == nil {
		pClient := &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 4 * time.Second,
		}
		pReq, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/"+repo+"/releases/latest", nil)
		if err == nil {
			pReq.Header.Set("User-Agent", "CubeX-Daemon/1.0")
			pReq.Header.Set("Accept", "application/vnd.github.v3+json")
			pResp, err := pClient.Do(pReq)
			if err == nil && pResp.StatusCode == http.StatusOK {
				var rel struct {
					TagName string `json:"tag_name"`
				}
				if json.NewDecoder(pResp.Body).Decode(&rel) == nil && rel.TagName != "" {
					pResp.Body.Close()
					return rel.TagName, nil
				}
				pResp.Body.Close()
			}
		}
	}

	// Tier 1: Web 302 Redirect via verified fast GitHub mirrors
	redirectCandidates := []string{
		"https://ghfast.top/https://github.com/" + repo + "/releases/latest",
		"https://ghproxy.cn/https://github.com/" + repo + "/releases/latest",
		"https://github.com/" + repo + "/releases/latest",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, targetURL := range redirectCandidates {
		req, err := http.NewRequest(http.MethodHead, targetURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

		resp, err := client.Do(req)
		if err != nil {
			req.Method = http.MethodGet
			resp, err = client.Do(req)
			if err != nil {
				continue
			}
		}

		loc := resp.Header.Get("Location")
		resp.Body.Close()

		if loc != "" && strings.Contains(loc, "/releases/tag/") {
			u, err := url.Parse(loc)
			var targetPath string
			if err == nil {
				targetPath = u.Path
			} else {
				targetPath = loc
			}
			tag := path.Base(targetPath)
			if tag != "" && tag != "." && tag != "/" && tag != "tag" {
				return tag, nil
			}
		}
	}

	// Tier 2: GitHub API fallback (direct)
	apiCandidates := []string{
		"https://api.github.com/repos/" + repo + "/releases/latest",
	}
	apiClient := &http.Client{Timeout: 6 * time.Second}
	var lastErr error
	for _, targetURL := range apiCandidates {
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("User-Agent", "CubeX-Daemon/1.0")
		req.Header.Set("Accept", "application/vnd.github.v3+json")

		resp, err := apiClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, targetURL)
			continue
		}

		var release struct {
			TagName string `json:"tag_name"`
		}
		err = json.NewDecoder(resp.Body).Decode(&release)
		resp.Body.Close()
		if err == nil && release.TagName != "" {
			return release.TagName, nil
		}
	}

	// Tier 3: If repo is CubeX itself, fallback to raw meta.json on main branch
	if strings.Contains(strings.ToLower(repo), "cubex") {
		metaURLs := []string{
			"https://ghfast.top/https://raw.githubusercontent.com/" + repo + "/main/meta.json",
			"https://raw.githubusercontent.com/" + repo + "/main/meta.json",
		}
		for _, mURL := range metaURLs {
			req, err := http.NewRequest(http.MethodGet, mURL, nil)
			if err != nil {
				continue
			}
			req.Header.Set("User-Agent", "CubeX-Daemon/1.0")
			resp, err := client.Do(req)
			if err == nil && resp.StatusCode == http.StatusOK {
				var meta struct {
					Version string `json:"version"`
				}
				if json.NewDecoder(resp.Body).Decode(&meta) == nil && meta.Version != "" {
					resp.Body.Close()
					return meta.Version, nil
				}
				resp.Body.Close()
			}
		}
	}

	return "", lastErr
}

func (m *Manager) fetchLatestPanelVersion() (string, error) {
	tag, err := fetchGitHubLatestTag("zyeyefyass/CubeX")
	if err != nil || tag == "" {
		return m.getLocalPanelVersion(), nil
	}
	return tag, nil
}

func (m *Manager) getLocalMihomoVersion() string {
	mihomoPath := filepath.Join(m.binDir, "mihomo")
	out, err := exec.Command(mihomoPath, "-v").CombinedOutput()
	if err != nil {
		return "v1.19.33"
	}
	s := string(out)
	for _, field := range strings.Fields(s) {
		if strings.HasPrefix(field, "v") && strings.Contains(field, ".") {
			return field
		}
	}
	return "v1.19.33"
}

func (m *Manager) fetchLatestMihomoVersion() (string, error) {
	tag, err := fetchGitHubLatestTag("MetaCubeX/mihomo")
	if err != nil || tag == "" {
		return "v1.19.33", err
	}
	return tag, nil
}

func (m *Manager) getLocalSingboxVersion() string {
	singboxPath := filepath.Join(m.binDir, "sing-box")
	out, err := exec.Command(singboxPath, "version").CombinedOutput()
	if err != nil {
		return "v1.14.3"
	}
	s := string(out)
	for _, field := range strings.Fields(s) {
		if strings.HasPrefix(field, "v") && strings.Contains(field, ".") {
			return field
		}
	}
	return "v1.14.3"
}

func (m *Manager) fetchLatestSingboxVersion() (string, error) {
	tag, err := fetchGitHubLatestTag("SagerNet/sing-box")
	if err != nil || tag == "" {
		return "v1.14.3", err
	}
	return tag, nil
}

func (m *Manager) checkROSConnection() (connected bool, version string) {
	if m.config.Host == "" || m.config.User == "" {
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
			Version       string `json:"version"`
			BoardName     string `json:"board-name"`
			FreeHddSpace  int64  `json:"free-hdd-space"`
			TotalHddSpace int64  `json:"total-hdd-space"`
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
			if resource.FreeHddSpace > 0 {
				m.lastFreeHddSpace = resource.FreeHddSpace
			}
			if resource.TotalHddSpace > 0 {
				m.lastTotalHddSpace = resource.TotalHddSpace
			}
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

	isLargeDatabase := strings.Contains(destPath, "geoip.metadb") || strings.Contains(destPath, "geosite.dat") || strings.Contains(destPath, "GeoIP.dat") || strings.Contains(destPath, "GeoSite.dat")

	// 1. If local proxy 127.0.0.1:7890 is active, try direct download via local proxy first! (15s timeout)
	proxyURL, pErr := url.Parse("http://127.0.0.1:7890")
	if pErr == nil {
		pClient := &http.Client{
			Transport: &http.Transport{
				Proxy:           http.ProxyURL(proxyURL),
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
			Timeout: 15 * time.Second,
		}
		pReq, err := http.NewRequest(http.MethodGet, cleanGitHubURL, nil)
		if err == nil {
			pReq.Header.Set("User-Agent", "CubeX-Gateway/1.5")
			pResp, err := pClient.Do(pReq)
			if err == nil && pResp.StatusCode == http.StatusOK {
				bodyBytes, readErr := io.ReadAll(pResp.Body)
				pResp.Body.Close()
				if readErr == nil && len(bodyBytes) >= 100 {
					headSample := strings.ToLower(string(bodyBytes[:int(math.Min(float64(len(bodyBytes)), 256))]))
					if !strings.Contains(headSample, "<!doctype") && !strings.Contains(headSample, "<html") && !strings.Contains(headSample, "<!--") {
						if !isLargeDatabase || len(bodyBytes) >= 1000000 {
							if err := os.WriteFile(destPath, bodyBytes, 0755); err == nil {
								return nil
							}
						}
					}
				}
			} else if pResp != nil {
				pResp.Body.Close()
			}
		}
	}

	// 2. Candidates with fast CDN mirrors (ghproxy.cn excluded due to captive HTML challenge pages)
	var candidates []string
	if strings.Contains(rawURL, "jsdelivr.net") {
		candidates = []string{
			rawURL,
			strings.ReplaceAll(rawURL, "gcore.jsdelivr.net", "fastly.jsdelivr.net"),
			strings.ReplaceAll(rawURL, "gcore.jsdelivr.net", "cdn.jsdelivr.net"),
		}
	} else {
		candidates = []string{
			"https://ghfast.top/" + cleanGitHubURL,
			"https://ghproxy.net/" + cleanGitHubURL,
			"https://mirror.ghproxy.com/" + cleanGitHubURL,
			cleanGitHubURL,
		}
	}

	client := &http.Client{Timeout: 15 * time.Second}
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

		bodyBytes, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}

		// Reject HTML error/block pages for binary rule databases
		if len(bodyBytes) < 100 {
			lastErr = fmt.Errorf("payload too small (%d bytes)", len(bodyBytes))
			continue
		}
		if isLargeDatabase && len(bodyBytes) < 1000000 {
			lastErr = fmt.Errorf("database payload too small (%d bytes, minimum 1MB required)", len(bodyBytes))
			continue
		}
		headSample := strings.ToLower(string(bodyBytes[:int(math.Min(float64(len(bodyBytes)), 256))]))
		if strings.Contains(headSample, "<!doctype") || strings.Contains(headSample, "<html") || strings.Contains(headSample, "<!--") {
			lastErr = fmt.Errorf("received HTML page instead of binary database")
			continue
		}

		if err := os.WriteFile(destPath, bodyBytes, 0755); err != nil {
			lastErr = err
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

func (m *Manager) downloadAndExtractTarGz(rawURL, targetBinaryName, destPath string) error {
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

	client := &http.Client{Timeout: 45 * time.Second}
	var lastErr error

	for _, targetURL := range candidates {
		log.Printf("[ROS] Downloading core from: %s", targetURL)
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
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, targetURL)
			continue
		}

		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("gzip reader error: %w", err)
			continue
		}

		tarReader := tar.NewReader(gzReader)
		found := false

		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				lastErr = fmt.Errorf("tar read error: %w", err)
				break
			}

			base := filepath.Base(header.Name)
			if base == targetBinaryName && (header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA) {
				outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
				if err != nil {
					lastErr = fmt.Errorf("cannot create destination: %w", err)
					break
				}
				_, copyErr := io.Copy(outFile, tarReader)
				outFile.Close()
				if copyErr != nil {
					lastErr = fmt.Errorf("copy failed: %w", copyErr)
					_ = os.Remove(destPath)
					break
				}
				found = true
				break
			}
		}

		gzReader.Close()
		resp.Body.Close()

		if found {
			log.Printf("[ROS] Successfully extracted %s to %s", targetBinaryName, destPath)
			return nil
		}
	}
	return fmt.Errorf("all download mirrors failed: %w", lastErr)
}

func (m *Manager) downloadAndExtractZip(rawURL, targetBinaryName, destPath string) error {
	tmpZip := destPath + ".dl.zip"
	defer os.Remove(tmpZip)
	if err := m.downloadFile(rawURL, tmpZip); err != nil {
		return err
	}
	zr, err := zip.OpenReader(tmpZip)
	if err != nil {
		return fmt.Errorf("open zip failed: %w", err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if base == targetBinaryName || base == targetBinaryName+".exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			outFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				rc.Close()
				return err
			}
			_, copyErr := io.Copy(outFile, rc)
			outFile.Close()
			rc.Close()
			if copyErr != nil {
				_ = os.Remove(destPath)
				return copyErr
			}
			log.Printf("[Upgrade] Successfully extracted %s from zip to %s", base, destPath)
			return nil
		}
	}
	return fmt.Errorf("target binary %s not found in zip archive", targetBinaryName)
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
	m.mu.RLock()
	host := m.config.Host
	port := m.config.Port
	user := m.config.User
	pass := m.config.Password
	m.mu.RUnlock()

	if host == "" {
		return nil, fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, host, port, strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	client := m.getHTTPClient()
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
	m.mu.RLock()
	host := m.config.Host
	port := m.config.Port
	user := m.config.User
	pass := m.config.Password
	m.mu.RUnlock()

	if host == "" {
		return fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, host, port, strings.TrimPrefix(path, "/"))
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(user, pass)
	client := m.getHTTPClient()
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
	m.mu.RLock()
	host := m.config.Host
	port := m.config.Port
	user := m.config.User
	pass := m.config.Password
	m.mu.RUnlock()

	if host == "" {
		return nil, fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, host, port, strings.TrimPrefix(path, "/"))
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
	req.SetBasicAuth(user, pass)
	client := m.getHTTPClient()
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
	m.mu.RLock()
	host := m.config.Host
	port := m.config.Port
	user := m.config.User
	pass := m.config.Password
	m.mu.RUnlock()

	if host == "" {
		return fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, host, port, strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, pass)
	client := m.getHTTPClient()
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
	m.mu.RLock()
	host := m.config.Host
	port := m.config.Port
	user := m.config.User
	pass := m.config.Password
	m.mu.RUnlock()

	if host == "" {
		return nil, fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, host, port, strings.TrimPrefix(path, "/"))
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
	req.SetBasicAuth(user, pass)
	client := m.getHTTPClient()
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

// GetROS executes an authenticated GET request against RouterOS REST API.

// GetContainerMemoryCurrent returns the live memory-current (in MB) of the CubeX container as reported by RouterOS.
func (m *Manager) GetContainerMemoryCurrent() (float64, error) {
	m.mu.RLock()
	if time.Since(m.lastContainerMemAt) < 5*time.Second && m.lastContainerMem > 0 {
		cached := m.lastContainerMem
		m.mu.RUnlock()
		return cached, nil
	}
	cached := m.lastContainerMem
	m.mu.RUnlock()

	data, err := m.GetROS("/container")
	if err != nil {
		if cached > 0 {
			return cached, nil
		}
		return 0, err
	}
	var containers []map[string]any
	if err := json.Unmarshal(data, &containers); err != nil {
		if cached > 0 {
			return cached, nil
		}
		return 0, err
	}
	for _, c := range containers {
		name, _ := c["name"].(string)
		comment, _ := c["comment"].(string)
		running, _ := c["running"].(string)
		// Match CubeX container or if only 1 running container
		if strings.Contains(strings.ToLower(name), "cubex") || strings.Contains(strings.ToLower(comment), "cubex") || running == "true" || len(containers) == 1 {
			if memVal, ok := c["memory-current"]; ok {
				var bytesVal float64
				switch v := memVal.(type) {
				case string:
					bytesVal, _ = strconv.ParseFloat(v, 64)
				case float64:
					bytesVal = v
				}
				if bytesVal > 0 {
					memMB := bytesVal / (1024 * 1024)
					m.mu.Lock()
					m.lastContainerMem = memMB
					m.lastContainerMemAt = time.Now()
					m.mu.Unlock()
					return memMB, nil
				}
			}
		}
	}
	if cached > 0 {
		return cached, nil
	}
	return 0, fmt.Errorf("container memory-current not found")
}

func (m *Manager) GetROS(path string) ([]byte, error) {
	m.mu.RLock()
	host := m.config.Host
	port := m.config.Port
	user := m.config.User
	pass := m.config.Password
	m.mu.RUnlock()

	if host == "" {
		return nil, fmt.Errorf("RouterOS not configured")
	}
	scheme := "https"
	if port == "80" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s:%s/rest/%s", scheme, host, port, strings.TrimPrefix(path, "/"))
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	client := m.getHTTPClient()
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
// Geo Data & 217heidai MRS/SRS Rule Providers Management
// ──────────────────────────────────────────────

type GeoRuleFile struct {
	Name        string `json:"name"`
	Filename    string `json:"filename"`
	SizeBytes   int64  `json:"size_bytes"`
	SizeStr     string `json:"size_str"`
	Category    string `json:"category"`
	UpdatedAt   string `json:"updated_at"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Engine      string `json:"engine"` // "Mihomo" | "sing-box" | "双核通用"
	Group       string `json:"group"`  // 对应策略组
}

type GeoDataStatus struct {
	TotalRules     int           `json:"total_rules"`
	Version        string        `json:"version"`
	LastUpdated    string        `json:"last_updated"`
	GeoIPSize      int64         `json:"geoip_size"`
	GeoIPSizeStr   string        `json:"geoip_size_str"`
	GeoIPUpdated   string        `json:"geoip_updated"`
	GeoSiteSize    int64         `json:"geosite_size"`
	GeoSiteSizeStr string        `json:"geosite_size_str"`
	GeoSiteUpdated string        `json:"geosite_updated"`
	RulesCount     int           `json:"rules_count"`
	FilesCount     int           `json:"files_count"`
	Files          []GeoRuleFile `json:"files"`
	RecentLogs     []string      `json:"recent_logs"`
	IsUpdating     bool          `json:"is_updating"`
}

type GeoUpdateResult struct {
	Success      bool          `json:"success"`
	TotalRules   int           `json:"total_rules"`
	Version      string        `json:"version"`
	FilesUpdated int           `json:"files_updated"`
	Logs         []string      `json:"logs"`
	UpdatedAt    string        `json:"updated_at"`
	Files        []GeoRuleFile `json:"files"`
}

type RulesMeta struct {
	TotalRules int      `json:"total_rules"`
	Version    string   `json:"version"`
	UpdatedAt  string   `json:"updated_at"`
	RecentLogs []string `json:"recent_logs"`
}

type ruleItemDef struct {
	Name        string
	Filename    string
	URL         string
	Category    string
	Description string
	Engine      string
	Group       string
}

var allGeoRuleDefs = []ruleItemDef{
	{
		Name:        "Mihomo 高性能 GeoIP 数据库 (geoip.metadb)",
		Filename:    "geoip.metadb",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/release/geoip.metadb",
		Category:    "基础数据库",
		Description: "全球 IP 归属地高精度数据库 (MaxMind 格式，Mihomo 专享)",
		Engine:      "Mihomo",
		Group:       "双核基础IP库",
	},
	{
		Name:        "官方全量 GeoSite 域名分类库 (geosite.dat)",
		Filename:    "geosite.dat",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/release/geosite.dat",
		Category:    "基础数据库",
		Description: "全量分类域名数据库，包含国内外各大服务与 CDN 顶级标识",
		Engine:      "Mihomo",
		Group:       "双核基础域名库",
	},
	{
		Name:        "官方全量 GeoIP 数据库 (GeoIP.dat)",
		Filename:    "GeoIP.dat",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/release/GeoIP.dat",
		Category:    "基础数据库",
		Description: "全量全球 IP 数据库 (MaxMind MMDB 兼容格式)",
		Engine:      "双核通用",
		Group:       "双核基础IP库",
	},
	{
		Name:        "217heidai Mihomo 专享去广告规则 (ads_domain.mrs)",
		Filename:    "rules/ads_domain.mrs",
		URL:         "https://gcore.jsdelivr.net/gh/217heidai/adblockfilters@main/rules/adblockmihomo.mrs",
		Category:    "广告与恶意追踪拦截",
		Description: "Mihomo 专用 MRS 格式 217heidai 全网去广告与恶意追踪过滤",
		Engine:      "Mihomo",
		Group:       "🛑 全网广告拦截",
	},
	{
		Name:        "217heidai sing-box 专享去广告规则 (adblocksingbox.srs)",
		Filename:    "rules/adblocksingbox.srs",
		URL:         "https://gcore.jsdelivr.net/gh/217heidai/adblockfilters@main/rules/adblocksingbox.srs",
		Category:    "广告与恶意追踪拦截",
		Description: "sing-box 1.14+ 专用 SRS 格式 217heidai 全网去广告过滤",
		Engine:      "sing-box",
		Group:       "🛑 全网广告拦截",
	},
	{
		Name:        "局域网与私网保留规则集 (privacy_domain.mrs)",
		Filename:    "rules/privacy_domain.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/privacy_domain.mrs",
		Category:    "广告与恶意追踪拦截",
		Description: "局域网保留 IP、本地私有子网、mDNS 及路由管理网段直连直出",
		Engine:      "Mihomo",
		Group:       "🛑 全网广告拦截",
	},
	{
		Name:        "局域网反追踪规则集 (privacy.srs)",
		Filename:    "rules/privacy.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/privacy.srs",
		Category:    "广告与恶意追踪拦截",
		Description: "sing-box 专用反指纹嗅探与本地隐私保护二进制规则",
		Engine:      "sing-box",
		Group:       "🛑 全网广告拦截",
	},
	{
		Name:        "国内直连域名规则集 (cn_domain.mrs)",
		Filename:    "rules/cn_domain.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/cn_domain.mrs",
		Category:    "基础路由分流",
		Description: "中国大陆各大互联网公司、政企官网、银行及云服务器直连",
		Engine:      "Mihomo",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "国内直连域名规则集 (cn_domain.srs)",
		Filename:    "rules/cn_domain.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/cn_domain.srs",
		Category:    "基础路由分流",
		Description: "中国大陆各大互联网公司、政企官网直连 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "国内 IP 分流规则集 (cn_ip.mrs)",
		Filename:    "rules/cn_ip.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geoip/rules/cn_ip.mrs",
		Category:    "基础路由分流",
		Description: "中国电信/联通/移动/广电真实公网 IP 地址段零延迟物理直连",
		Engine:      "Mihomo",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "国内 IP 分流规则集 (cn_ip.srs)",
		Filename:    "rules/cn_ip.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/rules/cn_ip.srs",
		Category:    "基础路由分流",
		Description: "中国电信/联通/移动/广电真实公网 IP 地址段物理直连 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "国外加速域名规则集 (cn_not_domain.mrs)",
		Filename:    "rules/cn_not_domain.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/cn_not_domain.mrs",
		Category:    "基础路由分流",
		Description: "非国内境外网站与全球互联网节点加速分流",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "国外加速域名规则集 (cn_not_domain.srs)",
		Filename:    "rules/cn_not_domain.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/cn_not_domain.srs",
		Category:    "基础路由分流",
		Description: "非国内境外网站与全球互联网节点加速分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "GFW 拦截域名规则集 (gfw_domain.mrs)",
		Filename:    "rules/gfw_domain.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/gfw_domain.mrs",
		Category:    "基础路由分流",
		Description: "GFW 精准匹配走代理出站，确保敏感受限网站秒级穿透",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "GFW 拦截域名规则集 (gfw.srs)",
		Filename:    "rules/gfw.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/gfw.srs",
		Category:    "基础路由分流",
		Description: "GFW 精准匹配走代理出站 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "Telegram 全球专有 IP 段 (telegram_ip.mrs)",
		Filename:    "rules/telegram_ip.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geoip/rules/telegram_ip.mrs",
		Category:    "基础路由分流",
		Description: "Telegram 官方 DC1~DC5 全球数据中心专有网段秒级路由",
		Engine:      "Mihomo",
		Group:       "Telegram",
	},
	{
		Name:        "Telegram 全球专有 IP 段 (telegram_ip.srs)",
		Filename:    "rules/telegram_ip.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geoip/rules/telegram_ip.srs",
		Category:    "基础路由分流",
		Description: "Telegram 官方 DC1~DC5 全球数据中心专有网段 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Telegram",
	},
	{
		Name:        "OpenAI / ChatGPT (openai.mrs)",
		Filename:    "rules/openai.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/openai.mrs",
		Category:    "应用专线分流",
		Description: "ChatGPT、Codex、Sora 及 OpenAI 官方 API 智能分流",
		Engine:      "Mihomo",
		Group:       "AI",
	},
	{
		Name:        "OpenAI / ChatGPT (openai.srs)",
		Filename:    "rules/openai.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/openai.srs",
		Category:    "应用专线分流",
		Description: "ChatGPT、Codex、Sora 及 OpenAI 官方 API 智能分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "AI",
	},
	{
		Name:        "Google Gemini (google-gemini.mrs)",
		Filename:    "rules/google-gemini.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/google-gemini.mrs",
		Category:    "应用专线分流",
		Description: "Google Gemini 对话界面、多模态与核心大模型分流",
		Engine:      "Mihomo",
		Group:       "Gemini & Antigravity",
	},
	{
		Name:        "Google Gemini (google-gemini.srs)",
		Filename:    "rules/google-gemini.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/google-gemini.srs",
		Category:    "应用专线分流",
		Description: "Google Gemini 对话界面、多模态与核心大模型分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Gemini & Antigravity",
	},
	{
		Name:        "Google DeepMind & Antigravity (google-deepmind.mrs)",
		Filename:    "rules/google-deepmind.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/google-deepmind.mrs",
		Category:    "应用专线分流",
		Description: "Antigravity、AI Studio、MakerSuite 与 DeepMind API 极速专线",
		Engine:      "Mihomo",
		Group:       "Gemini & Antigravity",
	},
	{
		Name:        "Google DeepMind & Antigravity (google-deepmind.srs)",
		Filename:    "rules/google-deepmind.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/google-deepmind.srs",
		Category:    "应用专线分流",
		Description: "Antigravity、AI Studio、MakerSuite 与 DeepMind API 极速专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Gemini & Antigravity",
	},
	{
		Name:        "YouTube 视频流 (youtube.mrs)",
		Filename:    "rules/youtube.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/youtube.mrs",
		Category:    "应用专线分流",
		Description: "YouTube 官方主站、Googlevideo 4K/8K 视频分流",
		Engine:      "Mihomo",
		Group:       "YouTube",
	},
	{
		Name:        "YouTube 视频流 (youtube.srs)",
		Filename:    "rules/youtube.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/youtube.srs",
		Category:    "应用专线分流",
		Description: "YouTube 官方主站、Googlevideo 4K/8K 视频分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "YouTube",
	},
	{
		Name:        "Google 核心服务 (google.mrs)",
		Filename:    "rules/google.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/google.mrs",
		Category:    "应用专线分流",
		Description: "Google 搜索、Gmail、Play 商店及全球基础设施分流",
		Engine:      "Mihomo",
		Group:       "Google",
	},
	{
		Name:        "Google 核心服务 (google.srs)",
		Filename:    "rules/google.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/google.srs",
		Category:    "应用专线分流",
		Description: "Google 搜索、Gmail、Play 商店及全球基础设施分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Google",
	},
	{
		Name:        "Apple 官方服务 (apple.mrs)",
		Filename:    "rules/apple.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/apple.mrs",
		Category:    "应用专线分流",
		Description: "Apple 固件更新、iCloud 同步与 App Store 官方分流",
		Engine:      "Mihomo",
		Group:       "Apple 服务",
	},
	{
		Name:        "Apple 官方服务 (apple.srs)",
		Filename:    "rules/apple.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/apple.srs",
		Category:    "应用专线分流",
		Description: "Apple 固件更新、iCloud 同步与 App Store 官方分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Apple 服务",
	},
	{
		Name:        "Microsoft 微软 (microsoft.mrs)",
		Filename:    "rules/microsoft.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/microsoft.mrs",
		Category:    "应用专线分流",
		Description: "Windows Update、Office 365 及 Azure 云服务分流",
		Engine:      "Mihomo",
		Group:       "Microsoft",
	},
	{
		Name:        "Microsoft 微软 (microsoft.srs)",
		Filename:    "rules/microsoft.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/microsoft.srs",
		Category:    "应用专线分流",
		Description: "Windows Update、Office 365 及 Azure 云服务分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Microsoft",
	},
	{
		Name:        "OneDrive 云盘 (onedrive.mrs)",
		Filename:    "rules/onedrive.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/onedrive.mrs",
		Category:    "应用专线分流",
		Description: "微软 OneDrive 个人与企业版大文件极速同步分流",
		Engine:      "Mihomo",
		Group:       "Microsoft",
	},
	{
		Name:        "OneDrive 云盘 (onedrive.srs)",
		Filename:    "rules/onedrive.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/onedrive.srs",
		Category:    "应用专线分流",
		Description: "微软 OneDrive 个人与企业版大文件极速同步分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Microsoft",
	},
	{
		Name:        "GitHub 开发协同 (github.mrs)",
		Filename:    "rules/github.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/github.mrs",
		Category:    "应用专线分流",
		Description: "GitHub 代码托管、Git Clone、Releases 极速加速",
		Engine:      "Mihomo",
		Group:       "GitHub & Docker",
	},
	{
		Name:        "GitHub 开发协同 (github.srs)",
		Filename:    "rules/github.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/github.srs",
		Category:    "应用专线分流",
		Description: "GitHub 代码托管、Git Clone、Releases 极速加速 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "GitHub & Docker",
	},
	{
		Name:        "Telegram 官方域名 (telegram.mrs)",
		Filename:    "rules/telegram.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/telegram.mrs",
		Category:    "应用专线分流",
		Description: "Telegram 官方网页端、API 节点及文件直连分流",
		Engine:      "Mihomo",
		Group:       "Telegram",
	},
	{
		Name:        "Telegram 官方域名 (telegram.srs)",
		Filename:    "rules/telegram.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/telegram.srs",
		Category:    "应用专线分流",
		Description: "Telegram 官方网页端、API 节点及文件直连分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Telegram",
	},
	{
		Name:        "Twitter / X (twitter.mrs)",
		Filename:    "rules/twitter.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/twitter.mrs",
		Category:    "应用专线分流",
		Description: "Twitter 社交平台、X 媒体流及 Twimg 资源加速分流",
		Engine:      "Mihomo",
		Group:       "Twitter (X)",
	},
	{
		Name:        "Twitter / X (twitter.srs)",
		Filename:    "rules/twitter.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/twitter.srs",
		Category:    "应用专线分流",
		Description: "Twitter 社交平台、X 媒体流及 Twimg 资源加速分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Twitter (X)",
	},
	{
		Name:        "Netflix 流媒体 (netflix.mrs)",
		Filename:    "rules/netflix.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/netflix.mrs",
		Category:    "应用专线分流",
		Description: "奈飞全球原生流媒体与 CDN 节点定向解锁分流",
		Engine:      "Mihomo",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "Netflix 流媒体 (netflix.srs)",
		Filename:    "rules/netflix.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/netflix.srs",
		Category:    "应用专线分流",
		Description: "奈飞全球原生流媒体与 CDN 节点定向解锁分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "Disney+ 流媒体 (disney.mrs)",
		Filename:    "rules/disney.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/disney.mrs",
		Category:    "应用专线分流",
		Description: "迪士尼 Disney+、Star+ 全球专线加速分流",
		Engine:      "Mihomo",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "Disney+ 流媒体 (disney.srs)",
		Filename:    "rules/disney.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/disney.srs",
		Category:    "应用专线分流",
		Description: "迪士尼 Disney+、Star+ 全球专线加速分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "Spotify 音乐 (spotify.mrs)",
		Filename:    "rules/spotify.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/spotify.mrs",
		Category:    "应用专线分流",
		Description: "Spotify 高品质流媒体音乐与歌曲数据加载分流",
		Engine:      "Mihomo",
		Group:       "Spotify",
	},
	{
		Name:        "Spotify 音乐 (spotify.srs)",
		Filename:    "rules/spotify.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/spotify.srs",
		Category:    "应用专线分流",
		Description: "Spotify 高品质流媒体音乐与歌曲数据加载分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Spotify",
	},
	{
		Name:        "Steam 游戏平台 (steam.mrs)",
		Filename:    "rules/steam.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/steam.mrs",
		Category:    "应用专线分流",
		Description: "Steam 商店、社区讨论区及游戏下载 CDN 分流",
		Engine:      "Mihomo",
		Group:       "Steam 游戏",
	},
	{
		Name:        "Steam 游戏平台 (steam.srs)",
		Filename:    "rules/steam.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/steam.srs",
		Category:    "应用专线分流",
		Description: "Steam 商店、社区讨论区及游戏下载 CDN 分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Steam 游戏",
	},
	{
		Name:        "Epic Games (epicgames.mrs)",
		Filename:    "rules/epicgames.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/epicgames.mrs",
		Category:    "应用专线分流",
		Description: "Epic 游戏平台、虚幻引擎商城领取代连",
		Engine:      "Mihomo",
		Group:       "Steam 游戏",
	},
	{
		Name:        "Epic Games (epicgames.srs)",
		Filename:    "rules/epicgames.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/epicgames.srs",
		Category:    "应用专线分流",
		Description: "Epic 游戏平台、虚幻引擎商城领取代连 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Steam 游戏",
	},
	{
		Name:        "腾讯生态 / 微信 / QQ (tencent.mrs)",
		Filename:    "rules/tencent.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/tencent.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "微信、QQ、腾讯会议、腾讯视频及腾讯云直连分流",
		Engine:      "Mihomo",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "腾讯生态 / 微信 / QQ (tencent.srs)",
		Filename:    "rules/tencent.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/tencent.srs",
		Category:    "常用生态与自定义扩展",
		Description: "微信、QQ、腾讯会议、腾讯视频及腾讯云直连分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "阿里生态 / 淘宝 / 支付宝 (alibaba.mrs)",
		Filename:    "rules/alibaba.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/alibaba.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "淘宝、天猫、支付宝、阿里云及高德地图服务直连",
		Engine:      "Mihomo",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "阿里生态 / 淘宝 / 支付宝 (alibaba.srs)",
		Filename:    "rules/alibaba.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/alibaba.srs",
		Category:    "常用生态与自定义扩展",
		Description: "淘宝、天猫、支付宝、阿里云及高德地图服务直连 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "百度服务 / 百度网盘 (baidu.mrs)",
		Filename:    "rules/baidu.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/baidu.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "百度搜索、百度网盘下载加速与百度智能云直连",
		Engine:      "Mihomo",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "百度服务 / 百度网盘 (baidu.srs)",
		Filename:    "rules/baidu.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/baidu.srs",
		Category:    "常用生态与自定义扩展",
		Description: "百度搜索、百度网盘下载加速与百度智能云直连 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "字节跳动 / 抖音 / 飞书 (bytedance.mrs)",
		Filename:    "rules/bytedance.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/bytedance.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "抖音、今日头条、飞书协作及火山引擎加速分流",
		Engine:      "Mihomo",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "字节跳动 / 抖音 / 飞书 (bytedance.srs)",
		Filename:    "rules/bytedance.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/bytedance.srs",
		Category:    "常用生态与自定义扩展",
		Description: "抖音、今日头条、飞书协作及火山引擎加速分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "网易云音乐 / 网易游戏 (netease.mrs)",
		Filename:    "rules/netease.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/netease.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "网易云音乐无损音频、有道词典与网易游戏直连",
		Engine:      "Mihomo",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "网易云音乐 / 网易游戏 (netease.srs)",
		Filename:    "rules/netease.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/netease.srs",
		Category:    "常用生态与自定义扩展",
		Description: "网易云音乐无损音频、有道词典与网易游戏直连 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "Bilibili 哔哩哔哩 (bilibili.mrs)",
		Filename:    "rules/bilibili.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/bilibili.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "哔哩哔哩视频流与海外番剧解锁分流",
		Engine:      "Mihomo",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "Bilibili 哔哩哔哩 (bilibili.srs)",
		Filename:    "rules/bilibili.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/bilibili.srs",
		Category:    "常用生态与自定义扩展",
		Description: "哔哩哔哩视频流与海外番剧解锁分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🎯 国内流量",
	},
	{
		Name:        "Discord 语音与社群 (discord.mrs)",
		Filename:    "rules/discord.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/discord.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "Discord 极客社群、语音频道与 Bot 机器人高速分流",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "Discord 语音与社群 (discord.srs)",
		Filename:    "rules/discord.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/discord.srs",
		Category:    "常用生态与自定义扩展",
		Description: "Discord 极客社群、语音频道与 Bot 机器人高速分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "WhatsApp 通讯 (whatsapp.mrs)",
		Filename:    "rules/whatsapp.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/whatsapp.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "WhatsApp 即时文字、高清语音与视频通话专线",
		Engine:      "Mihomo",
		Group:       "Telegram",
	},
	{
		Name:        "WhatsApp 通讯 (whatsapp.srs)",
		Filename:    "rules/whatsapp.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/whatsapp.srs",
		Category:    "常用生态与自定义扩展",
		Description: "WhatsApp 即时文字、高清语音与视频通话专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Telegram",
	},
	{
		Name:        "LINE 通讯服务 (line.mrs)",
		Filename:    "rules/line.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/line.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "LINE 亚洲流行即时通讯与表情包加载专线",
		Engine:      "Mihomo",
		Group:       "Telegram",
	},
	{
		Name:        "LINE 通讯服务 (line.srs)",
		Filename:    "rules/line.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/line.srs",
		Category:    "常用生态与自定义扩展",
		Description: "LINE 亚洲流行即时通讯与表情包加载专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Telegram",
	},
	{
		Name:        "Signal 加密通讯 (signal.mrs)",
		Filename:    "rules/signal.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/signal.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "Signal 端到端强加密隐私通讯低延迟专线",
		Engine:      "Mihomo",
		Group:       "Telegram",
	},
	{
		Name:        "Signal 加密通讯 (signal.srs)",
		Filename:    "rules/signal.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/signal.srs",
		Category:    "常用生态与自定义扩展",
		Description: "Signal 端到端强加密隐私通讯低延迟专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Telegram",
	},
	{
		Name:        "Amazon Prime Video (primevideo.mrs)",
		Filename:    "rules/primevideo.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/primevideo.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "亚马逊 Prime Video 原生流媒体与 4K 超清影视专线",
		Engine:      "Mihomo",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "Amazon Prime Video (primevideo.srs)",
		Filename:    "rules/primevideo.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/primevideo.srs",
		Category:    "常用生态与自定义扩展",
		Description: "亚马逊 Prime Video 原生流媒体与 4K 超清影视专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "HBO Max / Max (hbo.mrs)",
		Filename:    "rules/hbo.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/hbo.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "HBO Max 原创美剧与华纳影视极速解锁分流",
		Engine:      "Mihomo",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "HBO Max / Max (hbo.srs)",
		Filename:    "rules/hbo.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/hbo.srs",
		Category:    "常用生态与自定义扩展",
		Description: "HBO Max 原创美剧与华纳影视极速解锁分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "Twitch 游戏直播 (twitch.mrs)",
		Filename:    "rules/twitch.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/twitch.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "Twitch 全球游戏电竞赛事与互动直播流畅分流",
		Engine:      "Mihomo",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "Twitch 游戏直播 (twitch.srs)",
		Filename:    "rules/twitch.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/twitch.srs",
		Category:    "常用生态与自定义扩展",
		Description: "Twitch 全球游戏电竞赛事与互动直播流畅分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "巴哈姆特动画疯 (bahamut.mrs)",
		Filename:    "rules/bahamut.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/bahamut.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "巴哈姆特动画疯台湾地区独占番剧解锁分流",
		Engine:      "Mihomo",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "巴哈姆特动画疯 (bahamut.srs)",
		Filename:    "rules/bahamut.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/bahamut.srs",
		Category:    "常用生态与自定义扩展",
		Description: "巴哈姆特动画疯台湾地区独占番剧解锁分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Netflix & 流媒体",
	},
	{
		Name:        "PlayStation / PSN (playstation.mrs)",
		Filename:    "rules/playstation.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/playstation.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "索尼 PS4/PS5 商店下载、联机对战与 PSN 会员专线",
		Engine:      "Mihomo",
		Group:       "Steam 游戏",
	},
	{
		Name:        "PlayStation / PSN (playstation.srs)",
		Filename:    "rules/playstation.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/playstation.srs",
		Category:    "常用生态与自定义扩展",
		Description: "索尼 PS4/PS5 商店下载、联机对战与 PSN 会员专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Steam 游戏",
	},
	{
		Name:        "Nintendo 任天堂 (nintendo.mrs)",
		Filename:    "rules/nintendo.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/nintendo.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "任天堂 Switch eShop 商店下载与 NSO 联机专线",
		Engine:      "Mihomo",
		Group:       "Steam 游戏",
	},
	{
		Name:        "Nintendo 任天堂 (nintendo.srs)",
		Filename:    "rules/nintendo.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/nintendo.srs",
		Category:    "常用生态与自定义扩展",
		Description: "任天堂 Switch eShop 商店下载与 NSO 联机专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Steam 游戏",
	},
	{
		Name:        "EA 艺电平台 (ea.mrs)",
		Filename:    "rules/ea.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/ea.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "EA App / Origin 战地、Apex 英雄联机专线",
		Engine:      "Mihomo",
		Group:       "Steam 游戏",
	},
	{
		Name:        "EA 艺电平台 (ea.srs)",
		Filename:    "rules/ea.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/ea.srs",
		Category:    "常用生态与自定义扩展",
		Description: "EA App / Origin 战地、Apex 英雄联机专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Steam 游戏",
	},
	{
		Name:        "暴雪战网 / Battle.net (blizzard.mrs)",
		Filename:    "rules/blizzard.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/blizzard.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "暴雪战网国服/外服魔兽、暗黑4、守望先锋专线",
		Engine:      "Mihomo",
		Group:       "Steam 游戏",
	},
	{
		Name:        "暴雪战网 / Battle.net (blizzard.srs)",
		Filename:    "rules/blizzard.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/blizzard.srs",
		Category:    "常用生态与自定义扩展",
		Description: "暴雪战网国服/外服魔兽、暗黑4、守望先锋专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Steam 游戏",
	},
	{
		Name:        "币安 Binance (binance.mrs)",
		Filename:    "rules/binance.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/binance.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "全球最大加密货币交易所 Web/App 与 API 毫秒交易专线",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "币安 Binance (binance.srs)",
		Filename:    "rules/binance.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/binance.srs",
		Category:    "常用生态与自定义扩展",
		Description: "全球最大加密货币交易所 Web/App 与 API 毫秒交易专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "欧易 OKX (okx.mrs)",
		Filename:    "rules/okx.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/okx.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "OKX 欧易交易平台与 Web3 钱包极速连接专线",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "欧易 OKX (okx.srs)",
		Filename:    "rules/okx.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/okx.srs",
		Category:    "常用生态与自定义扩展",
		Description: "OKX 欧易交易平台与 Web3 钱包极速连接专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "PayPal 贝宝 (paypal.mrs)",
		Filename:    "rules/paypal.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/paypal.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "PayPal 全球跨境支付与电商结账安全直通",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "PayPal 贝宝 (paypal.srs)",
		Filename:    "rules/paypal.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/paypal.srs",
		Category:    "常用生态与自定义扩展",
		Description: "PayPal 全球跨境支付与电商结账安全直通 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "GitLab 源码托管 (gitlab.mrs)",
		Filename:    "rules/gitlab.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/gitlab.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "GitLab 企业与个人自建/官方代码仓库极速同步",
		Engine:      "Mihomo",
		Group:       "GitHub & Docker",
	},
	{
		Name:        "GitLab 源码托管 (gitlab.srs)",
		Filename:    "rules/gitlab.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/gitlab.srs",
		Category:    "常用生态与自定义扩展",
		Description: "GitLab 企业与个人自建/官方代码仓库极速同步 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "GitHub & Docker",
	},
	{
		Name:        "Notion 笔记协作 (notion.mrs)",
		Filename:    "rules/notion.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/notion.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "Notion 全球云端知识库与协作看板低延迟分流",
		Engine:      "Mihomo",
		Group:       "Microsoft",
	},
	{
		Name:        "Notion 笔记协作 (notion.srs)",
		Filename:    "rules/notion.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/notion.srs",
		Category:    "常用生态与自定义扩展",
		Description: "Notion 全球云端知识库与协作看板低延迟分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Microsoft",
	},
	{
		Name:        "Zoom 视频会议 (zoom.mrs)",
		Filename:    "rules/zoom.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/zoom.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "Zoom 全球音视频跨国会议与屏幕共享分流",
		Engine:      "Mihomo",
		Group:       "Microsoft",
	},
	{
		Name:        "Zoom 视频会议 (zoom.srs)",
		Filename:    "rules/zoom.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/zoom.srs",
		Category:    "常用生态与自定义扩展",
		Description: "Zoom 全球音视频跨国会议与屏幕共享分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "Microsoft",
	},
	{
		Name:        "Amazon / AWS 云服务 (amazon.mrs)",
		Filename:    "rules/amazon.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/amazon.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "AWS 云控制台、S3 存储桶与亚马逊全球电商专线",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "Amazon / AWS 云服务 (amazon.srs)",
		Filename:    "rules/amazon.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/amazon.srs",
		Category:    "常用生态与自定义扩展",
		Description: "AWS 云控制台、S3 存储桶与亚马逊全球电商专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "Speedtest 测速节点 (speedtest.mrs)",
		Filename:    "rules/speedtest.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/speedtest.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "Ookla Speedtest 全球网络测速服务器分流",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "Speedtest 测速节点 (speedtest.srs)",
		Filename:    "rules/speedtest.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/speedtest.srs",
		Category:    "常用生态与自定义扩展",
		Description: "Ookla Speedtest 全球网络测速服务器分流 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "TikTok 国际版 (tiktok.mrs)",
		Filename:    "rules/tiktok.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/tiktok.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "字节跳动 TikTok 全球短视频与电商专线",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "TikTok 国际版 (tiktok.srs)",
		Filename:    "rules/tiktok.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/tiktok.srs",
		Category:    "常用生态与自定义扩展",
		Description: "字节跳动 TikTok 全球短视频与电商专线 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "Reddit 极客社区 (reddit.mrs)",
		Filename:    "rules/reddit.mrs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/geo/geosite/rules/reddit.mrs",
		Category:    "常用生态与自定义扩展",
		Description: "Reddit 极客社区讨论区与媒体流高速加载",
		Engine:      "Mihomo",
		Group:       "🌐 国外流量",
	},
	{
		Name:        "Reddit 极客社区 (reddit.srs)",
		Filename:    "rules/reddit.srs",
		URL:         "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/sing/geo/geosite/rules/reddit.srs",
		Category:    "常用生态与自定义扩展",
		Description: "Reddit 极客社区讨论区与媒体流高速加载 (sing-box 专享)",
		Engine:      "sing-box",
		Group:       "🌐 国外流量",
	},
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// GetGeoDataStatus returns the file sizes, modification timestamps, total rules, and full file list.
func (m *Manager) GetGeoDataStatus() *GeoDataStatus {
	m.mu.Lock()
	isUp := m.isUpdatingGeo
	m.mu.Unlock()

	st := &GeoDataStatus{
		TotalRules:  236255,
		Version:     "最新稳定版",
		IsUpdating:  isUp,
		Files:       make([]GeoRuleFile, 0, len(allGeoRuleDefs)),
		RecentLogs:  make([]string, 0),
	}
	etcDir := filepath.Join(m.rootDir, "etc")
	metaPath := filepath.Join(etcDir, "rules_meta.json")

	// 1. Try reading cached metadata
	if data, err := os.ReadFile(metaPath); err == nil {
		var meta RulesMeta
		if json.Unmarshal(data, &meta) == nil {
			if meta.TotalRules > 0 {
				st.TotalRules = meta.TotalRules
			}
			if meta.Version != "" {
				st.Version = meta.Version
			}
			if meta.UpdatedAt != "" {
				st.LastUpdated = meta.UpdatedAt
			}
			if len(meta.RecentLogs) > 0 {
				st.RecentLogs = meta.RecentLogs
			}
		}
	}

	// 2. Check geoip.metadb
	geoipFile := filepath.Join(etcDir, "geoip.metadb")
	if fi, err := os.Stat(geoipFile); err == nil {
		st.GeoIPSize = fi.Size()
		st.GeoIPSizeStr = formatBytes(fi.Size())
		st.GeoIPUpdated = fi.ModTime().Format("2006-01-02 15:04:05")
		if st.LastUpdated == "" {
			st.LastUpdated = st.GeoIPUpdated
		}
	} else if fi, err := os.Stat(filepath.Join(etcDir, "Country.mmdb")); err == nil {
		st.GeoIPSize = fi.Size()
		st.GeoIPSizeStr = formatBytes(fi.Size())
		st.GeoIPUpdated = fi.ModTime().Format("2006-01-02 15:04:05")
		if st.LastUpdated == "" {
			st.LastUpdated = st.GeoIPUpdated
		}
	}

	// 3. Check geosite.dat
	geositeFile := filepath.Join(etcDir, "geosite.dat")
	if fi, err := os.Stat(geositeFile); err == nil {
		st.GeoSiteSize = fi.Size()
		st.GeoSiteSizeStr = formatBytes(fi.Size())
		st.GeoSiteUpdated = fi.ModTime().Format("2006-01-02 15:04:05")
		if st.LastUpdated == "" || st.GeoSiteUpdated > st.LastUpdated {
			st.LastUpdated = st.GeoSiteUpdated
		}
	}

	// 4. Inspect each rule file
	installedCount := 0
	for _, def := range allGeoRuleDefs {
		targetPath := filepath.Join(etcDir, def.Filename)
		rf := GeoRuleFile{
			Name:        def.Name,
			Filename:    filepath.Base(def.Filename),
			Category:    def.Category,
			Description: def.Description,
			Engine:      def.Engine,
			Group:       def.Group,
		}
		if fi, err := os.Stat(targetPath); err == nil {
			rf.SizeBytes = fi.Size()
			rf.SizeStr = formatBytes(fi.Size())
			rf.UpdatedAt = fi.ModTime().Format("2006-01-02 15:04:05")
			rf.Status = "已装载生效"
			installedCount++
		} else {
			rf.SizeStr = "未装载"
			rf.Status = "未装载"
		}
		st.Files = append(st.Files, rf)
	}

	st.RulesCount = len(allGeoRuleDefs)
	st.FilesCount = installedCount

	return st
}

// fetch217HeidaiMeta attempts to read Blocked domains and Version from 217heidai rules header.
func (m *Manager) fetch217HeidaiMeta() (int, string) {
	urls := []string{
		"https://gcore.jsdelivr.net/gh/217heidai/adblockfilters@main/rules/adblockmihomo.yaml",
		"https://raw.githubusercontent.com/217heidai/adblockfilters/main/rules/adblockmihomo.yaml",
	}
	client := &http.Client{Timeout: 6 * time.Second}
	for _, u := range urls {
		targetURL := u
		if m.URLOptimizer != nil {
			targetURL = m.URLOptimizer(targetURL)
		}
		req, err := http.NewRequest(http.MethodGet, targetURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Range", "bytes=0-2048")
		req.Header.Set("User-Agent", "CubeX-Gateway/1.5")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		content := string(body)

		totalRules := 0
		version := ""
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "# Blocked domains:") {
				val := strings.TrimSpace(strings.TrimPrefix(line, "# Blocked domains:"))
				if n, err := strconv.Atoi(val); err == nil && n > 0 {
					totalRules = n
				}
			} else if strings.HasPrefix(line, "# Version:") {
				version = strings.TrimSpace(strings.TrimPrefix(line, "# Version:"))
			}
		}
		if totalRules > 0 {
			return totalRules, version
		}
	}
	return 236255, "20261021"
}

// UpdateGeoData downloads the latest GeoIP, GeoSite, and 39 217heidai MRS/SRS rules, saves metadata, and reloads core.
func (m *Manager) UpdateGeoData(reloadCoreFn func() error) (*GeoUpdateResult, error) {
	m.mu.Lock()
	if m.isUpdatingGeo {
		m.mu.Unlock()
		return nil, fmt.Errorf("规则库正在全速同步中，请稍候...")
	}
	m.isUpdatingGeo = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.isUpdatingGeo = false
		m.mu.Unlock()
	}()

	res := &GeoUpdateResult{
		Success: false,
		Logs:    make([]string, 0),
		Files:   make([]GeoRuleFile, 0),
	}

	etcDir := filepath.Join(m.rootDir, "etc")
	rulesDir := filepath.Join(etcDir, "rules")
	_ = os.MkdirAll(rulesDir, 0755)

	// Step 1: Probe 217heidai upstream metadata
	totalRules, ruleVer := m.fetch217HeidaiMeta()
	res.TotalRules = totalRules
	res.Version = ruleVer
	res.Logs = append(res.Logs, fmt.Sprintf("📡 连线 217heidai 权威规则源成功 (版本: %s, 规则总计: %d 条)", ruleVer, totalRules))

	// Step 2: Download queue
	type downloadJob struct {
		def  ruleItemDef
		dest string
	}
	jobs := make(chan downloadJob, len(allGeoRuleDefs))
	for _, def := range allGeoRuleDefs {
		jobs <- downloadJob{
			def:  def,
			dest: filepath.Join(etcDir, def.Filename),
		}
	}
	close(jobs)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0

	workerCount := 5
	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				tmpPath := job.dest + ".tmp"
				targetURL := job.def.URL
				if m.URLOptimizer != nil {
					targetURL = m.URLOptimizer(targetURL)
				}
				if err := m.downloadFile(targetURL, tmpPath); err != nil {
					mu.Lock()
					res.Logs = append(res.Logs, fmt.Sprintf("⚠️ %s 同步失败: %v", job.def.Name, err))
					mu.Unlock()
					_ = os.Remove(tmpPath)
					continue
				}
				_ = os.Rename(tmpPath, job.dest)
				fi, statErr := os.Stat(job.dest)
				sizeStr := ""
				if statErr == nil {
					sizeStr = formatBytes(fi.Size())
				}
				mu.Lock()
				successCount++
				res.Logs = append(res.Logs, fmt.Sprintf("✅ %s (%s) 已同步", job.def.Name, sizeStr))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	nowStr := time.Now().Format("2006-01-02 15:04:05")
	res.FilesUpdated = successCount
	res.UpdatedAt = nowStr

	// Save metadata to etc/rules_meta.json
	meta := RulesMeta{
		TotalRules: totalRules,
		Version:    ruleVer,
		UpdatedAt:  nowStr,
		RecentLogs: res.Logs,
	}
	if metaBytes, err := json.MarshalIndent(meta, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(etcDir, "rules_meta.json"), metaBytes, 0644)
	}

	if successCount > 0 {
		res.Success = true
		if reloadCoreFn != nil {
			if err := reloadCoreFn(); err != nil {
				res.Logs = append(res.Logs, fmt.Sprintf("⚠️ 路由内核热重载提示: %v", err))
			} else {
				res.Logs = append(res.Logs, "✨ 路由内核已热重载最新 Geo 数据库与 217heidai 规则体系！")
			}
		}
	} else {
		return res, fmt.Errorf("所有规则数据包下载均未成功，请检查外网连接或代理节点状态")
	}

	// Update live status for response
	st := m.GetGeoDataStatus()
	res.Files = st.Files

	return res, nil
}
