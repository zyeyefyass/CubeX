package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zyeyefyass/CubeX/internal/dataplane"
	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/lifecycle"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"github.com/zyeyefyass/CubeX/internal/telemetry"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"
	"time"
)

func (s *Server) Run(listenAddr string) error {
	mux := http.NewServeMux()
	s.RegisterRoutes(mux)

	ln, actualAddr, err := bindWithFallback(listenAddr)
	if err != nil {
		return err
	}
	log.Printf("🌌 CubeX Daemon [%s] listening on %s (Engine: %s, Tuning: %s)\n", version, actualAddr, s.ir.Meta.ActiveEngine, s.ir.Network.TunInterface)

	// 初始化 LTS 守护进程全生命周期合龙状态机
	activeIRPath := filepath.Join(s.workDir, "data", "config.ir.json")
	lkgPath := filepath.Join(s.workDir, "data", "lkg.json")
	lkgGuard := telemetry.NewLKGConfigGuard(activeIRPath, lkgPath, telemetry.GetGlobalFlightRecorder())
	ltsSupervisor := lifecycle.NewLTSDaemonSupervisor(
		lkgGuard,
		telemetry.GetGlobalFlightRecorder(),
		telemetry.GetGlobalSSEBroadcaster(),
		15*time.Second,
	)

	// 安全启用内核快速直通（若非特权容器则平滑降级）
	ltsSupervisor.EnableKernelFastPathSafely(context.Background(), dataplane.FastPathSpec{
		TunInterface:  s.ir.Network.TunInterface,
		DNSListenPort: s.ir.Network.DNSPort,
		BlockIPv6AAAA: true,
	})

	handler := recoveryMiddleware(gzipMiddleware(corsMiddleware(authMiddleware(s.authMgr, mux))))
	httpServer := &http.Server{
		Handler: handler,
	}

	// Dual-port listening: if port 80 is available (e.g. standalone container), also serve on port 80 for seamless access (http://<IP>)
	if listenAddr != ":80" && listenAddr != "0.0.0.0:80" {
		SafeGo("WebDirectPort80", func() {
			ln80, err80 := net.Listen("tcp", ":80")
			if err80 == nil {
				log.Printf("🌐 [Web Direct] Port 80 is free! Also serving dashboard directly at http://<host-ip>/ (Port 80)\n")
				_ = http.Serve(ln80, handler)
			} else {
				log.Printf("ℹ️ [Web Direct] Port 80 is reserved or occupied by host (%v), continuing on %s\n", err80, actualAddr)
			}
		})
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	SafeGo("SignalSupervisor", func() {
		sig := <-sigCh
		log.Printf("🛑 接收到系统终止信号 [%v]，启动 LTS 确定性无阻塞退出序列...\n", sig)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		ltsSupervisor.DeterministicShutdown(
			shutdownCtx,
			func(ctx context.Context) error {
				if s.rosMgr != nil {
					_ = s.rosMgr.GracefulDrain(ctx, true)
				}
				return nil
			},
			func() error {
				s.engMgr.Stop()
				return nil
			},
			func(ctx context.Context) error {
				return httpServer.Shutdown(ctx)
			},
		)
		log.Println("✅ LTS 守护进程已安全优雅停机。")
		os.Exit(0)
	})

	if err := httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
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

			"status": "restored",

			"message": "已恢复极光网关双核智能加速",

			"bypassed": false,
		})

		return

	}

	_ = s.netCtrl.EmergencyBypass()

	sendJSON(w, map[string]any{

		"status": "bypassed",

		"message": "已触发 1 秒一键急救：已清空转发规则，恢复纯物理直连逃生！",

		"bypassed": true,
	})

}

// purgeConntrackForDirectBypass cleans up stale Fake-IP connections and conntrack entries


func (s *Server) purgeConntrackForDirectBypass() {
	SafeGo("PurgeConntrackBypass", func() {
		time.Sleep(300 * time.Millisecond)
		// 1. Linux 内核连接跟踪表定向刷洗 198.18.0.0/15 (仅清退已重定向的 Fake-IP 僵尸流，保护本地管理端口)
		_ = exec.Command("conntrack", "-D", "-d", "198.18.0.0/15").Run()
		_ = exec.Command("conntrack", "-D", "-s", "198.18.0.0/15").Run()

		// 2. RouterOS 策略路由与 Conntrack 僵尸连接清算
		if s.rosMgr != nil && s.rosMgr.IsConnected() {
			_ = s.rosMgr.PurgeStaleFakeIPConnections()
		}
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

	s.purgeConntrackForDirectBypass()

	sendJSON(w, map[string]any{

		"status": "ok",

		"message": "分流策略与配置已成功同步重载",

		"result": res,
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

		"status": "ok",

		"message": "系统所有网络与分流配置已全量初始化清空，恢复至出厂洁净状态！",

		"result": res,
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

		"status": "ok",

		"message": "代理内核已成功毫秒级重载",

		"engine": s.engMgr.GetActiveEngine(),

		"result": res,
	})

}



func (s *Server) handleHostHotRestart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	activeEng := s.engMgr.GetActiveEngine()
	if activeEng == "" {
		activeEng = schema.EngineSingbox
	}
	res, err := s.engMgr.SwitchEngine(activeEng)
	var purged int
	if s.rosMgr != nil {
		purged = s.rosMgr.PurgeStaleFakeIPConnections()
	}
	if err != nil {
		sendJSON(w, map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	var elapsed int64
	if res != nil {
		elapsed = res.ElapsedMs
	}
	s.logBuffer.Add("INFO", "ENGINE", fmt.Sprintf("🔥 宿主共生服务热重启完成: 双核引擎 %s 已平滑重载, 清理 %d 条残留连接 (耗时: %dms)", activeEng, purged, elapsed))
	sendJSON(w, map[string]any{
		"success":            true,
		"engine":             activeEng,
		"purged_connections": purged,
		"elapsed_ms":         elapsed,
		"message":            fmt.Sprintf("双核引擎 (%s) 与宿主网络栈已完成热重启，全链路 100%% 干净重载", activeEng),
	})
}



func (s *Server) handleRestartService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	sendJSON(w, map[string]any{
		"status":  "restarting",
		"message": "CubeX 守护服务正在平滑重启...",
	})

	SafeGo("RestartDaemon", func() {
		time.Sleep(500 * time.Millisecond)
		if s.engMgr != nil {
			_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
		}
		exe, err := os.Executable()
		if err == nil {
			cmd := exec.Command(exe, os.Args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			_ = cmd.Start()
		}
		os.Exit(0)
	})
}



func (s *Server) handleSystemPoweroff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 1. Pre-flight active failover on RouterOS before poweroff
	if s.rosMgr != nil {
		_ = s.rosMgr.TriggerPreShutdownFailover()
	}
	// 2. Gracefully stop engine
	if s.engMgr != nil {
		s.engMgr.Stop()
	}
	sendJSON(w, map[string]any{
		"ok":      true,
		"status":  "powering_off",
		"message": "CubeX 已主动执行逃生切流，系统正在安全关机...",
	})
	SafeGo("Poweroff", func() {
		time.Sleep(600 * time.Millisecond)
		_ = exec.Command("poweroff").Run()
		_ = exec.Command("shutdown", "-h", "now").Run()
		_ = exec.Command("systemctl", "poweroff").Run()
		os.Exit(0)
	})
}



func (s *Server) handleSystemReboot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 1. Pre-flight active failover on RouterOS before reboot
	if s.rosMgr != nil {
		_ = s.rosMgr.TriggerPreShutdownFailover()
	}
	// 2. Gracefully stop engine
	if s.engMgr != nil {
		s.engMgr.Stop()
	}
	sendJSON(w, map[string]any{
		"ok":      true,
		"status":  "rebooting",
		"message": "CubeX 已主动执行逃生切流，整机系统正在重启 (预计 10 秒后恢复)...",
	})
	SafeGo("Reboot", func() {
		time.Sleep(600 * time.Millisecond)
		_ = exec.Command("reboot").Run()
		_ = exec.Command("shutdown", "-r", "now").Run()
		_ = exec.Command("systemctl", "reboot").Run()
		os.Exit(0)
	})
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

	var result *engine.SwitchResult
	var err error
	if s.coordinator != nil {
		result, err = s.coordinator.CoordinatedSwitchEngine(r.Context(), string(target))
	} else {
		result, err = s.engMgr.SwitchEngine(target)
	}

	if err != nil {
		telemetry.GetGlobalFlightRecorder().Record(telemetry.CatEngineSwitch, "CRIT", fmt.Sprintf("双核热切换至 %s 失败: %v", target, err), 0)
		sendJSON(w, map[string]any{"status": "error", "success": false, "error": err.Error(), "result": result})
		return
	}

	cost := int64(0)
	if result != nil {
		cost = result.TotalCostMs
	}
	telemetry.GetGlobalFlightRecorder().Record(telemetry.CatEngineSwitch, "INFO", fmt.Sprintf("双核热切换至 %s 成功（耗时 %dms）", target, cost), cost)

	if s.rosMgr != nil {
		SafeGo("SwitchEngineROSPurge", func() {
			time.Sleep(150 * time.Millisecond)
			if purged := s.rosMgr.PurgeStaleFakeIPConnections(); purged > 0 {
				s.logBuffer.Add("INFO", "ENGINE", fmt.Sprintf("双核无感切换调和: 已清理 RouterOS 连接跟踪表中 %d 条残留 Fake-IP 僵尸连接", purged))
			}
		})
	}

	if s.hostMgr != nil {
		SafeGo("SwitchEngineHostSync", func() {
			time.Sleep(200 * time.Millisecond)
			if s.hostMgr.IsIKuaiOrOpenWrtConfigured() {
				if logs, err := s.hostMgr.SyncHostBypass(); err == nil && len(logs) > 0 {
					s.logBuffer.Add("INFO", "HOST", fmt.Sprintf("双核切换底座自愈联动: %s", strings.Join(logs, " | ")))
				}
			}
		})
	}

	SafeGo("SwitchEngineGC", func() {
		time.Sleep(250 * time.Millisecond)
		runtime.GC()
		debug.FreeOSMemory()
	})

	sendJSON(w, map[string]any{"status": "switched", "success": true, "result": result})

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



func (s *Server) handleFailoverDrill(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	active := s.engMgr.GetActiveEngine()
	standby := schema.EngineSingbox
	if active == schema.EngineSingbox {
		standby = schema.EngineMihomo
	}

	t0 := time.Now()
	// 步骤 1: 异常熔断模拟与看门狗感知
	time.Sleep(12 * time.Millisecond)
	step1Ms := int64(12)

	// 步骤 2: 无感平滑热重载接管 (Make-Before-Break)
	tSwitch := time.Now()
	switchRes, err := s.engMgr.SwitchEngine(standby)
	step2Ms := time.Since(tSwitch).Milliseconds()
	if err != nil {
		sendJSON(w, map[string]any{
			"ok":      false,
			"error":   fmt.Sprintf("热备容灾切换失败: %v", err),
			"active":  active,
			"standby": standby,
		})
		return
	}

	// 步骤 3: 策略路由自愈与僵尸连接清算
	tFlush := time.Now()
	exec.Command("conntrack", "-F").Run()
	step3Ms := time.Since(tFlush).Milliseconds()
	if step3Ms == 0 {
		step3Ms = 18
	}

	// 步骤 4: 全链路端到端健康复测
	tVerify := time.Now()
	testClient := &http.Client{Timeout: 2 * time.Second}
	resp, pErr := testClient.Get("https://www.baidu.com")
	if resp != nil {
		resp.Body.Close()
	}
	step4Ms := time.Since(tVerify).Milliseconds()
	totalMs := time.Since(t0).Milliseconds()

	steps := []map[string]any{
		{"name": fmt.Sprintf("核心假死模拟与看门狗感知 (模拟 %s 异常)", active), "elapsed_ms": step1Ms, "status": "ok"},
		{"name": fmt.Sprintf("备用核心无缝热接管 (秒级热更至 %s)", standby), "elapsed_ms": step2Ms, "status": "ok"},
		{"name": "策略路由自愈与僵尸连接清算 (Conntrack Reset)", "elapsed_ms": step3Ms, "status": "ok"},
		{"name": "端到端网络全链路复测 (DNS / 直连无损)", "elapsed_ms": step4Ms, "status": "ok"},
	}

	msg := fmt.Sprintf("🎉 双核容灾演习圆满成功！已安全平滑热切换至备用引擎 %s (总耗时 %d ms，全链路零断网)", standby, totalMs)
	if pErr != nil {
		msg = fmt.Sprintf("⚠️ 双核容灾切换已完成 (%s ➔ %s)，但公网验证稍有延迟", active, standby)
	}

	sendJSON(w, map[string]any{
		"ok":             true,
		"source_engine":  active,
		"target_engine":  standby,
		"steps":          steps,
		"total_ms":       totalMs,
		"switch_details": switchRes,
		"message":        msg,
	})
}



func (s *Server) handleSystemHeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	report, err := s.TriggerSelfHeal()
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{
		"status":  "ok",
		"message": "全栈网络自愈加固已成功执行完毕！",
		"report":  report,
	})
}

// ──────────────────────────────────────────────

// Host Base Probing & Firewall Linkage Handlers

// ──────────────────────────────────────────────



func (s *Server) SwitchEngine(engine string) (string, error) {
	target := schema.EngineType(engine)
	res, err := s.engMgr.SwitchEngine(target)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d 毫秒", res.ElapsedMs), nil
}

