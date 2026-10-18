package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

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
type Manager struct {
	mu           sync.Mutex
	ir           *schema.ConfigIR
	workDir      string
	activeEngine schema.EngineType
	currentProc  *os.Process
	clashAPI     string
	httpClient   *http.Client
	lastResult   *SwitchResult
	statusCh     chan schema.EngineType
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
		_ = os.WriteFile(irFile, cleanData, 0644)
	}
	apiAddr := fmt.Sprintf("http://127.0.0.1:%d", initialIR.Network.ClashAPIPort)
	return &Manager{
		ir:           initialIR,
		workDir:      workDir,
		activeEngine: initialIR.Meta.ActiveEngine,
		clashAPI:     apiAddr,
		httpClient:   &http.Client{Timeout: 3 * time.Second},
		statusCh:     make(chan schema.EngineType, 10),
	}
}

// GetActiveEngine returns the currently active persona.
func (m *Manager) GetActiveEngine() schema.EngineType {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeEngine
}

// GetLastResult returns the telemetry of the most recent switch.
func (m *Manager) GetLastResult() *SwitchResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastResult
}

// GetIR returns a reference to the active IR config.
func (m *Manager) GetIR() *schema.ConfigIR {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ir
}

// UpdateIR safely updates the configuration IR in memory and persists to disk.
func (m *Manager) UpdateIR(newIR *schema.ConfigIR) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ir = newIR
	_ = os.MkdirAll(filepath.Join(m.workDir, "data"), 0755)
	if data, err := json.MarshalIndent(newIR, "", "  "); err == nil {
		_ = os.WriteFile(filepath.Join(m.workDir, "data", "config.ir.json"), data, 0644)
	}
}

// SnapshotClashState snapshots active selector node choices from 127.0.0.1:9090/proxies.
func (m *Manager) SnapshotClashState() map[string]string {
	snapshot := make(map[string]string)
	url := fmt.Sprintf("%s/proxies", m.clashAPI)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return snapshot
	}

	resp, err := m.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Fallback to IR defaults
		for _, g := range m.ir.Groups {
			if g.Now != "" {
				snapshot[g.Name] = g.Now
			}
		}
		return snapshot
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return snapshot
	}

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
	return snapshot
}

// RestoreClashState injects cached state into the newly launched core's Clash API.
func (m *Manager) RestoreClashState(snapshot map[string]string) int {
	restored := 0
	for groupName, targetNode := range snapshot {
		url := fmt.Sprintf("%s/proxies/%s", m.clashAPI, groupName)
		payload, _ := json.Marshal(map[string]string{"name": targetNode})
		req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, url, bytes.NewReader(payload))
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := m.httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
				restored++
			}
		}
	}
	return restored
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

func (m *Manager) buildCmd(eng schema.EngineType, binPath, etcDir, configPath string) *exec.Cmd {
	var cmd *exec.Cmd
	if eng == schema.EngineMihomo {
		cmd = exec.Command(binPath, "-d", etcDir, "-f", configPath)
	} else {
		cmd = exec.Command(binPath, "run", "-c", configPath)
		cmd.Dir = etcDir
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// OOM Defense & Garbage Collection tuning for tight router RAM (CHR 448MB)
	cmd.Env = append(os.Environ(),
		"GOMEMLIMIT=45MiB",
		"GOGC=50",
		"GODEBUG=madvdontneed=1",
	)
	return cmd
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
	if err != nil {
		outStr := strings.TrimSpace(string(out))
		return fmt.Errorf("%s dry-run failed: %s (%w)", eng, outStr, err)
	}
	return nil
}

func (m *Manager) stopCurrentCore(binName string) {
	if m.currentProc != nil && m.currentProc.Pid > 0 {
		pid := m.currentProc.Pid
		log.Printf("🛑 [Engine] Gracefully stopping active core (PID %d)...", pid)
		if runtime.GOOS != "windows" {
			_ = m.currentProc.Signal(syscall.SIGTERM)
			done := make(chan struct{})
			go func() {
				_, _ = m.currentProc.Wait()
				close(done)
			}()
			select {
			case <-done:
				log.Printf("✅ [Engine] Core (PID %d) gracefully exited", pid)
			case <-time.After(500 * time.Millisecond):
				log.Printf("⚠️ [Engine] Core (PID %d) did not terminate in 500ms, force killing...", pid)
				_ = m.currentProc.Kill()
				<-done
			}
		} else {
			_ = m.currentProc.Kill()
			_, _ = m.currentProc.Wait()
		}
		m.currentProc = nil
	}
	if runtime.GOOS != "windows" && binName != "" {
		_ = exec.Command("pkill", "-15", binName).Run()
	}
	time.Sleep(100 * time.Millisecond)
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
	if err := cmd.Start(); err == nil {
		m.currentProc = cmd.Process
		log.Printf("🛡️ [Engine] Rollback process for %s started with PID %d", eng, cmd.Process.Pid)
		go func(proc *os.Process, e schema.EngineType) {
			state, err := proc.Wait()
			log.Printf("⚠️ [Engine] Rollback core %s (PID %d) exited: %v (state: %v)", e, proc.Pid, err, state)
		}(cmd.Process, eng)
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
	defer m.mu.Unlock()

	startTime := time.Now()
	fromEngine := m.activeEngine

	res := &SwitchResult{
		FromEngine: fromEngine,
		ToEngine:   target,
		Timestamp:  startTime,
	}

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
	_ = os.MkdirAll(etcDir, 0755)
	targetConfigPath := filepath.Join(etcDir, configFileName)
	if err := os.WriteFile(targetConfigPath, configData, 0644); err != nil {
		res.Success = false
		res.ErrorMessage = "failed to write config: " + err.Error()
		m.lastResult = res
		return res, err
	}

	binPath := m.findBinary(binName)

	// Step 3: Strict Pre-Switch Dry-Run Validation (Make-Before-Break)
	// If target config is invalid, abort immediately to protect the currently running engine!
	if err := m.dryRunCheck(target, binPath, etcDir, targetConfigPath); err != nil {
		log.Printf("⚠️ [Engine] Pre-flight dry-run failed for %s: %v. Aborting switch to protect active network!", target, err)
		res.Success = false
		res.ErrorMessage = fmt.Sprintf("pre-flight validation failed: %v", err)
		m.lastResult = res
		return res, err
	}

	// Step 4: Graceful shutdown of old engine and release port 53, port 9090, port 7893 and TUN
	prevBin := "mihomo"
	if fromEngine == schema.EngineSingbox {
		prevBin = "sing-box"
	}
	m.stopCurrentCore(prevBin)

	// Step 5: Start new engine
	cmd := m.buildCmd(target, binPath, etcDir, targetConfigPath)
	if err := cmd.Start(); err != nil {
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
	log.Printf("🚀 [Engine] Core engine %s started with PID %d", target, cmd.Process.Pid)
	go func(proc *os.Process, eng schema.EngineType) {
		state, err := proc.Wait()
		log.Printf("⚠️ [Engine] Core process %s (PID %d) exited: %v (state: %v)", eng, proc.Pid, err, state)
	}(cmd.Process, target)

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
