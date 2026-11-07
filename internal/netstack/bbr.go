package netstack

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// BBRStatus provides real-time telemetry about host TCP congestion control.
type BBRStatus struct {
	AvailableAlgorithms []string `json:"available_algorithms"`
	CurrentAlgorithm    string   `json:"current_algorithm"`
	IsBBREnabled        bool     `json:"is_bbr_enabled"`
	BBRVersion          string   `json:"bbr_version"` // "bbrv3", "bbrv1", "cubic", etc.
	DefaultQdisc        string   `json:"default_qdisc"`
	FastOpenCode        int      `json:"fast_open_code"` // 3 = client + server enabled
	KernelVersion       string   `json:"kernel_version"`
	LastOptimizedAt     string   `json:"last_optimized_at"`
	OptimizationApplied bool     `json:"optimization_applied"`
}

// BBRManager manages TCP BBR congestion control and socket buffers.
type BBRManager struct {
	mu     sync.RWMutex
	status BBRStatus
}

var (
	globalBBRManager *BBRManager
	bbrOnce          sync.Once
)

// GetBBRManager returns the singleton BBRManager.
func GetBBRManager() *BBRManager {
	bbrOnce.Do(func() {
		globalBBRManager = &BBRManager{}
		globalBBRManager.Detect()
	})
	return globalBBRManager
}

// Detect inspects current kernel TCP congestion control without mutating state.
func (m *BBRManager) Detect() BBRStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	status := BBRStatus{
		CurrentAlgorithm: "cubic",
		DefaultQdisc:     "fq_codel",
		FastOpenCode:     0,
		KernelVersion:    runtime.GOOS + "/" + runtime.GOARCH,
	}

	if runtime.GOOS != "linux" {
		status.CurrentAlgorithm = "default"
		status.BBRVersion = "unsupported_os"
		m.status = status
		return status
	}

	// 1. Read available algorithms
	if data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_available_congestion_control"); err == nil {
		algs := strings.Fields(strings.TrimSpace(string(data)))
		status.AvailableAlgorithms = algs
	}

	// 2. Read current algorithm
	if data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_congestion_control"); err == nil {
		status.CurrentAlgorithm = strings.TrimSpace(string(data))
	}

	// 3. Read default qdisc
	if data, err := os.ReadFile("/proc/sys/net/core/default_qdisc"); err == nil {
		status.DefaultQdisc = strings.TrimSpace(string(data))
	}

	// 4. Read TCP Fast Open
	if data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_fastopen"); err == nil {
		var code int
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &code)
		status.FastOpenCode = code
	}

	// 5. Check if BBR / BBRv3 is enabled
	hasBBR3 := false
	hasBBR := false
	for _, alg := range status.AvailableAlgorithms {
		if alg == "bbr3" {
			hasBBR3 = true
		} else if alg == "bbr" {
			hasBBR = true
		}
	}

	if strings.Contains(status.CurrentAlgorithm, "bbr") {
		status.IsBBREnabled = true
		if status.CurrentAlgorithm == "bbr3" || hasBBR3 {
			status.BBRVersion = "bbrv3"
		} else {
			status.BBRVersion = "bbrv1"
		}
	} else if hasBBR3 {
		status.BBRVersion = "bbr3_available"
	} else if hasBBR {
		status.BBRVersion = "bbr_available"
	} else {
		status.BBRVersion = status.CurrentAlgorithm
	}

	// Read kernel release
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		status.KernelVersion = strings.TrimSpace(string(data))
	}

	m.status = status
	return status
}

// Optimize applies BBRv3/BBR congestion control, FQ qdisc, and TCP Fast Open.
func (m *BBRManager) Optimize() (BBRStatus, error) {
	if runtime.GOOS != "linux" {
		return m.Detect(), nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Target algorithm: prioritize bbr3, fallback to bbr
	targetAlg := "bbr"
	for _, a := range m.status.AvailableAlgorithms {
		if a == "bbr3" {
			targetAlg = "bbr3"
			break
		}
	}

	// 1. Try loading tcp_bbr kernel module if bbr not present
	if !m.status.IsBBREnabled {
		_ = exec.Command("modprobe", "tcp_bbr").Run()
	}

	// 2. Set default qdisc to fq (BBR requirement)
	_ = os.WriteFile("/proc/sys/net/core/default_qdisc", []byte("fq\n"), 0644)

	// 3. Set TCP congestion control to bbr / bbr3
	_ = os.WriteFile("/proc/sys/net/ipv4/tcp_congestion_control", []byte(targetAlg+"\n"), 0644)

	// 4. Enable TCP Fast Open (3 = client & server)
	_ = os.WriteFile("/proc/sys/net/ipv4/tcp_fastopen", []byte("3\n"), 0644)

	// 5. Optimize socket buffer limits (up to 32MB buffers for 2.5G/10G line-rate)
	_ = os.WriteFile("/proc/sys/net/core/rmem_max", []byte("33554432\n"), 0644)
	_ = os.WriteFile("/proc/sys/net/core/wmem_max", []byte("33554432\n"), 0644)
	_ = os.WriteFile("/proc/sys/net/ipv4/tcp_rmem", []byte("4096 87380 33554432\n"), 0644)
	_ = os.WriteFile("/proc/sys/net/ipv4/tcp_wmem", []byte("4096 65536 33554432\n"), 0644)

	// Re-detect to confirm changes
	status := m.status
	if data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_congestion_control"); err == nil {
		status.CurrentAlgorithm = strings.TrimSpace(string(data))
		if strings.Contains(status.CurrentAlgorithm, "bbr") {
			status.IsBBREnabled = true
			status.BBRVersion = targetAlg
			status.OptimizationApplied = true
			status.LastOptimizedAt = time.Now().Format("2006-01-02 15:04:05")
		}
	}
	if data, err := os.ReadFile("/proc/sys/net/core/default_qdisc"); err == nil {
		status.DefaultQdisc = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile("/proc/sys/net/ipv4/tcp_fastopen"); err == nil {
		fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &status.FastOpenCode)
	}

	m.status = status
	return status, nil
}

// GetStatus returns the current status.
func (m *BBRManager) GetStatus() BBRStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}
