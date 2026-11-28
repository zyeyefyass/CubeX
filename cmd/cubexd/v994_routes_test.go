package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func newTestServerWithDir(t *testing.T) *Server {
	t.Helper()
	tmpDir := t.TempDir()
	engMgr := engine.NewManager(tmpDir, schema.NewDefaultIR())
	srv := &Server{
		workDir:   tmpDir,
		engMgr:    engMgr,
		ir:        engMgr.GetIR(),
		logBuffer: NewRingLogBuffer(50),
		stopCh:    make(chan struct{}),
	}
	t.Cleanup(func() {
		srv.Close()
		engMgr.Stop()
	})
	return srv
}

func TestAPI1_ConfigFakeIPFilter(t *testing.T) {
	srv := newTestServerWithDir(t)
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// 1. GET defaults
	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/fakeip-filter", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET fakeip-filter status = %d, want 200", rec.Code)
	}
	var getResp struct {
		BuiltinEnabled bool     `json:"builtin_enabled"`
		BuiltinCount   int      `json:"builtin_count"`
		CustomDomains  []string `json:"custom_domains"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &getResp); err != nil {
		t.Fatal(err)
	}
	if !getResp.BuiltinEnabled || getResp.BuiltinCount < 32 {
		t.Fatalf("unexpected default fakeip-filter response: %+v", getResp)
	}

	// 2. POST invalid domain -> 400
	badReq := httptest.NewRequest(http.MethodPost, "/api/v1/config/fakeip-filter", strings.NewReader(`{"builtin_enabled":true,"custom_domains":["bad domain/path"]}`))
	badRec := httptest.NewRecorder()
	mux.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 on invalid domain, got %d (%s)", badRec.Code, badRec.Body.String())
	}

	// 3. POST valid update -> persists to IR and returns normalized response
	postBody := `{"builtin_enabled":false,"custom_domains":[" NAS.Local. ","+.Corp.Internal","nas.local"]}`
	postReq := httptest.NewRequest(http.MethodPost, "/api/v1/config/fakeip-filter", strings.NewReader(postBody))
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("POST fakeip-filter status = %d, body=%s", postRec.Code, postRec.Body.String())
	}

	var postResp struct {
		BuiltinEnabled bool     `json:"builtin_enabled"`
		BuiltinCount   int      `json:"builtin_count"`
		CustomDomains  []string `json:"custom_domains"`
	}
	if err := json.Unmarshal(postRec.Body.Bytes(), &postResp); err != nil {
		t.Fatal(err)
	}
	if postResp.BuiltinEnabled {
		t.Fatalf("expected builtin_enabled=false after update")
	}
	if len(postResp.CustomDomains) != 2 || postResp.CustomDomains[0] != "nas.local" || postResp.CustomDomains[1] != "+.corp.internal" {
		t.Fatalf("unexpected normalized custom_domains: %v", postResp.CustomDomains)
	}

	// Verify persisted on disk in config.ir.json
	rawIR, err := os.ReadFile(filepath.Join(srv.workDir, "data", "config.ir.json"))
	if err != nil {
		t.Fatal(err)
	}
	var diskIR schema.ConfigIR
	if err := json.Unmarshal(rawIR, &diskIR); err != nil {
		t.Fatal(err)
	}
	if diskIR.DNS.IsFakeIPBuiltinEnabled() {
		t.Fatalf("expected persisted IR fakeip_builtin_enabled=false")
	}
}

func startMockDNSServer(t *testing.T, replyIP [4]byte) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = pc.Close() })

	SafeGo("TestMockDNS", func() {
		buf := make([]byte, 512)
		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			if n < 12 {
				continue
			}
			// Echo query + append 1 A-record answer with pointer 0xC00C
			resp := make([]byte, 0, n+16)
			resp = append(resp, buf[:n]...)
			resp[2] = 0x81 // QR=1, RD=1
			resp[3] = 0x80 // RA=1
			binary.BigEndian.PutUint16(resp[6:8], 1)
			resp = append(resp,
				0xC0, 0x0C, // Name pointer to question
				0x00, 0x01, // Type A
				0x00, 0x01, // Class IN
				0x00, 0x00, 0x00, 0x3C, // TTL 60
				0x00, 0x04, // RDLENGTH 4
				replyIP[0], replyIP[1], replyIP[2], replyIP[3],
			)
			_, _ = pc.WriteTo(resp, addr)
		}
	})
	return pc.LocalAddr().String()
}

func TestAPI2_ThreeLightCheck(t *testing.T) {
	srv := newTestServerWithDir(t)

	// 1. Mock domestic TCP target
	domLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer domLn.Close()
	SafeGo("TestMockDomesticTCP", func() {
		for {
			c, err := domLn.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	})

	// 2. Mock HTTP proxy server returning 204
	proxySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer proxySrv.Close()
	proxyURL, _ := url.Parse(proxySrv.URL)

	// 3. Mock Fake-IP UDP DNS server returning 198.18.0.42
	dnsAddr := startMockDNSServer(t, [4]byte{198, 18, 0, 42})

	srv.threeLightCfg = &threeLightProbeConfig{
		DomesticTargets:  []string{domLn.Addr().String()},
		ProxyAddr:        proxyURL.Host,
		ProxyProbeURL:    "http://cp.cloudflare.com/generate_204",
		DNSAddr:          dnsAddr,
		FakeIPTestDomain: "www.google.com",
		ProbeTimeout:     1 * time.Second,
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	// Without proxy headers -> dual_proxy_hint should be false
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/three-light-check", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("three-light-check status = %d", rec.Code)
	}

	var resp struct {
		Domestic      threeLightConnResult `json:"domestic"`
		Proxy         threeLightConnResult `json:"proxy"`
		FakeIPDNS     threeLightDNSResult  `json:"fakeip_dns"`
		DualProxyHint bool                 `json:"dual_proxy_hint"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.Domestic.OK {
		t.Fatalf("expected domestic ok=true, got %+v", resp.Domestic)
	}
	if !resp.Proxy.OK {
		t.Fatalf("expected proxy ok=true, got %+v", resp.Proxy)
	}
	if !resp.FakeIPDNS.OK || resp.FakeIPDNS.ResolvedIP != "198.18.0.42" {
		t.Fatalf("expected fakeip_dns ok=true with 198.18.0.42, got %+v", resp.FakeIPDNS)
	}
	if resp.DualProxyHint {
		t.Fatalf("expected dual_proxy_hint=false without proxy headers")
	}

	// With Via header -> dual_proxy_hint should be true
	reqVia := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/three-light-check", nil)
	reqVia.Header.Set("Via", "1.1 clash-client")
	recVia := httptest.NewRecorder()
	mux.ServeHTTP(recVia, reqVia)
	var respVia map[string]any
	if err := json.Unmarshal(recVia.Body.Bytes(), &respVia); err != nil {
		t.Fatal(err)
	}
	if hint, _ := respVia["dual_proxy_hint"].(bool); !hint {
		t.Fatalf("expected dual_proxy_hint=true when Via header present, got %v", respVia["dual_proxy_hint"])
	}

	// Non-Fake-IP DNS response (1.2.3.4) -> fakeip_dns.ok == false
	srv.threeLightCfg.DNSAddr = startMockDNSServer(t, [4]byte{1, 2, 3, 4})
	recNonFake := httptest.NewRecorder()
	mux.ServeHTTP(recNonFake, httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/three-light-check", nil))
	var respNonFake struct {
		FakeIPDNS threeLightDNSResult `json:"fakeip_dns"`
	}
	if err := json.Unmarshal(recNonFake.Body.Bytes(), &respNonFake); err != nil {
		t.Fatal(err)
	}
	if respNonFake.FakeIPDNS.OK || respNonFake.FakeIPDNS.ResolvedIP != "1.2.3.4" {
		t.Fatalf("expected fakeip_dns ok=false with resolved_ip=1.2.3.4, got %+v", respNonFake.FakeIPDNS)
	}
}

func TestAPI3_SystemBackupExportAndImport(t *testing.T) {
	srv := newTestServerWithDir(t)
	dataDir := filepath.Join(srv.workDir, "data")
	if err := schema.WriteFileAtomicSecure(filepath.Join(dataDir, "ui_state.json"), []byte(`{"minimal":true}`)); err != nil {
		t.Fatal(err)
	}
	if err := schema.WriteFileAtomicSecure(filepath.Join(dataDir, "runtime.db"), []byte("do-not-export")); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)
	handler := gzipMiddleware(mux)

	// 1. Export with Accept-Encoding: gzip (verifies gzipMiddleware does NOT double-compress!)
	expReq := httptest.NewRequest(http.MethodGet, "/api/v1/system/backup/export", nil)
	expReq.Header.Set("Accept-Encoding", "gzip")
	expRec := httptest.NewRecorder()
	handler.ServeHTTP(expRec, expReq)

	if expRec.Code != http.StatusOK {
		t.Fatalf("export status = %d", expRec.Code)
	}
	if ct := expRec.Header().Get("Content-Type"); ct != "application/gzip" {
		t.Fatalf("Content-Type = %q, want application/gzip", ct)
	}
	if cd := expRec.Header().Get("Content-Disposition"); !strings.Contains(cd, "cubex-config-backup.tar.gz") {
		t.Fatalf("unexpected Content-Disposition: %q", cd)
	}
	if ce := expRec.Header().Get("Content-Encoding"); ce != "" {
		t.Fatalf("expected empty Content-Encoding (no double gzip), got %q", ce)
	}

	tarGzBytes := expRec.Body.Bytes()
	gr, err := gzip.NewReader(bytes.NewReader(tarGzBytes))
	if err != nil {
		t.Fatalf("exported payload is not valid single-layer gzip: %v", err)
	}
	tr := tar.NewReader(gr)
	foundUI := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if hdr.Name == "ui_state.json" {
			foundUI = true
		}
		if strings.HasSuffix(hdr.Name, ".db") {
			t.Fatalf("exported archive must not include .db file: %s", hdr.Name)
		}
	}
	_ = gr.Close()
	if !foundUI {
		t.Fatalf("expected ui_state.json in exported archive")
	}

	// 2. Import via multipart/form-data into a clean server
	srv2 := newTestServerWithDir(t)
	mux2 := http.NewServeMux()
	srv2.RegisterRoutes(mux2)

	var mpBody bytes.Buffer
	mw := multipart.NewWriter(&mpBody)
	fw, err := mw.CreateFormFile("file", "cubex-config-backup.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write(tarGzBytes)
	_ = mw.Close()

	impReq := httptest.NewRequest(http.MethodPost, "/api/v1/system/backup/import", &mpBody)
	impReq.Header.Set("Content-Type", mw.FormDataContentType())
	impRec := httptest.NewRecorder()
	mux2.ServeHTTP(impRec, impReq)
	if impRec.Code != http.StatusOK {
		t.Fatalf("multipart import status = %d, body=%s", impRec.Code, impRec.Body.String())
	}
	var impResp struct {
		OK            bool     `json:"ok"`
		RestoredItems []string `json:"restored_items"`
	}
	if err := json.Unmarshal(impRec.Body.Bytes(), &impResp); err != nil {
		t.Fatal(err)
	}
	if !impResp.OK || len(impResp.RestoredItems) == 0 {
		t.Fatalf("unexpected import response: %+v", impResp)
	}

	// 3. Import via raw gzip body
	rawReq := httptest.NewRequest(http.MethodPost, "/api/v1/system/backup/import", bytes.NewReader(tarGzBytes))
	rawReq.Header.Set("Content-Type", "application/gzip")
	rawRec := httptest.NewRecorder()
	mux2.ServeHTTP(rawRec, rawReq)
	if rawRec.Code != http.StatusOK {
		t.Fatalf("raw gzip import status = %d, body=%s", rawRec.Code, rawRec.Body.String())
	}
}

func TestAPI4_ProxyActiveRates(t *testing.T) {
	srv := newTestServerWithDir(t)
	ir := srv.engMgr.GetIR()
	ir.Proxies = append(ir.Proxies,
		schema.ProxyNode{Name: "HK-Edge-01", Type: "vless", Server: "hk.example.com", Port: 443},
		schema.ProxyNode{Name: "JP-Edge-02", Type: "trojan", Server: "jp.example.com", Port: 443},
	)
	ir.Groups = append(ir.Groups, schema.StrategyGroup{Name: "🌐 国外流量", Type: "select", Proxies: []string{"HK-Edge-01", "JP-Edge-02"}})

	// 1. When kernel Clash API is unreachable -> 200 OK with empty map {} (never 500)
	ir.Network.ClashAPIPort = 1 // unreachable local port
	srv.engMgr.UpdateIR(ir)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	recEmpty := httptest.NewRecorder()
	mux.ServeHTTP(recEmpty, httptest.NewRequest(http.MethodGet, "/api/v1/proxy/active-rates", nil))
	if recEmpty.Code != http.StatusOK {
		t.Fatalf("expected 200 when kernel not ready, got %d", recEmpty.Code)
	}
	var emptyPayload struct {
		ActiveNodeRates map[string]ActiveNodeRate `json:"active_node_rates"`
	}
	if err := json.Unmarshal(recEmpty.Body.Bytes(), &emptyPayload); err != nil {
		t.Fatal(err)
	}
	if emptyPayload.ActiveNodeRates == nil || len(emptyPayload.ActiveNodeRates) != 0 {
		t.Fatalf("expected empty non-nil active_node_rates map, got %+v", emptyPayload.ActiveNodeRates)
	}

	// 2. With live mock Clash API /connections -> aggregates real-time rates per outbound node
	var step atomic.Int32
	mockClash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/connections" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		s := step.Load()
		up1, down1 := int64(1000), int64(4000)
		up2, down2 := int64(500), int64(1500)
		if s >= 1 {
			up1 += 2048
			down1 += 8192
			up2 += 1024
			down2 += 4096
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"connections": []map[string]any{
				{"id": "c1", "upload": up1, "download": down1, "chains": []string{"🌐 国外流量", "HK-Edge-01"}},
				{"id": "c2", "upload": up2, "download": down2, "chains": []string{"JP-Edge-02"}},
				{"id": "c3", "upload": 9999, "download": 9999, "chains": []string{"DIRECT"}},
			},
		})
	}))
	defer mockClash.Close()
	u, _ := url.Parse(mockClash.URL)
	clashPort, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	ir.Network.ClashAPIPort = clashPort
	srv.engMgr.UpdateIR(ir)

	// First sample initializes baseline
	rec1 := httptest.NewRecorder()
	mux.ServeHTTP(rec1, httptest.NewRequest(http.MethodGet, "/api/v1/proxy/active-rates", nil))
	if rec1.Code != http.StatusOK {
		t.Fatalf("first active-rates call failed: %d", rec1.Code)
	}

	step.Store(1)
	time.Sleep(80 * time.Millisecond)

	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/api/v1/proxy/active-rates", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("second active-rates call failed: %d", rec2.Code)
	}
	var ratesResp struct {
		ActiveNodeRates map[string]ActiveNodeRate `json:"active_node_rates"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &ratesResp); err != nil {
		t.Fatal(err)
	}
	if hk, ok := ratesResp.ActiveNodeRates["HK-Edge-01"]; !ok || hk.UpBps <= 0 || hk.DownBps <= 0 {
		t.Fatalf("expected positive rates for HK-Edge-01, got %+v (all=%+v)", hk, ratesResp.ActiveNodeRates)
	}
	if jp, ok := ratesResp.ActiveNodeRates["JP-Edge-02"]; !ok || jp.UpBps <= 0 || jp.DownBps <= 0 {
		t.Fatalf("expected positive rates for JP-Edge-02, got %+v", jp)
	}
	if _, hasDirect := ratesResp.ActiveNodeRates["DIRECT"]; hasDirect {
		t.Fatalf("DIRECT connections must be excluded from active_node_rates")
	}
}

func TestStartupTiming_ReadinessGateAndAdaptiveMemoryLimit(t *testing.T) {
	// 1. Respect preconfigured GOMEMLIMIT env var
	t.Setenv("GOMEMLIMIT", "64MiB")
	limit := configureAdaptiveMemoryLimit()
	if limit <= 0 {
		t.Fatalf("expected positive memory limit, got %d", limit)
	}

	// 2. Engine readiness starts false until core boots
	srv := newTestServerWithDir(t)
	if srv.engMgr.IsCoreReady() {
		t.Fatalf("expected IsCoreReady() == false on fresh engine manager")
	}
	srv.engMgr.SetCoreReady(true)
	if !srv.engMgr.IsCoreReady() {
		t.Fatalf("expected IsCoreReady() == true after SetCoreReady(true)")
	}
}
