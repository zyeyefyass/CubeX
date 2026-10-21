package host

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestTestSSH_RouterOSPort80_AuthFailure(t *testing.T) {
	// Mock HTTP server returning 401 Unauthorized like RouterOS REST API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/system/resource" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":   401,
				"message": "Unauthorized",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	// Extract port and host from ts.URL
	parts := strings.Split(strings.TrimPrefix(ts.URL, "http://"), ":")
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])

	m := NewManager(t.TempDir())
	cfg := HostConfig{
		Host:     host,
		Port:     port,
		User:     "zyy",
		Password: "wrongpassword",
	}

	_, err := m.TestSSH(cfg)
	if err == nil {
		t.Fatalf("expected error on 401 Unauthorized, got nil")
	}

	// Must contain explicit authentication failure, NOT "ssh: handshake failed"
	if !strings.Contains(err.Error(), "鉴权失败") && !strings.Contains(err.Error(), "401") {
		t.Fatalf("expected clear 401 auth error, got: %v", err)
	}
	if strings.Contains(err.Error(), "ssh: handshake failed") {
		t.Fatalf("unexpected misleading SSH error on HTTP port: %v", err)
	}
}

func TestTestSSH_RouterOSPort80_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/system/resource" {
			user, pass, ok := r.BasicAuth()
			if !ok || user != "zyy" || pass != "820130" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"platform":          "MikroTik",
				"architecture-name": "x86_64",
				"board-name":        "CHR",
				"version":           "7.24.5 (stable)",
				"cpu-count":         "4",
				"cpu-load":          "1",
				"total-memory":      "2080374784",
				"free-memory":       "1521696768",
				"uptime":            "2h43m9s",
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer ts.Close()

	parts := strings.Split(strings.TrimPrefix(ts.URL, "http://"), ":")
	host := parts[0]
	port, _ := strconv.Atoi(parts[1])

	m := NewManager(t.TempDir())
	cfg := HostConfig{
		Host:     host,
		Port:     port,
		User:     "zyy",
		Password: "820130",
	}

	res, err := m.TestSSH(cfg)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if res["platform_id"] != "routeros" {
		t.Errorf("expected platform_id routeros, got: %v", res["platform_id"])
	}
	if res["kernel"] != "7.24.5 (stable)" {
		t.Errorf("expected kernel version 7.24.5 (stable), got: %v", res["kernel"])
	}
}
