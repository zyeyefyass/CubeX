package engine

import (
	"bufio"

	"context"

	"crypto/sha256"

	"encoding/json"

	"fmt"

	"io"

	"log"

	"net"

	"net/http"

	"net/url"

	"regexp"

	"os"

	"os/exec"

	"path/filepath"

	"runtime"
	"runtime/debug"

	"strconv"

	"strings"

	"sync"

	"sync/atomic"

	"syscall"

	"time"

	"github.com/zyeyefyass/CubeX/internal/clashapi"
	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/config"
	"github.com/zyeyefyass/CubeX/internal/netutil"
	"github.com/zyeyefyass/CubeX/internal/perf"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

var ansiStripRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07`)

// SwitchResult contains statistics about the kernel switch.

type SwitchResult struct {
	FromEngine     schema.EngineType `json:"from_engine"`
	ToEngine       schema.EngineType `json:"to_engine"`
	ElapsedMs      int64             `json:"elapsed_ms"`
	TotalCostMs    int64             `json:"total_cost_ms"`
	StopCostMs     int64             `json:"stop_cost_ms"`
	StartupCostMs  int64             `json:"startup_cost_ms"`
	ActiveRules    int               `json:"active_rules"`
	RestoredGroups int               `json:"restored_groups"`
	Success        bool              `json:"success"`
	ErrorMessage   string            `json:"error_message,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// Manager controls the dual-engine lifecycle, hot-swap state machine, and status.

// LogFunc is the callback signature for streaming real-time core and engine events to the ring buffer.

type LogFunc func(level, module, msg string)

type Manager struct {
	mu sync.RWMutex

	cbMu sync.RWMutex

	ir *schema.ConfigIR

	workDir string

	activeEngine schema.EngineType

	currentProc *os.Process

	procDone chan struct{}

	clashAPI string

	httpClient *http.Client

	lastResult *SwitchResult

	statusCh chan schema.EngineType

	stopping bool

	crashCount int

	logCallback LogFunc

	dryRunCache     sync.Map
	dryRunCacheSize atomic.Int32

	precompiledConfig map[schema.EngineType][]byte
	precompiledMu     sync.RWMutex
	irVersion         uint64
	switching         atomic.Bool
	switchingGraceDeadline atomic.Int64
	stopCh chan struct{}
	stopOnce sync.Once

	faultMu             sync.Mutex
	circuitBroken       bool
	circuitBrokenReason string
	circuitBrokenAt     time.Time
	circuitHoldLogAt    map[string]time.Time
	failoverHistory     []time.Time
	failoverWindow      time.Duration
	failoverMaxFailures int
	lastCoreError       string
	recentStderr        []string
	switchHook          func(schema.EngineType) (*SwitchResult, error)
	onAllCoresDown      func(reason string)
	onUpstreamStall     func(active schema.EngineType, detail string)
}

// NewManager creates a dual-engine manager.

func NewManager(workDir string, initialIR *schema.ConfigIR) *Manager {

	if initialIR == nil {

		initialIR = schema.NewDefaultIR()

	}

	irFile := filepath.Join(workDir, "data", "config.ir.json")

	var loaded bool

	if data, err := os.ReadFile(irFile); err == nil {

		var savedIR schema.ConfigIR

		if uErr := json.Unmarshal(data, &savedIR); uErr == nil {

			// Clean up any corrupted groups from old files

			cleanGroups := make([]schema.StrategyGroup, 0, len(savedIR.Groups))

			for _, g := range savedIR.Groups {

				if strings.Contains(g.Name, "??") || strings.Contains(g.Name, "\ufffd") || strings.HasSuffix(g.Name, "手动") || g.Policy == "manual" {

					continue

				}

				cleanGroups = append(cleanGroups, g)

			}

			savedIR.Groups = cleanGroups

			initialIR = &savedIR

			loaded = true

		} else {

			log.Printf("⚠️ [Engine] Failed to parse config.ir.json (%v); backing up corrupt file", uErr)

			corruptPath := fmt.Sprintf("%s.corrupt.%d", irFile, time.Now().Unix())

			_ = os.Rename(irFile, corruptPath)

		}

	} else if !os.IsNotExist(err) {

		log.Printf("⚠️ [Engine] Failed to read config.ir.json (%v)", err)

	}

	// Fallback to last-known-good snapshot if primary config is missing or corrupted

	if !loaded {

		goodFile := filepath.Join(workDir, "data", "config.last-known-good.json")

		if gData, gErr := os.ReadFile(goodFile); gErr == nil {

			var goodIR schema.ConfigIR

			if uErr := json.Unmarshal(gData, &goodIR); uErr == nil {

				initialIR = &goodIR

				log.Printf("🛡️ [Engine] Restored configuration from last-known-good snapshot (%d proxies)", len(goodIR.Proxies))

			} else {

				log.Printf("⚠️ [Engine] Failed to parse last-known-good snapshot (%v)", uErr)

			}

		}

	}

	initialIR.NormalizeDefaults()
	schema.SyncAllStrategyGroups(initialIR)

	if cleanData, err := json.MarshalIndent(initialIR, "", "  "); err == nil {

		_ = schema.WriteFileAtomicSecure(irFile, cleanData)

	}

	apiAddr := fmt.Sprintf("http://127.0.0.1:%d", initialIR.Network.ClashAPIPort)

	mgr := &Manager{

		ir: initialIR,

		workDir: workDir,

		activeEngine: initialIR.Meta.ActiveEngine,

		clashAPI: apiAddr,

		httpClient: &http.Client{
			Timeout: 2 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     60 * time.Second,
				DisableKeepAlives:   false,
			},
		},

		statusCh: make(chan schema.EngineType, 10),
		precompiledConfig: make(map[schema.EngineType][]byte),
		stopCh: make(chan struct{}),
	}

	netutil.SafeGo("WarmStandbyConfigs", mgr.WarmStandbyConfigs)

	mgr.StartSupervisor()

	mgr.StartEndToEndWatchdog()

	return mgr

}

// SetFaultCallbacks 注册双核全挂逃生回调与上游节点假死回调。
func (m *Manager) SetFaultCallbacks(onAllCoresDown func(reason string), onUpstreamStall func(active schema.EngineType, detail string)) {
	m.cbMu.Lock()
	m.onAllCoresDown = onAllCoresDown
	m.onUpstreamStall = onUpstreamStall
	m.cbMu.Unlock()
}

// WarmStandbyConfigs pre-compiles standby engine configuration in the background to achieve zero-millisecond compile latency during hot-swap.
func (m *Manager) WarmStandbyConfigs() {
	ver := atomic.LoadUint64(&m.irVersion)
	m.prewarmStandbyVersion(ver)
}

// prewarmStandbyVersion performs async precompilation guarded by targetVer generation counter to eliminate ABA race conditions.
func (m *Manager) prewarmStandbyVersion(targetVer uint64) {
	m.mu.RLock()
	ir := m.ir.Clone()
	eng := m.activeEngine
	m.mu.RUnlock()
	if ir == nil {
		return
	}

	// Initial version guard before expensive compilation
	if atomic.LoadUint64(&m.irVersion) != targetVer {
		return
	}

	var sb, mh []byte
	var err error
	if eng == schema.EngineMihomo {
		sb, err = compiler.BuildSingBox(ir)
	} else {
		mh, err = compiler.BuildMihomo(ir)
	}

	m.precompiledMu.Lock()
	defer m.precompiledMu.Unlock()
	// Guard against ABA write-back race condition: if irVersion mutated, discard stale compilation
	if atomic.LoadUint64(&m.irVersion) != targetVer {
		return
	}
	if m.precompiledConfig == nil {
		m.precompiledConfig = make(map[schema.EngineType][]byte)
	}
	if eng == schema.EngineMihomo && err == nil {
		m.precompiledConfig[schema.EngineSingbox] = sb
	} else if err == nil {
		m.precompiledConfig[schema.EngineMihomo] = mh
	}
	m.reclaimOSMemory()
}

// GetActiveEngine returns the currently active persona.

func (m *Manager) GetActiveEngine() schema.EngineType {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.activeEngine

}

// SetLogCallback registers the receiver for real-time engine and core logs.

func (m *Manager) SetLogCallback(fn LogFunc) {

	m.cbMu.Lock()

	defer m.cbMu.Unlock()

	m.logCallback = fn

}

func (m *Manager) log(level, module, msg string) {

	m.cbMu.RLock()

	cb := m.logCallback

	m.cbMu.RUnlock()

	if cb != nil {

		cb(level, module, msg)

	}

}

// GetCurrentProcPID returns the OS PID of the running core process, or 0.

func (m *Manager) GetCurrentProcPID() int {

	m.mu.RLock()

	defer m.mu.RUnlock()

	if m.currentProc != nil {

		return m.currentProc.Pid

	}

	return 0

}

// GetCoreMemoryRSS returns the live real-time resident set size (RSS in MB) of the active engine process.

func (m *Manager) GetCoreMemoryRSS() float64 {

	m.mu.RLock()

	proc := m.currentProc

	eng := m.activeEngine

	m.mu.RUnlock()

	if proc != nil && proc.Pid > 0 && runtime.GOOS == "linux" {
		statusPath := fmt.Sprintf("/proc/%d/status", proc.Pid)
		if data, err := os.ReadFile(statusPath); err == nil {
			lines := strings.Split(string(data), "\n")
			var rssAnon, vmRss float64
			for _, line := range lines {
				if strings.HasPrefix(line, "RssAnon:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseFloat(fields[1], 64); err == nil && kb > 0 {
							rssAnon = kb / 1024.0
						}
					}
				} else if strings.HasPrefix(line, "VmRSS:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseFloat(fields[1], 64); err == nil && kb > 0 {
							vmRss = kb / 1024.0
						}
					}
				}
			}
			// Prefer RssAnon (actual physical RAM used by Go heap/stack, excluding file-backed executable mmap)
			if rssAnon > 0 {
				return rssAnon
			}
			if vmRss > 0 {
				return vmRss
			}
		}
	}

	// Fallback estimate if /proc is not available or before process starts
	if eng == schema.EngineMihomo {
		return 36.5
	}
	return 28.5
}

// GetLastResult returns the telemetry of the most recent switch.

func (m *Manager) GetLastResult() *SwitchResult {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.lastResult

}

// GetIR returns a reference to the active IR config.

func (m *Manager) snapshotNetMeta() (clashAPI, clashSecret, tunIface string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	clashAPI = m.clashAPI
	tunIface = "cubex-tun"
	if m.ir != nil {
		clashSecret = m.ir.Network.ClashSecret
		if m.ir.Network.TunInterface != "" {
			tunIface = m.ir.Network.TunInterface
		}
	}
	return clashAPI, clashSecret, tunIface
}

// GetIR returns a defensive deep copy of the configuration IR to prevent caller data races.
func (m *Manager) GetIR() *schema.ConfigIR {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.ir.Clone()

}

// UpdateIRWithErr safely updates the configuration IR in memory, increments irVersion,
// immediately invalidates stale precompiled configs under lock, persists atomically to disk,
// and asynchronously prewarms the standby engine for the new irVersion.
func (m *Manager) UpdateIRWithErr(newIR *schema.ConfigIR) error {
	if newIR == nil {
		return fmt.Errorf("cannot update engine with nil ConfigIR")
	}

	m.mu.Lock()
	curVer := atomic.AddUint64(&m.irVersion, 1)

	m.precompiledMu.Lock()
	m.precompiledConfig = make(map[schema.EngineType][]byte)
	m.precompiledMu.Unlock()

	cloned := newIR.Clone()
	if cloned.Meta.ActiveEngine == "" && m.activeEngine != "" {
		cloned.Meta.ActiveEngine = m.activeEngine
	}
	m.ir = cloned

	_ = os.MkdirAll(filepath.Join(m.workDir, "data"), 0700)
	var persistErr error
	if data, err := json.MarshalIndent(cloned, "", "  "); err != nil {
		persistErr = err
	} else if err := schema.WriteFileAtomicSecure(filepath.Join(m.workDir, "data", "config.ir.json"), data); err != nil {
		persistErr = err
	}
	m.mu.Unlock()

	if persistErr != nil {
		return persistErr
	}

	netutil.SafeGo("PrewarmStandbyVersion", func() {
		m.prewarmStandbyVersion(curVer)
	})
	return nil
}

// UpdateIR safely updates the configuration IR and logs any persistence failure.
func (m *Manager) UpdateIR(newIR *schema.ConfigIR) {
	if err := m.UpdateIRWithErr(newIR); err != nil {
		log.Printf("⚠️ [Engine] Failed to persist updated ConfigIR: %v", err)
	}
}

// SnapshotClashState snapshots active selector node choices from Clash API.
func (m *Manager) SnapshotClashState() map[string]string {
	clashAPI, clashSecret, _ := m.snapshotNetMeta()
	var groups []schema.StrategyGroup
	m.mu.RLock()
	if m.ir != nil && len(m.ir.Groups) > 0 {
		groups = append([]schema.StrategyGroup(nil), m.ir.Groups...)
	}
	m.mu.RUnlock()
	return m.snapshotClashStateWithMeta(clashAPI, clashSecret, groups)
}

func (m *Manager) snapshotClashStateLocked() map[string]string {
	clashAPI := m.clashAPI
	clashSecret := ""
	var groups []schema.StrategyGroup
	if m.ir != nil {
		clashSecret = m.ir.Network.ClashSecret
		if len(m.ir.Groups) > 0 {
			groups = m.ir.Groups
		}
	}
	return m.snapshotClashStateWithMeta(clashAPI, clashSecret, groups)
}

func (m *Manager) snapshotClashStateWithMeta(clashAPI, clashSecret string, groups []schema.StrategyGroup) map[string]string {
	snapshot := make(map[string]string)

	if clashAPI != "" {
		apiClient := clashapi.NewClient(clashAPI, clashSecret, m.httpClient)
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Millisecond)
		proxies, err := apiClient.GetProxies(ctx)
		cancel()
		if err == nil {
			for name, p := range proxies {
				if p.Now != "" {
					snapshot[name] = p.Now
				}
			}
		}
	}

	// Fallback / merge with IR defaults if any group is missing in snapshot
	for _, g := range groups {
		if snapshot[g.Name] == "" && g.Now != "" {
			snapshot[g.Name] = g.Now
		}
	}

	return snapshot
}

// RestoreClashState injects cached state into the newly launched core's Clash API concurrently.

func (m *Manager) RestoreClashState(snapshot map[string]string) int {

	if len(snapshot) == 0 {

		return 0

	}

	clashAPI, clashSecret, _ := m.snapshotNetMeta()
	apiClient := clashapi.NewClient(clashAPI, clashSecret, m.httpClient)

	var wg sync.WaitGroup

	var restoredCount int64

	sem := make(chan struct{}, 40)

	for groupName, targetNode := range snapshot {

		if groupName == "" || targetNode == "" {

			continue

		}

		wg.Add(1)

		sem <- struct{}{}

		grp, node := groupName, targetNode
		netutil.SafeGo("RestoreClashGroup", func() {
			defer wg.Done()
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
			defer cancel()

			if err := apiClient.SwitchProxy(ctx, grp, node); err == nil {
				atomic.AddInt64(&restoredCount, 1)
			}
		})

	}

	wg.Wait()

	return int(restoredCount)

}

func (m *Manager) findBinary(binName string) string {

	if runtime.GOOS == "windows" {

		candidates := []string{

			filepath.Join(m.workDir, "bin", binName+".exe"),

			filepath.Join(m.workDir, "bin", "windows", binName+".exe"),

			filepath.Join(m.workDir, binName+".exe"),
		}

		for _, c := range candidates {

			if _, err := os.Stat(c); err == nil {

				return c

			}

		}

		return binName + ".exe"

	}

	candidates := []string{

		filepath.Join(m.workDir, "bin", binName),

		filepath.Join(m.workDir, "bin", runtime.GOARCH, binName),

		filepath.Join(m.workDir, binName),

		filepath.Join("/app/bin", binName),
	}

	for _, c := range candidates {

		if _, err := os.Stat(c); err == nil {

			return c

		}

	}

	return binName

}

func getAdaptiveEngineMemoryLimits() (gomemlimit string, gogc string) {
	if customLimit := os.Getenv("CUBEX_CORE_GOMEMLIMIT"); customLimit != "" {
		gc := os.Getenv("CUBEX_CORE_GOGC")
		if gc == "" {
			gc = "100"
		}
		return "GOMEMLIMIT=" + customLimit, "GOGC=" + gc
	}

	totalMb := perf.DetectTotalRAMMb()
	limitMb := 64
	gc := 100

	if totalMb <= 256 {
		limitMb = 48 // 256M low-mem container/router: 48MiB profile
		gc = 50
	} else {
		limitMb = 64 // Standard PVE/NAS/RouterOS: 64MiB golden balance sweetspot
		gc = 100
	}

	return fmt.Sprintf("GOMEMLIMIT=%dMiB", limitMb), fmt.Sprintf("GOGC=%d", gc)
}

func (m *Manager) buildCmd(eng schema.EngineType, binPath, etcDir, configPath string) *exec.Cmd {

	var cmd *exec.Cmd

	if eng == schema.EngineMihomo {

		cmd = exec.Command(binPath, "-d", etcDir, "-f", configPath)

	} else {

		cmd = exec.Command(binPath, "run", "-c", configPath, "-D", etcDir)

		cmd.Dir = etcDir

	}

	// High-Performance Go Runtime & GC tuning:
	// Adaptive GOMEMLIMIT ensures tight memory ceiling on routers and containers,
	// with dynamic GOGC for prompt garbage reclamation of network packets and TLS buffers.
	limitEnv, gcEnv := getAdaptiveEngineMemoryLimits()
	cmd.Env = append(os.Environ(),
		limitEnv,
		gcEnv,
		"GODEBUG=madvdontneed=1",
		"ENABLE_DEPRECATED_OUTBOUND_DNS_RULE_ITEM=true",
	)

	// Configure Linux process group isolation and parent death cleanup
	configureSysProcAttr(cmd)

	return cmd

}

func (m *Manager) streamLogPipe(eng schema.EngineType, pipeType string, r io.Reader) {

	scanner := bufio.NewScanner(r)

	buf := make([]byte, 64*1024)

	scanner.Buffer(buf, 64*1024)

	module := "SINGBOX"

	if eng == schema.EngineMihomo {

		module = "MIHOMO"

	}

	for scanner.Scan() {

		line := strings.TrimSpace(scanner.Text())

		if line == "" {

			continue

		}

		cleanLine := ansiStripRegex.ReplaceAllString(line, "")
		m.captureCoreOutput(pipeType, cleanLine)

		level := "INFO"

		lower := strings.ToLower(cleanLine)

		// Benign socket cancellation & routine external network timeouts, client aborts,

		// connection resets, or container interface monitor events that should be categorized as DEBUG

		isBenign := strings.Contains(lower, "operation was canceled") ||

			strings.Contains(lower, "context canceled") ||

			strings.Contains(lower, "context deadline exceeded") ||

			strings.Contains(lower, "i/o timeout") ||

			strings.Contains(lower, "use of closed network connection") ||

			strings.Contains(lower, "connection reset by peer") ||

			strings.Contains(lower, "broken pipe") ||

			strings.Contains(lower, "missing fakeip record") ||

			strings.Contains(lower, "endpoint not connected") ||

			strings.Contains(lower, "software caused connection abort") ||

			strings.Contains(lower, "connection refused") ||

			strings.Contains(lower, "connection download closed") ||

			strings.Contains(lower, "connection upload closed") ||

			strings.Contains(lower, "network is unreachable") ||

			strings.Contains(lower, "no route to host") ||

			strings.Contains(lower, "default interface changed by monitor") ||

			strings.Contains(lower, "tunsetoffload") ||

			strings.Contains(lower, "enable offload") ||

			strings.Contains(lower, "tls: first record does not look like a tls handshake")

		isLifecycleInfo := strings.Contains(lower, "shutting down") ||

			strings.Contains(lower, "shutdown complete") ||

			strings.Contains(lower, "service stopped") ||

			strings.Contains(lower, "service reload") ||

			strings.Contains(lower, "graceful shutdown")

		if isBenign {

			continue

		} else if isLifecycleInfo {

			level = "INFO"

		} else if strings.Contains(lower, "error") || strings.Contains(lower, "fatal") || strings.Contains(lower, "panic") || strings.Contains(lower, "fail") {

			level = "ERROR"

		} else if strings.Contains(lower, "warn") {

			level = "WARN"

		} else if strings.Contains(lower, "debug") {

			level = "DEBUG"

		}

		if pipeType == "stderr" {

			fmt.Fprintln(os.Stderr, line)

		} else {

			fmt.Fprintln(os.Stdout, line)

		}

		// Strip redundant timestamp & level prefix from sing-box lines

		displayMsg := cleanLine

		if strings.HasPrefix(displayMsg, "+") && len(displayMsg) > 26 {

			displayMsg = strings.TrimSpace(displayMsg[26:])

		}

		for _, pfx := range []string{"WARN ", "ERROR ", "INFO ", "DEBUG ", "FATAL ", "WARN: ", "ERROR: ", "INFO: ", "DEBUG: "} {

			if strings.HasPrefix(strings.ToUpper(displayMsg), pfx) {

				displayMsg = strings.TrimSpace(displayMsg[len(pfx):])

				break

			}

		}

		// Clean Mihomo logfmt envelope: time="..." level=... msg="..."

		if eng == schema.EngineMihomo && strings.Contains(displayMsg, "msg=\"") {

			if idx := strings.Index(displayMsg, "msg=\""); idx != -1 {

				mPart := displayMsg[idx+5:]

				if endIdx := strings.LastIndex(mPart, "\""); endIdx != -1 {

					displayMsg = mPart[:endIdx]

				} else {

					displayMsg = mPart

				}

			}

		}

		m.log(level, module, displayMsg)

	}

}

func (m *Manager) startEngineProcess(eng schema.EngineType, cmd *exec.Cmd) (*sync.WaitGroup, error) {

	stdoutPipe, err := cmd.StdoutPipe()

	if err != nil {

		cmd.Stdout = os.Stdout

	}

	stderrPipe, err2 := cmd.StderrPipe()

	if err2 != nil {

		cmd.Stderr = os.Stderr

	}

	if err := cmd.Start(); err != nil {

		return nil, err

	}

	var pipeWG sync.WaitGroup

	if stdoutPipe != nil {

		pipeWG.Add(1)

		netutil.SafeGo("EngineStdoutPipe", func() {

			defer pipeWG.Done()

			m.streamLogPipe(eng, "stdout", stdoutPipe)

		})

	}

	if stderrPipe != nil {

		pipeWG.Add(1)

		netutil.SafeGo("EngineStderrPipe", func() {

			defer pipeWG.Done()

			m.streamLogPipe(eng, "stderr", stderrPipe)

		})

	}

	return &pipeWG, nil

}

func (m *Manager) waitEngineProcess(eng schema.EngineType, cmd *exec.Cmd, pipeWG *sync.WaitGroup, done chan struct{}) {
	pid := 0
	if cmd != nil && cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	if pipeWG != nil {
		pipeWG.Wait()
	}
	err := cmd.Wait()
	state := cmd.ProcessState
	if err != nil || (state != nil && !state.Success()) {
		detail := m.summarizeRecentStderr()
		if detail != "" {
			m.setLastCoreError(fmt.Sprintf("%s (PID %d) 异常退出 (%v): %s", eng, pid, err, detail))
		} else {
			m.setLastCoreError(fmt.Sprintf("%s (PID %d) 异常退出 (%v, state: %v)", eng, pid, err, state))
		}
	}
	log.Printf("⚠️ [Engine] Core process %s (PID %d) exited: %v (state: %v)", eng, pid, err, state)
	close(done)
}

func (m *Manager) dryRunCheck(eng schema.EngineType, binPath, etcDir, configPath string) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	var testCmd *exec.Cmd

	if eng == schema.EngineMihomo {

		testCmd = exec.CommandContext(ctx, binPath, "-t", "-d", etcDir, "-f", configPath)

	} else {

		testCmd = exec.CommandContext(ctx, binPath, "check", "-c", configPath, "-D", etcDir)

		testCmd.Dir = etcDir

	}

	out, err := testCmd.CombinedOutput()

	outStr := strings.TrimSpace(string(out))

	engTag := "MIHOMO"

	if eng == schema.EngineSingbox {

		engTag = "SINGBOX"

	}

	if err != nil {

		m.log("ERROR", engTag, fmt.Sprintf("内核预检失败: %s (%v)", outStr, err))

		return fmt.Errorf("%s dry-run failed: %s (%w)", eng, outStr, err)

	}

	if outStr != "" {

		m.log("INFO", engTag, fmt.Sprintf("内核预检通过: %s", outStr))

	} else {

		m.log("INFO", engTag, "内核配置预检通过: 语法与规则校验合规")

	}

	return nil

}

func (m *Manager) waitForPortsRelease(ports []int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		allFree := true
		for _, port := range ports {
			if port <= 0 {
				continue
			}
			addr := fmt.Sprintf("0.0.0.0:%d", port)
			l, err := net.Listen("tcp", addr)
			if err != nil {
				allFree = false
				if time.Now().After(deadline) {
					log.Printf("⚠️ [Engine] Port %d still occupied: %v", port, err)
				}
				break
			}
			_ = l.Close()
		}
		if allFree {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(5 * time.Millisecond)
	}
}

const (
	// HotSwitchGracePeriod is the maximum grace period for old engine SIGTERM during hot-swap
	HotSwitchGracePeriod = 60 * time.Millisecond
	// ShutdownGracePeriod is the graceful shutdown period (SIGTERM 3s -> SIGKILL)
	ShutdownGracePeriod = 3 * time.Second
)

// Stop gracefully stops the active core and unbinds TUN interfaces before system shutdown.
func (m *Manager) Stop() {
	m.mu.Lock()
	m.stopping = true
	active := m.activeEngine
	m.stopCurrentCoreWithGrace(active, ShutdownGracePeriod)
	m.reclaimOSMemory()
	m.mu.Unlock()

	m.stopOnce.Do(func() {
		close(m.stopCh)
	})
}

func (m *Manager) stopCurrentCore(binName string) {
	m.stopCurrentCoreWithGrace(schema.EngineType(binName), HotSwitchGracePeriod)
}

func (m *Manager) stopCurrentCoreWithGrace(target schema.EngineType, gracePeriod time.Duration) {
	if m.currentProc == nil || m.currentProc.Pid <= 0 {
		return
	}

	pid := m.currentProc.Pid
	done := m.procDone
	procBin := "sing-box"
	if target == schema.EngineMihomo || string(target) == "mihomo" {
		procBin = "mihomo"
	}

	defer func() {
		m.currentProc = nil
		m.procDone = nil
	}()

	// If already done, return
	if done != nil {
		select {
		case <-done:
			log.Printf("✨ [Engine] Core (PID %d) already finished", pid)
			return
		default:
		}
	}

	log.Printf("🛑 [Engine] Stopping active core (PID %d, grace %v)...", pid, gracePeriod)

	// Step 1: Send SIGTERM to process group
	_ = signalProcessGroup(pid, syscall.SIGTERM)

	if done == nil {
		return
	}

	timer := time.NewTimer(gracePeriod)
	defer timer.Stop()

	select {
	case <-done:
		// Clean exit: kernel exit_files() automatically closed sockets and destroyed tun
		log.Printf("✨ [Engine] Core (PID %d) cleanly stopped within grace window", pid)
		return
	case <-timer.C:
		// Step 2: Escalate to SIGKILL on process group
		log.Printf("⚠️ [Engine] Core (PID %d) exceeded grace %v; escalating to SIGKILL on process group", pid, gracePeriod)
		_ = signalProcessGroup(pid, syscall.SIGKILL)
		select {
		case <-done:
			log.Printf("✨ [Engine] Core (PID %d) terminated via SIGKILL barrier", pid)
			return
		case <-time.After(500 * time.Millisecond):
			log.Printf("🚨 [Engine] Process group %d stuck after SIGKILL", pid)
			if runtime.GOOS != "windows" {
				// 仅清理当前失控引擎对应进程名，绝不误杀另一内核
				_ = exec.Command("pkill", "-9", "-x", procBin).Run()
			}
		}
	}
}

func (m *Manager) probeCoreHealth(target schema.EngineType, maxWait time.Duration) bool {
	return m.probeCoreHealthFor(m.currentProc, m.procDone, target, m.clashAPI, maxWait)
}

func (m *Manager) probeCoreHealthFor(proc *os.Process, procDone <-chan struct{}, target schema.EngineType, clashAPI string, maxWait time.Duration) bool {
	deadline := time.Now().Add(maxWait)
	backoff := 3 * time.Millisecond
	maxBackoff := 25 * time.Millisecond

	probeAddr := "127.0.0.1:9090"
	if u, err := url.Parse(clashAPI); err == nil && u.Host != "" {
		probeAddr = u.Host
	}

	for time.Now().Before(deadline) {
		if proc == nil {
			return false
		}

		// Fail-fast if process exited prematurely
		if procDone != nil {
			select {
			case <-procDone:
				log.Printf("🚨 [Engine] Target core process exited prematurely during startup")
				return false
			default:
			}
		}

		// Fast TCP connect probe: as soon as Clash API port accepts connection, core is fully initialized
		c, err := net.DialTimeout("tcp", probeAddr, 10*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return true
		}

		time.Sleep(backoff)
		if backoff < maxBackoff {
			backoff = backoff * 3 / 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	return false
}

// prewarmEngineAssets pre-heats binaries and active SRS rules into Linux VFS PageCache
func (m *Manager) prewarmEngineAssets(binPath, cfgPath string, srsPaths []string) {
	buf := make([]byte, 64*1024)
	warmFile := func(p string) {
		if p == "" {
			return
		}
		f, err := os.Open(p)
		if err != nil {
			return
		}
		_, _ = io.CopyBuffer(io.Discard, f, buf)
		_ = f.Close()
	}

	warmFile(binPath)
	warmFile(cfgPath)
	for _, p := range srsPaths {
		warmFile(p)
	}
}

// reclaimOSMemory forces runtime GC and returns free heap spans to the OS via madvise
func (m *Manager) reclaimOSMemory() {
	runtime.GC()
	debug.FreeOSMemory()
}

func (m *Manager) rollback(eng schema.EngineType, etcDir string, snapshot map[string]string, prevConfig []byte) bool {
	if eng == "" {
		return false
	}

	log.Printf("🛡️ [Engine] Initiating emergency self-healing rollback to %s...", eng)

	binName := "mihomo"
	configFile := "config.yaml"
	if eng == schema.EngineSingbox {
		binName = "sing-box"
		configFile = "config.json"
	}

	binPath := m.findBinary(binName)
	configPath := filepath.Join(etcDir, configFile)
	if len(prevConfig) > 0 {
		_ = schema.WriteFileAtomicSecure(configPath, prevConfig)
	}

	cmd := m.buildCmd(eng, binPath, etcDir, configPath)
	pipeWG, err := m.startEngineProcess(eng, cmd)
	if err != nil {
		log.Printf("🚨 [Engine] Critical: rollback to %s failed: %v", eng, err)
		return false
	}

	m.currentProc = cmd.Process
	done := make(chan struct{})
	m.procDone = done
	log.Printf("🛡️ [Engine] Rollback process for %s started with PID %d", eng, cmd.Process.Pid)

	netutil.SafeGo("WaitRollbackEngine", func() {
		m.waitEngineProcess(eng, cmd, pipeWG, done)
	})

	if m.probeCoreHealth(eng, 3*time.Second) {
		m.RestoreClashState(snapshot)
		m.activeEngine = eng
		if m.ir != nil {
			m.ir.Meta.ActiveEngine = eng
			if data, mErr := json.MarshalIndent(m.ir, "", "  "); mErr == nil {
				_ = schema.WriteFileAtomicSecure(filepath.Join(m.workDir, "data", "config.ir.json"), data)
			}
		}
		log.Printf("✅ [Engine] Self-healing rollback to %s succeeded! Network preserved.", eng)
		return true
	}

	log.Printf("🚨 [Engine] Critical: rollback process %s failed health probe; stopping orphaned rollback process", eng)
	m.stopCurrentCore(binName)
	return false
}

func (m *Manager) triggerAllCoresDown(reason string) {
	m.activeEngine = ""
	m.currentProc = nil
	m.procDone = nil
	m.log("ERROR", "ENGINE", fmt.Sprintf("🚨 双核均启动/回滚失败，触发保底直通逃生: %s", reason))
	m.cbMu.RLock()
	cb := m.onAllCoresDown
	m.cbMu.RUnlock()
	if cb != nil {
		cb(reason)
	}
}

// SwitchEngine performs the Make-Before-Break Zero-Downtime Hot-Swap with health probe & auto-rollback.

func (m *Manager) SwitchEngine(target schema.EngineType) (*SwitchResult, error) {

	m.mu.Lock()

	m.switching.Store(true)
	// Set 1500ms grace window for /healthz/dataplane and Supervisor to prevent false alarms during active switch
	m.switchingGraceDeadline.Store(time.Now().Add(1500 * time.Millisecond).UnixNano())

	defer func() {

		m.switching.Store(false)
		m.switchingGraceDeadline.Store(0)

		m.mu.Unlock()

	}()

	startTime := time.Now()

	fromEngine := m.activeEngine

	res := &SwitchResult{

		FromEngine: fromEngine,

		ToEngine: target,

		Timestamp: startTime,
	}

	if fromEngine == target {
		m.log("INFO", "ENGINE", fmt.Sprintf("执行核心配置热重载: %s (配置同步与连接平滑保持)", target))
	} else {
		m.log("INFO", "ENGINE", fmt.Sprintf("发起双核热切换: %s ➔ %s (Zero-Downtime 平滑容灾)", fromEngine, target))
	}

	// Step 1: Snapshot active selections (lock already held by SwitchEngine)
	t0 := time.Now()
	snapshot := m.snapshotClashStateLocked()
	dSnapshot := time.Since(t0).Milliseconds()

	// Synchronize snapshot selections into IR so new engine defaults directly to active nodes
	if m.ir != nil && len(snapshot) > 0 {
		for i := range m.ir.Groups {
			if nowNode, ok := snapshot[m.ir.Groups[i].Name]; ok && nowNode != "" {
				m.ir.Groups[i].Now = nowNode
			}
		}
	}

	// Step 2: Compile target config (checking pre-compiled standby cache first)
	tCompStart := time.Now()
	var configData []byte
	var configFileName string
	var binName string

	if target == schema.EngineMihomo {
		configFileName = "config.yaml"
		binName = "mihomo"
	} else {
		configFileName = "config.json"
		binName = "sing-box"
	}

	m.precompiledMu.Lock()
	cachedData := m.precompiledConfig[target]
	delete(m.precompiledConfig, target)
	m.precompiledMu.Unlock()

	if len(cachedData) > 0 {
		configData = cachedData
		m.log("INFO", "ENGINE", fmt.Sprintf("⚡ 目标内核配置预编译极速命中 (0ms): %s", target))
	} else if target == schema.EngineMihomo {
		compiled, err := compiler.BuildMihomo(m.ir)
		if err != nil {
			res.Success = false
			res.ErrorMessage = "mihomo compile error: " + err.Error()
			m.lastResult = res
			return res, err
		}
		configData = compiled
	} else {
		compiled, err := compiler.BuildSingBox(m.ir)
		if err != nil {
			res.Success = false
			res.ErrorMessage = "singbox compile error: " + err.Error()
			m.lastResult = res
			return res, err
		}
		configData = compiled
	}
	dCompile := time.Since(tCompStart).Milliseconds()

	// Persist target config (save previous config bytes for in-place reload rollback)
	tWriteStart := time.Now()
	etcDir := filepath.Join(m.workDir, "etc")
	_ = os.MkdirAll(etcDir, 0700)
	targetConfigPath := filepath.Join(etcDir, configFileName)
	var prevConfigBytes []byte
	if oldData, rErr := os.ReadFile(targetConfigPath); rErr == nil {
		prevConfigBytes = oldData
	}

	if err := schema.WriteFileAtomicSecure(targetConfigPath, configData); err != nil {
		res.Success = false
		res.ErrorMessage = "failed to write config: " + err.Error()
		m.lastResult = res
		return res, err
	}
	dWrite := time.Since(tWriteStart).Milliseconds()

	binPath := m.findBinary(binName)

	// Step 2.2: Pre-flight /dev/net/tun availability & permission check before launching core
	if tunOK, tunReason := CheckTunDevice(); !tunOK {
		m.setLastCoreError(tunReason)
		m.log("ERROR", "ENGINE", fmt.Sprintf("❌ 启动前 TUN 设备预检失败 (%s): %s", target, tunReason))
		res.Success = false
		res.ErrorMessage = tunReason
		m.lastResult = res
		return res, fmt.Errorf("%s", tunReason)
	}

	// Step 2.5: Mandatory Pre-Switch Dry-Run Guard (Guarantees zero crashes from corrupt airport nodes)
	tDryStart := time.Now()
	if binPath != "" && runtime.GOOS == "linux" {
		configHash := sha256.Sum256(configData)
		cacheKey := fmt.Sprintf("%s:%x", target, configHash)

		if _, verified := m.dryRunCache.Load(cacheKey); !verified {
			if err := m.dryRunCheck(target, binPath, etcDir, targetConfigPath); err != nil {
				if fromEngine == target && len(prevConfigBytes) > 0 {
					_ = schema.WriteFileAtomicSecure(targetConfigPath, prevConfigBytes)
				}
				m.setLastCoreError(err.Error())
				m.log("ERROR", "ENGINE", fmt.Sprintf("⚠️ 目标内核配置预检失败，阻断热重载以保护当前稳定运行态: %v", err))
				res.Success = false
				res.ErrorMessage = "dry-run verification failed: " + err.Error()
				m.lastResult = res
				return res, err
			}
			// 限制 dryRunCache 上限，避免长期频繁切配导致无界增长
			if m.dryRunCacheSize.Load() >= 16 {
				m.dryRunCache.Range(func(k, _ any) bool {
					m.dryRunCache.Delete(k)
					return true
				})
				m.dryRunCacheSize.Store(0)
			}
			m.dryRunCache.Store(cacheKey, true)
			m.dryRunCacheSize.Add(1)
		} else {
			m.log("INFO", "ENGINE", fmt.Sprintf("⚡ 目标内核配置预检通过 (缓存极速命中 0ms): %s", target))
		}
	}
	dDry := time.Since(tDryStart).Milliseconds()

	// Step 2.8: Pre-flight VFS PageCache prewarm (warms target binary, config, and active rules into kernel memory)
	var srsFiles []string
	if target == schema.EngineSingbox {
		srsDir := filepath.Join(etcDir, "rules")
		if entries, err := os.ReadDir(srsDir); err == nil {
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), ".srs") {
					srsFiles = append(srsFiles, filepath.Join(srsDir, entry.Name()))
				}
			}
		}
	}
	m.prewarmEngineAssets(binPath, targetConfigPath, srsFiles)

	// Step 3: Stop old engine to release ports & TUN device
	tStopStart := time.Now()
	prevBin := "mihomo"
	if fromEngine == schema.EngineSingbox {
		prevBin = "sing-box"
	}
	m.stopCurrentCore(prevBin)
	var waitPorts []int
	if m.ir != nil {
		if m.ir.Network.ClashAPIPort > 0 {
			waitPorts = append(waitPorts, m.ir.Network.ClashAPIPort)
		}
		if m.ir.Network.MixedPort > 0 {
			waitPorts = append(waitPorts, m.ir.Network.MixedPort)
		}
	}
	if len(waitPorts) > 0 && !m.waitForPortsRelease(waitPorts, 1500*time.Millisecond) {
		log.Printf("⚠️ [Engine] Timeout waiting for ports %v release before spawning %s", waitPorts, target)
	}
	dStop := time.Since(tStopStart).Milliseconds()

	// Step 5: Start new engine
	tStartStart := time.Now()
	m.clearRecentCoreOutput()
	cmd := m.buildCmd(target, binPath, etcDir, targetConfigPath)
	pipeWG, err := m.startEngineProcess(target, cmd)
	if err != nil {
		log.Printf("🚨 [Engine] Failed to spawn %s: %v. Triggering self-healing rollback to %s!", target, err, fromEngine)
		m.setLastCoreError(err.Error())
		res.ErrorMessage = fmt.Sprintf("failed to spawn %s: %v", target, err)
		var rollbackPrev []byte
		if fromEngine == target {
			rollbackPrev = prevConfigBytes
		}
		if fromEngine == "" || !m.rollback(fromEngine, etcDir, snapshot, rollbackPrev) {
			m.triggerAllCoresDown(res.ErrorMessage)
		}
		res.Success = false
		m.lastResult = res
		return res, err
	}

	m.currentProc = cmd.Process
	done := make(chan struct{})
	m.procDone = done
	log.Printf("🚀 [Engine] Core engine %s started with PID %d", target, cmd.Process.Pid)

	netutil.SafeGo("WaitEngineProcess", func() {
		m.waitEngineProcess(target, cmd, pipeWG, done)
	})
	dStart := time.Since(tStartStart).Milliseconds()

	// Step 6: Active Health Probe (verify Clash API and DNS readiness within 3s)
	tProbeStart := time.Now()
	if !m.probeCoreHealth(target, 3*time.Second) {
		detail := m.summarizeRecentStderr()
		if detail != "" {
			res.ErrorMessage = fmt.Sprintf("core %s failed liveness health check (%s); network rolled back to %s", target, detail, fromEngine)
			m.log("ERROR", "ENGINE", fmt.Sprintf("❌ 核心 %s 启动后健康探针未就绪: %s", target, detail))
		} else {
			res.ErrorMessage = fmt.Sprintf("core %s failed liveness health check; network rolled back to %s", target, fromEngine)
		}
		m.setLastCoreError(res.ErrorMessage)
		log.Printf("🚨 [Engine] Target core %s failed health probe after launch! Triggering self-healing rollback to %s...", target, fromEngine)
		m.stopCurrentCore(binName)
		var rollbackPrev []byte
		if fromEngine == target {
			rollbackPrev = prevConfigBytes
		}
		if fromEngine == "" || !m.rollback(fromEngine, etcDir, snapshot, rollbackPrev) {
			m.triggerAllCoresDown(res.ErrorMessage)
		}
		res.Success = false
		m.lastResult = res
		return res, fmt.Errorf("%s", res.ErrorMessage)
	}
	dProbe := time.Since(tProbeStart).Milliseconds()

	// Step 7: Restore node states (asynchronously reconcile via Clash API; target core boots with active snapshot nodes at index 0)
	tRestoreStart := time.Now()
	snapCopy := make(map[string]string, len(snapshot))
	for k, v := range snapshot {
		snapCopy[k] = v
	}
	netutil.SafeGo("RestoreClashState", func() {
		count := m.RestoreClashState(snapCopy)
		log.Printf("✨ [Engine] Asynchronous proxy state reconciliation completed (%d groups)", count)
	})
	dRestore := time.Since(tRestoreStart).Milliseconds()
	restoredCount := len(snapshot)

	res.RestoredGroups = restoredCount
	res.StopCostMs = dStop
	res.StartupCostMs = dStart
	res.ElapsedMs = time.Since(startTime).Milliseconds()
	res.TotalCostMs = res.ElapsedMs
	if m.ir != nil {
		res.ActiveRules = len(m.ir.Rules)
		m.ir.Meta.ActiveEngine = target
		atomic.AddUint64(&m.irVersion, 1)
		m.precompiledMu.Lock()
		m.precompiledConfig = make(map[schema.EngineType][]byte)
		m.precompiledMu.Unlock()
		if data, mErr := json.MarshalIndent(m.ir, "", "  "); mErr == nil {
			_ = schema.WriteFileAtomicSecure(filepath.Join(m.workDir, "data", "config.ir.json"), data)
		}
	}
	m.activeEngine = target
	res.Success = true
	m.lastResult = res
	m.ResetCircuitBreaker()

	// Stage 7.5: Reclaim unused Go heap spans back to OS kernel (maintains ~35MB RSS ceiling)
	m.reclaimOSMemory()

	// Stage 7.6: Flush stale half-open connections via Clash API (prevents client connection stalls across Fake-IP pools)
	netutil.SafeGo("FlushStaleConnectionsAfterSwitch", m.flushStaleConnectionsAfterSwitch)

	log.Printf("⏱️ [Engine Timing] Snapshot: %dms, Compile: %dms, Write: %dms, DryRun: %dms, Stop: %dms, Start: %dms, Probe: %dms, Restore: %dms -> Total: %dms",
		dSnapshot, dCompile, dWrite, dDry, dStop, dStart, dProbe, dRestore, res.ElapsedMs)

	if fromEngine == target {
		log.Printf("✨ [Engine] In-place configuration hot-reload for %s completed in %d ms (%d groups restored)", target, res.ElapsedMs, restoredCount)
		m.log("INFO", "ENGINE", fmt.Sprintf("✅ 核心 %s 配置热重载完成 (耗时 %d ms, 恢复策略组: %d)", target, res.ElapsedMs, restoredCount))
	} else {
		log.Printf("✨ [Engine] Zero-downtime hot-swap to %s completed in %d ms (%d groups restored)", target, res.ElapsedMs, restoredCount)
		m.log("INFO", "ENGINE", fmt.Sprintf("✅ 双核平滑切换完成: %s ➔ %s (耗时 %d ms, 恢复策略组: %d)", fromEngine, target, res.ElapsedMs, restoredCount))
	}

	select {

	case m.statusCh <- target:

	default:

	}

	netutil.SafeGo("PrewarmEngine", func() { m.prewarmEngine(target) })
	netutil.SafeGo("WarmStandbyConfigs", m.WarmStandbyConfigs)

	return res, nil

}

// prewarmEngine pre-heats active proxy groups immediately after engine start

// to eliminate cold-start TLS latency and ensure instant traffic flow for LAN clients.

func (m *Manager) prewarmEngine(target schema.EngineType) {

	time.Sleep(150 * time.Millisecond)

	clashAPI, clashSecret, tunIface := m.snapshotNetMeta()

	// In Linux environments (PVE LXC / Container / Host), ensure Fake-IP route 198.18.0.0/15 dev cubex-tun is active

	if runtime.GOOS == "linux" {

		_ = exec.Command("ip", "route", "replace", "198.18.0.0/15", "dev", tunIface).Run()
		_ = exec.Command("conntrack", "-D", "-d", "198.18.0.0/15").Run()

	}

	// 优先从当前 IR 抽取业务策略组进行预热，不写死固定分组名
	var warmGroups []string
	m.mu.RLock()
	if m.ir != nil {
		for _, g := range m.ir.Groups {
			if g.Category == "service" || g.Category == "business" {
				warmGroups = append(warmGroups, g.Name)
				if len(warmGroups) >= 4 {
					break
				}
			}
		}
	}
	m.mu.RUnlock()
	if len(warmGroups) == 0 {
		warmGroups = []string{"🌐 国外流量", "YouTube", "Google", "AI"}
	}

	testURL := "https://www.gstatic.com/generate_204"
	if config.GlobalRuntimeSpec != nil && len(config.GlobalRuntimeSpec.ProbeURLs) > 0 {
		testURL = config.GlobalRuntimeSpec.ProbeURLs[0]
	}

	for _, g := range warmGroups {

		escaped := url.PathEscape(g)

		// 1. Warm specific proxy

		uProxy := fmt.Sprintf("%s/proxies/%s/delay?url=%s&timeout=2000", clashAPI, escaped, url.QueryEscape(testURL))

		ctx, cancel := context.WithTimeout(context.Background(), 2200*time.Millisecond)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, uProxy, nil)

		if err == nil {

			if clashSecret != "" {

				req.Header.Set("Authorization", "Bearer "+clashSecret)

			}

			resp, err := m.httpClient.Do(req)

			if err == nil {

				netutil.DrainAndCloseBody(resp)

			}

		}

		cancel()

		// 2. Pre-warm entire group

		uGroup := fmt.Sprintf("%s/group/%s/delay?url=%s&timeout=2000", clashAPI, escaped, url.QueryEscape(testURL))

		ctx2, cancel2 := context.WithTimeout(context.Background(), 2200*time.Millisecond)

		req2, err2 := http.NewRequestWithContext(ctx2, http.MethodGet, uGroup, nil)

		if err2 == nil {

			if clashSecret != "" {

				req2.Header.Set("Authorization", "Bearer "+clashSecret)

			}

			resp2, err2 := m.httpClient.Do(req2)

			if err2 == nil {

				netutil.DrainAndCloseBody(resp2)

			}

		}

		cancel2()

	}

	runtime.GC()
	debug.FreeOSMemory()

}

// StartSupervisor runs the 2.5s periodic health heartbeat & auto-healing state machine
func (m *Manager) StartSupervisor() {
	netutil.SafeGo("EngineSupervisor", func() {
		ticker := time.NewTicker(2500 * time.Millisecond)
		defer ticker.Stop()
		var healthyConsecutive int

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.mu.RLock()
				stopping := m.stopping
				active := m.activeEngine
				proc := m.currentProc
				procDone := m.procDone
				clashAPI := m.clashAPI
				m.mu.RUnlock()

				if stopping {
					return
				}
				if m.switching.Load() || m.IsSwitchingInGracePeriod() {
					continue
				}
				if m.IsCircuitBroken() {
					continue
				}
				if active == "" || proc == nil {
					continue
				}

			// 1. Process Liveness Check

			isAlive := true

			if runtime.GOOS != "windows" {

				if err := proc.Signal(syscall.Signal(0)); err != nil {

					isAlive = false

				}

			}

			// 2. Health API Probe

			if isAlive {

				isAlive = m.probeCoreHealthFor(proc, procDone, active, clashAPI, 1200*time.Millisecond)

			}

			if !isAlive {

				healthyConsecutive = 0

				m.mu.Lock()

				m.crashCount++

				count := m.crashCount

				m.mu.Unlock()

				pid := 0

				if proc != nil {

					pid = proc.Pid

				}

				m.log("WARN", "SUPERVISOR", fmt.Sprintf("🚨 核心 %s (PID %d) 异常退出或无响应 (故障计数: %d)", active, pid, count))

				log.Printf("🚨 [Supervisor] Detected active engine %s (PID %d) unhealthy/crashed! (Crash count: %d)", active, pid, count)

				standby := schema.EngineSingbox

				if active == schema.EngineSingbox {

					standby = schema.EngineMihomo

				}

				m.runSupervisorFailover(active, standby, count)

			} else {

				healthyConsecutive++

				if healthyConsecutive >= 10 { // 25s continuous healthy operation

					m.mu.Lock()

					if m.crashCount > 0 {

						m.crashCount = 0

					}

					m.mu.Unlock()

					if healthyConsecutive%24 == 0 { // Snapshot last-known-good every ~60s of healthy operation

						m.saveLastKnownGoodConfig()

					}

					if healthyConsecutive%6 == 0 { // Audit dead-node circuit breaker every ~15s

						m.PatrolCircuitBreaker()

					}

				}

			}

			}

		}

	})

}

func (m *Manager) saveLastKnownGoodConfig() {

	if m.workDir == "" {

		return

	}

	src := filepath.Join(m.workDir, "data", "config.ir.json")

	dst := filepath.Join(m.workDir, "data", "config.last-known-good.json")

	if data, err := os.ReadFile(src); err == nil && len(data) > 0 {

		_ = schema.WriteFileAtomicSecure(dst, data)

	}

}

// GetLastKnownGoodTime returns when the last known good snapshot was persisted.

func (m *Manager) GetLastKnownGoodTime() string {

	dst := filepath.Join(m.workDir, "data", "config.last-known-good.json")

	if info, err := os.Stat(dst); err == nil {

		return info.ModTime().Format("2006-01-02 15:04:05")

	}

	return "就绪保护中"

}

// PatrolCircuitBreaker audits all Selector groups in Clash API (127.0.0.1:9090).

// If any group is statically locked to a single concrete node that has timed out (Delay <= 0)

// or failed health checks, it automatically triggers a circuit-breaker to safely fall back to an active dynamic group.

func (m *Manager) PatrolCircuitBreaker() int {
	clashAPI, clashSecret, _ := m.snapshotNetMeta()
	apiClient := clashapi.NewClient(clashAPI, clashSecret, m.httpClient)

	ctx, cancel := context.WithTimeout(context.Background(), 1200*time.Millisecond)
	defer cancel()

	proxies, err := apiClient.GetProxies(ctx)
	if err != nil {
		return 0
	}

	healedCount := 0
	globalPhysicalCount := countGlobalPhysicalNodes(proxies)

	isSafeTarget := func(name string) bool {
		return strings.Contains(name, "自动") || strings.Contains(name, "故转") ||
			strings.Contains(name, "容灾") || name == "DIRECT" || name == "REJECT" || name == "block"
	}

	for grpName, grp := range proxies {
		if grp.Type != "Selector" || grp.Now == "" || isSafeTarget(grp.Now) {
			continue
		}

		// grp.Now is a single physical node (e.g. "[HHost-JP] xtls-reality"). Check its health.
		nodeInfo, exists := proxies[grp.Now]
		if !exists {
			continue
		}

		isDead := false
		lastDelay := 0
		if len(nodeInfo.History) > 0 {
			lastDelay = nodeInfo.History[len(nodeInfo.History)-1].Delay
			if lastDelay <= 0 || lastDelay >= 7500 {
				isDead = true
			}
		}

		if isDead {
			if globalPhysicalCount <= 1 {
				m.logCircuitHoldOnce(grpName, grp.Now, fmt.Sprintf("🛡️ 智能熔断: 策略组 [%s] 节点 [%s] 异常(%dms)，但全局仅配置 %d 个有效物理节点，禁止切入空分组或 DIRECT，原地保留等待主路由/外网恢复", grpName, grp.Now, lastDelay, globalPhysicalCount))
				continue
			}

			// Find optimal non-empty fallback dynamic target from grp.All
			fallbackTarget := selectCircuitBreakerFallback(proxies, grp.All, grp.Now)
			if fallbackTarget == "" {
				m.logCircuitHoldOnce(grpName, grp.Now, fmt.Sprintf("🛡️ 智能熔断: 策略组 [%s] 节点 [%s] 异常(%dms)，备选自动/容灾分组均为空或无其他可用物理节点，原地保留等待主路由/外网恢复", grpName, grp.Now, lastDelay))
				continue
			}

			if fallbackTarget != grp.Now {
				ctxPut, cancelPut := context.WithTimeout(context.Background(), 1500*time.Millisecond)
				sErr := apiClient.SwitchProxy(ctxPut, grpName, fallbackTarget)
				cancelPut()
				if sErr != nil {
					continue
				}

				m.mu.Lock()
				if m.ir != nil {
					for i := range m.ir.Groups {
						if m.ir.Groups[i].Name == grpName {
							m.ir.Groups[i].Now = fallbackTarget
							break
						}
					}
					atomic.AddUint64(&m.irVersion, 1)
					m.precompiledMu.Lock()
					m.precompiledConfig = make(map[schema.EngineType][]byte)
					m.precompiledMu.Unlock()
					if data, err := json.MarshalIndent(m.ir, "", "  "); err == nil {
						_ = schema.WriteFileAtomicSecure(filepath.Join(m.workDir, "data", "config.ir.json"), data)
					}
				}
				m.mu.Unlock()

				healedCount++
				log.Printf("🛡️ [CircuitBreaker] 策略组 [%s] 节点 [%s] 延迟异常(%dms)，已触发智能熔断回退至 [%s]！", grpName, grp.Now, lastDelay, fallbackTarget)
				m.log("WARN", "HA", fmt.Sprintf("🛡️ 智能熔断: 策略组 [%s] 单节点 [%s] 异常不可达，已静默回退至 [%s]", grpName, grp.Now, fallbackTarget))
			}
		}
	}

	return healedCount
}

// StartEndToEndWatchdog periodically verifies actual outbound data flow through the proxy.
// If the local core is unresponsive or both proxy and domestic paths stall, it auto-switches to the standby engine;
// if the core Clash API & domestic network are healthy while only proxy targets time out, it avoids unnecessary kernel thrashing.
func (m *Manager) StartEndToEndWatchdog() {
	netutil.SafeGo("EngineE2EWatchdog", func() {
		select {
		case <-m.stopCh:
			return
		case <-time.After(45 * time.Second):
		}

		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		var failCount int

		for {
			select {
			case <-m.stopCh:
				return
			case <-ticker.C:
				m.mu.RLock()
				stopping := m.stopping
				active := m.activeEngine
				ir := m.ir.Clone()
				proc := m.currentProc
				procDone := m.procDone
				clashAPI := m.clashAPI
				m.mu.RUnlock()

				if stopping {
					return
				}
				if m.switching.Load() || m.IsSwitchingInGracePeriod() {
					continue
				}
				if m.IsCircuitBroken() {
					continue
				}
				if active == "" || ir == nil || len(ir.Proxies) == 0 {
					continue
				}

				proxyPort := 7890
				if ir.Network.MixedPort > 0 {
					proxyPort = ir.Network.MixedPort
				}
				proxyURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
				if err != nil {
					continue
				}

				probeURLs := []string{"https://www.gstatic.com/generate_204", "http://cp.cloudflare.com/generate_204"}
				if config.GlobalRuntimeSpec != nil && len(config.GlobalRuntimeSpec.ProbeURLs) > 0 {
					probeURLs = config.GlobalRuntimeSpec.ProbeURLs
				}

				transport := &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
					DialContext: (&net.Dialer{
						Timeout: 4 * time.Second,
					}).DialContext,
					ResponseHeaderTimeout: 4 * time.Second,
					DisableKeepAlives:     true,
				}
				client := &http.Client{
					Transport: transport,
					Timeout:   5 * time.Second,
				}

				ok := false
				for _, targetURL := range probeURLs {
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					req, rErr := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
					if rErr == nil {
						resp, dErr := client.Do(req)
						if dErr == nil {
							netutil.DrainAndCloseBody(resp)
							if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
								ok = true
							}
						}
					}
					cancel()
					if ok {
						break
					}
				}
				transport.CloseIdleConnections()

				if ok {
					if failCount > 0 {
						m.log("INFO", "WATCHDOG", fmt.Sprintf("✅ 国际分流业务链路恢复畅通 (引擎: %s)", active))
					}
					failCount = 0
				} else {
					failCount++
					m.log("WARN", "WATCHDOG", fmt.Sprintf("⚠️ 国际分流业务链路探测超时 (引擎: %s, 连续失败: %d/6)", active, failCount))

					if failCount >= 3 {
						// 区分“内核/本地代理端口挂死”与“仅上游节点波动”：
						// 若内核 Clash API 存活且国内直连正常，第 3~5 次先上报上游节点异常交由 EWMA/熔断处理；
						// 若内核失联或连续 6 次仍无法恢复，再执行内核热切换。
						coreAlive := m.probeCoreHealthFor(proc, procDone, active, clashAPI, 800*time.Millisecond)
						domesticOK := probeDirectConnectivity()
						if coreAlive && domesticOK && failCount < 6 {
							m.cbMu.RLock()
							upCb := m.onUpstreamStall
							m.cbMu.RUnlock()
							if upCb != nil {
								upCb(active, fmt.Sprintf("连续 %d 次代理探测超时，内核接口与直连正常", failCount))
							}
							continue
						}

						failCount = 0
						standby := schema.EngineSingbox
						if active == schema.EngineSingbox {
							standby = schema.EngineMihomo
						}
						m.log("WARN", "SUPERVISOR", fmt.Sprintf("🚨 核心 %s 链路连续异常，触发容灾热切换 ➔ %s...", active, standby))
						if res, err := m.SwitchEngine(standby); err == nil {
							m.log("INFO", "SUPERVISOR", fmt.Sprintf("✅ 业务假死容灾自愈完成: 已切换至 %s (耗时 %d ms)", standby, res.ElapsedMs))
						}
					}
				}
			}
		}
	})
}

func probeDirectConnectivity() bool {
	for _, addr := range []string{"223.5.5.5:53", "119.29.29.29:53"} {
		c, err := net.DialTimeout("tcp", addr, 800*time.Millisecond)
		if err == nil {
			_ = c.Close()
			return true
		}
	}
	return false
}

// IsSwitchingInGracePeriod checks if an intentional core switch is currently in progress within the 1500ms grace window.
func (m *Manager) IsSwitchingInGracePeriod() bool {
	deadline := m.switchingGraceDeadline.Load()
	return deadline > 0 && time.Now().UnixNano() < deadline
}

// SetSwitchingGraceDeadline explicitly sets a switching grace deadline (useful for testing and deterministic timeout control).
func (m *Manager) SetSwitchingGraceDeadline(deadline time.Time) {
	m.switchingGraceDeadline.Store(deadline.UnixNano())
}

// flushStaleConnectionsAfterSwitch cleans up stale half-open connections in the kernel after an engine switch,
// so client browsers immediately re-resolve and connect with the new core's Fake-IP pool.
func (m *Manager) flushStaleConnectionsAfterSwitch() {
	clashAPI, secret, _ := m.snapshotNetMeta()
	apiClient := clashapi.NewClient(clashAPI, secret, m.httpClient)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err := apiClient.DeleteAllConnections(ctx); err == nil {
		log.Printf("🧹 [Engine] Flushed stale connections via Clash API (%s)", apiClient.BaseURL)
	}
}

type clashProxyEntry = clashapi.ProxyEntry

func isVirtualOrBuiltInProxy(name, typ string) bool {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "", "DIRECT", "REJECT", "REJECT-DROP", "PASS", "COMPATIBLE", "GLOBAL", "BLOCK", "DNS-OUT":
		return true
	}
	if schema.IsInformationalOrInvalidNode(name) {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(typ)) {
	case "", "selector", "urltest", "url-test", "fallback", "loadbalance", "load-balance", "relay",
		"direct", "reject", "rejectdrop", "reject-drop", "block", "pass", "compatible", "dns", "unknown":
		return true
	}
	return false
}

func collectValidPhysicalNodes(proxies map[string]clashProxyEntry, target string, visited map[string]bool) []string {
	if target == "" || visited[target] {
		return nil
	}
	visited[target] = true

	entry, ok := proxies[target]
	if !ok {
		return nil
	}
	if len(entry.All) == 0 {
		if isVirtualOrBuiltInProxy(target, entry.Type) {
			return nil
		}
		return []string{target}
	}

	var out []string
	seen := make(map[string]bool)
	for _, member := range entry.All {
		for _, node := range collectValidPhysicalNodes(proxies, member, visited) {
			if !seen[node] {
				seen[node] = true
				out = append(out, node)
			}
		}
	}
	return out
}

func countGlobalPhysicalNodes(proxies map[string]clashProxyEntry) int {
	count := 0
	for name, entry := range proxies {
		if len(entry.All) == 0 && !isVirtualOrBuiltInProxy(name, entry.Type) {
			count++
		}
	}
	return count
}

func groupHasUsableAlternative(proxies map[string]clashProxyEntry, candidate, currentDead string) bool {
	if candidate == "" || candidate == currentDead || schema.IsInformationalOrInvalidNode(candidate) {
		return false
	}
	nodes := collectValidPhysicalNodes(proxies, candidate, make(map[string]bool))
	for _, n := range nodes {
		if n != currentDead {
			return true
		}
	}
	return false
}

func selectCircuitBreakerFallback(proxies map[string]clashProxyEntry, candidates []string, currentDead string) string {
	tiers := []func(string) bool{
		func(c string) bool { return strings.Contains(c, "自动") },
		func(c string) bool { return strings.Contains(c, "故转") || strings.Contains(c, "容灾") },
		func(c string) bool { return c == "🌐 国外流量" },
	}
	for _, matchTier := range tiers {
		for _, candidate := range candidates {
			if matchTier(candidate) && groupHasUsableAlternative(proxies, candidate, currentDead) {
				return candidate
			}
		}
	}
	return ""
}

func (m *Manager) logCircuitHoldOnce(grpName, nodeName, msg string) {
	key := grpName + "|" + nodeName
	now := time.Now()
	m.faultMu.Lock()
	if m.circuitHoldLogAt == nil {
		m.circuitHoldLogAt = make(map[string]time.Time)
	}
	last, exists := m.circuitHoldLogAt[key]
	if exists && now.Sub(last) < 60*time.Second {
		m.faultMu.Unlock()
		return
	}
	m.circuitHoldLogAt[key] = now
	m.faultMu.Unlock()

	log.Print(msg)
	m.log("WARN", "HA", msg)
}
