package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/host"
	"github.com/zyeyefyass/CubeX/internal/ros"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestShouldRunROSPatrol_SilentWhenUnconfiguredOrNonROS(t *testing.T) {
	tmpDir := t.TempDir()
	rosMgr := ros.NewManager(tmpDir)
	defer rosMgr.Close()
	hostMgr := host.NewManager(tmpDir)
	srv := &Server{
		workDir:   tmpDir,
		rosMgr:    rosMgr,
		hostMgr:   hostMgr,
		logBuffer: NewRingLogBuffer(100),
	}

	// 1. Unconfigured ROS credentials (standalone /小米/华硕/独立模式): must skip silently
	if srv.shouldRunROSPatrol() {
		t.Fatalf("expected shouldRunROSPatrol() == false when ROS host/user is empty")
	}

	// 2. Calling ROS guard helpers directly when unconfigured must be silent no-ops
	if restored := rosMgr.EnsureFakeIPRoute("10.0.0.7"); restored {
		t.Fatalf("expected EnsureFakeIPRoute to return false when ROS is unconfigured")
	}
	if err := rosMgr.RestoreRulesAfterBoot(); err != nil {
		t.Fatalf("expected RestoreRulesAfterBoot to return nil when ROS is unconfigured, got %v", err)
	}
	if purged := rosMgr.PurgeStaleFakeIPConnections(); purged != 0 {
		t.Fatalf("expected PurgeStaleFakeIPConnections to return 0 when ROS is unconfigured, got %d", purged)
	}

	// 3. Even if ROS config has host/user, when companion host platform is iKuai/OpenWrt, ROS patrol must skip
	if err := rosMgr.SaveConfig(map[string]string{
		"host":     "192.168.1.1",
		"port":     "80",
		"user":     "admin",
		"password": "pw",
	}); err != nil {
		t.Fatal(err)
	}
	if err := hostMgr.SaveConfig(host.HostConfig{
		Enabled:      true,
		Host:         "192.168.1.1",
		Port:         80,
		User:         "admin",
		PlatformType: "ikuai",
	}); err != nil {
		t.Fatal(err)
	}
	if srv.shouldRunROSPatrol() {
		t.Fatalf("expected shouldRunROSPatrol() == false when active platform_type is ikuai")
	}

	// 4. Switch to routeros -> patrol enabled
	if err := hostMgr.SaveConfig(host.HostConfig{
		Enabled:      true,
		Host:         "192.168.1.1",
		Port:         80,
		User:         "admin",
		PlatformType: "routeros",
	}); err != nil {
		t.Fatal(err)
	}
	if !srv.shouldRunROSPatrol() {
		t.Fatalf("expected shouldRunROSPatrol() == true when ROS configured and platform_type is routeros")
	}

	for _, e := range srv.logBuffer.GetLogs(100) {
		if e.Module == "ROS" {
			t.Fatalf("expected zero ROS warning logs in standalone/non-ROS checks, got %+v", e)
		}
	}
}

func TestHandleEnginePauseAndResumePreservesPrePauseMode(t *testing.T) {
	tmpDir := t.TempDir()
	ir := schema.NewDefaultIR()
	ir.Meta.ProxyMode = schema.ModeGlobal
	engMgr := engine.NewManager(tmpDir, ir)
	defer engMgr.Stop()

	srv := &Server{
		workDir:   tmpDir,
		engMgr:    engMgr,
		ir:        engMgr.GetIR(),
		logBuffer: NewRingLogBuffer(50),
	}

	// Pause -> switches to direct passthrough & preserves ModeGlobal
	reqPause := httptest.NewRequest(http.MethodPost, "/api/v1/engine/pause", bytes.NewReader([]byte(`{"paused":true}`)))
	recPause := httptest.NewRecorder()
	srv.handleEnginePause(recPause, reqPause)
	if recPause.Code != http.StatusOK {
		t.Fatalf("expected 200 on pause, got %d: %s", recPause.Code, recPause.Body.String())
	}

	pausedIR := engMgr.GetIR()
	if !pausedIR.Meta.ProxyPaused || pausedIR.Meta.ProxyMode != schema.ModeDirect || pausedIR.Meta.PrePauseMode != schema.ModeGlobal {
		t.Fatalf("unexpected paused IR meta: %+v", pausedIR.Meta)
	}

	// Reload from disk (simulate restart persistence)
	engMgr2 := engine.NewManager(tmpDir, schema.NewDefaultIR())
	defer engMgr2.Stop()
	reloadedIR := engMgr2.GetIR()
	if !reloadedIR.Meta.ProxyPaused || reloadedIR.Meta.ProxyMode != schema.ModeDirect {
		t.Fatalf("expected paused state to persist across daemon restart, got %+v", reloadedIR.Meta)
	}

	// Resume -> restores ModeGlobal
	srv2 := &Server{
		workDir:   tmpDir,
		engMgr:    engMgr2,
		ir:        reloadedIR,
		logBuffer: NewRingLogBuffer(50),
	}
	reqResume := httptest.NewRequest(http.MethodPost, "/api/v1/engine/pause", bytes.NewReader([]byte(`{"paused":false}`)))
	recResume := httptest.NewRecorder()
	srv2.handleEnginePause(recResume, reqResume)
	if recResume.Code != http.StatusOK {
		t.Fatalf("expected 200 on resume, got %d", recResume.Code)
	}
	resumedIR := engMgr2.GetIR()
	if resumedIR.Meta.ProxyPaused || resumedIR.Meta.ProxyMode != schema.ModeGlobal {
		t.Fatalf("expected ModeGlobal restored after resume, got %+v", resumedIR.Meta)
	}
}

func TestExecuteNodeRealDelayProbe_FallbackAndHysteria2Timeout(t *testing.T) {
	var seenTimeout atomic.Value
	var callCount atomic.Int32

	mockClash := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		seenTimeout.Store(r.URL.Query().Get("timeout"))
		probeURL := r.URL.Query().Get("url")
		// First probe fails/times out, second probe succeeds
		if n == 1 || strings.Contains(probeURL, "gstatic") {
			w.WriteHeader(http.StatusGatewayTimeout)
			_, _ = w.Write([]byte(`{"message":"An error occurred in the delay test: context deadline exceeded"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"delay":142}`))
	}))
	defer mockClash.Close()

	u, err := url.Parse(mockClash.URL)
	if err != nil {
		t.Fatal(err)
	}
	var port int
	for _, c := range u.Port() {
		port = port*10 + int(c-'0')
	}

	srv := &Server{}
	node := schema.ProxyNode{
		Name:   "[自建] hy2",
		Type:   "hysteria2",
		Server: "example.com",
		Port:   443,
	}

	res := srv.executeNodeRealDelayProbe(node, port, "")
	if res.Delay != 142 || res.Status != "ok" {
		t.Fatalf("expected fallback probe to succeed with delay 142, got %+v", res)
	}
	if gotTimeout, _ := seenTimeout.Load().(string); gotTimeout != "4500" {
		t.Fatalf("expected hysteria2 delay timeout 4500, got %q", gotTimeout)
	}
}
