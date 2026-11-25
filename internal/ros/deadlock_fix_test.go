package ros

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestROSManagerNoDeadlockAndNoRace(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/rest/system/resource" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"7.25.2","board-name":"x86","free-hdd-space":"1073741824","total-hdd-space":"2147483648"}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir)
	defer mgr.Close()

	if err := mgr.SaveConfig(map[string]string{
		"host":     host,
		"port":     port,
		"user":     "tester",
		"password": "secret",
	}); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	mgr.SetURLOptimizer(func(rawURL string) string { return rawURL })

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				for j := 0; j < 3; j++ {
					ov := mgr.GetOverview()
					if ov == nil {
						t.Errorf("GetOverview returned nil")
					}
					_ = mgr.IsConnected()
					_, _ = mgr.GetFullPlatformInfo()
					_ = mgr.optimizeURL(fmt.Sprintf("https://example.com/%d/%d", idx, j))
					_ = mgr.GetConfig()
				}
			}(i)
		}
		wg.Wait()
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("deadlock detected: ros.Manager concurrent calls timed out")
	}

	if !mgr.IsConnected() {
		t.Fatalf("expected IsConnected() to be true against mock TLS REST server")
	}
	conn, plat := mgr.GetFullPlatformInfo()
	if !conn || plat == "" {
		t.Fatalf("expected GetFullPlatformInfo connected=true, got conn=%v plat=%q", conn, plat)
	}
}
