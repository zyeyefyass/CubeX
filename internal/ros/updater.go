package ros

import (
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

	singboxVer := m.getLocalSingboxVersion()
	latestSingbox, _ := m.fetchLatestSingboxVersion()

	normLatestPanel := normalizeVersion(latestPanel)
	normCurrPanel := normalizeVersion(panelVer)
	panelUpdateAvail := latestPanel != "" && normLatestPanel != normCurrPanel

	normLatestMihomo := normalizeVersion(latestMihomo)
	normCurrMihomo := normalizeVersion(mihomoVer)
	mihomoUpdateAvail := latestMihomo != "" && normLatestMihomo != normCurrMihomo

	normLatestSingbox := normalizeVersion(latestSingbox)
	normCurrSingbox := normalizeVersion(singboxVer)
	singboxUpdateAvail := latestSingbox != "" && normLatestSingbox != normCurrSingbox

	result := map[string]any{
		"ros_connected":          rosConnected,
		"ros_version":            rosVersion,
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
func (m *Manager) UpgradePanel(restartFn func() error) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	latestTag, err := m.fetchLatestPanelVersion()
	if err != nil {
		return "", fmt.Errorf("cannot check latest panel release: %w", err)
	}

	currTag := m.getLocalPanelVersion()
	normLatest := normalizeVersion(latestTag)
	normCurr := normalizeVersion(currTag)
	if normLatest == normCurr && latestTag != "" {
		return currTag, fmt.Errorf("panel is already at the latest version %s", currTag)
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

	hasUpdate := latest != "" && installed != "" && latest != installed
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
	data, err := os.ReadFile(m.metaFile)
	if err == nil {
		var meta struct {
			Version string `json:"version"`
		}
		if json.Unmarshal(data, &meta) == nil && meta.Version != "" {
			return meta.Version
		}
	}
	return "Test-v1.3"
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

func fetchGitHubLatestTag(repo string) (string, error) {
	// Tier 1: Web 302 Redirect (Zero API Rate Limit, supported by domestic GHProxy mirrors)
	redirectCandidates := []string{
		"https://ghproxy.net/https://github.com/" + repo + "/releases/latest",
		"https://mirror.ghproxy.com/https://github.com/" + repo + "/releases/latest",
		"https://github.com/" + repo + "/releases/latest",
	}

	client := &http.Client{
		Timeout: 6 * time.Second,
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

	// Tier 2: GitHub API fallback
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
		return "v1.19.32"
	}
	s := string(out)
	for _, field := range strings.Fields(s) {
		if strings.HasPrefix(field, "v") && strings.Contains(field, ".") {
			return field
		}
	}
	return "v1.19.32"
}

func (m *Manager) fetchLatestMihomoVersion() (string, error) {
	tag, err := fetchGitHubLatestTag("MetaCubeX/mihomo")
	if err != nil || tag == "" {
		return "v1.19.32", err
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

// GetROS executes an authenticated GET request against RouterOS REST API.
func (m *Manager) GetROS(path string) ([]byte, error) {
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
		// ── 基础数据库 ──
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

		// ── 7 套核心基础 MRS/SRS 规则包 ──
		{
			name: "1. 217heidai 高速去广告过滤规则 (ads_domain.mrs)",
			url:  "https://gcore.jsdelivr.net/gh/217heidai/adblockfilters@main/rules/adblockmihomo.mrs",
			dest: filepath.Join(rulesDir, "ads_domain.mrs"),
		},
		{
			name: "1b. 217heidai sing-box 独占去广告规则 (adblocksingbox.srs)",
			url:  "https://gcore.jsdelivr.net/gh/217heidai/adblockfilters@main/rules/adblocksingbox.srs",
			dest: filepath.Join(rulesDir, "adblocksingbox.srs"),
		},
		{
			name: "2. 局域网与私网保留规则集 (privacy_domain.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/private.mrs",
			dest: filepath.Join(rulesDir, "privacy_domain.mrs"),
		},
		{
			name: "3. 国内直连域名规则集 (cn_domain.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/cn.mrs",
			dest: filepath.Join(rulesDir, "cn_domain.mrs"),
		},
		{
			name: "4. 国内 IP 分流规则集 (cn_ip.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geoip/cn.mrs",
			dest: filepath.Join(rulesDir, "cn_ip.mrs"),
		},
		{
			name: "5. 国外加速域名规则集 (cn_not_domain.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/geolocation-!cn.mrs",
			dest: filepath.Join(rulesDir, "cn_not_domain.mrs"),
		},
		{
			name: "6. GFW 拦截域名规则集 (gfw_domain.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/gfw.mrs",
			dest: filepath.Join(rulesDir, "gfw_domain.mrs"),
		},
		{
			name: "7. Telegram 全球专有 IP 段规则集 (telegram_ip.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geoip/telegram.mrs",
			dest: filepath.Join(rulesDir, "telegram_ip.mrs"),
		},

		// ── 已选及备用业务分流策略组官方 MRS 规则集 ──
		{
			name: "业务 MRS: OpenAI / ChatGPT (openai.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/openai.mrs",
			dest: filepath.Join(rulesDir, "openai.mrs"),
		},
		{
			name: "业务 MRS: Claude / Anthropic (claude.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/claude.mrs",
			dest: filepath.Join(rulesDir, "claude.mrs"),
		},
		{
			name: "业务 MRS: YouTube (youtube.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/youtube.mrs",
			dest: filepath.Join(rulesDir, "youtube.mrs"),
		},
		{
			name: "业务 MRS: Netflix (netflix.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/netflix.mrs",
			dest: filepath.Join(rulesDir, "netflix.mrs"),
		},
		{
			name: "业务 MRS: Disney+ (disney.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/disney.mrs",
			dest: filepath.Join(rulesDir, "disney.mrs"),
		},
		{
			name: "业务 MRS: Google (google.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/google.mrs",
			dest: filepath.Join(rulesDir, "google.mrs"),
		},
		{
			name: "业务 MRS: GitHub (github.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/github.mrs",
			dest: filepath.Join(rulesDir, "github.mrs"),
		},
		{
			name: "业务 MRS: Telegram 域名 (telegram.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/telegram.mrs",
			dest: filepath.Join(rulesDir, "telegram.mrs"),
		},
		{
			name: "业务 MRS: Twitter (X) (twitter.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/twitter.mrs",
			dest: filepath.Join(rulesDir, "twitter.mrs"),
		},
		{
			name: "业务 MRS: Spotify (spotify.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/spotify.mrs",
			dest: filepath.Join(rulesDir, "spotify.mrs"),
		},
		{
			name: "业务 MRS: Steam 游戏 (steam.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/steam.mrs",
			dest: filepath.Join(rulesDir, "steam.mrs"),
		},
		{
			name: "业务 MRS: TikTok (tiktok.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/tiktok.mrs",
			dest: filepath.Join(rulesDir, "tiktok.mrs"),
		},
		{
			name: "业务 MRS: Bilibili (bilibili.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/bilibili.mrs",
			dest: filepath.Join(rulesDir, "bilibili.mrs"),
		},
		{
			name: "业务 MRS: Discord (discord.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/discord.mrs",
			dest: filepath.Join(rulesDir, "discord.mrs"),
		},
		{
			name: "业务 MRS: Bahamut 巴哈姆特 (bahamut.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/bahamut.mrs",
			dest: filepath.Join(rulesDir, "bahamut.mrs"),
		},
		{
			name: "业务 MRS: Emby 影视 (emby.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/emby.mrs",
			dest: filepath.Join(rulesDir, "emby.mrs"),
		},
		{
			name: "业务 MRS: Epic Games (epicgames.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/epicgames.mrs",
			dest: filepath.Join(rulesDir, "epicgames.mrs"),
		},
		{
			name: "业务 MRS: PlayStation (playstation.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/playstation.mrs",
			dest: filepath.Join(rulesDir, "playstation.mrs"),
		},
		{
			name: "业务 MRS: Nintendo (nintendo.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/nintendo.mrs",
			dest: filepath.Join(rulesDir, "nintendo.mrs"),
		},
		{
			name: "业务 MRS: Amazon (amazon.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/amazon.mrs",
			dest: filepath.Join(rulesDir, "amazon.mrs"),
		},
		{
			name: "业务 MRS: PayPal (paypal.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/paypal.mrs",
			dest: filepath.Join(rulesDir, "paypal.mrs"),
		},
		{
			name: "业务 MRS: Speedtest (speedtest.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/speedtest.mrs",
			dest: filepath.Join(rulesDir, "speedtest.mrs"),
		},
		{
			name: "业务 MRS: Reddit (reddit.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/reddit.mrs",
			dest: filepath.Join(rulesDir, "reddit.mrs"),
		},
		{
			name: "业务 MRS: Notion (notion.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/notion.mrs",
			dest: filepath.Join(rulesDir, "notion.mrs"),
		},
		{
			name: "业务 MRS: Binance (binance.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/binance.mrs",
			dest: filepath.Join(rulesDir, "binance.mrs"),
		},
		{
			name: "业务 MRS: Apple 服务 (apple.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/apple.mrs",
			dest: filepath.Join(rulesDir, "apple.mrs"),
		},
		{
			name: "业务 MRS: Microsoft (microsoft.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/microsoft.mrs",
			dest: filepath.Join(rulesDir, "microsoft.mrs"),
		},
		{
			name: "业务 MRS: OneDrive (onedrive.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/onedrive.mrs",
			dest: filepath.Join(rulesDir, "onedrive.mrs"),
		},
		{
			name: "业务 MRS: Zoom (zoom.mrs)",
			url:  "https://github.com/MetaCubeX/meta-rules-dat/raw/refs/heads/meta/geo/geosite/zoom.mrs",
			dest: filepath.Join(rulesDir, "zoom.mrs"),
		},
	}

	jobs := make(chan downloadItem, len(items))
	for _, it := range items {
		jobs <- it
	}
	close(jobs)

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0

	workerCount := 5
	if len(items) < workerCount {
		workerCount = len(items)
	}

	for w := 0; w < workerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				tmpPath := item.dest + ".tmp"
				targetURL := item.url
				if m.URLOptimizer != nil {
					targetURL = m.URLOptimizer(targetURL)
				}
				if err := m.downloadFile(targetURL, tmpPath); err != nil {
					mu.Lock()
					res.Logs = append(res.Logs, fmt.Sprintf("⚠️ %s 下载失败: %v", item.name, err))
					mu.Unlock()
					_ = os.Remove(tmpPath)
					continue
				}
				_ = os.Rename(tmpPath, item.dest)
				mu.Lock()
				successCount++
				res.Logs = append(res.Logs, fmt.Sprintf("✅ %s 已更新", item.name))
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

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

