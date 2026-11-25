package ros

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildROSURLFrom_RejectsSSHPort22(t *testing.T) {
	if _, err := buildROSURLFrom("192.168.2.253", "22", "/routing/table"); err == nil {
		t.Fatalf("expected error when port 22 is passed to buildROSURLFrom, got nil")
	} else if !strings.Contains(err.Error(), "22") {
		t.Fatalf("expected error mentioning port 22, got: %v", err)
	}

	u80, err := buildROSURLFrom("192.168.2.253", "80", "/routing/table")
	if err != nil || u80 != "http://192.168.2.253:80/rest/routing/table" {
		t.Fatalf("unexpected port 80 url=%q err=%v", u80, err)
	}

	uEmpty, err := buildROSURLFrom("192.168.2.253", "", "/ip/route")
	if err != nil || uEmpty != "http://192.168.2.253:80/rest/ip/route" {
		t.Fatalf("unexpected empty port url=%q err=%v", uEmpty, err)
	}

	u443, err := buildROSURLFrom("192.168.2.253", "443", "ip/route")
	if err != nil || u443 != "https://192.168.2.253:443/rest/ip/route" {
		t.Fatalf("unexpected port 443 url=%q err=%v", u443, err)
	}
}

func TestROSManager_SanitizesPort22OnLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}
	legacyCfg := `{"host":"192.168.2.253","port":"22","user":"admin","password":"secret"}`
	if err := os.WriteFile(filepath.Join(dataDir, "ros-config.json"), []byte(legacyCfg), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager(tmpDir)
	cfg := mgr.GetConfig()
	if cfg["port"] != "80" {
		t.Fatalf("expected legacy port 22 sanitized to 80 on load, got %q", cfg["port"])
	}

	if err := mgr.SaveConfig(map[string]string{
		"host": "192.168.2.253",
		"port": "22",
	}); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}
	if mgr.GetConfig()["port"] == "22" {
		t.Fatalf("SaveConfig must not persist SSH port 22 into ros-config.json")
	}
}

func TestVersionComparison_V991AndHotUpgrade(t *testing.T) {
	if !IsVersionNewer("v9.91", "v9.9.9") {
		t.Fatalf("expected v9.91 to be newer than v9.9.9")
	}
	if IsVersionNewer("v9.9.9", "v9.91") {
		t.Fatalf("expected v9.9.9 NOT to be newer than v9.91")
	}
	if IsVersionNewer("v9.91", "v9.91") {
		t.Fatalf("expected identical v9.91 not newer")
	}
	if IsVersionNewer("v9.91", "v9.91.0") {
		t.Fatalf("expected v9.91 and v9.91.0 to be equivalent")
	}
	if !IsVersionNewer("v9.92", "v9.91") {
		t.Fatalf("expected v9.92 to be newer than v9.91")
	}

	nums := extractVersionNumbers("v9.91-rc1")
	if len(nums) != 2 || nums[0] != 9 || nums[1] != 91 {
		t.Fatalf("expected [9, 91] for v9.91-rc1, got %v", nums)
	}

	// Hot-upgrade allowance matrix
	if allow, _ := ShouldAllowPanelUpgrade("v9.91", "v9.9.9", false); !allow {
		t.Fatalf("expected v9.9.9 -> v9.91 upgrade allowed")
	}
	if allow, _ := ShouldAllowPanelUpgrade("v9.91", "v9.91", false); allow {
		t.Fatalf("expected v9.91 -> v9.91 (non-force) blocked as already latest")
	}
	if allow, _ := ShouldAllowPanelUpgrade("v9.91-hotfix", "v9.91", false); !allow {
		t.Fatalf("expected custom suffix tag v9.91-hotfix from v9.91 allowed")
	}
	if allow, _ := ShouldAllowPanelUpgrade("v9.9.9", "v9.91", false); allow {
		t.Fatalf("expected downgrade v9.91 -> v9.9.9 blocked without force")
	}
	if allow, _ := ShouldAllowPanelUpgrade("v9.9.9", "v9.91", true); !allow {
		t.Fatalf("expected forced downgrade/reinstall allowed")
	}

	// Verify effective local version reads meta.json cleanly (for isolated v9.9.9 -> v9.91 simulation)
	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "meta.json"), []byte(`{"version":"v9.9.9"}`), 0644)
	mgr := NewManager(tmpDir)
	if got := mgr.getEffectiveLocalPanelVersion(); got != "v9.9.9" {
		t.Fatalf("expected effective local version v9.9.9 from meta.json, got %q", got)
	}
	if allow, _ := ShouldAllowPanelUpgrade("v9.91", gotVersion(mgr), false); !allow {
		t.Fatalf("expected upgrade from meta v9.9.9 to v9.91 allowed")
	}
}

func gotVersion(m *Manager) string {
	return m.getEffectiveLocalPanelVersion()
}

func TestUpgradePanel_CrossSegmentV999ToV991(t *testing.T) {
	// Verify UpgradePanel version gate + atomic binary staging with mock release server
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	_ = os.MkdirAll(binDir, 0755)
	_ = os.WriteFile(filepath.Join(tmpDir, "meta.json"), []byte(`{"version":"v9.9.9"}`), 0644)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/download/v9.91/cubexd-linux-amd64") ||
			strings.Contains(r.URL.Path, "/releases/download/v9.91/") {
			w.WriteHeader(http.StatusOK)
			payload := make([]byte, 5*1024*1024+64)
			copy(payload, []byte("MOCK-CUBEXD-V991-BINARY"))
			_, _ = w.Write(payload)
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	mgr := NewManager(tmpDir)
	mgr.cachedLatestPanel = "v9.91"
	mgr.URLOptimizer = func(rawURL string) string {
		if idx := strings.Index(rawURL, "/releases/download/"); idx >= 0 {
			return ts.URL + rawURL[idx:]
		}
		return rawURL
	}

	// Simulate check + download path using ShouldAllowPanelUpgrade
	curr := mgr.getEffectiveLocalPanelVersion()
	if allow, reason := ShouldAllowPanelUpgrade("v9.91", curr, false); !allow {
		t.Fatalf("expected v9.9.9 -> v9.91 upgrade allowed, got allow=%v reason=%s", allow, reason)
	}

	destPath := filepath.Join(binDir, "cubexd.new")
	dlURL := mgr.URLOptimizer(fmt.Sprintf("https://github.com/zyeyefyass/CubeX/releases/download/%s/cubexd-linux-amd64", "v9.91"))
	if err := mgr.downloadFile(dlURL, destPath); err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}
	gotBytes, err := os.ReadFile(destPath)
	if err != nil || len(gotBytes) != 5*1024*1024+64 || !strings.HasPrefix(string(gotBytes), "MOCK-CUBEXD-V991-BINARY") {
		t.Fatalf("unexpected downloaded content len=%d err=%v", len(gotBytes), err)
	}
	mgr.updateLocalMetaVersion("v9.91")
	if after := mgr.getEffectiveLocalPanelVersion(); after != "v9.91" {
		t.Fatalf("expected meta version updated to v9.91, got %q", after)
	}
	// Re-checking v9.91 -> v9.91 without force must cleanly report already_latest
	if allow, reason := ShouldAllowPanelUpgrade("v9.91", mgr.getEffectiveLocalPanelVersion(), false); allow || reason != "already_latest" {
		t.Fatalf("expected v9.91 -> v9.91 blocked as already_latest, got allow=%v reason=%s", allow, reason)
	}
}
