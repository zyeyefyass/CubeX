package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/host"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestConsoleAuthAndMiddleware(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := NewConsoleAuthManager(tmpDir)

	if mgr.IsEnabled() {
		t.Fatalf("expected auth to be disabled by default")
	}

	protected := authMiddleware(mgr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	// 1. Unlocked by default
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/ir", nil)
	rec := httptest.NewRecorder()
	protected.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when unlocked, got %d", rec.Code)
	}

	// 2. Enable password lock
	if err := mgr.SetPassword("", "my-secret-pass"); err != nil {
		t.Fatalf("SetPassword failed: %v", err)
	}
	if !mgr.IsEnabled() {
		t.Fatalf("expected auth to be enabled")
	}

	// Reject short password change if wrong current password
	if err := mgr.SetPassword("wrong", "new-pass-123"); err == nil {
		t.Fatalf("expected error on wrong current password")
	}

	// Unauthenticated request to /api/v1/config/ir must return 401
	rec2 := httptest.NewRecorder()
	protected.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/api/v1/config/ir", nil))
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when locked, got %d", rec2.Code)
	}

	// Exempt /api/health and /api/v1/auth/status
	recHealth := httptest.NewRecorder()
	protected.ServeHTTP(recHealth, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected 200 for /api/health, got %d", recHealth.Code)
	}

	// Authenticated via cookie
	token, err := mgr.CreateSession()
	if err != nil || token == "" {
		t.Fatalf("CreateSession failed: %v", err)
	}
	reqAuth := httptest.NewRequest(http.MethodGet, "/api/v1/config/ir", nil)
	reqAuth.AddCookie(&http.Cookie{Name: consoleAuthCookieName, Value: token})
	recAuth := httptest.NewRecorder()
	protected.ServeHTTP(recAuth, reqAuth)
	if recAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 with valid cookie, got %d", recAuth.Code)
	}

	// Disable lock with valid current password
	if err := mgr.SetPassword("my-secret-pass", ""); err != nil {
		t.Fatalf("failed to disable password: %v", err)
	}
	if mgr.IsEnabled() {
		t.Fatalf("expected auth disabled after clearing password")
	}
}

func TestIsTrustedOrigin(t *testing.T) {
	cases := []struct {
		origin string
		host   string
		want   bool
	}{
		{"", "10.0.0.7:2026", true},
		{"http://localhost:5173", "127.0.0.1:2026", true},
		{"http://192.168.31.10:2026", "192.168.31.10:2026", true},
		{"http://10.0.0.5", "10.0.0.7:2026", true},
		{"https://cubex.lan", "cubex.lan", true},
		{"https://evil-attacker.example.com", "10.0.0.7:2026", false},
	}
	for _, tc := range cases {
		if got := isTrustedOrigin(tc.origin, tc.host); got != tc.want {
			t.Errorf("isTrustedOrigin(%q, %q) = %v, want %v", tc.origin, tc.host, got, tc.want)
		}
	}
	if strings.Contains("abc", "xyz") {
		t.Fail()
	}
}

func TestSecretMaskingAndRestore(t *testing.T) {
	stored := host.HostConfig{
		Enabled:    true,
		Host:       "192.168.10.1",
		Port:       22,
		User:       "root",
		Password:   "super-secret-pass",
		PrivateKey: "-----BEGIN PRIVATE KEY-----\nABC",
	}
	masked := maskHostConfigResponse(stored)
	if masked.Password != schema.SecretMaskPlaceholder || masked.PrivateKey != schema.SecretMaskPlaceholder {
		t.Fatalf("expected masked secrets, got pass=%q key=%q", masked.Password, masked.PrivateKey)
	}
	if masked.Host != stored.Host || masked.User != stored.User {
		t.Fatalf("non-secret fields altered: %+v", masked)
	}

	incoming := host.HostConfig{
		Host:       "192.168.10.1",
		User:       "root",
		Password:   schema.SecretMaskPlaceholder,
		PrivateKey: schema.SecretMaskPlaceholder,
	}
	restoreHostMaskedSecrets(&incoming, stored)
	if incoming.Password != stored.Password || incoming.PrivateKey != stored.PrivateKey {
		t.Fatalf("expected stored secrets restored, got pass=%q key=%q", incoming.Password, incoming.PrivateKey)
	}

	rosMasked := maskROSConfigResponse(map[string]string{
		"host":     "192.168.10.1",
		"password": "ros-password",
	})
	if rosMasked["password"] != schema.SecretMaskPlaceholder {
		t.Fatalf("expected ros password masked, got %q", rosMasked["password"])
	}
}

func TestConsoleAuth_CorruptConfigFailsClosedAndEngineModeValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "console-auth.json"), []byte("{corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	mgr := NewConsoleAuthManager(tmpDir)
	if !mgr.IsEnabled() {
		t.Fatalf("expected corrupt console-auth.json to fail closed (IsEnabled=true)")
	}
	if mgr.VerifyPassword("anything") {
		t.Fatalf("expected corrupt auth config to reject any login attempt")
	}

	engMgr := engine.NewManager(tmpDir, schema.NewDefaultIR())
	defer engMgr.Stop()
	srv := &Server{workDir: tmpDir, engMgr: engMgr, ir: engMgr.GetIR()}

	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/engine/mode", strings.NewReader(`{"mode":"invalid-mode"}`))
	badRec := httptest.NewRecorder()
	srv.handleEngineMode(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid proxy mode, got %d", badRec.Code)
	}

	okReq := httptest.NewRequest(http.MethodPost, "/api/v1/engine/mode", strings.NewReader(`{"mode":"global"}`))
	okRec := httptest.NewRecorder()
	srv.handleEngineMode(okRec, okReq)
	if okRec.Code != http.StatusOK && okRec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 200 (or 500 only if switch fails without binary), got %d", okRec.Code)
	}
	if engMgr.GetIR().Meta.ProxyMode != schema.ModeGlobal {
		t.Fatalf("expected ProxyMode persisted to IR as global, got %s", engMgr.GetIR().Meta.ProxyMode)
	}
}
