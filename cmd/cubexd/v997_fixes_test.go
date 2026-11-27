package main

import (
	"encoding/json"
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

	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestExecutePanelRestart_ContainerDoesNotSelfSpawn(t *testing.T) {
	t.Setenv("CUBEX_SUPERVISED", "1")
	if !isSupervisedContainerEnv() {
		t.Fatalf("expected isSupervisedContainerEnv() == true when CUBEX_SUPERVISED=1")
	}

	tmpDir := t.TempDir()
	engMgr := engine.NewManager(tmpDir, schema.NewDefaultIR())
	srv := &Server{
		workDir:   tmpDir,
		engMgr:    engMgr,
		ir:        engMgr.GetIR(),
		logBuffer: NewRingLogBuffer(20),
	}

	var serviceCalled, spawnCalled atomic.Bool
	var exitCode atomic.Int32
	exitCode.Store(-1)

	executePanelRestart(
		srv,
		isSupervisedContainerEnv(),
		func() bool {
			serviceCalled.Store(true)
			return true
		},
		func(exe string, args []string) error {
			spawnCalled.Store(true)
			return nil
		},
		func(code int) {
			exitCode.Store(int32(code))
		},
	)

	if serviceCalled.Load() {
		t.Fatalf("expected serviceRestartFn NOT called in supervised container env")
	}
	if spawnCalled.Load() {
		t.Fatalf("CRITICAL: expected spawnDetachedFn NOT called in supervised container env (split-brain guard)")
	}
	if exitCode.Load() != 0 {
		t.Fatalf("expected exitCode 0, got %d", exitCode.Load())
	}

	// Verify /api/v1/system/status exposes in_docker=true
	rec := httptest.NewRecorder()
	srv.handleSystemStatus(rec, httptest.NewRequest(http.MethodGet, "/api/v1/system/status", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from handleSystemStatus, got %d", rec.Code)
	}
	var st map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if inDocker, ok := st["in_docker"].(bool); !ok || !inDocker {
		t.Fatalf("expected in_docker=true in /api/v1/system/status, got %v", st["in_docker"])
	}

	// Also verify /.dockerenv marker detection works even without CUBEX_SUPERVISED
	t.Setenv("CUBEX_SUPERVISED", "")
	marker := filepath.Join(tmpDir, ".dockerenv")
	if err := os.WriteFile(marker, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}
	origMarker := dockerEnvMarkerPath
	dockerEnvMarkerPath = marker
	defer func() { dockerEnvMarkerPath = origMarker }()
	if !isSupervisedContainerEnv() {
		t.Fatalf("expected isSupervisedContainerEnv() == true when /.dockerenv exists")
	}
}

func TestTestNodeRealDelay_NoFailedCacheAndManualBypass(t *testing.T) {
	var calls atomic.Int32
	var shouldFail atomic.Bool
	shouldFail.Store(true)

	mockClash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if shouldFail.Load() {
			w.WriteHeader(http.StatusGatewayTimeout)
			_, _ = w.Write([]byte(`{"message":"timeout"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"delay":88}`))
	}))
	defer mockClash.Close()

	u, _ := url.Parse(mockClash.URL)
	port, _ := strconv.Atoi(u.Port())

	srv := &Server{
		delayCache:   make(map[string]delayCacheEntry),
		delaySFGroup: make(map[string]*delayFlightCall),
	}
	node := schema.ProxyNode{Name: "[测试] 节点 01", Type: "vless", Server: "example.com", Port: 443}

	// 1. First probe fails -> Delay == 0 must NOT be cached
	res1 := srv.testNodeRealDelay(node, port, "")
	if res1.Delay != 0 {
		t.Fatalf("expected Delay=0 on timeout, got %+v", res1)
	}
	if _, exists := srv.delayCache[node.Name]; exists {
		t.Fatalf("failed probe result (Delay=0) must NOT be stored in delayCache")
	}

	// 2. Recover immediately -> next testNodeRealDelay must NOT hit 20s failed cache and must return 88ms
	shouldFail.Store(false)
	callsBefore := calls.Load()
	res2 := srv.testNodeRealDelay(node, port, "")
	if res2.Delay != 88 || calls.Load() == callsBefore {
		t.Fatalf("expected live retry returning 88ms after previous failure, got %+v (calls before=%d after=%d)", res2, callsBefore, calls.Load())
	}

	// 3. Manual fresh probe must bypass positive 20s cache
	callsBeforeFresh := calls.Load()
	res3 := srv.testNodeRealDelayFresh(node, port, "")
	if res3.Delay != 88 || calls.Load() == callsBeforeFresh {
		t.Fatalf("expected testNodeRealDelayFresh to bypass positive cache and hit mockClash, calls before=%d after=%d", callsBeforeFresh, calls.Load())
	}
}

func TestEncodeClashProxyPathAndSlashNodeFallback(t *testing.T) {
	encoded := encodeClashProxyPath("🇭🇰 [专线] 香港/日本 01")
	if strings.Contains(encoded, "[") || strings.Contains(encoded, "]") || strings.Contains(encoded, " ") || strings.Contains(encoded, "/") {
		t.Fatalf("expected brackets, spaces and slashes encoded, got %q", encoded)
	}

	// Spin up a local TCP listener to act as the node server:port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			_ = c.Close()
		}
	}()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	nodePort, _ := strconv.Atoi(portStr)

	// Mock Clash API that returns 404 on /proxies/.../delay for '/' node name, but lists it in /proxies
	nodeName := "香港/日本 专线-01"
	mockClash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/proxies" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"proxies": map[string]any{
					nodeName: map[string]any{"name": nodeName, "type": "VLESS"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	defer mockClash.Close()

	u, _ := url.Parse(mockClash.URL)
	clashPort, _ := strconv.Atoi(u.Port())

	srv := &Server{}
	res := srv.executeNodeRealDelayProbe(schema.ProxyNode{
		Name:   nodeName,
		Type:   "vless",
		Server: "127.0.0.1",
		Port:   nodePort,
	}, clashPort, "")
	if res.Delay <= 0 || res.Status != "ok" {
		t.Fatalf("expected slash-node fallback probe to succeed with positive RTT, got %+v", res)
	}
}

func TestHandleProxyBurstSpeedtest_RealPayloadAndDeadNodeZero(t *testing.T) {
	var proxyAlive atomic.Bool
	proxyAlive.Store(true)

	// Mock payload server returning 128KB real bytes
	payloadSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 128*1024))
	}))
	defer payloadSrv.Close()

	// Mock Clash API
	mockClash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !proxyAlive.Load() {
			w.WriteHeader(http.StatusGatewayTimeout)
			_, _ = w.Write([]byte(`{"message":"timeout"}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"delay": 35})
	}))
	defer mockClash.Close()

	// Mock mixedPort HTTP proxy forwarding to payloadSrv (or failing when dead)
	mixedLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer mixedLn.Close()
	mixedSrv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !proxyAlive.Load() {
				http.Error(w, "upstream dead", http.StatusBadGateway)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(make([]byte, 256*1024))
		}),
	}
	go func() { _ = mixedSrv.Serve(mixedLn) }()
	defer mixedSrv.Close()

	clashU, _ := url.Parse(mockClash.URL)
	clashPort, _ := strconv.Atoi(clashU.Port())
	_, mixedPortStr, _ := net.SplitHostPort(mixedLn.Addr().String())
	mixedPort, _ := strconv.Atoi(mixedPortStr)

	origURLs := burstSpeedtestPayloadURLs
	burstSpeedtestPayloadURLs = []string{payloadSrv.URL}
	defer func() { burstSpeedtestPayloadURLs = origURLs }()

	tmpDir := t.TempDir()
	ir := schema.NewDefaultIR()
	ir.Network.ClashAPIPort = clashPort
	ir.Network.MixedPort = mixedPort
	ir.Proxies = []schema.ProxyNode{{Name: "测试节点-01", Type: "vless", Server: "example.com", Port: 443}}
	engMgr := engine.NewManager(tmpDir, ir)
	defer engMgr.Stop()

	srv := &Server{
		workDir:      tmpDir,
		engMgr:       engMgr,
		ir:           ir,
		logBuffer:    NewRingLogBuffer(50),
		delayCache:   make(map[string]delayCacheEntry),
		delaySFGroup: make(map[string]*delayFlightCall),
	}

	// 1. Alive node with real payload -> non-zero bytes and real positive speed_mbps
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/config/proxies/burst-speedtest", strings.NewReader(`{"node":"测试节点-01"}`))
	rec1 := httptest.NewRecorder()
	srv.handleProxyBurstSpeedtest(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec1.Code)
	}
	var out1 map[string]any
	_ = json.Unmarshal(rec1.Body.Bytes(), &out1)
	if out1["bytes"].(float64) != 256*1024 || out1["speed_mbps"].(float64) <= 0 || out1["delay_ms"].(float64) != 35 {
		t.Fatalf("unexpected alive burst speedtest output: %+v", out1)
	}

	// 2. Dead node -> must honestly report 0 bytes, 0 delay, 0 speed_mbps (NO fake 25Mbps)
	proxyAlive.Store(false)
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/config/proxies/burst-speedtest", strings.NewReader(`{"node":"测试节点-01"}`))
	rec2 := httptest.NewRecorder()
	srv.handleProxyBurstSpeedtest(rec2, req2)
	var out2 map[string]any
	_ = json.Unmarshal(rec2.Body.Bytes(), &out2)
	if out2["bytes"].(float64) != 0 || out2["speed_mbps"].(float64) != 0 || out2["delay_ms"].(float64) != 0 {
		t.Fatalf("expected dead node burst speedtest to report 0/0/0, got %+v", out2)
	}
}
