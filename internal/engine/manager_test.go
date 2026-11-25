package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestStreamLogPipeClassification(t *testing.T) {
	mgr := &Manager{}

	type logEntry struct {
		Level   string
		Module  string
		Message string
	}
	var captured []logEntry
	mgr.SetLogCallback(func(level, module, msg string) {
		captured = append(captured, logEntry{
			Level:   level,
			Module:  module,
			Message: msg,
		})
	})

	tests := []struct {
		engine     schema.EngineType
		input      string
		wantLevel  string
		wantModule string
		wantMsgSub string
		shouldDrop bool
	}{
		{
			engine:     schema.EngineSingbox,
			input:      "ERROR [2876942641 3.87s] connection: connection download closed: close tcp 172.19.0.1:59936->198.18.0.73:443: endpoint not connected\n",
			shouldDrop: true,
		},
		{
			engine:     schema.EngineSingbox,
			input:      "router: missing fakeip record, try enable experimental.cache_file\n",
			shouldDrop: true,
		},
		{
			engine:     schema.EngineSingbox,
			input:      "FATAL sing-box crashed: cannot parse configuration json\n",
			wantLevel:  "ERROR",
			wantModule: "SINGBOX",
			wantMsgSub: "sing-box crashed",
		},
		{
			engine:     schema.EngineMihomo,
			input:      `time="2026-10-22T21:47:09.917846991+08:00" level=warning msg="deprecated rule set format"` + "\n",
			wantLevel:  "WARN",
			wantModule: "MIHOMO",
			wantMsgSub: "deprecated rule set format",
		},
		{
			engine:     schema.EngineMihomo,
			input:      `time="2026-10-24T10:09:26Z" level=warning msg="Mihomo shutting down"` + "\n",
			wantLevel:  "INFO",
			wantModule: "MIHOMO",
			wantMsgSub: "Mihomo shutting down",
		},
	}

	for _, tc := range tests {
		captured = nil
		mgr.streamLogPipe(tc.engine, "stdout", strings.NewReader(tc.input))
		if tc.shouldDrop {
			if len(captured) != 0 {
				t.Fatalf("expected log to be dropped for input %q, but got %d logs", tc.input, len(captured))
			}
			continue
		}
		if len(captured) != 1 {
			t.Fatalf("expected 1 log captured for input %q, got %d", tc.input, len(captured))
		}
		got := captured[0]
		if got.Level != tc.wantLevel {
			t.Errorf("input %q: got Level %s, want %s", tc.input, got.Level, tc.wantLevel)
		}
		if got.Module != tc.wantModule {
			t.Errorf("input %q: got Module %s, want %s", tc.input, got.Module, tc.wantModule)
		}
		if !strings.Contains(got.Message, tc.wantMsgSub) {
			t.Errorf("input %q: got Message %q, expected to contain %q", tc.input, got.Message, tc.wantMsgSub)
		}
	}
}

func TestWaitForPortsReleaseFast(t *testing.T) {
	mgr := &Manager{}
	// Empty or zero ports should return true immediately
	ports := []int{0, -1}
	if !mgr.waitForPortsRelease(ports, 50*time.Millisecond) {
		t.Fatalf("expected true for empty ports")
	}
}

func TestSwitchingGracePeriod(t *testing.T) {
	mgr := &Manager{}
	if mgr.IsSwitchingInGracePeriod() {
		t.Error("expected false initially")
	}

	// Set deadline 200ms in the future
	mgr.switchingGraceDeadline.Store(time.Now().Add(200 * time.Millisecond).UnixNano())
	if !mgr.IsSwitchingInGracePeriod() {
		t.Error("expected true within grace period")
	}

	// Wait for grace period to expire
	time.Sleep(250 * time.Millisecond)
	if mgr.IsSwitchingInGracePeriod() {
		t.Error("expected false after grace period expiration")
	}
}

func TestCheckTunDevice_MissingAndPermissionErrors(t *testing.T) {
	ok, msg := probeTunDevicePath("/nonexistent/dev/net/tun-cubex-test")
	if ok {
		t.Fatalf("expected missing tun device check to fail")
	}
	if !strings.Contains(msg, "--device /dev/net/tun") || !strings.Contains(msg, "tun.ko") {
		t.Fatalf("expected actionable guidance in tun error, got: %s", msg)
	}

	permMsg := classifyTunOpenError("/dev/net/tun", errString("operation not permitted"))
	if !strings.Contains(permMsg, "NET_ADMIN") || !strings.Contains(permMsg, "--device /dev/net/tun") {
		t.Fatalf("expected NET_ADMIN guidance for permission error, got: %s", permMsg)
	}

	nodevMsg := classifyTunOpenError("/dev/net/tun", errString("no such device"))
	if !strings.Contains(nodevMsg, "insmod /lib/modules/tun.ko") {
		t.Fatalf("expected insmod guidance for ENODEV error, got: %s", nodevMsg)
	}
}

type errString string

func (e errString) Error() string { return string(e) }

func TestSwitchEngine_BlockedWhenTunUnavailable(t *testing.T) {
	origProbe := tunProbeFunc
	defer func() { tunProbeFunc = origProbe }()

	tunProbeFunc = func() (bool, string) {
		return false, "缺少 TUN 虚拟网卡设备 (/dev/net/tun 不存在)"
	}

	tmpDir := t.TempDir()
	mgr := NewManager(tmpDir, schema.NewDefaultIR())
	res, err := mgr.SwitchEngine(schema.EngineSingbox)
	if err == nil || res == nil || res.Success {
		t.Fatalf("expected SwitchEngine to block when /dev/net/tun is unavailable, got res=%+v err=%v", res, err)
	}
	if !strings.Contains(res.ErrorMessage, "/dev/net/tun") {
		t.Fatalf("expected ErrorMessage to contain /dev/net/tun, got %q", res.ErrorMessage)
	}
	st := mgr.GetEngineFaultStatus()
	if st.TunReady {
		t.Fatalf("expected TunReady=false, got true")
	}
	if !strings.Contains(st.LastCoreError, "/dev/net/tun") {
		t.Fatalf("expected LastCoreError to preserve tun failure, got %q", st.LastCoreError)
	}
}

func TestDualEngineFailoverCircuitBreaker_BothUnavailable(t *testing.T) {
	mgr := &Manager{}
	var calls []schema.EngineType
	mgr.switchHook = func(target schema.EngineType) (*SwitchResult, error) {
		calls = append(calls, target)
		if target == schema.EngineSingbox {
			return &SwitchResult{Success: false}, errString("sing-box exit status 1: no /dev/net/tun")
		}
		return &SwitchResult{Success: false}, errString("mihomo dry-run failed: This program can only be run on AMD64 processors with v3 microarchitecture support.")
	}

	// Simulate singbox active failure count == 1 -> tries quick restart singbox, then standby mihomo -> both fail -> MUST TRIP IMMEDIATELY
	mgr.runSupervisorFailover(schema.EngineSingbox, schema.EngineMihomo, 1)
	if len(calls) != 2 {
		t.Fatalf("expected exactly 2 switch attempts (singbox then mihomo), got %d (%v)", len(calls), calls)
	}
	if !mgr.IsCircuitBroken() {
		t.Fatalf("expected circuit breaker to trip when both active and standby cores fail")
	}

	st := mgr.GetEngineFaultStatus()
	if !strings.Contains(st.CircuitBrokenReason, "v3 microarchitecture") || !strings.Contains(st.CircuitBrokenReason, "/dev/net/tun") {
		t.Fatalf("expected CircuitBrokenReason to expose both core failures clearly, got: %s", st.CircuitBrokenReason)
	}

	// Further supervisor invocations while tripped MUST NOT invoke switchHook (no infinite loop!)
	mgr.runSupervisorFailover(schema.EngineSingbox, schema.EngineMihomo, 2)
	mgr.runSupervisorFailover(schema.EngineSingbox, schema.EngineMihomo, 3)
	if len(calls) != 2 {
		t.Fatalf("expected zero additional switch attempts once circuit broken, got total %d", len(calls))
	}

	// Resetting circuit breaker clears broken state
	mgr.ResetCircuitBreaker()
	if mgr.IsCircuitBroken() {
		t.Fatalf("expected ResetCircuitBreaker to clear circuitBroken")
	}
}

func TestDualEngineFailoverCircuitBreaker_RapidThrashingLimit(t *testing.T) {
	mgr := &Manager{
		failoverWindow:      60 * time.Second,
		failoverMaxFailures: 4,
	}
	var calls int
	mgr.switchHook = func(target schema.EngineType) (*SwitchResult, error) {
		calls++
		// Simulate core starts briefly (returns nil) but keeps crashing immediately afterward
		return &SwitchResult{Success: true, ElapsedMs: 10}, nil
	}

	for i := 1; i <= 6; i++ {
		mgr.runSupervisorFailover(schema.EngineSingbox, schema.EngineMihomo, i)
	}
	if !mgr.IsCircuitBroken() {
		t.Fatalf("expected rapid thrashing (4 crashes within 60s) to trip circuit breaker")
	}
	if calls != 4 {
		t.Fatalf("expected circuit breaker to stop loop at exactly 4 failover attempts, got %d", calls)
	}
}

func TestStreamLogPipe_CapturesV3StderrError(t *testing.T) {
	mgr := &Manager{}
	v3Msg := "This program can only be run on AMD64 processors with v3 microarchitecture support."
	mgr.streamLogPipe(schema.EngineMihomo, "stderr", strings.NewReader(v3Msg+"\n"))

	st := mgr.GetEngineFaultStatus()
	if st.LastCoreError != v3Msg {
		t.Fatalf("expected LastCoreError=%q, got %q", v3Msg, st.LastCoreError)
	}
}

func TestNewManager_EmptyProxiesPreservedAndCorruptBackedUp(t *testing.T) {
	// 1. Empty proxies with custom rule/port must NOT be discarded by default IR
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}
	customJSON := `{"meta":{"active_engine":"sing-box","proxy_mode":"global"},"network":{"mixed_port":17890,"dns_port":53,"clash_api_port":9090},"proxies":[],"rules":[{"type":"DOMAIN-SUFFIX","payload":"example.com","target":"DIRECT"}]}`
	irPath := filepath.Join(dataDir, "config.ir.json")
	if err := os.WriteFile(irPath, []byte(customJSON), 0600); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(tmpDir, schema.NewDefaultIR())
	defer mgr.Stop()
	got := mgr.GetIR()
	if got.Network.MixedPort != 17890 || got.Meta.ProxyMode != schema.ModeGlobal {
		t.Fatalf("expected empty-proxy custom config preserved, got mixed_port=%d mode=%s", got.Network.MixedPort, got.Meta.ProxyMode)
	}

	// 2. Corrupt config.ir.json must be backed up as .corrupt.*
	corruptDir := t.TempDir()
	cDataDir := filepath.Join(corruptDir, "data")
	if err := os.MkdirAll(cDataDir, 0700); err != nil {
		t.Fatal(err)
	}
	cIrPath := filepath.Join(cDataDir, "config.ir.json")
	if err := os.WriteFile(cIrPath, []byte("{corrupt json"), 0600); err != nil {
		t.Fatal(err)
	}
	mgr2 := NewManager(corruptDir, schema.NewDefaultIR())
	defer mgr2.Stop()
	matches, _ := filepath.Glob(cIrPath + ".corrupt.*")
	if len(matches) == 0 {
		t.Fatalf("expected corrupt config.ir.json to be backed up with .corrupt.* suffix")
	}
}

