package themes

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

var (
	validIDRegex    = regexp.MustCompile(`^[a-z0-9_-]+$`)
	validFileRegex  = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	tokenKeyRegex   = regexp.MustCompile(`^--cbx-[a-z0-9_-]+$`)
	tokenColorRegex = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|rgba?\([0-9.,\s%]+\)|hsla?\([0-9.,\s%deg]+\))$`)
)

const (
	MaxPackageSizeBytes = 35 * 1024 * 1024 // 35 MB hard limit
	DefaultThemeID      = "twilight-default"
	LegacyThemeID       = "legacy-wallpapers"
)

// Manager handles CubeX theme discovery, asset delivery, import, activation, and persistence.
type Manager struct {
	mu            sync.RWMutex
	baseDir       string
	wallpapersDir string
	configFile    string
	activeID      string
	playbackMode  string // "auto", "always", "poster"
	cachedThemes  map[string]ThemeMeta
}

// NewManager creates and initializes the Theme Manager.
func NewManager(baseDir, wallpapersDir string) (*Manager, error) {
	if baseDir == "" {
		baseDir = "/etc/cubex/data/themes"
	}
	if wallpapersDir == "" {
		wallpapersDir = "/etc/cubex/data/wallpapers"
	}

	// Smart fallback: if custom baseDir or wallpapersDir is empty, check system standard /etc/cubex/data
	if baseDir != "/etc/cubex/data/themes" {
		if entries, err := os.ReadDir(baseDir); err != nil || len(entries) == 0 {
			if etcEntries, err := os.ReadDir("/etc/cubex/data/themes"); err == nil && len(etcEntries) > 0 {
				baseDir = "/etc/cubex/data/themes"
			}
		}
	}
	if wallpapersDir != "/etc/cubex/data/wallpapers" {
		if !hasLegacyWallpapers(wallpapersDir) && hasLegacyWallpapers("/etc/cubex/data/wallpapers") {
			wallpapersDir = "/etc/cubex/data/wallpapers"
		}
	}

	_ = os.MkdirAll(baseDir, 0755)
	_ = os.MkdirAll(wallpapersDir, 0755)

	m := &Manager{
		baseDir:       baseDir,
		wallpapersDir: wallpapersDir,
		configFile:    filepath.Join(baseDir, "config.json"),
		activeID:      DefaultThemeID,
		playbackMode:  "auto",
		cachedThemes:  make(map[string]ThemeMeta),
	}

	m.loadConfig()
	m.rescanThemesLocked()
	return m, nil
}

func (m *Manager) loadConfig() {
	data, err := os.ReadFile(m.configFile)
	if err != nil {
		return
	}
	var cfg ThemeConfig
	if err := json.Unmarshal(data, &cfg); err == nil {
		if cfg.ActiveThemeID != "" {
			m.activeID = cfg.ActiveThemeID
		}
		if cfg.PlaybackMode != "" {
			m.playbackMode = cfg.PlaybackMode
		}
	}
}

func (m *Manager) saveConfigLocked() error {
	cfg := ThemeConfig{
		ActiveThemeID: m.activeID,
		PlaybackMode:  m.playbackMode,
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmpFile := m.configFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpFile, m.configFile)
}

func (m *Manager) rescanThemesLocked() {
	themes := make(map[string]ThemeMeta)

	// 1. Built-in default Twilight Crystal theme (vector canvas, zero bandwidth)
	themes[DefaultThemeID] = ThemeMeta{
		ID:          DefaultThemeID,
		Name:        "暮光冰晶 (默认)",
		Version:     "1.0.0",
		Type:        "vector",
		Tier:        "builtin",
		Description: "原创 Twilight Crystal 暮光冰晶几何矢量流光画布 · 0显存开销 · 极致轻量",
		Accent:      "#00F5D4",
		Assets:      map[string]string{},
		Tokens: map[string]string{
			"--cbx-theme-accent":      "#00F5D4",
			"--cbx-theme-accent-glow": "rgba(0, 245, 212, 0.35)",
		},
		IsBuiltin: true,
	}

	// 2. Seamless v9.9.4 Legacy Wallpapers compatibility (/etc/cubex/data/wallpapers/)
	if hasLegacyWallpapers(m.wallpapersDir) {
		themes[LegacyThemeID] = ThemeMeta{
			ID:          LegacyThemeID,
			Name:        "全景主题 (本地相册)",
			Version:     "1.0.0",
			Type:        "multi-image",
			Tier:        "legacy",
			Description: "v9.9.4 本地挂载全景海报 · 随功能标签自动切换 · 0 侵权纯本地映射",
			Accent:      "#38bdf8",
			Assets: map[string]string{
				"overview":    "bg-tab-overview.webp",
				"routing":     "bg-tab-routing.webp",
				"rules":       "bg-tab-rules.webp",
				"system":      "bg-tab-system.webp",
				"diagnostics": "bg-tab-diagnostics.webp",
			},
			Tokens: map[string]string{
				"--cbx-theme-accent":      "#38bdf8",
				"--cbx-theme-accent-glow": "rgba(56, 189, 248, 0.35)",
			},
			IsBuiltin: true,
		}
	}

	// 3. Scan installed themes in baseDir
	entries, err := os.ReadDir(m.baseDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			metaPath := filepath.Join(m.baseDir, entry.Name(), "theme.json")
			metaData, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}
			var meta ThemeMeta
			if err := json.Unmarshal(metaData, &meta); err != nil {
				continue
			}
			if meta.ID == "" {
				meta.ID = entry.Name()
			}
			meta.IsBuiltin = false
			themes[meta.ID] = meta
		}
	}

	m.cachedThemes = themes

	// If current activeID no longer exists, safely fallback to DefaultThemeID
	if _, ok := themes[m.activeID]; !ok {
		m.activeID = DefaultThemeID
		_ = m.saveConfigLocked()
	}
}

func hasLegacyWallpapers(dir string) bool {
	if dir == "" {
		return false
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		if strings.HasSuffix(name, ".webp") || strings.HasSuffix(name, ".png") || strings.HasSuffix(name, ".jpg") {
			return true
		}
	}
	return false
}

// ListThemes returns the list of all available themes and current playback configuration.
func (m *Manager) ListThemes() ThemesResponse {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []ThemeMeta
	// Put active or default first
	for id, t := range m.cachedThemes {
		t.IsActive = (id == m.activeID)
		list = append(list, t)
	}

	return ThemesResponse{
		ActiveThemeID: m.activeID,
		PlaybackMode:  m.playbackMode,
		Themes:        list,
	}
}

// ActivateTheme sets the active theme and playback mode, persisting it atomically.
func (m *Manager) ActivateTheme(id, playbackMode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.cachedThemes[id]; !ok {
		return fmt.Errorf("theme not found: %s", id)
	}

	m.activeID = id
	if playbackMode != "" {
		switch playbackMode {
		case "auto", "always", "poster":
			m.playbackMode = playbackMode
		default:
			return fmt.Errorf("invalid playback mode: %s (must be auto, always, or poster)", playbackMode)
		}
	}

	return m.saveConfigLocked()
}

// GetThemeAsset locates the physical file on disk for a theme asset with strict path traversal defense.
func (m *Manager) GetThemeAsset(id, filename string) (string, error) {
	if !validIDRegex.MatchString(id) {
		return "", errors.New("invalid theme id")
	}

	cleanFile := filepath.Base(filename)
	if cleanFile != filename || !validFileRegex.MatchString(cleanFile) {
		return "", errors.New("invalid asset filename")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var fullPath string
	if id == LegacyThemeID {
		fullPath = filepath.Join(m.wallpapersDir, cleanFile)
	} else {
		fullPath = filepath.Join(m.baseDir, id, cleanFile)
	}

	cleanTarget := filepath.Clean(fullPath)
	expectedRoot := m.baseDir
	if id == LegacyThemeID {
		expectedRoot = m.wallpapersDir
	}

	if !strings.HasPrefix(cleanTarget, filepath.Clean(expectedRoot)) {
		return "", errors.New("access denied: path traversal detected")
	}

	info, err := os.Stat(cleanTarget)
	if err != nil || info.IsDir() {
		return "", os.ErrNotExist
	}

	return cleanTarget, nil
}

// ImportTheme parses and unpacks a .cbxtheme package with strict security gates.
func (m *Manager) ImportTheme(r io.Reader, size int64) (*ThemeMeta, error) {
	if size > MaxPackageSizeBytes {
		return nil, fmt.Errorf("package exceeds maximum size limit (35 MB): got %d bytes", size)
	}

	// 1. Stage zip to a temp file
	tmpZip, err := os.CreateTemp("", "cbx_theme_pkg_*.zip")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		_ = tmpZip.Close()
		_ = os.Remove(tmpZip.Name())
	}()

	written, err := io.Copy(tmpZip, r)
	if err != nil {
		return nil, fmt.Errorf("failed to read package stream: %w", err)
	}
	if written > MaxPackageSizeBytes {
		return nil, fmt.Errorf("stream exceeds maximum size limit (35 MB): got %d bytes", written)
	}

	// 2. Open Zip archive
	zr, err := zip.OpenReader(tmpZip.Name())
	if err != nil {
		return nil, fmt.Errorf("invalid zip archive: %w", err)
	}
	defer zr.Close()

	// 3. Extract to staging directory with ZipSlip & Extension Whitelist checks
	stagingDir, err := os.MkdirTemp("", "cbx_theme_unpack_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create staging dir: %w", err)
	}
	defer os.RemoveAll(stagingDir)

	allowedExts := map[string]bool{
		".json": true,
		".mp4":  true,
		".webm": true,
		".webp": true,
		".png":  true,
		".jpg":  true,
		".jpeg": true,
	}

	for _, f := range zr.File {
		// Strict ZipSlip check
		cleanName := filepath.Clean(f.Name)
		targetPath := filepath.Join(stagingDir, cleanName)
		if !strings.HasPrefix(targetPath, filepath.Clean(stagingDir)+string(os.PathSeparator)) && targetPath != stagingDir {
			return nil, fmt.Errorf("security violation: ZipSlip attack detected in file '%s'", f.Name)
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0755)
			continue
		}

		// Whitelist check
		ext := strings.ToLower(filepath.Ext(cleanName))
		if !allowedExts[ext] {
			return nil, fmt.Errorf("unauthorized file type in theme package: '%s'", cleanName)
		}

		_ = os.MkdirAll(filepath.Dir(targetPath), 0755)

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to write extracted file: %w", err)
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return nil, fmt.Errorf("failed to read zip member '%s': %w", f.Name, err)
		}

		_, copyErr := io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if copyErr != nil {
			return nil, fmt.Errorf("failed to extract file '%s': %w", f.Name, copyErr)
		}
	}

	// 4. Validate theme.json
	themeJSONPath := filepath.Join(stagingDir, "theme.json")
	data, err := os.ReadFile(themeJSONPath)
	if err != nil {
		return nil, errors.New("missing theme.json in theme package")
	}

	var meta ThemeMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("invalid theme.json syntax: %w", err)
	}

	if meta.ID == "" || !validIDRegex.MatchString(meta.ID) {
		return nil, fmt.Errorf("invalid theme ID in theme.json: '%s'", meta.ID)
	}
	if meta.ID == DefaultThemeID || meta.ID == LegacyThemeID {
		return nil, fmt.Errorf("cannot overwrite built-in reserved theme ID: '%s'", meta.ID)
	}

	// 5. CSS Token Double-Ended Whitelist & Injection Defense
	for k, v := range meta.Tokens {
		if !tokenKeyRegex.MatchString(k) {
			return nil, fmt.Errorf("invalid CSS token key: '%s' (must start with --cbx-)", k)
		}
		cleanVal := strings.TrimSpace(v)
		lowerVal := strings.ToLower(cleanVal)
		if strings.Contains(lowerVal, "url(") ||
			strings.Contains(lowerVal, ";") ||
			strings.Contains(lowerVal, "expression") ||
			strings.Contains(lowerVal, "<") ||
			strings.Contains(lowerVal, ">") {
			return nil, fmt.Errorf("CSS token injection attack blocked in '%s': '%s'", k, v)
		}
		if !tokenColorRegex.MatchString(cleanVal) {
			return nil, fmt.Errorf("CSS token value not a valid color in '%s': '%s'", k, v)
		}
	}

	// 6. Atomic Install: os.Rename into baseDir/meta.ID
	destDir := filepath.Join(m.baseDir, meta.ID)
	_ = os.RemoveAll(destDir)
	if err := os.Rename(stagingDir, destDir); err != nil {
		return nil, fmt.Errorf("failed to finalize theme installation: %w", err)
	}

	// 7. Update in-memory registry & auto-activate
	m.mu.Lock()
	meta.IsBuiltin = false
	meta.IsActive = true
	m.cachedThemes[meta.ID] = meta
	m.activeID = meta.ID
	_ = m.saveConfigLocked()
	m.mu.Unlock()

	return &meta, nil
}

// DeleteTheme removes a custom theme and resets to default if it was active.
func (m *Manager) DeleteTheme(id string) error {
	if id == DefaultThemeID || id == LegacyThemeID {
		return errors.New("cannot delete built-in protected theme")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	targetDir := filepath.Join(m.baseDir, id)
	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("failed to remove theme directory: %w", err)
	}

	delete(m.cachedThemes, id)

	if m.activeID == id {
		m.activeID = DefaultThemeID
		_ = m.saveConfigLocked()
	}

	return nil
}
