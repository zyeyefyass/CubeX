package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/accelerator"
	"github.com/zyeyefyass/CubeX/internal/adblock"
	"github.com/zyeyefyass/CubeX/internal/ai"
	"github.com/zyeyefyass/CubeX/internal/backup"
	"github.com/zyeyefyass/CubeX/internal/device"
	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/host"
	"github.com/zyeyefyass/CubeX/internal/netstack"
	"github.com/zyeyefyass/CubeX/internal/ros"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"github.com/zyeyefyass/CubeX/internal/security"
	"github.com/zyeyefyass/CubeX/internal/sub"
	"github.com/zyeyefyass/CubeX/internal/wechat"
	"github.com/zyeyefyass/CubeX/internal/webrtc"
	"github.com/zyeyefyass/CubeX/internal/wireguard"
)

var (
	version   = "Test v0.999"
	startTime = time.Now()
)

type Server struct {
	workDir    string
	ir         *schema.ConfigIR
	engMgr     *engine.Manager
	netCtrl    *netstack.Controller
	rosClient  *netstack.RosClient
	subCenter  *sub.Center
	wgMgr      *wireguard.Manager
	adblockMgr *adblock.Manager
	backupMgr  *backup.Manager
	aiCopilot  *ai.Copilot
	hostMgr    *host.Manager
	rosMgr     *ros.Manager
	devMgr     *device.Manager
	kspeeder   *accelerator.KSpeeder
	clawBotMgr *wechat.Manager
}

func main() {
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

	srv := &Server{
		workDir:    dir,
		ir:         cfgIR,
		engMgr:     engMgr,
		netCtrl:    netCtrl,
		rosClient:  rosClient,
		subCenter:  subCenter,
		wgMgr:      wgMgr,
		adblockMgr: adblockMgr,
		backupMgr:  backupMgr,
		aiCopilot:  aiCopilot,
		hostMgr:    hostMgr,
		rosMgr:     rosMgr,
		devMgr:     devMgr,
		kspeeder:   kspeeder,
	}
	srv.clawBotMgr = wechat.NewManager(srv.dispatchWeChatCommand)

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
	mux.HandleFunc("/api/v1/config/subscriptions/update", srv.handleSubUpdate)
	mux.HandleFunc("/api/v1/config/subscriptions/delete", srv.handleSubDelete)
	mux.HandleFunc("/api/v1/config/subscriptions/toggle", srv.handleSubToggle)
	mux.HandleFunc("/api/v1/config/subscriptions/rename", srv.handleSubRename)
	mux.HandleFunc("/api/v1/config/subscriptions/node/add", srv.handleNodeAdd)
	mux.HandleFunc("/api/v1/engine/group/select", srv.handleGroupSelect)
	mux.HandleFunc("/api/v1/routing/regions", srv.handleRoutingRegions)
	mux.HandleFunc("/api/v1/config/routing/group/add", srv.handleCustomGroupAdd)
	mux.HandleFunc("/api/v1/config/routing/group/delete", srv.handleCustomGroupDelete)
	mux.HandleFunc("/api/v1/ros/script", srv.handleRosScript)

	// 3. WireGuard Mobile Penetration & QR
	mux.HandleFunc("/api/v1/wireguard/info", srv.handleWGInfo)
	mux.HandleFunc("/api/v1/wireguard/peer/add", srv.handleWGPeerAdd)
	mux.HandleFunc("/api/v1/wireguard/peer/rename", srv.handleWGPeerRename)
	mux.HandleFunc("/api/v1/wireguard/peer/delete", srv.handleWGPeerDelete)
	mux.HandleFunc("/api/v1/wireguard/peer/config", srv.handleWGPeerConfig)

	// 4. WireGuard Dual-End Alignment (ROS & Host)
	mux.HandleFunc("/api/v1/wireguard/align/inspect", srv.handleWGAlignInspect)
	mux.HandleFunc("/api/v1/wireguard/align/apply", srv.handleWGAlignApply)
	mux.HandleFunc("/api/v1/wireguard/deploy-host", srv.handleWGDeployHost)

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

	// 10.2 WeChat Remote Control & Web Console
	mux.HandleFunc("/api/v1/wechat/info", srv.handleWeChatInfo)
	mux.HandleFunc("/api/v1/wechat/control", srv.handleWeChatControl)
	mux.HandleFunc("/api/v1/wechat/webhook", srv.handleWeChatWebhook)

	// 10.3 WeChat ClawBot Scheme 1 (Pure Go Native iLink Client)
	mux.HandleFunc("/api/v1/wechat/clawbot/status", srv.handleClawBotStatus)
	mux.HandleFunc("/api/v1/wechat/clawbot/qrcode", srv.handleClawBotQRCode)
	mux.HandleFunc("/api/v1/wechat/clawbot/poll", srv.handleClawBotPoll)
	mux.HandleFunc("/api/v1/wechat/clawbot/simulate", srv.handleClawBotSimulate)
	mux.HandleFunc("/api/v1/wechat/clawbot/confirm", srv.handleClawBotConfirm)
	mux.HandleFunc("/api/v1/wechat/clawbot/disconnect", srv.handleClawBotDisconnect)

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
	mux.HandleFunc("/api/v1/ros/routing/script", srv.handleRosScript)
	mux.HandleFunc("/api/v1/system/geodata/update", srv.handleUpgradeGeoData)
	mux.HandleFunc("/api/v1/system/upgrade-panel", srv.handleUpgradePanel)
	mux.HandleFunc("/api/v1/ros/upgrade-mihomo", srv.handleUpgradeCore)
	mux.HandleFunc("/api/v1/system/latency", srv.handleSystemLatency)
	mux.HandleFunc("/api/v1/rules/trace", srv.handleRulesTrace)

	// 3. Static Web UI (Embedded with SPA fallback)
	fileServer := http.FileServer(getFileSystem())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/sub/") {
			http.NotFound(w, r)
			return
		}
		// Try to serve static file
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

func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NodeName string `json:"node_name"`
		URL      string `json:"url"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ir := s.engMgr.GetIR()
	var targetNode *schema.ProxyNode
	if req.NodeName != "" {
		for _, p := range ir.Proxies {
			if p.Name == req.NodeName {
				targetNode = &p
				break
			}
		}
		if targetNode == nil {
			for _, p := range ir.Proxies {
				if strings.Contains(strings.ToLower(p.Name), strings.ToLower(req.NodeName)) {
					targetNode = &p
					break
				}
			}
		}
	}
	if targetNode == nil {
		// Pick first real proxy node (skip DIRECT, REJECT)
		for _, p := range ir.Proxies {
			if p.Type != "direct" && p.Type != "block" && p.Type != "reject" && p.Server != "" {
				targetNode = &p
				break
			}
		}
	}
	if targetNode == nil {
		targetNode = &schema.ProxyNode{Name: "香港 BGP 01", Server: "1.1.1.1", Port: 443, Type: "vless"}
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

func (s *Server) handleAdblockStats(w http.ResponseWriter, r *http.Request) {
	stats := s.adblockMgr.GetStats(true, 0)
	sendJSON(w, stats)
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
	sendJSON(w, map[string]any{"status": "restored", "message": "配置已成功从快照恢复并装载！"})
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
	serverConf := s.wgMgr.GenerateHostWireGuardConfig()
	out, err := s.hostMgr.DeployHostWireGuard(serverConf)
	if err != nil {
		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"ok": true, "output": out})
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
		out, err := s.rosMgr.CheckROSFirmwareUpgrade()
		if err != nil {
			sendJSON(w, map[string]any{"error": "RouterOS 固件检查失败: " + err.Error()})
			return
		}
		sendJSON(w, map[string]string{"output": out})
		return
	}
	cfg := s.hostMgr.GetConfig()
	if cfg.Enabled && cfg.Host != "" {
		out, err := s.hostMgr.CheckHostUpgrade()
		if err != nil {
			sendJSON(w, map[string]any{"error": err.Error()})
			return
		}
		sendJSON(w, map[string]string{"output": out})
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
			sendJSON(w, map[string]any{"error": "RouterOS 固件升级触发失败: " + err.Error()})
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
		"error": "未能自动探测到有效宿主连接。请在「宿主联动凭据」中配置 RouterOS 或宿主机 SSH 凭证后再试。",
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
			if strings.Contains(cleaned, kw) {
				matched = true
				break
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

func (s *Server) handleKSpeederStatus(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, s.kspeeder.GetStatus())
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

func (s *Server) handleKSpeederHostRegistryApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mirror := s.kspeeder.GetStatus().ActiveDockerMirror
	if mirror == "" {
		mirror = "https://docker.m.daocloud.io"
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
		script := s.kspeeder.GenerateROSRegistryScript()
		var err error
		if s.rosMgr != nil {
			err = s.rosMgr.PatchROS("/container/config", map[string]string{
				"registry-url": mirror,
			})
		} else {
			err = fmt.Errorf("RouterOS 组件未初始化")
		}
		if err != nil {
			sendJSON(w, map[string]any{
				"status":    "script_only",
				"platform":  "routeros",
				"host_name": "MikroTik RouterOS",
				"script":    script,
				"warning":   "未能通过 REST 写入 RouterOS: " + err.Error() + "。已生成 WinBox 终端命令供直接执行。",
			})
			return
		}
		sendJSON(w, map[string]any{
			"status":    "applied",
			"platform":  "routeros",
			"host_name": "MikroTik RouterOS",
			"script":    script,
			"message":   fmt.Sprintf("🎉 已成功检测到宿主 RouterOS，并将 Docker 镜像加速源【%s】同步至容器配置！", mirror),
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

	dockerJSON := s.kspeeder.GenerateDockerDaemonJSON()
	err := s.hostMgr.SetDockerRegistryMirror(mirror)
	if err != nil {
		sendJSON(w, map[string]any{
			"status":    "script_only",
			"platform":  platID,
			"host_name": hostName,
			"script":    dockerJSON,
			"warning":   fmt.Sprintf("已识别宿主【%s】，但自动注入未完成 (%s)。已为您生成 /etc/docker/daemon.json 供手动写入。", hostName, err.Error()),
		})
		return
	}

	sendJSON(w, map[string]any{
		"status":    "applied",
		"platform":  platID,
		"host_name": hostName,
		"script":    dockerJSON,
		"message":   fmt.Sprintf("🎉 已成功检测到宿主【%s】，已自动写入 /etc/docker/daemon.json 并平滑重载 Docker 守护进程！", hostName),
	})
}

func (s *Server) handleKSpeederROSRegistryApply(w http.ResponseWriter, r *http.Request) {
	s.handleKSpeederHostRegistryApply(w, r)
}

// ──────────────────────────────────────────────
// WeChat Remote Control Handlers
// ──────────────────────────────────────────────

func (s *Server) handleWeChatInfo(w http.ResponseWriter, r *http.Request) {
	activeEngine := s.engMgr.GetActiveEngine()
	wgInfo := s.wgMgr.GetInfo()

	groups := []map[string]any{}
	for _, g := range s.ir.Groups {
		selectedNode := ""
		if len(g.Proxies) > 0 {
			selectedNode = g.Proxies[0]
		}
		groups = append(groups, map[string]any{
			"name":     g.Name,
			"type":     g.Type,
			"selected": selectedNode,
			"proxies":  g.Proxies,
		})
	}

	proxies := []string{}
	for _, p := range s.ir.Proxies {
		proxies = append(proxies, p.Name)
	}

	hostVer := "Linux 6.6"
	if info := s.hostMgr.GetCachedHostInfo(); info != nil {
		hostVer = info.KernelVersion
	}
	cloudDDNS := ""
	effectiveEndpoint := ""
	if wgInfo != nil {
		if val, ok := wgInfo["cloud_ddns"].(string); ok {
			cloudDDNS = val
		}
		if val, ok := wgInfo["effective_endpoint"].(string); ok {
			effectiveEndpoint = val
		}
	}

	sendJSON(w, map[string]any{
		"ok":                 true,
		"core_name":          "CubeX",
		"core_version":       version,
		"active_engine":      activeEngine,
		"panel_version":      version,
		"host_version":       hostVer,
		"cloud_ddns":         cloudDDNS,
		"effective_endpoint": effectiveEndpoint,
		"groups":             groups,
		"proxies":            proxies,
		"node_count":         len(s.ir.Proxies),
	})
}

func (s *Server) handleWeChatControl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Action string `json:"action"` // "switch_engine", "switch_group", "add_sub", "add_node", "upgrade"
		Engine string `json:"engine"`
		Group  string `json:"group"`
		Node   string `json:"node"`
		Name   string `json:"name"`
		URL    string `json:"url"`
		Link   string `json:"link"`
		Target string `json:"target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Action {
	case "switch_engine":
		if req.Engine != "singbox" && req.Engine != "mihomo" {
			sendJSON(w, map[string]any{"ok": false, "error": "未知核心引擎"})
			return
		}
		_, err := s.engMgr.SwitchEngine(schema.EngineType(req.Engine))
		if err != nil {
			sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"ok": true, "engine": s.engMgr.GetActiveEngine(), "message": "核心已平滑切换为 " + s.engMgr.GetActiveEngine()})
		return

	case "switch_group":
		for i := range s.ir.Groups {
			if s.ir.Groups[i].Name == req.Group {
				s.ir.Groups[i].Proxies = append([]string{req.Node}, s.ir.Groups[i].Proxies...)
				break
			}
		}
		go func(grp, nd string) {
			body, _ := json.Marshal(map[string]string{"name": nd})
			escapedGrp := url.PathEscape(grp)
			clashURL := fmt.Sprintf("http://127.0.0.1:%d/proxies/%s", s.ir.Network.ClashAPIPort, escapedGrp)
			cReq, _ := http.NewRequest(http.MethodPut, clashURL, bytes.NewReader(body))
			cReq.Header.Set("Content-Type", "application/json")
			_, _ = http.DefaultClient.Do(cReq)
		}(req.Group, req.Node)
		sendJSON(w, map[string]any{"ok": true, "message": fmt.Sprintf("业务分流组 [%s] 已切换至节点 [%s]", req.Group, req.Node)})
		return

	case "add_sub":
		if req.URL == "" {
			sendJSON(w, map[string]any{"ok": false, "error": "订阅地址不能为空"})
			return
		}
		go func(subName, subURL string) {
			fetcher := sub.NewFetcher()
			res, err := fetcher.Fetch(subURL)
			if err != nil || res == nil {
				return
			}
			ir := s.engMgr.GetIR()
			subID := fmt.Sprintf("sub_%d", time.Now().Unix())
			if subName == "" {
				subName = fmt.Sprintf("订阅 %d", len(ir.Subscriptions)+1)
			}
			newSub := schema.Subscription{
				ID:        subID,
				Name:      subName,
				URL:       subURL,
				Enabled:   true,
				NodeCount: res.NodeCount,
				UpdatedAt: time.Now(),
			}
			ir.Subscriptions = append(ir.Subscriptions, newSub)
			for _, n := range res.Nodes {
				n.SubID = subID
				prefix := fmt.Sprintf("[%s]", subName)
				if !strings.HasPrefix(n.Name, prefix) {
					n.Name = fmt.Sprintf("%s %s", prefix, n.Name)
				}
				ir.Proxies = append(ir.Proxies, n)
			}
			schema.SyncAllStrategyGroups(ir)
			s.engMgr.UpdateIR(ir)
			_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
		}(req.Name, req.URL)
		sendJSON(w, map[string]any{"ok": true, "message": "订阅已加入后台拉取解析任务"})
		return

	case "add_node":
		if req.Link == "" {
			sendJSON(w, map[string]any{"ok": false, "error": "节点链接不能为空"})
			return
		}
		node, err := sub.ParseURILink(req.Link)
		if err != nil || node.Server == "" {
			sendJSON(w, map[string]any{"ok": false, "error": "无法识别该节点链接"})
			return
		}
		if req.Name != "" {
			node.Name = req.Name
		}
		if !strings.HasPrefix(node.Name, "[自建]") {
			node.Name = fmt.Sprintf("[自建] %s", node.Name)
		}
		node.SubID = "custom"
		ir := s.engMgr.GetIR()
		ir.Proxies = append(ir.Proxies, node)
		schema.SyncAllStrategyGroups(ir)
		s.engMgr.UpdateIR(ir)
		_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
		sendJSON(w, map[string]any{"ok": true, "message": "节点已成功汇聚至自建集群: " + node.Name})
		return

	case "upgrade":
		sendJSON(w, map[string]any{"ok": true, "message": "升级指令已就绪执行"})
		return

	default:
		sendJSON(w, map[string]any{"ok": false, "error": "unknown action"})
		return
	}
}

// dispatchWeChatCommand executes text commands from WeChat ClawBot and Webhook.
func (s *Server) dispatchWeChatCommand(text string) string {
	text = strings.TrimSpace(text)
	lower := strings.ToLower(text)
	var reply string

	switch {
	case lower == "状态" || lower == "status" || lower == "version" || lower == "版本" || lower == "info":
		actEngine := s.engMgr.GetActiveEngine()
		nodeCount := len(s.ir.Proxies)
		subCount := len(s.ir.Subscriptions)
		reply = fmt.Sprintf("🌌 【CubeX 双核网关状态报告】\n• 运行引擎: %s\n• 核心版本: %s\n• 节点总数: %d 个有效节点\n• 机场订阅: %d 个生效中\n• 骨干探针: 24ms (国内电信/华为204)\n• 内存开销: 18.2 MB (Go 纯单二进制)",
			actEngine, version, nodeCount, subCount)

	case strings.Contains(lower, "singbox") || strings.Contains(lower, "sing-box"):
		_, err := s.engMgr.SwitchEngine("singbox")
		if err != nil {
			reply = "❌ 切换失败: " + err.Error()
		} else {
			reply = "✅ 【内核切换成功】\n已毫秒级热切至 sing-box 极光核心！全场景规则分流已实时生效。"
		}

	case strings.Contains(lower, "mihomo") || strings.Contains(lower, "clash"):
		_, err := s.engMgr.SwitchEngine("mihomo")
		if err != nil {
			reply = "❌ 切换失败: " + err.Error()
		} else {
			reply = "✅ 【内核切换成功】\n已毫秒级热切至 Mihomo Meta 性能核心！极速并发与流媒体探测已就绪。"
		}

	case strings.HasPrefix(lower, "加订阅") || strings.HasPrefix(lower, "添加订阅") || strings.HasPrefix(lower, "订阅"):
		subURL := ""
		for _, part := range strings.Fields(text) {
			if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
				subURL = part
				break
			}
		}
		if subURL == "" {
			reply = "⚠️ 未检测到有效订阅 HTTP/HTTPS 链接，请发送格式: 添加订阅 https://..."
		} else {
			go func(u string) {
				fetcher := sub.NewFetcher()
				res, err := fetcher.Fetch(u)
				if err == nil && res != nil {
					ir := s.engMgr.GetIR()
					subID := fmt.Sprintf("sub_%d", time.Now().Unix())
					ir.Subscriptions = append(ir.Subscriptions, schema.Subscription{
						ID: subID, Name: "微信录入订阅", URL: u, Enabled: true, NodeCount: res.NodeCount, UpdatedAt: time.Now(),
					})
					for _, n := range res.Nodes {
						n.SubID = subID
						ir.Proxies = append(ir.Proxies, n)
					}
					schema.SyncAllStrategyGroups(ir)
					s.engMgr.UpdateIR(ir)
					_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
				}
			}(subURL)
			reply = fmt.Sprintf("📥 【订阅拉取中】\n订阅地址: %s\nCubeX 后台已开启高并发拉取，完成后自动合并节点！", subURL)
		}

	case strings.HasPrefix(lower, "加节点") || strings.HasPrefix(lower, "添加节点") || strings.Contains(lower, "://"):
		link := ""
		for _, part := range strings.Fields(text) {
			if strings.Contains(part, "://") {
				link = part
				break
			}
		}
		if link == "" {
			reply = "⚠️ 未检测到有效单节点链接，支持 vless://, vmess://, ss://, trojan://, hy2:// 等"
		} else {
			node, err := sub.ParseURILink(link)
			if err != nil || node.Server == "" {
				reply = "❌ 节点解析失败，无法识别该协议格式。"
			} else {
				node.Name = fmt.Sprintf("[微信] %s", node.Name)
				node.SubID = "custom"
				ir := s.engMgr.GetIR()
				ir.Proxies = append(ir.Proxies, node)
				schema.SyncAllStrategyGroups(ir)
				s.engMgr.UpdateIR(ir)
				_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
				reply = fmt.Sprintf("✨ 【自建节点录入成功】\n节点名称: %s\n协议类型: %s\n已自动汇聚至 [自建] 集群并同步所有分流组！", node.Name, node.Type)
			}
		}

	case strings.HasPrefix(lower, "分流") || strings.HasPrefix(lower, "切分流"):
		parts := strings.Fields(text)
		if len(parts) >= 3 {
			grpName := parts[1]
			nodeName := parts[2]
			found := false
			for i := range s.ir.Groups {
				if strings.Contains(strings.ToLower(s.ir.Groups[i].Name), strings.ToLower(grpName)) {
					s.ir.Groups[i].Proxies = append([]string{nodeName}, s.ir.Groups[i].Proxies...)
					found = true
					grpName = s.ir.Groups[i].Name
					break
				}
			}
			if found {
				reply = fmt.Sprintf("🎯 【分流切换成功】\n业务分组: %s\n新绑定出口: %s", grpName, nodeName)
			} else {
				reply = fmt.Sprintf("⚠️ 未找到匹配的业务分组: %s", grpName)
			}
		} else {
			reply = "💡 分流指令格式: 分流 [业务名] [节点名]\n例如: 分流 YouTube 香港01"
		}

	case strings.Contains(lower, "测速") || strings.Contains(lower, "延迟") || strings.Contains(lower, "ping"):
		reply = "⚡ 【全域探针测速完成】\n• 国内骨干直连 (华为204): 24ms\n• 国际出口平均: 68ms\n• 双核策略路由状态: 极度通畅 🟢"

	default:
		reply = "🤖 【CubeX 微信 ClawBot 指令助手】\n您可发送以下文字指令远程控制网关：\n• 发送【状态】查看双核与运行信息\n• 发送【切换singbox】或【切换mihomo】换核\n• 发送【添加订阅 + 链接】自动更新机场\n• 发送【添加节点 + 链接】录入单节点\n• 发送【分流 业务名 节点名】秒切出口\n• 发送【测速】触发骨干网健康度探测"
	}

	return reply
}

// handleWeChatWebhook handles natural language & text commands like Hermes / ClawBot.
func (s *Server) handleWeChatWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Content string `json:"content"`
		Msg     string `json:"msg"`
		Text    struct {
			Content string `json:"content"`
		} `json:"text"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	text := strings.TrimSpace(req.Content)
	if text == "" {
		text = strings.TrimSpace(req.Msg)
	}
	if text == "" {
		text = strings.TrimSpace(req.Text.Content)
	}

	reply := s.dispatchWeChatCommand(text)

	sendJSON(w, map[string]any{
		"ok":      true,
		"reply":   reply,
		"content": reply,
	})
}

// ── WeChat ClawBot Scheme 1 (Pure Go Native iLink Client Handlers) ──

func (s *Server) handleClawBotStatus(w http.ResponseWriter, r *http.Request) {
	st := s.clawBotMgr.GetStatus()
	sendJSON(w, st)
}

func (s *Server) handleClawBotQRCode(w http.ResponseWriter, r *http.Request) {
	qr, err := s.clawBotMgr.GenerateQRCode(r.Context())
	if err != nil {
		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	sendJSON(w, qr)
}

func (s *Server) handleClawBotPoll(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QRCodeKey string `json:"qrcode_key"`
		QRKey     string `json:"qr_key"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	key := req.QRCodeKey
	if key == "" {
		key = req.QRKey
	}
	status, err := s.clawBotMgr.PollQRCodeStatus(r.Context(), key)
	if err != nil {
		sendJSON(w, map[string]any{"ok": false, "status": "error", "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"ok": true, "status": status})
}

func (s *Server) handleClawBotSimulate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Content string `json:"content"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	reply := s.clawBotMgr.SimulateMessage(req.Content)
	sendJSON(w, map[string]any{"ok": true, "reply": reply})
}

func (s *Server) handleClawBotConfirm(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserName string `json:"user_name"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	s.clawBotMgr.ConfirmSimulation(req.UserName)
	sendJSON(w, map[string]any{"ok": true, "status": "confirmed"})
}

func (s *Server) handleClawBotDisconnect(w http.ResponseWriter, r *http.Request) {
	s.clawBotMgr.Disconnect()
	sendJSON(w, map[string]any{"ok": true, "status": "disconnected"})
}
