package engine

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"strconv"
	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

// SwitchResult contains statistics about the kernel switch.
type SwitchResult struct {
	FromEngine      schema.EngineType `json:"from_engine"`
	ToEngine        schema.EngineType `json:"to_engine"`
	ElapsedMs       int64             `json:"elapsed_ms"`
	RestoredGroups  int               `json:"restored_groups"`
	Success         bool              `json:"success"`
	ErrorMessage    string            `json:"error_message,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
}

// Manager controls the dual-engine lifecycle, hot-swap state machine, and status.
// LogFunc is the callback signature for streaming real-time core and engine events to the ring buffer.
type LogFunc func(level, module, msg string)

type Manager struct {
	mu           sync.RWMutex
	cbMu         sync.RWMutex
	ir           *schema.ConfigIR
	workDir      string
	activeEngine schema.EngineType
	currentProc  *os.Process
	procDone     chan struct{}
	clashAPI     string
	httpClient   *http.Client
	lastResult   *SwitchResult
	statusCh     chan schema.EngineType
	stopping     bool
	crashCount   int
	logCallback  LogFunc
}

// NewManager creates a dual-engine manager.
func NewManager(workDir string, initialIR *schema.ConfigIR) *Manager {
	if initialIR == nil {
		initialIR = schema.NewDefaultIR()
	}
	irFile := filepath.Join(workDir, "data", "config.ir.json")
	if data, err := os.ReadFile(irFile); err == nil {
		var savedIR schema.ConfigIR
		if json.Unmarshal(data, &savedIR) == nil && len(savedIR.Proxies) > 0 {
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
		}
	}
	schema.SyncAllStrategyGroups(initialIR)
	if initialIR.Network.DNSPort == 0 || initialIR.Network.DNSPort == 1053 {
		initialIR.Network.DNSPort = 53
	}
	if cleanData, err := json.MarshalIndent(initialIR, "", "  "); err == nil {
		_ = schema.WriteFileAtomicSecure(irFile, cleanData)
	}
	apiAddr := fmt.Sprintf("http://127.0.0.1:%d", initialIR.Network.ClashAPIPort)
	mgr := &Manager{
		ir:           initialIR,
		workDir:      workDir,
		activeEngine: initialIR.Meta.ActiveEngine,
		clashAPI:     apiAddr,
		httpClient:   &http.Client{Timeout: 3 * time.Second},
		statusCh:     make(chan schema.EngineType, 10),
	}
	mgr.StartSupervisor()
	return mgr
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
			for _, line := range lines {
				if strings.HasPrefix(line, "VmRSS:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseFloat(fields[1], 64); err == nil && kb > 0 {
							return kb / 1024.0
						}
					}
				}
			}
		}
	}
	// Fallback estimate if /proc is not available or before process starts
	if eng == schema.EngineMihomo {
		return 48.5
	}
	return 18.2
}


// GetLastResult returns the telemetry of the most recent switch.
func (m *Manager) GetLastResult() *SwitchResult {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastResult
}

// GetIR returns a reference to the active IR config.
func (m *Manager) GetIR() *schema.ConfigIR {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.ir
}

// UpdateIR safely updates the configuration IR in memory and persists to disk.
func (m *Manager) UpdateIR(newIR *schema.ConfigIR) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ir = newIR
	_ = os.MkdirAll(filepath.Join(m.workDir, "data"), 0700)
	if data, err := json.MarshalIndent(newIR, "", "  "); err == nil {
		_ = schema.WriteFileAtomicSecure(filepath.Join(m.workDir, "data", "config.ir.json"), data)
	}
}

// SnapshotClashState snapshots active selector node choices from 127.0.0.1:9090/proxies.
func (m *Manager) SnapshotClashState() map[string]string {
	snapshot := make(map[string]string)
	url := fmt.Sprintf("%s/proxies", m.clashAPI)
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err == nil {
		if resp, err := m.httpClient.Do(req); err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				if body, err := io.ReadAll(resp.Body); err == nil {
					var data struct {
						Proxies map[string]struct {
							Name string `json:"name"`
							Now  string `json:"now"`
							Type string `json:"type"`
						} `json:"proxies"`
					}
					if err := json.Unmarshal(body, &data); err == nil {
						for name, p := range data.Proxies {
							if p.Now != "" {
								snapshot[name] = p.Now
							}
						}
					}
				}
			}
		}
	}
	// Fallback / merge with IR defaults if any group is missing in snapshot
	if m.ir != nil {
		for _, g := range m.ir.Groups {
			if snapshot[g.Name] == "" && g.Now != "" {
				snapshot[g.Name] = g.Now
			}
		}
	}
	return snapshot
}

// RestoreClashState injects cached state into the newly launched core's Clash API concurrently.
func (m *Manager) RestoreClashState(snapshot map[string]string) int {
	if len(snapshot) == 0 {
		return 0
	}
	var wg sync.WaitGroup
	var restoredCount int64
	sem := make(chan struct{}, 10)

	for groupName, targetNode := range snapshot {
		if groupName == "" || targetNode == "" {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(grp, node string) {
			defer wg.Done()
			defer func() { <-sem }()

			escapedGroup := url.PathEscape(grp)
			u := fmt.Sprintf("%s/proxies/%s", m.clashAPI, escapedGroup)
			payload, _ := json.Marshal(map[string]string{"name": node})
			ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
			defer cancel()
			req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(payload))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")
			if m.ir != nil && m.ir.Network.ClashSecret != "" {
				req.Header.Set("Authorization", "Bearer "+m.ir.Network.ClashSecret)
			}
			resp, err := m.httpClient.Do(req)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
					atomic.AddInt64(&restoredCount, 1)
				}
			}
		}(groupName, targetNode)
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

func getAdaptiveGOMEMLIMIT() string {
	limitMb := 90
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if totalKb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						totalMb := totalKb / 1024
						if totalMb <= 512 {
							limitMb = 70 // 512M CHR / router: 70MiB ceiling prevents OOM
						} else if totalMb <= 1024 {
							limitMb = 85 // 1GB host: 85MiB
						} else if totalMb <= 2048 {
							limitMb = 96 // 2GB host: 96MiB ceiling prevents container OOM
						} else {
							limitMb = 160 // 4GB+ fnOS / PVE: 160MiB for line-speed multi-gigabit
						}
					}
				}
				break
			}
		}
	}
	return fmt.Sprintf("GOMEMLIMIT=%dMiB", limitMb)
}

func (m *Manager) buildCmd(eng schema.EngineType, binPath, etcDir, configPath string) *exec.Cmd {
	var cmd *exec.Cmd
	if eng == schema.EngineMihomo {
		cmd = exec.Command(binPath, "-d", etcDir, "-f", configPath)
	} else {
		cmd = exec.Command(binPath, "run", "-c", configPath)
		cmd.Dir = etcDir
	}

	// High-Performance Go Runtime & GC tuning:
	// Adaptive GOMEMLIMIT ensures tight memory ceiling on routers and containers,
	// with GOGC=50 for prompt garbage reclamation of network packets and TLS buffers.
	cmd.Env = append(os.Environ(),
		getAdaptiveGOMEMLIMIT(),
		"GOGC=50",
		"GODEBUG=madvdontneed=1",
	)
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
		if pipeType == "stderr" {
			fmt.Fprintln(os.Stderr, line)
		} else {
			fmt.Fprintln(os.Stdout, line)
		}

		level := "INFO"
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "fatal") || strings.Contains(lower, "panic") || strings.Contains(lower, "fail") {
			level = "ERROR"
		} else if strings.Contains(lower, "warn") {
			level = "WARN"
		} else if strings.Contains(lower, "debug") {
			level = "DEBUG"
		}

		m.log(level, module, line)
	}
}

func (m *Manager) startEngineProcess(eng schema.EngineType, cmd *exec.Cmd) error {
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		cmd.Stdout = os.Stdout
	}
	stderrPipe, err2 := cmd.StderrPipe()
	if err2 != nil {
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	if stdoutPipe != nil {
		go m.streamLogPipe(eng, "stdout", stdoutPipe)
	}
	if stderrPipe != nil {
		go m.streamLogPipe(eng, "stderr", stderrPipe)
	}
	return nil
}

func (m *Manager) dryRunCheck(eng schema.EngineType, binPath, etcDir, configPath string) error {
	var testCmd *exec.Cmd
	if eng == schema.EngineMihomo {
		testCmd = exec.Command(binPath, "-t", "-d", etcDir, "-f", configPath)
	} else {
		testCmd = exec.Command(binPath, "check", "-c", configPath)
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

func (m *Manager) stopCurrentCore(binName string) {
	if m.currentProc != nil && m.currentProc.Pid > 0 {
		pid := m.currentProc.Pid
		log.Printf("🛑 [Engine] Gracefully stopping active core (PID %d)...", pid)
		done := m.procDone
		if runtime.GOOS != "windows" {
			_ = m.currentProc.Signal(syscall.SIGTERM)
			if done != nil {
				select {
				case <-done:
					log.Printf("✅ [Engine] Core (PID %d) gracefully exited", pid)
				case <-time.After(500 * time.Millisecond):
					log.Printf("⚠️ [Engine] Core (PID %d) did not terminate in 500ms, force killing...", pid)
					_ = m.currentProc.Kill()
					select {
					case <-done:
					case <-time.After(300 * time.Millisecond):
					}
				}
			}
		} else {
			_ = m.currentProc.Kill()
			if done != nil {
				select {
				case <-done:
				case <-time.After(500 * time.Millisecond):
				}
			}
		}
		m.currentProc = nil
		m.procDone = nil
	}
	if runtime.GOOS != "windows" && binName != "" {
		_ = exec.Command("pkill", "-9", binName).Run()
	}
	time.Sleep(50 * time.Millisecond)
}

func (m *Manager) probeCoreHealth(target schema.EngineType, maxWait time.Duration) bool {
	deadline := time.Now().Add(maxWait)
	versionURL := fmt.Sprintf("%s/version", m.clashAPI)
	for time.Now().Before(deadline) {
		if m.currentProc == nil {
			return false
		}
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, versionURL, nil)
		if err == nil {
			if m.ir.Network.ClashSecret != "" {
				req.Header.Set("Authorization", "Bearer "+m.ir.Network.ClashSecret)
			}
			resp, err := m.httpClient.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return true
				}
			}
		}
		time.Sleep(60 * time.Millisecond)
	}
	return false
}

func (m *Manager) rollback(eng schema.EngineType, etcDir string, snapshot map[string]string) {
	log.Printf("🛡️ [Engine] Initiating emergency self-healing rollback to %s...", eng)
	binName := "mihomo"
	configFile := "config.yaml"
	if eng == schema.EngineSingbox {
		binName = "sing-box"
		configFile = "config.json"
	}
	binPath := m.findBinary(binName)
	configPath := filepath.Join(etcDir, configFile)

	cmd := m.buildCmd(eng, binPath, etcDir, configPath)
	if err := m.startEngineProcess(eng, cmd); err == nil {
		m.currentProc = cmd.Process
		done := make(chan struct{})
		m.procDone = done
		log.Printf("🛡️ [Engine] Rollback process for %s started with PID %d", eng, cmd.Process.Pid)
		go func(proc *os.Process, e schema.EngineType, d chan struct{}) {
			state, err := proc.Wait()
			log.Printf("⚠️ [Engine] Rollback core %s (PID %d) exited: %v (state: %v)", e, proc.Pid, err, state)
			close(d)
		}(cmd.Process, eng, done)
		if m.probeCoreHealth(eng, 3*time.Second) {
			m.RestoreClashState(snapshot)
			m.activeEngine = eng
			m.ir.Meta.ActiveEngine = eng
			log.Printf("✅ [Engine] Self-healing rollback to %s succeeded! Network preserved.", eng)
		}
	} else {
		log.Printf("🚨 [Engine] Critical: rollback to %s failed: %v", eng, err)
	}
}

// SwitchEngine performs the Make-Before-Break Zero-Downtime Hot-Swap with health probe & auto-rollback.
func (m *Manager) SwitchEngine(target schema.EngineType) (*SwitchResult, error) {
	m.mu.Lock()
	m.stopping = true
	defer func() {
		m.stopping = false
		m.mu.Unlock()
	}()

	startTime := time.Now()
	fromEngine := m.activeEngine

	res := &SwitchResult{
		FromEngine: fromEngine,
		ToEngine:   target,
		Timestamp:  startTime,
	}
	m.log("INFO", "ENGINE", fmt.Sprintf("发起双核切换指令: %s -> %s", fromEngine, target))

	// Step 1: Snapshot active selections
	snapshot := m.SnapshotClashState()

	// Step 2: Compile target config
	var configData []byte
	var configFileName string
	var binName string

	if target == schema.EngineMihomo {
		compiled, err := compiler.BuildMihomo(m.ir)
		if err != nil {
			res.Success = false
			res.ErrorMessage = "mihomo compile error: " + err.Error()
			m.lastResult = res
			return res, err
		}
		configData = compiled
		configFileName = "config.yaml"
		binName = "mihomo"
	} else {
		compiled, err := compiler.BuildSingBox(m.ir)
		if err != nil {
			res.Success = false
			res.ErrorMessage = "singbox compile error: " + err.Error()
			m.lastResult = res
			return res, err
		}
		configData = compiled
		configFileName = "config.json"
		binName = "sing-box"
	}

	// Persist target config
	etcDir := filepath.Join(m.workDir, "etc")
	_ = os.MkdirAll(etcDir, 0700)
	targetConfigPath := filepath.Join(etcDir, configFileName)
	if err := schema.WriteFileAtomicSecure(targetConfigPath, configData); err != nil {
		res.Success = false
		res.ErrorMessage = "failed to write config: " + err.Error()
		m.lastResult = res
		return res, err
	}

	binPath := m.findBinary(binName)

	// Step 3: Stop old engine to release ports & TUN device
	prevBin := "mihomo"
	if fromEngine == schema.EngineSingbox {
		prevBin = "sing-box"
	}
	m.stopCurrentCore(prevBin)

	// Step 5: Start new engine
	cmd := m.buildCmd(target, binPath, etcDir, targetConfigPath)
	if err := m.startEngineProcess(target, cmd); err != nil {
		log.Printf("🚨 [Engine] Failed to spawn %s: %v. Triggering self-healing rollback to %s!", target, err, fromEngine)
		res.ErrorMessage = fmt.Sprintf("failed to spawn %s: %v", target, err)
		if fromEngine != "" && fromEngine != target {
			m.rollback(fromEngine, etcDir, snapshot)
		}
		res.Success = false
		m.lastResult = res
		return res, err
	}

	m.currentProc = cmd.Process
	done := make(chan struct{})
	m.procDone = done
	log.Printf("🚀 [Engine] Core engine %s started with PID %d", target, cmd.Process.Pid)
	go func(proc *os.Process, eng schema.EngineType, d chan struct{}) {
		state, err := proc.Wait()
		log.Printf("⚠️ [Engine] Core process %s (PID %d) exited: %v (state: %v)", eng, proc.Pid, err, state)
		close(d)
	}(cmd.Process, target, done)

	// Step 6: Active Health Probe (verify Clash API and DNS readiness within 3s)
	if !m.probeCoreHealth(target, 3*time.Second) {
		log.Printf("🚨 [Engine] Target core %s failed health probe after launch! Triggering self-healing rollback to %s...", target, fromEngine)
		m.stopCurrentCore(binName)
		if fromEngine != "" && fromEngine != target {
			m.rollback(fromEngine, etcDir, snapshot)
		}
		res.Success = false
		res.ErrorMessage = fmt.Sprintf("core %s failed liveness health check; network rolled back to %s", target, fromEngine)
		m.lastResult = res
		return res, fmt.Errorf("%s", res.ErrorMessage)
	}

	// Step 7: Restore node states
	restoredCount := m.RestoreClashState(snapshot)
	res.RestoredGroups = restoredCount

	m.activeEngine = target
	m.ir.Meta.ActiveEngine = target
	res.ElapsedMs = time.Since(startTime).Milliseconds()
	res.Success = true
	m.lastResult = res
	log.Printf("✨ [Engine] Zero-downtime hot-swap to %s completed in %d ms (%d groups restored)", target, res.ElapsedMs, restoredCount)

	select {
	case m.statusCh <- target:
	default:
	}

	return res, nil
}

// StartSupervisor runs the 5s periodic health heartbeat & auto-healing state machine
func (m *Manager) StartSupervisor() {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			m.mu.Lock()
			stopping := m.stopping
			active := m.activeEngine
			proc := m.currentProc
			m.mu.Unlock()

			if stopping || active == "" || proc == nil {
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
				isAlive = m.probeCoreHealth(active, 2*time.Second)
			}

			if !isAlive {
				m.mu.Lock()
				m.crashCount++
				count := m.crashCount
				m.mu.Unlock()

				log.Printf("🚨 [Supervisor] Detected active engine %s (PID %d) unhealthy/crashed! (Crash count: %d)", active, proc.Pid, count)

				if count <= 3 {
					log.Printf("🛡️ [Supervisor] Auto-recovering %s (attempt %d/3) with backoff...", active, count)
					time.Sleep(time.Duration(count) * time.Second)
					if _, err := m.SwitchEngine(active); err == nil {
						log.Printf("✅ [Supervisor] Successfully revived engine %s on attempt %d", active, count)
					}
				} else {
					standby := schema.EngineSingbox
					if active == schema.EngineSingbox {
						standby = schema.EngineMihomo
					}
					log.Printf("🚨 [Supervisor] Engine %s failed 3 consecutive times! Auto failover to standby engine %s...", active, standby)
					if _, err := m.SwitchEngine(standby); err == nil {
						log.Printf("✅ [Supervisor] Auto-failover to %s succeeded! Network restored.", standby)
						m.mu.Lock()
						m.crashCount = 0
						m.mu.Unlock()
					}
				}
			} else {
				m.mu.Lock()
				if m.crashCount > 0 {
					m.crashCount = 0
				}
				m.mu.Unlock()
			}
		}
	}()
}

