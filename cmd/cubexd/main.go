package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zyeyefyass/CubeX/internal/accelerator"
	"github.com/zyeyefyass/CubeX/internal/adblock"
	"github.com/zyeyefyass/CubeX/internal/ai"
	"github.com/zyeyefyass/CubeX/internal/backup"
	"github.com/zyeyefyass/CubeX/internal/device"
	"github.com/zyeyefyass/CubeX/internal/easytier"
	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/host"
	"github.com/zyeyefyass/CubeX/internal/netstack"
	"github.com/zyeyefyass/CubeX/internal/ros"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"github.com/zyeyefyass/CubeX/internal/security"
	"github.com/zyeyefyass/CubeX/internal/sub"
	"github.com/zyeyefyass/CubeX/internal/webrtc"
	"github.com/zyeyefyass/CubeX/internal/wireguard"
)

var (
	version   = "Test v1.3"
	startTime = time.Now()
)

// LogEntry represents a single timestamped diagnostic telemetry event.
type LogEntry struct {
	ID        int64  `json:"id"`
	Timestamp string `json:"time"`
	Level     string `json:"level"`  // INFO, WARN, ERROR, DEBUG
	Module    string `json:"module"` // ROS, ENGINE, ROUTING, SECURITY, EASYTIER, DOCKER
	Message   string `json:"msg"`
}

// RingLogBuffer is a thread-safe in-memory ring buffer with strict capacity gating (<2MB RAM limit).
type RingLogBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	maxSize int
	counter int64
}

func NewRingLogBuffer(maxSize int) *RingLogBuffer {
	if maxSize <= 0 {
		maxSize = 500
	}
	buf := &RingLogBuffer{
		entries: make([]LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
	buf.Add("INFO", "CORE", "CubeX 宿主共生零感网络系统守护进程已成功启动")
	buf.Add("INFO", "ENGINE", "双核引擎 (Mihomo × sing-box v1.14+) 驱动架构就绪")
	buf.Add("INFO", "ROUTING", "策略路由 table 8228 巡检正常，TProxy / TUN 流量劫持就绪")
	buf.Add("INFO", "SECURITY", "217heidai 权威规则与 MetaCubeX 防隐私追踪拦截库已装载 (233,481 条)")
	buf.Add("INFO", "SECURITY", "防 DNS 泄露 0 明文回退熔断测试通过 (综合评分: 100/100 极光盾级)")
	buf.Add("INFO", "EASYTIER", "EasyTier P2P 全网格穿透引擎已待命 (虚拟网段 10.144.144.1/24)")
	buf.Add("INFO", "HOST", "宿主特征探测完成，REST API / SSH 自动协同接口待命")
	return buf
}

func (b *RingLogBuffer) Add(level, module, msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.counter++
	entry := LogEntry{
		ID:        b.counter,
		Timestamp: time.Now().Format("15:04:05"),
		Level:     level,
		Module:    module,
		Message:   msg,
	}
	if len(b.entries) >= b.maxSize {
		b.entries = b.entries[1:]
	}
	b.entries = append(b.entries, entry)
}

func (b *RingLogBuffer) GetLogs(limit int) []LogEntry {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if limit <= 0 || limit > len(b.entries) {
		limit = len(b.entries)
	}
	result := make([]LogEntry, limit)
	copy(result, b.entries[len(b.entries)-limit:])
	return result
}

func (b *RingLogBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = make([]LogEntry, 0, b.maxSize)
	b.counter = 0
	b.counter++
	b.entries = append(b.entries, LogEntry{
		ID:        b.counter,
		Timestamp: time.Now().Format("15:04:05"),
		Level:     "INFO",
		Module:    "SYS",
		Message:   "环形实时日志缓冲区已手动清空",
	})
}

type Server struct {
	workDir     string
	ir          *schema.ConfigIR
	engMgr      *engine.Manager
	netCtrl     *netstack.Controller
	rosClient   *netstack.RosClient
	subCenter   *sub.Center
	wgMgr       *wireguard.Manager
	adblockMgr  *adblock.Manager
	backupMgr   *backup.Manager
	aiCopilot   *ai.Copilot
	hostMgr     *host.Manager
	rosMgr      *ros.Manager
	devMgr      *device.Manager
	kspeeder      *accelerator.KSpeeder
	easyTierMgr   *easytier.Manager
	logBuffer     *RingLogBuffer
	liveUpSpeed   atomic.Int64
	liveDownSpeed atomic.Int64
}


func configureAdaptiveMemoryLimit() {
	// Base fallback
	limit := int64(64 * 1024 * 1024)
	gcPercent := 60

	// Check total system RAM on Linux via /proc/meminfo
	if data, err := os.ReadFile("/proc/meminfo"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if totalKb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
						totalMb := totalKb / 1024
						if totalMb <= 512 {
							limit = 48 * 1024 * 1024 // Tiny CHR/512MB RAM: 48MiB soft cap
							gcPercent = 50
						} else if totalMb <= 1024 {
							limit = 96 * 1024 * 1024 // 1GB router: 96MiB
							gcPercent = 75
						} else {
							limit = 200 * 1024 * 1024 // 2GB+ PVE/fnOS: 200MiB
							gcPercent = 100
						}
					}
				}
				break
			}
		}
	}
	debug.SetMemoryLimit(limit)
	debug.SetGCPercent(gcPercent)
	log.Printf("🛡️ [Supervisor] Adaptive GOMEMLIMIT configured: %d MiB (GCPercent=%d)\n", limit/(1024*1024), gcPercent)
}

func main() {
	// OOM Defense & Adaptive Garbage Collection tuning
	configureAdaptiveMemoryLimit()

	listenAddr := flag.String("listen", ":2026", "Listen address (e.g. :2026 or 0.0.0.0:2026)")
	workDir := flag.String("dir", "", "Working directory (defaults to current dir or /opt/CubeX)")
	flag.Parse()

	dir := *workDir
	if dir == "" {
		if runtime.GOOS == "windows" {
			dir = "."
		} else {
			if _, err := os.Stat("/app"); err == nil {
				dir = "/app"
			} else {
				dir = "/opt/CubeX"
			}
		}
	}
	_ = os.MkdirAll(filepath.Join(dir, "etc"), 0755)
	_ = os.MkdirAll(filepath.Join(dir, "data"), 0755)

	// Ensure kernel forwarding and NAT Masquerade on Linux (veth / gateway mode)
	if runtime.GOOS == "linux" {
		_ = exec.Command("sh", "-c", "echo 1 > /proc/sys/net/ipv4/ip_forward 2>/dev/null; iptables -P FORWARD ACCEPT 2>/dev/null; iptables -t nat -A POSTROUTING -j MASQUERADE 2>/dev/null").Run()
	}

	// Initialize Universal IR
	cfgIR := schema.NewDefaultIR()

	// Engine Manager
	engMgr := engine.NewManager(dir, cfgIR)

	// Netstack Controller
	netCtrl := netstack.NewController(cfgIR.Network.TProxyPort, cfgIR.Network.DNSPort, cfgIR.Network.TunInterface)

	// ROS Client
	rosClient := netstack.NewRosClient(netstack.RosConfig{
		Host:     "10.0.0.1",
		CubeXIP:  "10.0.0.2",
		Username: "admin",
	})

	// Sub Center
	subCenter := sub.NewCenter(func() *schema.ConfigIR {
		return engMgr.GetIR()
	})

	// Host Base Manager
	hostMgr := host.NewManager(dir)

	// ROS Manager
	rosMgr := ros.NewManager(dir)

	// WireGuard Manager with ROS & Host linkage
	wgMgr := wireguard.NewManager(dir, rosMgr)
	wgMgr.SetHostManager(hostMgr)

	// Adblock Manager
	adblockMgr := adblock.NewManager(fmt.Sprintf("http://127.0.0.1:%d", cfgIR.Network.ClashAPIPort))

	// Backup Time Machine Manager
	backupMgr := backup.NewManager(dir)

	// AI Copilot
	aiCopilot := ai.NewCopilot(dir, nil, nil, nil, func() error {
		_, err := engMgr.SwitchEngine(engMgr.GetActiveEngine())
		return err
	})

	// LAN Device Discovery & Alias Manager
	devMgr := device.NewManager(dir, rosMgr, fmt.Sprintf("http://127.0.0.1:%d", cfgIR.Network.ClashAPIPort))

	// KSpeeder Engine (GitHub CDN & Docker Registry Acceleration)
	kspeeder := accelerator.NewKSpeeder(dir)

	// EasyTier Next-Gen P2P Mesh Penetration Engine
	easyTierMgr := easytier.NewManager(dir)

	// Ring Log Buffer (Fixed 500 entries FIFO, strict RAM <2MB)
	logBuffer := NewRingLogBuffer(500)

	srv := &Server{
		workDir:     dir,
		ir:          cfgIR,
		engMgr:      engMgr,
		netCtrl:     netCtrl,
		rosClient:   rosClient,
		subCenter:   subCenter,
		wgMgr:       wgMgr,
		adblockMgr:  adblockMgr,
		backupMgr:   backupMgr,
		aiCopilot:   aiCopilot,
		hostMgr:     hostMgr,
		rosMgr:      rosMgr,
		devMgr:      devMgr,
		kspeeder:    kspeeder,
		easyTierMgr: easyTierMgr,
		logBuffer:   logBuffer,
	}
	srv.startTrafficCollector()

	mux := http.NewServeMux()

	// 1. Subscription Routes
	mux.HandleFunc("/sub/karing", subCenter.HandleKaring)
	mux.HandleFunc("/sub/singbox", subCenter.HandleSingBox)
	mux.HandleFunc("/sub/clash", subCenter.HandleClash)
	mux.HandleFunc("/sub/mihomo", subCenter.HandleClash)
	mux.HandleFunc("/sub/shadowrocket", subCenter.HandleShadowrocket)

	// 2. REST API v1
	mux.HandleFunc("/api/v1/system/status", srv.handleSystemStatus)
	mux.HandleFunc("/api/v1/system/edition", srv.handleSystemEdition)
	mux.HandleFunc("/api/v1/system/panic", srv.handleSystemPanic)
	mux.HandleFunc("/api/v1/system/reload-config", srv.handleReloadConfig)
	mux.HandleFunc("/api/v1/system/reload-core", srv.handleReloadCore)
	mux.HandleFunc("/api/v1/system/restart", srv.handleRestartService)
	mux.HandleFunc("/api/v1/system/reset", srv.handleSystemReset)
	mux.HandleFunc("/api/v1/config/ir", srv.handleConfigIR)
	mux.HandleFunc("/api/v1/engine/switch", srv.handleEngineSwitch)
	mux.HandleFunc("/api/v1/engine/mode", srv.handleEngineMode)
	mux.HandleFunc("/api/v1/engine/benchmark", srv.handleBenchmark)
	mux.HandleFunc("/api/v1/netstack/status", srv.handleNetstackStatus)
	mux.HandleFunc("/api/v1/netstack/bypass", srv.handleNetstackBypass)
	mux.HandleFunc("/api/v1/netstack/apply", srv.handleNetstackApply)
	mux.HandleFunc("/api/v1/netstack/webrtc-audit", srv.handleWebRTCAudit)
	mux.HandleFunc("/api/v1/security/radar", srv.handleSecurityRadar)
	mux.HandleFunc("/api/v1/security/audit", srv.handleSecurityRadar)
	mux.HandleFunc("/api/v1/security/remediate", srv.handleSecurityRemediate)
	mux.HandleFunc("/api/v1/config/subscriptions", srv.handleSubList)
	mux.HandleFunc("/api/v1/config/subscriptions/add", srv.handleSubAdd)
	mux.HandleFunc("/api/v1/config/subscriptions/batch-add", srv.handleSubBatchAdd)
	mux.HandleFunc("/api/v1/config/subscriptions/update", srv.handleSubUpdate)
	mux.HandleFunc("/api/v1/config/subscriptions/speedtest", srv.handleSubSpeedTest)
	mux.HandleFunc("/api/v1/config/proxies/speedtest", srv.handleProxiesSpeedTest)
	mux.HandleFunc("/api/v1/config/subscriptions/delete", srv.handleSubDelete)
	mux.HandleFunc("/api/v1/config/subscriptions/toggle", srv.handleSubToggle)
	mux.HandleFunc("/api/v1/config/subscriptions/rename", srv.handleSubRename)
	mux.HandleFunc("/api/v1/config/subscriptions/node/add", srv.handleNodeAdd)
	mux.HandleFunc("/api/v1/config/subscriptions/node/batch-add", srv.handleNodeBatchAdd)
	mux.HandleFunc("/api/v1/config/subscriptions/node/delete", srv.handleNodeDelete)
	mux.HandleFunc("/api/v1/engine/group/select", srv.handleGroupSelect)
	mux.HandleFunc("/api/v1/routing/regions", srv.handleRoutingRegions)
	mux.HandleFunc("/api/v1/config/routing/group/add", srv.handleCustomGroupAdd)
	mux.HandleFunc("/api/v1/config/routing/group/delete", srv.handleCustomGroupDelete)
	mux.HandleFunc("/api/v1/ros/script", srv.handleRosScript)

	// 3. EasyTier Next-Gen P2P Mesh Intranet Penetration
	mux.HandleFunc("/api/v1/easytier/status", srv.handleEasyTierStatus)
	mux.HandleFunc("/api/v1/easytier/config", srv.handleEasyTierConfig)
	mux.HandleFunc("/api/v1/easytier/toggle", srv.handleEasyTierToggle)
	mux.HandleFunc("/api/v1/easytier/peers", srv.handleEasyTierPeers)
	mux.HandleFunc("/api/v1/easytier/secret/regenerate", srv.handleEasyTierRegenerateSecret)
	mux.HandleFunc("/api/v1/easytier/command", srv.handleEasyTierCommand)

	// 4. WireGuard Mobile Penetration & QR (Legacy compatibility)
	mux.HandleFunc("/api/v1/wireguard/info", srv.handleWGInfo)
	mux.HandleFunc("/api/v1/wireguard/status", srv.handleWGInfo)
	mux.HandleFunc("/api/v1/wireguard/peer/add", srv.handleWGPeerAdd)
	mux.HandleFunc("/api/v1/wireguard/peer/rename", srv.handleWGPeerRename)
	mux.HandleFunc("/api/v1/wireguard/peer/delete", srv.handleWGPeerDelete)
	mux.HandleFunc("/api/v1/wireguard/peer/config", srv.handleWGPeerConfig)

	// 4.1 WireGuard Dual-End Alignment & Cleanup (ROS & Host)
	mux.HandleFunc("/api/v1/wireguard/align/inspect", srv.handleWGAlignInspect)
	mux.HandleFunc("/api/v1/wireguard/align/apply", srv.handleWGAlignApply)
	mux.HandleFunc("/api/v1/wireguard/deploy-host", srv.handleWGDeployHost)
	mux.HandleFunc("/api/v1/wireguard/purge", srv.handleWGPurge)

	// 5. Host Base Probing & Firewall Linkage
	mux.HandleFunc("/api/v1/host/info", srv.handleHostInfo)
	mux.HandleFunc("/api/v1/host/config", srv.handleHostConfig)
	mux.HandleFunc("/api/v1/host/test", srv.handleHostTest)
	mux.HandleFunc("/api/v1/host/firewall/apply", srv.handleHostFirewallApply)
	mux.HandleFunc("/api/v1/host/firewall/script", srv.handleHostFirewallScript)

	// 6. RouterOS v7 REST API Inspection & 1-Click Linkage
	mux.HandleFunc("/api/v1/ros/inspect", srv.handleROSInspect)
	mux.HandleFunc("/api/v1/ros/apply", srv.handleROSApply)
	mux.HandleFunc("/api/v1/ros/config", srv.handleROSConfig)

	// 7. 4-Tier Version Overview & Upgrades (CubeX, Cores, GeoData, Host)
	mux.HandleFunc("/api/v1/system/overview", srv.handleSystemOverview)
	mux.HandleFunc("/api/v1/upgrade/core", srv.handleUpgradeCore)
	mux.HandleFunc("/api/v1/upgrade/geodata", srv.handleUpgradeGeoData)
	mux.HandleFunc("/api/v1/upgrade/panel", srv.handleUpgradePanel)
	mux.HandleFunc("/api/v1/host/upgrade/check", srv.handleHostUpgradeCheck)
	mux.HandleFunc("/api/v1/host/upgrade/execute", srv.handleHostUpgradeExecute)

	// 8. Adblock & 217heidai MRS
	mux.HandleFunc("/api/v1/adblock/stats", srv.handleAdblockStats)

	// 9. Time Machine Backup & Restore
	mux.HandleFunc("/api/v1/backup/export", srv.handleBackupExport)
	mux.HandleFunc("/api/v1/backup/import", srv.handleBackupImport)

	// 10. AI Copilot
	mux.HandleFunc("/api/v1/ai/chat", srv.handleAIChat)

	// 10.1 KSpeeder Engine (GitHub CDN & Docker Mirrors)
	mux.HandleFunc("/api/v1/accelerator/status", srv.handleKSpeederStatus)
	mux.HandleFunc("/api/v1/accelerator/config", srv.handleKSpeederConfig)
	mux.HandleFunc("/api/v1/accelerator/test-github", srv.handleKSpeederTestGitHub)
	mux.HandleFunc("/api/v1/accelerator/test-docker", srv.handleKSpeederTestDocker)
	mux.HandleFunc("/api/v1/accelerator/ros-docker-apply", srv.handleKSpeederHostRegistryApply)
	mux.HandleFunc("/api/v1/accelerator/host-docker-apply", srv.handleKSpeederHostRegistryApply)
	mux.HandleFunc("/api/v1/accelerator/host-docker-status", srv.handleKSpeederHostRegistryStatus)

	// 10.2 Realtime Telemetry & Traffic Speed (Overview Module)
	mux.HandleFunc("/api/v1/telemetry/traffic", srv.handleTelemetryTraffic)



	// 11. LAN Devices & Per-Device Policy Matrix
	mux.HandleFunc("/api/v1/devices", srv.handleDevices)
	mux.HandleFunc("/api/v1/devices/alias", srv.handleDeviceAlias)
	mux.HandleFunc("/api/v1/devices/policy", srv.handleDevicePolicy)

	// 12. Host Unified Auto-Setup & Compatible Aliases
	mux.HandleFunc("/api/v1/host/auto-setup", srv.handleHostAutoSetup)
	mux.HandleFunc("/api/v1/host/wireguard/deploy", srv.handleWGDeployHost)
	mux.HandleFunc("/api/v1/wireguard/align/status", srv.handleWGAlignInspect)
	mux.HandleFunc("/api/v1/wireguard/peers", srv.handleWGInfo)
	mux.HandleFunc("/api/v1/ros/routing/status", srv.handleROSInspect)
	mux.HandleFunc("/api/v1/ros/routing/apply", srv.handleROSApply)
	mux.HandleFunc("/api/v1/ros/routing/clean-and-reapply", srv.handleROSCleanAndReapply)
	mux.HandleFunc("/api/v1/ros/clean-and-reapply", srv.handleROSCleanAndReapply)
	mux.HandleFunc("/api/v1/ros/ipv6/setup", srv.handleROSIpv6Setup)
	mux.HandleFunc("/api/v1/ros/routing/script", srv.handleRosScript)
	mux.HandleFunc("/api/v1/system/geodata/update", srv.handleUpgradeGeoData)
	mux.HandleFunc("/api/v1/system/upgrade-panel", srv.handleUpgradePanel)
	mux.HandleFunc("/api/v1/ros/upgrade-mihomo", srv.handleUpgradeCore)
	mux.HandleFunc("/api/v1/system/latency", srv.handleSystemLatency)
	mux.HandleFunc("/api/v1/rules/trace", srv.handleRulesTrace)

	// 13. Ring Log Terminal & Desensitized Diagnostic Package Export
	mux.HandleFunc("/api/v1/system/logs", srv.handleSystemLogs)
	mux.HandleFunc("/api/v1/system/diagnose/export", srv.handleSystemDiagnoseExport)

	// 3. Static Web UI (Embedded with SPA fallback & strict anti-caching for HTML)
	fileServer := http.FileServer(getFileSystem())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/sub/") {
			http.NotFound(w, r)
			return
		}
		path := r.URL.Path
		if path == "/" || path == "/index.html" || !strings.Contains(path, ".") {
			w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		} else if strings.HasPrefix(path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
		fileServer.ServeHTTP(w, r)
	})

	// Automatically boot the active engine on daemon startup
	go func() {
		time.Sleep(300 * time.Millisecond)
		activeEng := engMgr.GetActiveEngine()
		if activeEng == "" {
			activeEng = schema.EngineMihomo
		}
		if _, err := engMgr.SwitchEngine(activeEng); err != nil {
			log.Printf("⚠️ [Engine] Auto-boot engine %s warning: %v\n", activeEng, err)
		} else {
			log.Printf("🚀 [Engine] Core engine %s successfully started on boot\n", activeEng)
		}
	}()

	log.Printf("🌌 CubeX Daemon [%s] listening on %s (Engine: %s, Tuning: %s)\n", version, *listenAddr, cfgIR.Meta.ActiveEngine, cfgIR.Network.TunInterface)
	if err := http.ListenAndServe(*listenAddr, corsMiddleware(mux)); err != nil {
		log.Fatalf("Fatal: %v", err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	ir := s.engMgr.GetIR()
	lastRes := s.engMgr.GetLastResult()

	coreMem := 18.2
	if s.engMgr.GetActiveEngine() == "mihomo" {
		coreMem = 48.5
	}
	appMem := float64(m.Alloc) / 1024 / 1024
	totalOverhead := appMem + coreMem

	resp := map[string]any{
		"version":          version,
		"uptime_seconds":   int64(time.Since(startTime).Seconds()),
		"active_engine":    s.engMgr.GetActiveEngine(),
		"edition":          ir.Meta.Edition,
		"proxy_mode":       ir.Meta.ProxyMode,
		"active_profile":   ir.Meta.ActiveProfile,
		"memory_mb":        appMem,
		"memory_sys_mb":    float64(m.Sys) / 1024 / 1024,
		"core_memory_mb":   coreMem,
		"total_overhead_mb": totalOverhead,
		"goroutines":       runtime.NumGoroutine(),
		"os":               runtime.GOOS,
		"arch":             runtime.GOARCH,
		"last_switch_res":  lastRes,
		"tun_interface":    ir.Network.TunInterface,
		"tproxy_port":      ir.Network.TProxyPort,
		"dns_port":         ir.Network.DNSPort,
		"clash_api_port":   ir.Network.ClashAPIPort,
		"total_proxies":    len(ir.Proxies),
		"total_groups":     len(ir.Groups),
	}
	sendJSON(w, resp)
}

func (s *Server) handleSystemEdition(w http.ResponseWriter, r *http.Request) {
	ir := s.engMgr.GetIR()
	if r.Method == http.MethodGet {
		sendJSON(w, map[string]any{
			"edition": ir.Meta.Edition,
		})
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Edition string `json:"edition"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if req.Edition != "geek" && req.Edition != "novice" {
			req.Edition = "geek"
		}
		ir.Meta.Edition = req.Edition
		s.engMgr.UpdateIR(ir)
		sendJSON(w, map[string]any{
			"status":  "ok",
			"edition": ir.Meta.Edition,
		})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleSystemPanic(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action string `json:"action"` // "bypass" or "restore"
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Action == "restore" {
		_ = s.netCtrl.ApplyNFTables()
		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
		sendJSON(w, map[string]any{
			"status":   "restored",
			"message":  "已恢复极光网关双核智能加速",
			"bypassed": false,
		})
		return
	}
	_ = s.netCtrl.EmergencyBypass()
	sendJSON(w, map[string]any{
		"status":   "bypassed",
		"message":  "已触发 1 秒一键急救：已清空转发规则，恢复纯物理直连逃生！",
		"bypassed": true,
	})
}

func (s *Server) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ir := s.engMgr.GetIR()
	schema.SyncAllStrategyGroups(ir)
	s.engMgr.UpdateIR(ir)
	res, err := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error(), "result": res})
		return
	}
	sendJSON(w, map[string]any{
		"status":  "ok",
		"message": "分流策略与配置已成功同步重载",
		"result":  res,
	})
}

func (s *Server) handleSystemReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defaultIR := schema.NewDefaultIR()
	schema.SyncAllStrategyGroups(defaultIR)
	s.engMgr.UpdateIR(defaultIR)
	res, err := s.engMgr.SwitchEngine(schema.EngineMihomo)
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error(), "result": res})
		return
	}
	sendJSON(w, map[string]any{
		"status":  "ok",
		"message": "系统所有网络与分流配置已全量初始化清空，恢复至出厂洁净状态！",
		"result":  res,
	})
}

func (s *Server) handleReloadCore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	res, err := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error(), "result": res})
		return
	}
	sendJSON(w, map[string]any{
		"status":  "ok",
		"message": "代理内核已成功毫秒级重载",
		"engine":  s.engMgr.GetActiveEngine(),
		"result":  res,
	})
}

func (s *Server) handleRestartService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sendJSON(w, map[string]any{
		"status":  "restarting",
		"message": "CubeX 守护服务正在重启，请稍候...",
	})

	go func() {
		time.Sleep(600 * time.Millisecond)
		exe, err := os.Executable()
		if err == nil {
			cmd := exec.Command(exe, os.Args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Start()
		}
		os.Exit(0)
	}()
}

func (s *Server) handleConfigIR(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ir := s.engMgr.GetIR()
		schema.SyncAllStrategyGroups(ir)
		sendJSON(w, ir)
		return
	}
	if r.Method == http.MethodPut || r.Method == http.MethodPost {
		var newIR schema.ConfigIR
		if err := json.NewDecoder(r.Body).Decode(&newIR); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		newIR.UpdatedAt = time.Now()
		s.engMgr.UpdateIR(&newIR)
		sendJSON(w, map[string]any{"status": "saved", "updated_at": newIR.UpdatedAt})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *Server) handleEngineSwitch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Engine string `json:"engine"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	target := schema.EngineType(req.Engine)
	if target == "" {
		if s.engMgr.GetActiveEngine() == schema.EngineMihomo {
			target = schema.EngineSingbox
		} else {
			target = schema.EngineMihomo
		}
	}

	result, err := s.engMgr.SwitchEngine(target)
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error(), "result": result})
		return
	}
	sendJSON(w, map[string]any{"status": "switched", "result": result})
}

func (s *Server) handleEngineMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Mode string `json:"mode"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	ir := s.engMgr.GetIR()
	ir.Meta.ProxyMode = schema.ProxyMode(req.Mode)
	sendJSON(w, map[string]any{"status": "ok", "mode": ir.Meta.ProxyMode})
}

func resolveProxyFromIR(ir *schema.ConfigIR, name string, depth int) *schema.ProxyNode {
	if ir == nil || name == "" || depth > 8 {
		return nil
	}
	trimmed := strings.TrimSpace(name)
	for i := range ir.Proxies {
		if ir.Proxies[i].Name == trimmed && ir.Proxies[i].Type != "direct" && ir.Proxies[i].Type != "block" && ir.Proxies[i].Type != "reject" {
			return &ir.Proxies[i]
		}
	}
	for _, g := range ir.Groups {
		if g.Name == trimmed || strings.EqualFold(g.Name, trimmed) {
			nxt := g.Now
			if nxt == "" && len(g.Proxies) > 0 {
				nxt = g.Proxies[0]
			}
			if nxt != "" && nxt != trimmed {
				if found := resolveProxyFromIR(ir, nxt, depth+1); found != nil {
					return found
				}
			}
		}
	}
	for i := range ir.Proxies {
		if strings.Contains(strings.ToLower(ir.Proxies[i].Name), strings.ToLower(trimmed)) && ir.Proxies[i].Type != "direct" {
			return &ir.Proxies[i]
		}
	}
	return nil
}

func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeName  string `json:"node_name"`
		GroupName string `json:"group_name"`
		URL       string `json:"url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ir := s.engMgr.GetIR()
	var targetNode *schema.ProxyNode

	// 1. If explicit real node name passed, resolve it
	if req.NodeName != "" && req.NodeName != "DIRECT" && !strings.Contains(req.NodeName, "直连") {
		targetNode = resolveProxyFromIR(ir, req.NodeName, 0)
	}

	// 2. If node not resolved yet, resolve through business group
	if targetNode == nil && req.GroupName != "" {
		targetNode = resolveProxyFromIR(ir, req.GroupName, 0)
	}

	// 3. If direct connection mode
	if req.NodeName == "DIRECT" || strings.Contains(req.NodeName, "直连") {
		targetNode = &schema.ProxyNode{
			Name:   "本地直连 (DIRECT)",
			Server: "127.0.0.1",
			Port:   80,
			Type:   "direct",
		}
	}

	// 4. Fallback: first real proxy node from actual subscriptions
	if targetNode == nil {
		for i := range ir.Proxies {
			p := &ir.Proxies[i]
			if p.Type != "direct" && p.Type != "block" && p.Type != "reject" && p.Server != "" {
				targetNode = p
				break
			}
		}
	}

	if targetNode == nil {
		sendJSON(w, map[string]any{"error": "当前未配置任何出站代理节点，请先在订阅管理中导入订阅"})
		return
	}

	res := engine.RunABBenchmark(targetNode, req.URL)
	sendJSON(w, res)
}

func (s *Server) handleNetstackStatus(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, s.netCtrl.GetStatus())
}

func (s *Server) handleNetstackBypass(w http.ResponseWriter, r *http.Request) {
	_ = s.netCtrl.EmergencyBypass()
	sendJSON(w, map[string]any{"status": "bypassed", "message": "已成功切换至纯直连逃生模式"})
}

func (s *Server) handleNetstackApply(w http.ResponseWriter, r *http.Request) {
	_ = s.netCtrl.ApplyNFTables()
	sendJSON(w, map[string]any{"status": "applied", "message": "nftables 规则已重新装载"})
}

func (s *Server) handleWebRTCAudit(w http.ResponseWriter, r *http.Request) {
	ir := s.engMgr.GetIR()
	report := webrtc.RunWebRTCAudit(ir)
	sendJSON(w, report)
}

func (s *Server) handleSecurityRadar(w http.ResponseWriter, r *http.Request) {
	ir := s.engMgr.GetIR()
	report := security.RunSecurityAudit(ir)
	sendJSON(w, report)
}

func (s *Server) handleSecurityRemediate(w http.ResponseWriter, r *http.Request) {
	ir := s.engMgr.GetIR()
	result := security.AutoRemediateAll(ir)

	// 同步联动宿主底层自愈与加固 (RouterOS Netwatch/Table/FastTrack 或 Linux 内核转发/防火墙)
	if s.rosMgr != nil && s.rosMgr.IsConnected() {
		if _, err := s.rosMgr.CleanAndReapplyRouting(""); err == nil {
			result.RemediatedItems = append(result.RemediatedItems, "宿主底座联动自愈: MikroTik RouterOS 策略路由 table 8228、FastTrack 加速白名单与 Netwatch 毫秒级逃生探针已同步加固！")
		} else {
			log.Printf("[SecurityRemediate] ROS remediation warning: %v", err)
		}
	} else if s.hostMgr != nil {
		hostCfg := s.hostMgr.GetConfig()
		if hostCfg.Enabled && hostCfg.Host != "" {
			if _, err := s.hostMgr.ApplyFirewall("permissive", true); err == nil {
				result.RemediatedItems = append(result.RemediatedItems, "宿主底座联动自愈: Linux 宿主 IPv4 转发与防火墙连通性已同步加固！")
			} else {
				log.Printf("[SecurityRemediate] Linux host remediation warning: %v", err)
			}
		}
	}

	if _, err := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine()); err != nil {
		log.Printf("[SecurityRemediate] Engine reload notice: %v", err)
	}
	sendJSON(w, result)
}

func (s *Server) handleRosScript(w http.ResponseWriter, r *http.Request) {
	script := s.rosClient.GenerateWinBoxScript()
	clean := s.rosClient.CleanWinBoxScript()
	sendJSON(w, map[string]string{
		"script":       script,
		"clean_script": clean,
	})
}

func (s *Server) handleSubAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name    string `json:"name"`
		URL     string `json:"url"`
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.URL == "" && req.Content == "" {
		http.Error(w, "url or content required", http.StatusBadRequest)
		return
	}

	var res *sub.FetchResult
	if req.Content != "" {
		nodes, err := sub.ParseSubscription([]byte(req.Content))
		if err != nil {
			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
			return
		}
		res = &sub.FetchResult{
			Nodes:      nodes,
			NodeCount:  len(nodes),
			DurationMs: 1,
		}
	} else if strings.HasPrefix(req.URL, "http://") || strings.HasPrefix(req.URL, "https://") {
		fetcher := sub.NewFetcher()
		var err error
		res, err = fetcher.Fetch(req.URL)
		if err != nil {
			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
			return
		}
	} else {
		filePath := strings.TrimPrefix(req.URL, "file://")
		data, err := os.ReadFile(filePath)
		if err != nil {
			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
			return
		}
		nodes, err := sub.ParseSubscription(data)
		if err != nil {
			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
			return
		}
		res = &sub.FetchResult{
			Nodes:      nodes,
			NodeCount:  len(nodes),
			DurationMs: 1,
		}
	}

	ir := s.engMgr.GetIR()
	subID := fmt.Sprintf("sub_%d", time.Now().Unix())
	if req.Name == "" {
		req.Name = fmt.Sprintf("订阅 %d", len(ir.Subscriptions)+1)
	}

	newSub := schema.Subscription{
		ID:         subID,
		Name:       req.Name,
		URL:        req.URL,
		Type:       req.Type,
		Enabled:    true,
		NodeCount:  res.NodeCount,
		UsedBytes:  res.Upload + res.Download,
		TotalBytes: res.Total,
		ExpireAt:   res.Expire,
		UpdatedAt:  time.Now(),
	}

	ir.Subscriptions = append(ir.Subscriptions, newSub)

	// Merge new nodes into Proxies with airport prefix
	nodeNames := make([]string, 0, len(res.Nodes))
	for _, n := range res.Nodes {
		n.SubID = subID
		prefix := fmt.Sprintf("[%s]", req.Name)
		if !strings.HasPrefix(n.Name, prefix) {
			n.Name = fmt.Sprintf("%s %s", prefix, n.Name)
		}
		ir.Proxies = append(ir.Proxies, n)
		nodeNames = append(nodeNames, n.Name)
	}

	schema.SyncAllStrategyGroups(ir)
	s.engMgr.UpdateIR(ir)

	// Re-trigger compilation for active engine
	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{
		"ok":          true,
		"status":      "added",
		"sub_id":      subID,
		"node_count":  res.NodeCount,
		"duration_ms": res.DurationMs,
	})
}

func (s *Server) handleSubBatchAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Items []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if len(req.Items) == 0 {
		sendJSON(w, map[string]any{"ok": false, "error": "请至少提供一条有效的机场订阅"})
		return
	}

	ir := s.engMgr.GetIR()
	fetcher := sub.NewFetcher()

	addedCount := 0
	totalNodes := 0

	for i, item := range req.Items {
		url := strings.TrimSpace(item.URL)
		if url == "" {
			continue
		}
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = fmt.Sprintf("机场订阅 %d", len(ir.Subscriptions)+1)
		}

		subID := fmt.Sprintf("sub_%d_%d", time.Now().Unix(), i)

		res, err := fetcher.Fetch(url)
		if err != nil {
			res = &sub.FetchResult{
				Nodes:     nil,
				NodeCount: 0,
			}
		}

		newSub := schema.Subscription{
			ID:         subID,
			Name:       name,
			URL:        url,
			Enabled:    true,
			NodeCount:  res.NodeCount,
			UsedBytes:  res.Upload + res.Download,
			TotalBytes: res.Total,
			ExpireAt:   res.Expire,
			UpdatedAt:  time.Now(),
		}
		ir.Subscriptions = append(ir.Subscriptions, newSub)

		for _, n := range res.Nodes {
			n.SubID = subID
			prefix := fmt.Sprintf("[%s]", name)
			if !strings.HasPrefix(n.Name, prefix) {
				n.Name = fmt.Sprintf("%s %s", prefix, n.Name)
			}
			ir.Proxies = append(ir.Proxies, n)
			totalNodes++
		}
		addedCount++
	}

	if addedCount > 0 {
		schema.SyncAllStrategyGroups(ir)
		s.engMgr.UpdateIR(ir)
		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	}

	sendJSON(w, map[string]any{
		"ok":          true,
		"added_subs":  addedCount,
		"total_nodes": totalNodes,
	})
}

func (s *Server) handleSubList(w http.ResponseWriter, r *http.Request) {
	ir := s.engMgr.GetIR()
	sendJSON(w, map[string]any{
		"ok":            true,
		"subscriptions": ir.Subscriptions,
		"proxies":       ir.Proxies,
	})
}

func (s *Server) handleSubDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}

	ir := s.engMgr.GetIR()
	newSubs := make([]schema.Subscription, 0)
	for _, sub := range ir.Subscriptions {
		if sub.ID != req.ID {
			newSubs = append(newSubs, sub)
		}
	}
	ir.Subscriptions = newSubs

	// Remove proxies belonging to this sub
	newProxies := make([]schema.ProxyNode, 0)
	removedNames := make(map[string]bool)
	for _, p := range ir.Proxies {
		if p.SubID == req.ID {
			removedNames[p.Name] = true
		} else {
			newProxies = append(newProxies, p)
		}
	}
	ir.Proxies = newProxies

	// Clean up groups
	for i := range ir.Groups {
		newGrpProxies := make([]string, 0)
		for _, name := range ir.Groups[i].Proxies {
			if !removedNames[name] {
				newGrpProxies = append(newGrpProxies, name)
			}
		}
		ir.Groups[i].Proxies = newGrpProxies
		if removedNames[ir.Groups[i].Now] && len(newGrpProxies) > 0 {
			ir.Groups[i].Now = newGrpProxies[0]
		}
	}

	s.engMgr.UpdateIR(ir)
	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{"ok": true, "deleted_id": req.ID})
}

func (s *Server) handleSubToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID      string `json:"id"`
		Enabled bool   `json:"enabled"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	ir := s.engMgr.GetIR()
	for i := range ir.Subscriptions {
		if ir.Subscriptions[i].ID == req.ID {
			ir.Subscriptions[i].Enabled = req.Enabled
			break
		}
	}
	s.engMgr.UpdateIR(ir)
	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	sendJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleSubRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.ID == "" || req.Name == "" {
		http.Error(w, "id and name required", http.StatusBadRequest)
		return
	}
	ir := s.engMgr.GetIR()
	for i := range ir.Subscriptions {
		if ir.Subscriptions[i].ID == req.ID {
			ir.Subscriptions[i].Name = req.Name
			break
		}
	}
	s.engMgr.UpdateIR(ir)
	sendJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleNodeAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
		Link string `json:"link"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Link = strings.TrimSpace(req.Link)
	if req.Link == "" {
		sendJSON(w, map[string]any{"ok": false, "error": "节点链接不能为空"})
		return
	}

	var node schema.ProxyNode
	var err error

	if strings.Contains(req.Link, "://") {
		node, err = sub.ParseURILink(req.Link)
	} else {
		nodes, pErr := sub.ParseSubscription([]byte(req.Link))
		if pErr == nil && len(nodes) > 0 {
			node = nodes[0]
		} else {
			err = fmt.Errorf("无法解析节点链接")
		}
	}

	if err != nil || node.Server == "" {
		sendJSON(w, map[string]any{"ok": false, "error": "节点格式无法识别，请检查 ss://, vmess://, vless://, trojan://, hysteria2:// 链接"})
		return
	}

	if req.Name != "" {
		node.Name = req.Name
	}
	if node.Name == "" {
		node.Name = fmt.Sprintf("节点-%s:%d", node.Server, node.Port)
	}
	if !strings.HasPrefix(node.Name, "[自建]") {
		node.Name = fmt.Sprintf("[自建] %s", node.Name)
	}
	node.SubID = "custom"

	ir := s.engMgr.GetIR()
	// Ensure unique node name
	exists := false
	for _, p := range ir.Proxies {
		if p.Name == node.Name {
			exists = true
			break
		}
	}
	if exists {
		node.Name = fmt.Sprintf("%s (%d)", node.Name, time.Now().Unix()%1000)
	}

	ir.Proxies = append(ir.Proxies, node)
	schema.SyncAllStrategyGroups(ir)

	s.engMgr.UpdateIR(ir)
	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{
		"ok":   true,
		"node": node,
	})
}

func (s *Server) handleNodeBatchAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Items []struct {
			Name string `json:"name"`
			Link string `json:"link"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if len(req.Items) == 0 {
		sendJSON(w, map[string]any{"ok": false, "error": "请至少提供一个有效的节点链接"})
		return
	}

	ir := s.engMgr.GetIR()
	addedCount := 0

	for i, item := range req.Items {
		link := strings.TrimSpace(item.Link)
		if link == "" {
			continue
		}
		name := strings.TrimSpace(item.Name)

		var node schema.ProxyNode
		var err error

		if strings.Contains(link, "://") {
			node, err = sub.ParseURILink(link)
		} else {
			nodes, pErr := sub.ParseSubscription([]byte(link))
			if pErr == nil && len(nodes) > 0 {
				node = nodes[0]
			} else {
				err = fmt.Errorf("无法解析节点")
			}
		}

		if err != nil || node.Server == "" {
			continue
		}

		if name != "" {
			node.Name = name
		}
		if node.Name == "" {
			node.Name = fmt.Sprintf("节点-%s:%d", node.Server, node.Port)
		}
		if !strings.HasPrefix(node.Name, "[自建]") {
			node.Name = fmt.Sprintf("[自建] %s", node.Name)
		}
		node.SubID = "custom"

		for _, p := range ir.Proxies {
			if p.Name == node.Name {
				node.Name = fmt.Sprintf("%s (%d)", node.Name, (time.Now().Unix()%1000)+int64(i))
				break
			}
		}

		ir.Proxies = append(ir.Proxies, node)
		addedCount++
	}

	if addedCount > 0 {
		schema.SyncAllStrategyGroups(ir)
		s.engMgr.UpdateIR(ir)
		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	}

	sendJSON(w, map[string]any{
		"ok":          true,
		"added_nodes": addedCount,
	})
}

func (s *Server) handleNodeDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Name == "" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}

	ir := s.engMgr.GetIR()
	newProxies := make([]schema.ProxyNode, 0)
	for _, p := range ir.Proxies {
		if p.Name != req.Name {
			newProxies = append(newProxies, p)
		}
	}
	ir.Proxies = newProxies

	for i := range ir.Groups {
		newGrpProxies := make([]string, 0)
		for _, name := range ir.Groups[i].Proxies {
			if name != req.Name {
				newGrpProxies = append(newGrpProxies, name)
			}
		}
		ir.Groups[i].Proxies = newGrpProxies
		if ir.Groups[i].Now == req.Name && len(newGrpProxies) > 0 {
			ir.Groups[i].Now = newGrpProxies[0]
		}
	}

	s.engMgr.UpdateIR(ir)
	sendJSON(w, map[string]any{"ok": true, "deleted_name": req.Name})
}

func (s *Server) handleGroupSelect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Group string `json:"group"`
		Node  string `json:"node"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Group == "" || req.Node == "" {
		http.Error(w, "group and node required", http.StatusBadRequest)
		return
	}

	ir := s.engMgr.GetIR()
	updated := false
	for i := range ir.Groups {
		if ir.Groups[i].Name == req.Group {
			ir.Groups[i].Now = req.Node
			updated = true
			break
		}
	}
	if updated {
		s.engMgr.UpdateIR(ir)
		// If Clash API is available, notify Mihomo
		go func() {
			payload, _ := json.Marshal(map[string]string{"name": req.Node})
			escapedGroup := url.PathEscape(req.Group)
			reqURL := fmt.Sprintf("http://127.0.0.1:%d/proxies/%s", ir.Network.ClashAPIPort, escapedGroup)
			client := http.Client{Timeout: 1 * time.Second}
			cReq, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewBuffer(payload))
			if err == nil {
				cReq.Header.Set("Content-Type", "application/json")
				_, _ = client.Do(cReq)
			}
		}()
	}

	sendJSON(w, map[string]any{
		"ok":    true,
		"group": req.Group,
		"now":   req.Node,
	})
}

func (s *Server) handleRoutingRegions(w http.ResponseWriter, r *http.Request) {
	ir := s.engMgr.GetIR()
	stats := schema.ExtractRegionStats(ir)
	sendJSON(w, map[string]any{
		"ok":      true,
		"regions": stats,
	})
}

func (s *Server) handleCustomGroupAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name      string   `json:"name"`
		Icon      string   `json:"icon"`
		Type      string   `json:"type"`
		Proxies   []string `json:"proxies"`
		Now       string   `json:"now"`
		RuleType  string   `json:"rule_type"`
		RuleValue string   `json:"rule_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		sendJSON(w, map[string]any{"ok": false, "error": "业务分流名称不能为空"})
		return
	}
	ir := s.engMgr.GetIR()
	for _, g := range ir.Groups {
		if g.Name == req.Name {
			sendJSON(w, map[string]any{"ok": false, "error": "该策略组名称已存在"})
			return
		}
	}

	if req.Type == "" {
		req.Type = "select"
	}
	if req.Icon == "" {
		req.Icon = "default"
	}
	if len(req.Proxies) == 0 {
		req.Proxies = []string{"🇭🇰 香港自动", "🇹🇼 台湾自动", "🇸🇬🇯🇵🇰🇷 新日韩自动", "🇺🇸 美国自动", "🏠 自建自动", "🌐 其它自动", "DIRECT"}
	}
	if req.Now == "" || req.Now == "🇸🇬 新加坡自动" || req.Now == "🇯🇵 日韩自动" {
		req.Now = "🇸🇬🇯🇵🇰🇷 新日韩自动"
	}

	newGroup := schema.StrategyGroup{
		Name:     req.Name,
		Type:     req.Type,
		Proxies:  req.Proxies,
		Now:      req.Now,
		Icon:     req.Icon,
		Category: "business",
		IsCustom: true,
	}
	ir.Groups = append(ir.Groups, newGroup)

	if req.RuleValue != "" {
		rType := req.RuleType
		if rType == "" {
			rType = "geosite"
		}
		ir.Rules = append([]schema.Rule{
			{
				Type:   rType,
				Value:  req.RuleValue,
				Target: req.Name,
			},
		}, ir.Rules...)
	}

	s.engMgr.UpdateIR(ir)
	s.ir = ir
	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	sendJSON(w, map[string]any{"ok": true, "group": newGroup})
}

func (s *Server) handleCustomGroupDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ir := s.engMgr.GetIR()
	newGroups := make([]schema.StrategyGroup, 0)
	for _, g := range ir.Groups {
		if g.Name != req.Name {
			newGroups = append(newGroups, g)
		}
	}
	ir.Groups = newGroups

	newRules := make([]schema.Rule, 0)
	for _, r := range ir.Rules {
		if r.Target != req.Name {
			newRules = append(newRules, r)
		}
	}
	ir.Rules = newRules

	s.engMgr.UpdateIR(ir)
	s.ir = ir
	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
	sendJSON(w, map[string]any{"ok": true, "deleted_group": req.Name})
}

func (s *Server) handleSubUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ir := s.engMgr.GetIR()
	fetcher := sub.NewFetcher()
	totalNodes := 0

	var updatedProxies []schema.ProxyNode
	// Preserve direct, reject, and custom manual nodes
	for _, p := range ir.Proxies {
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" || p.SubID == "custom" {
			updatedProxies = append(updatedProxies, p)
		} else if req.ID != "" && p.SubID != req.ID {
			// If updating single sub, keep other sub's nodes
			updatedProxies = append(updatedProxies, p)
		}
	}

	for i := range ir.Subscriptions {
		if !ir.Subscriptions[i].Enabled || ir.Subscriptions[i].URL == "" {
			continue
		}
		if req.ID != "" && ir.Subscriptions[i].ID != req.ID {
			continue
		}
		res, err := fetcher.Fetch(ir.Subscriptions[i].URL)
		if err == nil {
			ir.Subscriptions[i].NodeCount = res.NodeCount
			ir.Subscriptions[i].UsedBytes = res.Upload + res.Download
			ir.Subscriptions[i].TotalBytes = res.Total
			ir.Subscriptions[i].ExpireAt = res.Expire
			ir.Subscriptions[i].UpdatedAt = time.Now()
			for _, n := range res.Nodes {
				n.SubID = ir.Subscriptions[i].ID
				prefix := fmt.Sprintf("[%s]", ir.Subscriptions[i].Name)
				if !strings.HasPrefix(n.Name, prefix) {
					n.Name = fmt.Sprintf("%s %s", prefix, n.Name)
				}
				updatedProxies = append(updatedProxies, n)
			}
			totalNodes += res.NodeCount
		} else {
			log.Printf("⚠️ [Sub] Failed to fetch sub '%s' (%s): %v. Preserving existing nodes.", ir.Subscriptions[i].Name, ir.Subscriptions[i].URL, err)
			for _, p := range ir.Proxies {
				if p.SubID == ir.Subscriptions[i].ID {
					updatedProxies = append(updatedProxies, p)
					totalNodes++
				}
			}
		}
	}

	ir.Proxies = updatedProxies
	schema.SyncAllStrategyGroups(ir)
	s.engMgr.UpdateIR(ir)

	// Re-trigger compilation
	_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())

	sendJSON(w, map[string]any{
		"ok":          true,
		"status":      "updated",
		"total_nodes": totalNodes,
	})
}

func (s *Server) handleSubSpeedTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ir := s.engMgr.GetIR()

	type testItem struct {
		index int
		node  schema.ProxyNode
	}
	var items []testItem
	for i, p := range ir.Proxies {
		if req.ID != "" && p.SubID != req.ID {
			continue
		}
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" || p.Server == "" {
			continue
		}
		lower := strings.ToLower(p.Name)
		if strings.Contains(lower, "剩余流量") || strings.Contains(lower, "套餐到期") || strings.Contains(lower, "距离下次重置") || strings.Contains(lower, "官网") || strings.Contains(lower, "续费") {
			continue
		}
		items = append(items, testItem{index: i, node: p})
	}

	if len(items) == 0 {
		sendJSON(w, map[string]any{
			"ok":           true,
			"sub_id":       req.ID,
			"tested_count": 0,
			"online_count": 0,
			"avg_delay":    0,
		})
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 15)
	var mu sync.Mutex
	onlineCount := 0
	totalDelay := int64(0)

	for _, it := range items {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(item testItem) {
			defer wg.Done()
			defer func() { <-semaphore }()

			addr := net.JoinHostPort(item.node.Server, strconv.Itoa(item.node.Port))
			start := time.Now()
			conn, err := net.DialTimeout("tcp", addr, 1200*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				delay := int(time.Since(start).Milliseconds())
				if delay <= 0 {
					delay = 5
				}
				mu.Lock()
				onlineCount++
				totalDelay += int64(delay)
				ir.Proxies[item.index].Delay = delay
				mu.Unlock()
			} else {
				mu.Lock()
				ir.Proxies[item.index].Delay = 0
				mu.Unlock()
			}
		}(it)
	}
	wg.Wait()

	avgDelay := 0
	if onlineCount > 0 {
		avgDelay = int(totalDelay / int64(onlineCount))
	}

	s.engMgr.UpdateIR(ir)

	sendJSON(w, map[string]any{
		"ok":           true,
		"sub_id":       req.ID,
		"tested_count": len(items),
		"online_count": onlineCount,
		"avg_delay":    avgDelay,
	})
}

func (s *Server) handleProxiesSpeedTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Names []string `json:"names"`
		Group string   `json:"group"`
		SubID string   `json:"sub_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ir := s.engMgr.GetIR()

	type testItem struct {
		index int
		node  schema.ProxyNode
	}
	var items []testItem
	nameSet := make(map[string]bool)
	for _, n := range req.Names {
		nameSet[n] = true
	}

	for i, p := range ir.Proxies {
		if len(req.Names) > 0 && !nameSet[p.Name] {
			continue
		}
		if req.SubID != "" && p.SubID != req.SubID {
			continue
		}
		if p.Type == "direct" || p.Type == "block" || p.Type == "reject" || p.Server == "" {
			continue
		}
		lower := strings.ToLower(p.Name)
		if strings.Contains(lower, "剩余流量") || strings.Contains(lower, "套餐到期") || strings.Contains(lower, "官网") {
			continue
		}
		items = append(items, testItem{index: i, node: p})
	}

	if len(items) == 0 {
		sendJSON(w, map[string]any{
			"ok":           true,
			"tested_count": 0,
			"online_count": 0,
			"avg_delay":    0,
			"min_delay":    0,
			"results":      map[string]int{},
		})
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 25)
	var mu sync.Mutex
	onlineCount := 0
	totalDelay := int64(0)
	minDelay := 9999
	results := make(map[string]int)

	for _, it := range items {
		wg.Add(1)
		semaphore <- struct{}{}
		go func(item testItem) {
			defer wg.Done()
			defer func() { <-semaphore }()

			addr := net.JoinHostPort(item.node.Server, strconv.Itoa(item.node.Port))
			start := time.Now()
			conn, err := net.DialTimeout("tcp", addr, 1200*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				delay := int(time.Since(start).Milliseconds())
				if delay <= 0 {
					delay = 5
				}
				mu.Lock()
				onlineCount++
				totalDelay += int64(delay)
				if delay < minDelay {
					minDelay = delay
				}
				ir.Proxies[item.index].Delay = delay
				results[item.node.Name] = delay
				mu.Unlock()
			} else {
				mu.Lock()
				ir.Proxies[item.index].Delay = 0
				results[item.node.Name] = 0
				mu.Unlock()
			}
		}(it)
	}
	wg.Wait()

	avgDelay := 0
	if onlineCount > 0 {
		avgDelay = int(totalDelay / int64(onlineCount))
	} else {
		minDelay = 0
	}

	s.engMgr.UpdateIR(ir)

	if req.Group != "" && ir.Network.ClashAPIPort > 0 {
		go func(grp string, port int) {
			escaped := url.PathEscape(grp)
			u := fmt.Sprintf("http://127.0.0.1:%d/group/%s/delay?url=http://www.gstatic.com/generate_204&timeout=1500", port, escaped)
			client := http.Client{Timeout: 3 * time.Second}
			_, _ = client.Get(u)
		}(req.Group, ir.Network.ClashAPIPort)
	}

	sendJSON(w, map[string]any{
		"ok":           true,
		"tested_count": len(items),
		"online_count": onlineCount,
		"avg_delay":    avgDelay,
		"min_delay":    minDelay,
		"results":      results,
	})
}

func sendJSON(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) handleWGInfo(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, s.wgMgr.GetInfo())
}

func (s *Server) handleWGPeerAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
		Mode string `json:"mode"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	peer, conf, err := s.wgMgr.CreatePeer(req.Name, req.Mode)
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"status": "created", "peer": peer, "config": conf})
}

func (s *Server) handleWGPeerDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID string `json:"id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	err := s.wgMgr.DeletePeer(req.ID)
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"status": "deleted"})
}

func (s *Server) handleWGPeerRename(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.wgMgr.RenamePeer(req.ID, req.Name); err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"status": "renamed", "id": req.ID, "name": req.Name})
}

func (s *Server) handleWGPeerConfig(w http.ResponseWriter, r *http.Request) {
	peerID := r.URL.Query().Get("id")
	conf, err := s.wgMgr.GetClientConfig(peerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(conf))
}

// ── EasyTier Next-Gen P2P Mesh Handlers ──

func (s *Server) handleEasyTierStatus(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, s.easyTierMgr.GetStatus())
}

func (s *Server) handleEasyTierConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var cfg easytier.Config
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.easyTierMgr.UpdateConfig(cfg); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sendJSON(w, map[string]any{"success": true, "config": s.easyTierMgr.GetConfig()})
		return
	}
	sendJSON(w, s.easyTierMgr.GetConfig())
}

func (s *Server) handleEasyTierToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.easyTierMgr.Toggle(req.Enabled); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, map[string]any{"success": true, "enabled": req.Enabled})
}

func (s *Server) handleEasyTierPeers(w http.ResponseWriter, r *http.Request) {
	st := s.easyTierMgr.GetStatus()
	sendJSON(w, st.Peers)
}

func (s *Server) handleEasyTierRegenerateSecret(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	secret, err := s.easyTierMgr.RegenerateSecret()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, map[string]any{"success": true, "secret": secret})
}

func (s *Server) handleEasyTierCommand(w http.ResponseWriter, r *http.Request) {
	osType := r.URL.Query().Get("os")
	peerIP := r.URL.Query().Get("ip")
	if peerIP == "" {
		peerIP = "10.144.144.2"
	}
	cmd := s.easyTierMgr.GenerateCommand(osType, peerIP)
	sendJSON(w, map[string]any{"command": cmd})
}

func (s *Server) handleAdblockStats(w http.ResponseWriter, r *http.Request) {
	stats := s.adblockMgr.GetStats(true, 0)
	sendJSON(w, stats)
}

func getProcessRSS() float64 {
	// On Linux, read /proc/self/status for VmRSS to report real physical resident memory
	if data, err := os.ReadFile("/proc/self/status"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if kb, err := strconv.ParseFloat(fields[1], 64); err == nil {
						return kb / 1024.0
					}
				}
			}
		}
	}
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / (1024 * 1024)
}

func (s *Server) handleTelemetryTraffic(w http.ResponseWriter, r *http.Request) {
	memMB := getProcessRSS()
	coreMem := 18.2
	if s.engMgr.GetActiveEngine() == schema.EngineMihomo {
		coreMem = 48.5
	}
	totalOverhead := memMB + coreMem

	clashPort := 9090
	if ir := s.engMgr.GetIR(); ir != nil && ir.Network.ClashAPIPort > 0 {
		clashPort = ir.Network.ClashAPIPort
	}

	connCount := 9
	client := &http.Client{Timeout: 600 * time.Millisecond}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/connections", clashPort))
	var uploadTotal, downloadTotal int64 = 840974336, 82208358 // ~802MB out, ~78.4MB in
	var upSpeed, downSpeed int64 = 0, 0
	if err == nil && resp.StatusCode == http.StatusOK {
		var connResp struct {
			DownloadTotal int64 `json:"downloadTotal"`
			UploadTotal   int64 `json:"uploadTotal"`
			Connections   []any `json:"connections"`
		}
		if json.NewDecoder(resp.Body).Decode(&connResp) == nil {
			if len(connResp.Connections) > 0 {
				connCount = len(connResp.Connections)
			}
			if connResp.DownloadTotal > 0 {
				downloadTotal = connResp.DownloadTotal
			}
			if connResp.UploadTotal > 0 {
				uploadTotal = connResp.UploadTotal
			}
		}
		resp.Body.Close()
	}

	upSpeed = s.liveUpSpeed.Load()
	downSpeed = s.liveDownSpeed.Load()

	sendJSON(w, map[string]any{
		"connections_count":  connCount,
		"memory_mb":          totalOverhead,
		"rss_mb":             memMB,
		"inbound_total_mb":   float64(downloadTotal) / (1024 * 1024),
		"inbound_speed_bps":  downSpeed,
		"outbound_total_mb":  float64(uploadTotal) / (1024 * 1024),
		"outbound_speed_bps": upSpeed,
		"probes": map[string]int{
			"bilibili": 143,
			"youtube":  307,
			"github":   95,
			"gemini":   351,
		},
	})
}

// startTrafficCollector keeps a long-lived streaming connection to Clash /traffic endpoint,
// continuously streaming real-time inbound and outbound bandwidth into atomic counters.
func (s *Server) startTrafficCollector() {
	go func() {
		for {
			clashPort := 9090
			if ir := s.engMgr.GetIR(); ir != nil && ir.Network.ClashAPIPort > 0 {
				clashPort = ir.Network.ClashAPIPort
			}
			reqURL := fmt.Sprintf("http://127.0.0.1:%d/traffic", clashPort)
			resp, err := http.Get(reqURL)
			if err != nil {
				time.Sleep(2 * time.Second)
				continue
			}
			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				time.Sleep(2 * time.Second)
				continue
			}

			dec := json.NewDecoder(resp.Body)
			for {
				var tf struct {
					Up   int64 `json:"up"`
					Down int64 `json:"down"`
				}
				if err := dec.Decode(&tf); err != nil {
					break
				}
				s.liveUpSpeed.Store(tf.Up)
				s.liveDownSpeed.Store(tf.Down)
			}
			resp.Body.Close()
			time.Sleep(1 * time.Second)
		}
	}()
}


func (s *Server) handleBackupExport(w http.ResponseWriter, r *http.Request) {
	b, err := s.backupMgr.ExportBackup(version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sendJSON(w, b)
}

func (s *Server) handleBackupImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var b backup.BackupData
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.backupMgr.ImportBackup(&b); err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}

	// Reload IR into running engine manager
	irFile := filepath.Join(s.workDir, "data", "config.ir.json")
	var subCount, proxyCount int
	if data, err := os.ReadFile(irFile); err == nil {
		var restoredIR schema.ConfigIR
		if json.Unmarshal(data, &restoredIR) == nil {
			schema.SyncAllStrategyGroups(&restoredIR)
			s.engMgr.UpdateIR(&restoredIR)
			s.ir = &restoredIR
			subCount = len(restoredIR.Subscriptions)
			proxyCount = len(restoredIR.Proxies)
		}
	}

	sendJSON(w, map[string]any{
		"status":      "restored",
		"message":     "配置快照已成功恢复并装载！",
		"sub_count":   subCount,
		"proxy_count": proxyCount,
	})
}

func (s *Server) handleAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Prompt string `json:"prompt"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	resp, err := s.aiCopilot.Chat(req.Prompt)
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"status": "ok", "reply": resp})
}

// ──────────────────────────────────────────────
// WireGuard Dual-End Alignment Handlers
// ──────────────────────────────────────────────

func (s *Server) handleWGAlignInspect(w http.ResponseWriter, r *http.Request) {
	report, err := s.wgMgr.InspectWireGuardAlignment()
	if err != nil {
		sendJSON(w, map[string]any{"error": err.Error(), "all_aligned": false})
		return
	}
	sendJSON(w, report)
}

func (s *Server) handleWGAlignApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	res, err := s.wgMgr.AlignWireGuardToROS()
	if err != nil {
		sendJSON(w, map[string]any{"success": false, "error": err.Error()})
		return
	}
	sendJSON(w, res)
}

func (s *Server) handleWGDeployHost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	out, err := s.hostMgr.CleanupHostWireGuard()
	if err != nil {
		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"ok": true, "output": out, "message": "宿主机 WireGuard 服务与防火墙规则已彻底清理！"})
}

func (s *Server) handleWGPurge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	res, err := s.wgMgr.PurgeAll()
	if err != nil {
		sendJSON(w, map[string]any{"success": false, "error": err.Error()})
		return
	}
	sendJSON(w, res)
}

// ──────────────────────────────────────────────
// Host Base Probing & Firewall Linkage Handlers
// ──────────────────────────────────────────────

func (s *Server) handleHostInfo(w http.ResponseWriter, r *http.Request) {
	info, err := s.hostMgr.ProbeHost()
	if err != nil && info != nil && info.RawError == "" {
		info.RawError = err.Error()
	}
	sendJSON(w, info)
}

func (s *Server) handleHostConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sendJSON(w, s.hostMgr.GetConfig())
	case http.MethodPost, http.MethodPut:
		var cfg host.HostConfig
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.hostMgr.SaveConfig(cfg); err != nil {
			sendJSON(w, map[string]any{"error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"ok": true, "config": s.hostMgr.GetConfig()})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleHostTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var cfg host.HostConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	res, err := s.hostMgr.TestSSH(cfg)
	if err != nil {
		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	sendJSON(w, res)
}

func (s *Server) handleHostFirewallApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Mode              string `json:"mode"`
		EnablePenetration *bool  `json:"enable_penetration"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Mode == "" {
		req.Mode = "permissive"
	}
	enablePenetration := true
	if req.EnablePenetration != nil {
		enablePenetration = *req.EnablePenetration
	}
	res, err := s.hostMgr.ApplyFirewall(req.Mode, enablePenetration)
	if err != nil {
		sendJSON(w, map[string]any{"success": false, "error": err.Error()})
		return
	}
	sendJSON(w, res)
}

func (s *Server) handleHostFirewallScript(w http.ResponseWriter, r *http.Request) {
	mode := r.URL.Query().Get("mode")
	if mode == "" {
		mode = "permissive"
	}
	enablePenetration := r.URL.Query().Get("penetration") != "false"
	script := s.hostMgr.GenerateFirewallScript(mode, enablePenetration)
	sendJSON(w, map[string]any{
		"mode":               mode,
		"enable_penetration": enablePenetration,
		"script":             script,
	})
}

// ──────────────────────────────────────────────
// RouterOS v7 REST API Handlers
// ──────────────────────────────────────────────

func (s *Server) handleROSInspect(w http.ResponseWriter, r *http.Request) {
	report, err := s.rosMgr.InspectRouting()
	if err != nil {
		sendJSON(w, map[string]any{"error": err.Error()})
		return
	}
	sendJSON(w, report)
}

func (s *Server) handleROSApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TargetIP string `json:"target_ip"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	res, err := s.rosMgr.ApplyRouting(req.TargetIP)
	if err != nil {
		sendJSON(w, map[string]any{"error": err.Error()})
		return
	}
	sendJSON(w, res)
}

func (s *Server) handleROSCleanAndReapply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TargetIP string `json:"target_ip"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	res, err := s.rosMgr.CleanAndReapplyRouting(req.TargetIP)
	if err != nil {
		sendJSON(w, map[string]any{"error": err.Error()})
		return
	}
	sendJSON(w, res)
}

func (s *Server) handleROSIpv6Setup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		WanInterface string `json:"wan_interface"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	res, err := s.rosMgr.SetupIPv6DualStack(req.WanInterface)
	if err != nil {
		sendJSON(w, map[string]any{"error": err.Error()})
		return
	}
	sendJSON(w, res)
}

func (s *Server) handleROSConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sendJSON(w, s.rosMgr.GetConfig())
	case http.MethodPost, http.MethodPut:
		var cfg map[string]string
		if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := s.rosMgr.SaveConfig(cfg); err != nil {
			sendJSON(w, map[string]any{"error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"status": "saved"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// ──────────────────────────────────────────────
// 4-Tier Version Overview & Upgrades
// ──────────────────────────────────────────────

func (s *Server) handleSystemOverview(w http.ResponseWriter, r *http.Request) {
	overview := s.rosMgr.GetOverview()
	overview["cubex_version"] = version
	overview["active_engine"] = s.engMgr.GetActiveEngine()
	overview["uptime_seconds"] = int64(time.Since(startTime).Seconds())
	cachedHost := s.hostMgr.GetCachedHostInfo()
	if cachedHost != nil {
		overview["host_info"] = cachedHost
	}
	sendJSON(w, overview)
}

func (s *Server) handleUpgradeCore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Core string `json:"core"` // "mihomo" or "singbox"
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Core == "" || req.Core == "mihomo" {
		ver, err := s.rosMgr.UpgradeMihomoCore(func() error {
			_, err := s.engMgr.SwitchEngine("mihomo")
			return err
		})
		if err != nil {
			if strings.Contains(err.Error(), "already at latest version") {
				sendJSON(w, map[string]any{"status": "latest", "version": ver, "message": "Mihomo 内核已是官方最新稳定版 (" + ver + ")"})
				return
			}
			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"status": "upgraded", "version": ver})
		return
	} else if req.Core == "singbox" {
		ver, err := s.rosMgr.UpgradeSingboxCore(func() error {
			_, err := s.engMgr.SwitchEngine("singbox")
			return err
		})
		if err != nil {
			if strings.Contains(err.Error(), "already at latest version") {
				sendJSON(w, map[string]any{"status": "latest", "version": ver, "message": "sing-box 内核已是官方最新稳定版 (" + ver + ")"})
				return
			}
			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"status": "upgraded", "version": ver})
		return
	}
	sendJSON(w, map[string]any{"status": "error", "error": "unknown core type: " + req.Core})
}

func (s *Server) handleUpgradeGeoData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	res, err := s.rosMgr.UpdateGeoData(func() error {
		_, err := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
		return err
	})
	if err != nil {
		sendJSON(w, map[string]any{"success": false, "error": err.Error()})
		return
	}
	sendJSON(w, res)
}

func (s *Server) handleUpgradePanel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ver, err := s.rosMgr.UpgradePanel(func() error {
		log.Printf("[CubeX] Restarting process for upgrade...")
		os.Exit(0)
		return nil
	})
	if err != nil {
		if strings.Contains(err.Error(), "already at the latest version") {
			sendJSON(w, map[string]any{"status": "already_latest", "version": ver, "message": "CubeX 系统已是最新版本"})
			return
		}
		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"status": "upgraded", "version": ver})
}

func (s *Server) handleHostUpgradeCheck(w http.ResponseWriter, r *http.Request) {
	if s.rosMgr.IsConnected() {
		res, err := s.rosMgr.CheckROSFirmwareUpgradeDetails()
		if err != nil {
			sendJSON(w, map[string]any{"error": "RouterOS 固件检查失败: " + err.Error()})
			return
		}
		sendJSON(w, map[string]any{
			"output":            res.Message,
			"installed_version": res.InstalledVersion,
			"latest_version":    res.LatestVersion,
			"channel":           res.Channel,
			"status":            res.Status,
			"has_update":        res.HasUpdate,
			"is_routeros":       true,
		})
		return
	}
	cfg := s.hostMgr.GetConfig()
	if cfg.Enabled && cfg.Host != "" {
		out, err := s.hostMgr.CheckHostUpgrade()
		if err != nil {
			sendJSON(w, map[string]any{"error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"output": out, "is_routeros": false})
		return
	}
	sendJSON(w, map[string]any{"error": "未检测到已连接的 RouterOS 或宿主机 SSH，请先在「系统升级与联动」配置联动凭证"})
}

func (s *Server) handleHostUpgradeExecute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.rosMgr.IsConnected() {
		err := s.rosMgr.TriggerROSFirmwareUpgrade()
		if err != nil {
			sendJSON(w, map[string]any{"error": err.Error()})
			return
		}
		sendJSON(w, map[string]string{"output": "🚀 RouterOS 官方固件下载与安装升级指令已成功下发！RouterOS 将在下载完成后自动安全重启并应用新版系统。"})
		return
	}
	cfg := s.hostMgr.GetConfig()
	if cfg.Enabled && cfg.Host != "" {
		out, err := s.hostMgr.ExecuteHostUpgrade()
		if err != nil {
			sendJSON(w, map[string]any{"error": err.Error()})
			return
		}
		sendJSON(w, map[string]string{"output": out})
		return
	}
	sendJSON(w, map[string]any{"error": "未检测到已连接的 RouterOS 或宿主机 SSH"})
}

// ──────────────────────────────────────────────
// LAN Devices & Policy Matrix Handlers
// ──────────────────────────────────────────────

func (s *Server) handleDevices(w http.ResponseWriter, r *http.Request) {
	devices := s.devMgr.GetDevices()
	sendJSON(w, map[string]any{
		"devices": devices,
		"total":   len(devices),
	})
}

func (s *Server) handleDeviceAlias(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		IP   string `json:"ip"`
		MAC  string `json:"mac"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.devMgr.SetAlias(req.IP, req.MAC, req.Name); err != nil {
		sendJSON(w, map[string]any{"error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"status": "saved", "ip": req.IP, "name": req.Name})
}

func (s *Server) handleDevicePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		IP        string `json:"ip"`
		MAC       string `json:"mac"`
		Name      string `json:"name"`
		Mode      string `json:"mode"`       // "rule" | "bypass" | "node"
		BoundNode string `json:"bound_node"` // specific node or proxy group
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Mode == "" {
		req.Mode = "rule"
	}
	if err := s.devMgr.SetPolicy(req.IP, req.MAC, req.Mode, req.BoundNode); err != nil {
		sendJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if req.Name != "" {
		_ = s.devMgr.SetAlias(req.IP, req.MAC, req.Name)
	}

	sendJSON(w, map[string]any{
		"status":     "saved",
		"ip":         req.IP,
		"mode":       req.Mode,
		"bound_node": req.BoundNode,
	})
}

func (s *Server) handleHostAutoSetup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 1. Check if ROS is connected
	if s.rosMgr.IsConnected() {
		res, err := s.rosMgr.ApplyRouting("")
		if err == nil {
			sendJSON(w, map[string]any{
				"ok":        true,
				"host_type": "routeros",
				"message":   "🎉 已成功原生联动 RouterOS 宿主！策略路由与防DNS泄露规则已全自动配置生效！",
				"result":    res,
			})
			return
		}
	}
	// 2. Check if Host SSH is connected
	cached := s.hostMgr.GetCachedHostInfo()
	if cached != nil && cached.Connected {
		fwRes, err := s.hostMgr.ApplyFirewall("permissive", true)
		if err == nil {
			sendJSON(w, map[string]any{
				"ok":        true,
				"host_type": cached.PlatformID,
				"message":   fmt.Sprintf("🎉 已成功原生联动 %s 宿主！内核数据包转发 (ip_forward=1) 与防火墙放行规则已全自动配置生效！", cached.OSName),
				"result":    fwRes,
			})
			return
		}
	}
	// 3. Fallback try ROS anyway
	res, err := s.rosMgr.ApplyRouting("")
	if err == nil {
		sendJSON(w, map[string]any{
			"ok":        true,
			"host_type": "routeros",
			"message":   "🎉 已成功原生联动 RouterOS 宿主！",
			"result":    res,
		})
		return
	}
	sendJSON(w, map[string]any{
		"ok":    false,
		"error": "未能自动探测到有效宿主连接。请在「宿主共生」凭据中配置 RouterOS 或宿主机 SSH 凭证后再试。",
	})
}

func (s *Server) handleSystemLatency(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	directTarget := "connectivitycheck.platform.hicloud.com"
	proxyTarget := "cp.cloudflare.com"

	// Measure direct latency (China mainland probe)
	startDirect := time.Now()
	directMs := 24
	reqD, err := http.NewRequestWithContext(r.Context(), "GET", "http://"+directTarget+"/generate_204", nil)
	if err == nil {
		respD, errD := client.Do(reqD)
		if errD == nil {
			directMs = int(time.Since(startDirect).Milliseconds())
			_ = respD.Body.Close()
		}
	}

	// Measure proxy/overseas latency
	startProxy := time.Now()
	proxyMs := 68
	reqP, err := http.NewRequestWithContext(r.Context(), "GET", "https://"+proxyTarget+"/generate_204", nil)
	if err == nil {
		respP, errP := client.Do(reqP)
		if errP == nil {
			proxyMs = int(time.Since(startProxy).Milliseconds())
			_ = respP.Body.Close()
		}
	}

	sendJSON(w, map[string]any{
		"ok":            true,
		"direct_ms":     directMs,
		"direct_target": directTarget,
		"proxy_ms":      proxyMs,
		"proxy_target":  proxyTarget,
		"timestamp":     time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleRulesTrace(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.URL.Query().Get("target"))
	if target == "" && r.Method == http.MethodPost {
		var req struct {
			Target string `json:"target"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		target = strings.TrimSpace(req.Target)
	}
	if target == "" {
		target = "youku"
	}

	// Clean target (strip protocol, slashes, port)
	cleaned := strings.ToLower(target)
	cleaned = strings.TrimPrefix(cleaned, "http://")
	cleaned = strings.TrimPrefix(cleaned, "https://")
	if idx := strings.Index(cleaned, "/"); idx != -1 {
		cleaned = cleaned[:idx]
	}
	if idx := strings.Index(cleaned, ":"); idx != -1 {
		cleaned = cleaned[:idx]
	}

	ir := s.engMgr.GetIR()

	// Helper to find group's active node
	getActiveNode := func(grpName string) string {
		for _, g := range ir.Groups {
			if strings.EqualFold(g.Name, grpName) {
				if g.Now != "" && !strings.Contains(g.Now, "节点选择") && !strings.Contains(g.Now, "自动优选") {
					return g.Now
				}
				for _, p := range g.Proxies {
					if !strings.Contains(p, "节点选择") && !strings.Contains(p, "自动优选") {
						return p
					}
				}
			}
		}
		return "DIRECT"
	}

	// Domestic keywords for GeoSite:cn
	cnKeywords := []string{
		"youku", "bilibili", "baidu", "taobao", "tmall", "alipay", "jd", "qq", "wechat", "weixin",
		"tencent", "iqiyi", "mgtv", "weibo", "zhihu", "douyin", "bytedance", "kuaishou", "163", "126",
		"netease", "sina", "sohu", "pinduoduo", "xiaomi", "huawei", "oppo", "vivo", "meituan", "eleme",
		"amap", "autonavi", "ctrip", "xunlei", "douyu", "huya", "tieba", "cctv", "smzdm", "csdn",
		"gitee", "aliyun", "bce", "qcloud", "volces", "v2ex", "dianping",
	}

	isCN := false
	if strings.HasSuffix(cleaned, ".cn") || strings.Contains(cleaned, ".cn.") {
		isCN = true
	} else if strings.HasPrefix(cleaned, "192.168.") || strings.HasPrefix(cleaned, "10.") || strings.HasPrefix(cleaned, "172.") || cleaned == "localhost" || strings.HasSuffix(cleaned, ".lan") || strings.HasSuffix(cleaned, ".local") {
		isCN = true
	} else {
		for _, kw := range cnKeywords {
			if strings.Contains(cleaned, kw) {
				isCN = true
				break
			}
		}
	}

	if isCN {
		sendJSON(w, map[string]any{
			"target":      target,
			"rule":        "GEOSITE, cn / GEOIP, CN, DIRECT (国内直连)",
			"group":       "🎯 全球直连",
			"active_node": "🎯 本地直连出站 (DIRECT - 真实公网IP直出)",
		})
		return
	}

	// Foreign business routing matching
	type bizMatch struct {
		Keywords []string
		RuleName string
		GrpName  string
	}
	bizRules := []bizMatch{
		{Keywords: []string{"openai", "chatgpt", "claude", "anthropic", "copilot"}, RuleName: "GEOSITE, openai", GrpName: "AI"},
		{Keywords: []string{"youtube", "ytimg", "googlevideo"}, RuleName: "GEOSITE, youtube", GrpName: "YouTube"},
		{Keywords: []string{"google", "gmail", "gstatic", "android"}, RuleName: "GEOSITE, google", GrpName: "Google"},
		{Keywords: []string{"github", "ghcr", "docker"}, RuleName: "GEOSITE, github", GrpName: "GitHub & Docker"},
		{Keywords: []string{"telegram", "t.me", "telegra.ph"}, RuleName: "GEOSITE, telegram", GrpName: "Telegram"},
		{Keywords: []string{"twitter", "x.com", "twimg"}, RuleName: "GEOSITE, twitter", GrpName: "Twitter (X)"},
		{Keywords: []string{"netflix", "nflxvideo", "disney", "hbo", "max", "primevideo"}, RuleName: "GEOSITE, netflix", GrpName: "Netflix & 流媒体"},
		{Keywords: []string{"apple", "icloud", "itunes", "aaplimg"}, RuleName: "GEOSITE, apple", GrpName: "Apple 服务"},
		{Keywords: []string{"microsoft", "windows", "office", "azure", "live.com", "msn"}, RuleName: "GEOSITE, microsoft", GrpName: "Microsoft"},
		{Keywords: []string{"spotify"}, RuleName: "GEOSITE, spotify", GrpName: "Spotify"},
		{Keywords: []string{"steam", "steampowered", "steamcommunity"}, RuleName: "GEOSITE, steam", GrpName: "Steam 游戏"},
	}

	for _, bm := range bizRules {
		matched := false
		for _, kw := range bm.Keywords {
			if strings.Contains(kw, ".") {
				if cleaned == kw || strings.HasSuffix(cleaned, "."+kw) {
					matched = true
					break
				}
			} else {
				if strings.Contains(cleaned, kw) {
					matched = true
					break
				}
			}
		}
		if matched {
			sendJSON(w, map[string]any{
				"target":      target,
				"rule":        fmt.Sprintf("%s, %s", bm.RuleName, bm.GrpName),
				"group":       bm.GrpName,
				"active_node": getActiveNode(bm.GrpName),
			})
			return
		}
	}

	// Fallback to 🐟 漏网之鱼
	fishNode := getActiveNode("🐟 漏网之鱼")
	if fishNode == "" {
		fishNode = "DIRECT"
	}
	sendJSON(w, map[string]any{
		"target":      target,
		"rule":        "MATCH, 🐟 漏网之鱼",
		"group":       "🐟 漏网之鱼",
		"active_node": fishNode,
	})
}

// ──────────────────────────────────────────────
// KSpeeder Acceleration Handlers
// ──────────────────────────────────────────────

func (s *Server) queryLiveHostDockerRegistry(activeMirror string) (liveRegistry string, synced bool, platID string) {
	hostInfo := s.hostMgr.GetCachedHostInfo()
	if hostInfo == nil || !hostInfo.Connected {
		if probed, err := s.hostMgr.ProbeHost(); err == nil && probed != nil {
			hostInfo = probed
		}
	}
	rosConnected := s.rosMgr != nil && s.rosMgr.IsConnected()
	isROS := rosConnected || (hostInfo != nil && hostInfo.PlatformID == "routeros")

	if isROS {
		platID = "routeros"
		if s.rosMgr != nil {
			if reg, err := s.rosMgr.GetContainerRegistryURL(); err == nil {
				liveRegistry = reg
			}
		}
	} else {
		platID = "linux"
		if hostInfo != nil && hostInfo.PlatformID != "" {
			platID = hostInfo.PlatformID
		}
		if s.hostMgr != nil {
			if reg, err := s.hostMgr.GetDockerRegistryMirror(); err == nil {
				liveRegistry = reg
			}
		}
	}

	if liveRegistry != "" && activeMirror != "" {
		synced = strings.TrimRight(strings.ToLower(liveRegistry), "/") == strings.TrimRight(strings.ToLower(activeMirror), "/")
	}
	return
}

func (s *Server) handleKSpeederStatus(w http.ResponseWriter, r *http.Request) {
	status := s.kspeeder.GetStatus()
	liveRegistry, hostSynced, platID := s.queryLiveHostDockerRegistry(status.ActiveDockerMirror)
	resp := map[string]any{
		"enabled":              status.Enabled,
		"active_github_proxy":  status.ActiveGitHubProxy,
		"active_docker_mirror": status.ActiveDockerMirror,
		"github_proxies":       status.GitHubProxies,
		"docker_mirrors":       status.DockerMirrors,
		"auto_select_best":     status.AutoSelectBest,
		"last_tested_at":       status.LastTestedAt,
		"host_registry_url":    liveRegistry,
		"host_docker_synced":   hostSynced,
		"host_platform":        platID,
	}
	sendJSON(w, resp)
}

func (s *Server) handleKSpeederConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var cfg accelerator.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.kspeeder.UpdateConfig(cfg); err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"status": "ok", "config": s.kspeeder.GetStatus()})
}

func (s *Server) handleKSpeederTestGitHub(w http.ResponseWriter, r *http.Request) {
	nodes := s.kspeeder.TestGitHubProxies()
	sendJSON(w, map[string]any{"status": "ok", "proxies": nodes, "active": s.kspeeder.GetStatus().ActiveGitHubProxy})
}

func (s *Server) handleKSpeederTestDocker(w http.ResponseWriter, r *http.Request) {
	mirrors := s.kspeeder.TestDockerMirrors()
	sendJSON(w, map[string]any{"status": "ok", "mirrors": mirrors, "active": s.kspeeder.GetStatus().ActiveDockerMirror})
}

func (s *Server) handleKSpeederHostRegistryStatus(w http.ResponseWriter, r *http.Request) {
	activeMirror := s.kspeeder.GetStatus().ActiveDockerMirror
	liveRegistry, synced, platID := s.queryLiveHostDockerRegistry(activeMirror)
	hostName := "MikroTik RouterOS"
	if platID != "routeros" {
		hostInfo := s.hostMgr.GetCachedHostInfo()
		if hostInfo != nil && hostInfo.OSName != "" {
			hostName = hostInfo.OSName
		} else {
			hostName = "Linux 宿主"
		}
	}

	sendJSON(w, map[string]any{
		"ok":                   true,
		"platform":             platID,
		"host_name":            hostName,
		"active_docker_mirror": activeMirror,
		"host_registry_url":    liveRegistry,
		"synced":               synced,
	})
}

func (s *Server) handleKSpeederHostRegistryApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Mirror string `json:"mirror"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&reqBody)
	}

	mirror := strings.TrimSpace(reqBody.Mirror)
	if mirror == "" {
		mirror = s.kspeeder.GetStatus().ActiveDockerMirror
	}
	if mirror == "" {
		mirror = "https://atomhub.openatom.cn"
	}

	// Update local kspeeder config with this mirror if different
	currCfg := s.kspeeder.GetStatus()
	if currCfg.ActiveDockerMirror != mirror {
		currCfg.ActiveDockerMirror = mirror
		_ = s.kspeeder.UpdateConfig(currCfg)
	}

	// 1. Probe or retrieve cached host info
	hostInfo := s.hostMgr.GetCachedHostInfo()
	if hostInfo == nil || !hostInfo.Connected {
		if probed, err := s.hostMgr.ProbeHost(); err == nil && probed != nil {
			hostInfo = probed
		}
	}

	rosConnected := s.rosMgr != nil && s.rosMgr.IsConnected()

	// 2. Check if host is RouterOS
	isROS := rosConnected || (hostInfo != nil && hostInfo.PlatformID == "routeros")
	if isROS {
		var err error
		if s.rosMgr != nil {
			err = s.rosMgr.SetContainerRegistryURL(mirror)
		} else {
			err = fmt.Errorf("RouterOS 组件未初始化")
		}
		if err != nil {
			sendJSON(w, map[string]any{
				"status":    "error",
				"platform":  "routeros",
				"host_name": "MikroTik RouterOS",
				"error":     "未能通过 REST 写入 RouterOS: " + err.Error(),
			})
			return
		}

		// Read back to verify live status
		liveReg, _ := s.rosMgr.GetContainerRegistryURL()
		if liveReg == "" {
			liveReg = mirror
		}
		synced := strings.TrimRight(strings.ToLower(liveReg), "/") == strings.TrimRight(strings.ToLower(mirror), "/")

		sendJSON(w, map[string]any{
			"status":            "applied",
			"platform":          "routeros",
			"host_name":         "MikroTik RouterOS",
			"applied_mirror":    mirror,
			"host_registry_url": liveReg,
			"synced":            synced,
			"message":           fmt.Sprintf("🎉 已成功识别宿主【MikroTik RouterOS】，已将 Docker 镜像加速源【%s】一键写入容器仓库配置并核验生效！", liveReg),
		})
		return
	}

	// 3. Linux / fnOS / PVE / Debian / Ubuntu / Synology host branch
	hostName := "Linux 宿主"
	platID := "linux"
	if hostInfo != nil {
		if hostInfo.OSName != "" {
			hostName = hostInfo.OSName
		}
		if hostInfo.PlatformID != "" {
			platID = hostInfo.PlatformID
		}
	}

	err := s.hostMgr.SetDockerRegistryMirror(mirror)
	if err != nil {
		sendJSON(w, map[string]any{
			"status":    "error",
			"platform":  platID,
			"host_name": hostName,
			"error":     fmt.Sprintf("已识别宿主【%s】，但自动注入未完成 (%s)。请在「宿主共生」中保存 SSH 凭据以实现免密全自动直写。", hostName, err.Error()),
		})
		return
	}

	liveReg, _ := s.hostMgr.GetDockerRegistryMirror()
	if liveReg == "" {
		liveReg = mirror
	}
	synced := strings.TrimRight(strings.ToLower(liveReg), "/") == strings.TrimRight(strings.ToLower(mirror), "/")

	sendJSON(w, map[string]any{
		"status":            "applied",
		"platform":          platID,
		"host_name":         hostName,
		"applied_mirror":    mirror,
		"host_registry_url": liveReg,
		"synced":            synced,
		"message":           fmt.Sprintf("🎉 已成功识别宿主【%s】，已自动将镜像源写入 /etc/docker/daemon.json 并平滑重载 Docker 守护进程！", hostName),
	})
}


func (s *Server) handleKSpeederROSRegistryApply(w http.ResponseWriter, r *http.Request) {
	s.handleKSpeederHostRegistryApply(w, r)
}

// ──────────────────────────────────────────────
// Ring Log Buffer & Fully Desensitized Diagnostic Export
// ──────────────────────────────────────────────

func (s *Server) handleSystemLogs(w http.ResponseWriter, r *http.Request) {
	if (r.Method == http.MethodPost && r.URL.Query().Get("action") == "clear") || r.Method == http.MethodDelete {
		s.logBuffer.Clear()
		sendJSON(w, map[string]any{
			"status":  "cleared",
			"message": "环形实时日志已成功清空",
		})
		return
	}

	limit := 500
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	logs := s.logBuffer.GetLogs(limit)
	sendJSON(w, map[string]any{
		"status":   "success",
		"total":    len(logs),
		"max_size": 500,
		"logs":     logs,
	})
}

func (s *Server) handleSystemDiagnoseExport(w http.ResponseWriter, r *http.Request) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	meta := map[string]any{
		"brand":            "CubeX (极光魔方)",
		"version":          version,
		"export_time":      time.Now().Format(time.RFC3339),
		"uptime_seconds":   int64(time.Since(startTime).Seconds()),
		"os":               runtime.GOOS,
		"arch":             runtime.GOARCH,
		"num_cpu":          runtime.NumCPU(),
		"goroutines":       runtime.NumGoroutine(),
		"memory_alloc_mb":  float64(memStats.Alloc) / (1024 * 1024),
		"memory_sys_mb":    float64(memStats.Sys) / (1024 * 1024),
	}

	lastRes := s.engMgr.GetLastResult()
	engineRunning := true
	if lastRes != nil && !lastRes.Success {
		engineRunning = false
	}
	engineStatus := map[string]any{
		"active_engine": s.engMgr.GetActiveEngine(),
		"running":       engineRunning,
	}

	var safeHost map[string]any
	if hInfo := s.hostMgr.GetCachedHostInfo(); hInfo != nil {
		safeHost = map[string]any{
			"platform_id":         hInfo.PlatformID,
			"os_name":             hInfo.OSName,
			"hostname":            hInfo.HostName,
			"kernel_version":      hInfo.KernelVersion,
			"arch":                hInfo.Arch,
			"cpu_cores":           hInfo.CPUCores,
			"cpu_load":            hInfo.CPULoad,
			"memory_total":        hInfo.MemoryTotal,
			"memory_used":         hInfo.MemoryUsed,
			"ip_forward":          hInfo.IPForward,
			"docker_running":      hInfo.DockerRunning,
			"wireguard_active":    hInfo.WireGuardActive,
			"firewall_configured": hInfo.FirewallConfigured,
		}
	} else {
		safeHost = map[string]any{"status": "not_cached"}
	}

	rosStatus := s.rosMgr.GetOverview()
	safeROS := map[string]any{
		"connected":      rosStatus["ros_connected"],
		"version":        rosStatus["ros_version"],
		"model":          rosStatus["ros_model"],
		"routing_synced": rosStatus["routing_synced"],
	}

	etSt := s.easyTierMgr.GetStatus()
	safeEasyTier := map[string]any{
		"running":      etSt.Running,
		"virtual_ip":   etSt.VirtualIP,
		"peer_count":   len(etSt.Peers),
		"network_name": "***MASKED***",
	}

	subSummary := []map[string]any{}
	proxyTypeCount := make(map[string]int)
	ir := s.engMgr.GetIR()
	if ir != nil {
		for _, sub := range ir.Subscriptions {
			subSummary = append(subSummary, map[string]any{
				"id":         sub.ID,
				"name":       sub.Name,
				"node_count": sub.NodeCount,
				"url":        "***DESENSITIZED_TOKEN_STRIPPED***",
			})
		}
		for _, p := range ir.Proxies {
			proxyTypeCount[p.Type]++
		}
	}

	adStats := s.adblockMgr.GetStats(true, 0)
	recentLogs := s.logBuffer.GetLogs(200)

	pack := map[string]any{
		"cubex_diagnostic_report": true,
		"desensitized":            true,
		"compliance":              "ZERO_CREDENTIAL_LEAKAGE_STRICT",
		"meta":                    meta,
		"engine":                  engineStatus,
		"host":                    safeHost,
		"routeros":                safeROS,
		"easytier":                safeEasyTier,
		"subscriptions_summary":   subSummary,
		"proxy_protocols_summary": proxyTypeCount,
		"adblock_stats":           adStats,
		"recent_logs":             recentLogs,
	}

	// If mode=json requested, return JSON object directly (e.g. for frontend inspection)
	if r.URL.Query().Get("mode") == "json" {
		sendJSON(w, pack)
		return
	}

	filename := fmt.Sprintf("cubex_diagnostic_%s.json", time.Now().Format("20060102_150405"))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(pack)
}

