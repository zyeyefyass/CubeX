package themes

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func createTestZip(t *testing.T, files map[string]string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatalf("failed to write zip content %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestThemesManager_BasicAndLegacyFallback(t *testing.T) {
	tempBase := t.TempDir()
	tempWallpapers := t.TempDir()

	// Seed a legacy wallpaper
	_ = os.WriteFile(filepath.Join(tempWallpapers, "bg-tab-overview.webp"), []byte("RIFFfakeWEBPdata"), 0644)

	mgr, err := NewManager(tempBase, tempWallpapers)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	resp := mgr.ListThemes()
	if resp.ActiveThemeID != DefaultThemeID {
		t.Errorf("expected active theme %s, got %s", DefaultThemeID, resp.ActiveThemeID)
	}

	hasDefault := false
	hasLegacy := false
	for _, th := range resp.Themes {
		if th.ID == DefaultThemeID {
			hasDefault = true
		}
		if th.ID == LegacyThemeID {
			hasLegacy = true
		}
	}
	if !hasDefault {
		t.Errorf("missing default theme in response")
	}
	if !hasLegacy {
		t.Errorf("missing legacy wallpapers theme in response")
	}

	// Test asset retrieval for legacy wallpaper
	assetPath, err := mgr.GetThemeAsset(LegacyThemeID, "bg-tab-overview.webp")
	if err != nil {
		t.Fatalf("GetThemeAsset for legacy failed: %v", err)
	}
	if assetPath == "" || !filepath.IsAbs(assetPath) {
		t.Errorf("invalid assetPath: %s", assetPath)
	}

	// Test path traversal defense
	_, err = mgr.GetThemeAsset(LegacyThemeID, "../../../etc/passwd")
	if err == nil {
		t.Errorf("expected error for path traversal, got nil")
	}
}

func TestThemesManager_Import_ValidAndActivate(t *testing.T) {
	tempBase := t.TempDir()
	tempWallpapers := t.TempDir()
	mgr, _ := NewManager(tempBase, tempWallpapers)

	meta := ThemeMeta{
		ID:          "dark-skull-luxury",
		Name:        "暗夜曜金 · 奢华骷髅",
		Version:     "1.0.0",
		Type:        "video",
		Tier:        "free-fan",
		Description: "黑金奢华动态主题",
		Accent:      "#E5B95C",
		Assets: map[string]string{
			"video":   "bg.mp4",
			"poster":  "poster.webp",
			"preview": "preview.webp",
		},
		Tokens: map[string]string{
			"--cbx-theme-accent":      "#E5B95C",
			"--cbx-theme-accent-glow": "rgba(229, 185, 92, 0.35)",
			"--cbx-theme-card-bg":     "rgba(10, 12, 18, 0.75)",
		},
	}
	metaJSON, _ := json.Marshal(meta)

	validZip := createTestZip(t, map[string]string{
		"theme.json":   string(metaJSON),
		"bg.mp4":       "fake-mp4-data-0123456789",
		"poster.webp":  "fake-poster-data",
		"preview.webp": "fake-preview-data",
	})

	imported, err := mgr.ImportTheme(bytes.NewReader(validZip), int64(len(validZip)))
	if err != nil {
		t.Fatalf("ImportTheme failed on valid zip: %v", err)
	}
	if imported.ID != "dark-skull-luxury" {
		t.Errorf("expected imported ID dark-skull-luxury, got %s", imported.ID)
	}

	// Verify it auto-activated
	resp := mgr.ListThemes()
	if resp.ActiveThemeID != "dark-skull-luxury" {
		t.Errorf("expected active theme dark-skull-luxury, got %s", resp.ActiveThemeID)
	}

	// Test activation & mode update
	if err := mgr.ActivateTheme("dark-skull-luxury", "always"); err != nil {
		t.Fatalf("ActivateTheme failed: %v", err)
	}
	resp = mgr.ListThemes()
	if resp.PlaybackMode != "always" {
		t.Errorf("expected playback_mode always, got %s", resp.PlaybackMode)
	}

	// Test invalid playback mode
	if err := mgr.ActivateTheme("dark-skull-luxury", "invalid_mode"); err == nil {
		t.Errorf("expected error on invalid playback mode, got nil")
	}
}

func TestThemesManager_Security_ZipSlip(t *testing.T) {
	tempBase := t.TempDir()
	mgr, _ := NewManager(tempBase, "")

	metaJSON := `{"id":"zipslip-theme","name":"Evil","type":"video"}`
	zipSlipData := createTestZip(t, map[string]string{
		"theme.json":              metaJSON,
		"../../../../etc/evil.sh": "echo hacked",
	})

	_, err := mgr.ImportTheme(bytes.NewReader(zipSlipData), int64(len(zipSlipData)))
	if err == nil {
		t.Fatalf("expected ZipSlip attack to be blocked, but import succeeded")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("ZipSlip")) {
		t.Logf("ZipSlip blocked with message: %v", err)
	}
}

func TestThemesManager_Security_UnauthorizedFileType(t *testing.T) {
	tempBase := t.TempDir()
	mgr, _ := NewManager(tempBase, "")

	metaJSON := `{"id":"bad-ext-theme","name":"BadExt","type":"video"}`
	badExtData := createTestZip(t, map[string]string{
		"theme.json": metaJSON,
		"trojan.exe": "binary executable code",
	})

	_, err := mgr.ImportTheme(bytes.NewReader(badExtData), int64(len(badExtData)))
	if err == nil {
		t.Fatalf("expected executable file to be blocked, but import succeeded")
	}
	t.Logf("Unauthorized file blocked: %v", err)
}

func TestThemesManager_Security_CSSTokenInjection(t *testing.T) {
	tempBase := t.TempDir()
	mgr, _ := NewManager(tempBase, "")

	injections := []struct {
		name   string
		tokens map[string]string
	}{
		{
			name:   "url external tracking injection",
			tokens: map[string]string{"--cbx-theme-accent": "url(https://attacker.com/leak)"},
		},
		{
			name:   "semicolon statement injection",
			tokens: map[string]string{"--cbx-theme-accent": "#000; position: fixed;"},
		},
		{
			name:   "expression injection",
			tokens: map[string]string{"--cbx-theme-accent": "expression(alert(1))"},
		},
		{
			name:   "invalid token key prefix",
			tokens: map[string]string{"color": "#ffffff"},
		},
	}

	for _, tc := range injections {
		t.Run(tc.name, func(t *testing.T) {
			meta := ThemeMeta{
				ID:     "inject-theme",
				Name:   "Inject",
				Type:   "video",
				Tokens: tc.tokens,
			}
			metaJSON, _ := json.Marshal(meta)
			zipData := createTestZip(t, map[string]string{
				"theme.json": string(metaJSON),
			})

			_, err := mgr.ImportTheme(bytes.NewReader(zipData), int64(len(zipData)))
			if err == nil {
				t.Fatalf("expected CSS injection '%s' to be blocked, but import succeeded", tc.name)
			}
			t.Logf("CSS injection blocked as expected: %v", err)
		})
	}
}

func TestThemesManager_Security_Oversized(t *testing.T) {
	tempBase := t.TempDir()
	mgr, _ := NewManager(tempBase, "")

	oversized := int64(36 * 1024 * 1024) // 36 MB
	_, err := mgr.ImportTheme(bytes.NewReader([]byte("fake")), oversized)
	if err == nil {
		t.Fatalf("expected oversized package to be blocked, got nil")
	}
}

func TestThemesManager_HTTP206Range(t *testing.T) {
	tempBase := t.TempDir()
	mgr, _ := NewManager(tempBase, "")

	// Create test video payload of 4096 bytes
	testPayload := make([]byte, 4096)
	for i := range testPayload {
		testPayload[i] = byte(i % 256)
	}

	themeDir := filepath.Join(tempBase, "range-test")
	_ = os.MkdirAll(themeDir, 0755)
	_ = os.WriteFile(filepath.Join(themeDir, "bg.mp4"), testPayload, 0644)

	assetPath, err := mgr.GetThemeAsset("range-test", "bg.mp4")
	if err != nil {
		t.Fatalf("GetThemeAsset failed: %v", err)
	}

	// Setup HTTP handler using os.Open + http.ServeContent
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open(assetPath)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			http.Error(w, "stat error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeContent(w, r, "bg.mp4", info.ModTime(), f)
	})

	// 1. Request first 1024 bytes (Range: bytes=0-1023)
	req := httptest.NewRequest("GET", "/api/themes/range-test/assets/bg.mp4", nil)
	req.Header.Set("Range", "bytes=0-1023")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPartialContent {
		t.Fatalf("expected HTTP 206 Partial Content, got %d", rr.Code)
	}

	cr := rr.Header().Get("Content-Range")
	expectedCR := "bytes 0-1023/4096"
	if cr != expectedCR {
		t.Errorf("expected Content-Range %s, got %s", expectedCR, cr)
	}

	if rr.Body.Len() != 1024 {
		t.Errorf("expected 1024 bytes body, got %d", rr.Body.Len())
	}

	// Verify exact byte equality
	if !bytes.Equal(rr.Body.Bytes(), testPayload[0:1024]) {
		t.Errorf("partial content byte slice mismatch")
	}

	// 2. Request last 512 bytes (Range: bytes=3584-4095)
	req2 := httptest.NewRequest("GET", "/api/themes/range-test/assets/bg.mp4", nil)
	req2.Header.Set("Range", "bytes=3584-4095")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusPartialContent {
		t.Fatalf("expected HTTP 206 Partial Content, got %d", rr2.Code)
	}

	if !bytes.Equal(rr2.Body.Bytes(), testPayload[3584:4096]) {
		t.Errorf("last 512 bytes mismatch")
	}
}
