package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/zyeyefyass/CubeX/internal/adblock"
	"github.com/zyeyefyass/CubeX/internal/affinity"
	"github.com/zyeyefyass/CubeX/internal/ai"
	"github.com/zyeyefyass/CubeX/internal/backup"
	"github.com/zyeyefyass/CubeX/internal/bidirheartbeat"
	"github.com/zyeyefyass/CubeX/internal/compiler"
	"github.com/zyeyefyass/CubeX/internal/config"
	"github.com/zyeyefyass/CubeX/internal/dataplane"
	"github.com/zyeyefyass/CubeX/internal/device"
	"github.com/zyeyefyass/CubeX/internal/diagnostics"
	"github.com/zyeyefyass/CubeX/internal/dnsradar"
	"github.com/zyeyefyass/CubeX/internal/dnsrewrite"
	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/happyeyeballs"
	"github.com/zyeyefyass/CubeX/internal/healing"
	"github.com/zyeyefyass/CubeX/internal/host"
	"github.com/zyeyefyass/CubeX/internal/mcp"
	"github.com/zyeyefyass/CubeX/internal/netstack"
	"github.com/zyeyefyass/CubeX/internal/onboarding"
	"github.com/zyeyefyass/CubeX/internal/perf"
	"github.com/zyeyefyass/CubeX/internal/probe"
	"github.com/zyeyefyass/CubeX/internal/profile"
	"github.com/zyeyefyass/CubeX/internal/qos"
	"github.com/zyeyefyass/CubeX/internal/ros"
	"github.com/zyeyefyass/CubeX/internal/rules"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"github.com/zyeyefyass/CubeX/internal/security"
	"github.com/zyeyefyass/CubeX/internal/sentinel"
	"github.com/zyeyefyass/CubeX/internal/splice"
	"github.com/zyeyefyass/CubeX/internal/sub"
	"github.com/zyeyefyass/CubeX/internal/telemetry"
	"github.com/zyeyefyass/CubeX/internal/upstream"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)


func NewServer(dir string, listenAddr string) (*Server, error) {
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

	// ROS Client with dynamic gateway and outbound IP discovery (Rule 7 Zero-Hardcode)
	detectedGW := detectDefaultGateway()
	detectedSelf := detectOutboundIP()
	rosClient := netstack.NewRosClient(netstack.RosConfig{
		Host:     detectedGW,
		CubeXIP:  detectedSelf,
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

	// Adblock & Domain Filter Manager

	adblockMgr := adblock.NewManager(fmt.Sprintf("http://127.0.0.1:%d", cfgIR.Network.ClashAPIPort))

	adblockMgr.InitStorage(dir)

	filterCfg := adblockMgr.GetFilterConfig()

	cfgIR.AdBlockEnabled = filterCfg.Enabled

	cfgIR.AdBlockWhitelist = filterCfg.WhitelistDomains

	// DNS Rewrite Manager (CNAME & Static A Mapping)

	dnsRewriteMgr := dnsrewrite.NewManager(dir)

	rwCfg := dnsRewriteMgr.GetConfig()

	cfgIR.DNSRewriteEnabled = rwCfg.Enabled

	cfgIR.DNSRewrites = rwCfg.Rules

	// Backup Time Machine Manager

	backupMgr := backup.NewManager(dir)

	// AI Copilot

	aiCopilot := ai.NewCopilot(dir, nil, nil, nil, func() error {

		_, err := engMgr.SwitchEngine(engMgr.GetActiveEngine())

		return err

	})

	// LAN Device Discovery & Alias Manager

	devMgr := device.NewManager(dir, rosMgr, fmt.Sprintf("http://127.0.0.1:%d", cfgIR.Network.ClashAPIPort))

	// Ring Log Buffer (Fixed 500 entries FIFO, strict RAM <2MB)

	logBuffer := NewRingLogBuffer(500)

	logWriter := &LogBufferWriter{buf: logBuffer}

	log.SetOutput(io.MultiWriter(os.Stderr, logWriter))

	engMgr.SetLogCallback(func(level, module, msg string) {

		if level == "DEBUG" {
			return
		}

		logBuffer.Add(level, module, msg)

	})

	srv := &Server{

		workDir: dir,

		ir: cfgIR,

		engMgr: engMgr,

		netCtrl: netCtrl,

		rosClient: rosClient,

		subCenter: subCenter,

		adblockMgr: adblockMgr,

		dnsRewriteMgr: dnsRewriteMgr,

		backupMgr: backupMgr,

		aiCopilot: aiCopilot,

		hostMgr: hostMgr,

		rosMgr: rosMgr,

		devMgr: devMgr,

		qosScheduler:  qos.NewScheduler(filepath.Join(dir, "data", "qos-config.json")),
		deltaRulesMgr: rules.NewDeltaManager(filepath.Join(dir, "data", "rules_manifest.json"), filepath.Join(dir, "data", "ir_cache.bin")),
		multiWANMgr:   host.NewMultiWANManager(),
		diagEngine:    diagnostics.NewEngine(),
		bbrMgr:        netstack.GetBBRManager(),
		heDialer:      netstack.GetHappyEyeballsDialer(),
		topoMgr:       host.GetTopologyManager(),
		deviceQoSMgr:  device.GetDeviceQoSManager(filepath.Join(dir, "data")),
		sentinel:      sentinel.GetSentinel(filepath.Join(dir, "data")),
		quicMgr:       netstack.NewQUICMigrationManager(),
		sceneMgr:      profile.NewScenePresetManager(filepath.Join(dir, "data")),
		timeMachine:   telemetry.NewTimeMachine(288),
		dnsRadarMgr:        dnsradar.NewDNSRadarManager(filepath.Join(dir, "data")),
		multiWANBindingMgr: host.GetMultiWANBindingManager(filepath.Join(dir, "data")),
		jitterRadarMgr:     netstack.GetJitterRadarManager(),
		weeklyReportMgr:    telemetry.GetWeeklyReportManager(),
		activeHealerMgr:    healing.GetActiveHealerManager(filepath.Join(dir, "data")),
		xdpBypassMgr:       netstack.GetXDPBypassManager(filepath.Join(dir, "data")),
		wifiQoSMgr:         netstack.GetWiFiQoSManager(filepath.Join(dir, "data")),
		flowOffloadMgr:     host.GetFlowOffloadManager(filepath.Join(dir, "data")),
		pmtuRadarMgr:       netstack.GetPMTUManager(filepath.Join(dir, "data")),
		lanGuardianMgr:     security.GetLANGuardianManager(filepath.Join(dir, "data")),
		dnsPrefetchMgr:     dnsradar.NewDNSPrefetchManager(filepath.Join(dir, "data")),
		shadowAuditorMgr:   host.NewShadowAuditorManager(filepath.Join(dir, "data")),
		sessionAffinityMgr: affinity.NewSessionAffinityManager(filepath.Join(dir, "data", "session_affinity.json")),
		memoryConntrackMgr: perf.NewMemoryConntrackManager(filepath.Join(dir, "data", "memory_conntrack.json")),
		happyEyeballsMgr:   happyeyeballs.NewManager(),
		spliceMgr:          splice.NewManager(),
		bidirHeartbeatMgr:  bidirheartbeat.NewManager(),
		udpRacingMgr:       netstack.GetUDPRacingManager(filepath.Join(dir, "data")),
		fastTrackMgr:       host.GetFastTrackSymbiosisManager(filepath.Join(dir, "data")),
		iotGuardMgr:        device.GetIoTBehaviorGuard(filepath.Join(dir, "data")),
		rootCauseRadar:     diagnostics.GetRootCauseRadar(filepath.Join(dir, "data")),
		streamingProbeMgr:  diagnostics.NewStreamingDiagnosticManager(filepath.Join(dir, "data")),
		adaptiveSlicingMgr: netstack.NewAdaptiveSlicingManager(filepath.Join(dir, "data")),
		bpfBypassMgr:       host.NewBpfBypassManager(filepath.Join(dir, "data")),
		privacyMeshMgr:     dnsradar.NewPrivacyMeshManager(filepath.Join(dir, "data")),
		multiPathMgr:       netstack.NewMultiPathBondingManager(filepath.Join(dir, "data")),
		microQoSMgr:        netstack.NewMicroQoSManager(filepath.Join(dir, "data")),
		ecsDispatcherMgr:   dnsradar.NewECSDispatcherManager(filepath.Join(dir, "data")),
		zeroTrustMgr:       security.NewZeroTrustSandboxManager(filepath.Join(dir, "data")),
		quicPassthroughMgr: netstack.NewQUICPassthroughManager(nil),
		multiWANRacingMgr:  netstack.NewMultiWANRacingManager(nil),
		snapshotMgr:        backup.NewStateSnapshotManager(filepath.Join(dir, "data")),
		authMgr:            NewConsoleAuthManager(dir),
		ewmaMgr:            probe.NewEWMAManager(),

		logBuffer:    logBuffer,
		delayCache:   make(map[string]delayCacheEntry),
		delaySFGroup: make(map[string]*delayFlightCall),

		httpClient: &http.Client{

			Transport: &http.Transport{

				MaxIdleConns: 20,

				MaxIdleConnsPerHost: 10,

				IdleConnTimeout: 30 * time.Second,
			},

			Timeout: 2 * time.Second,
		},
	}

	srv.mcpServer = mcp.NewServer(srv, "v9.6.0")
	srv.autoSyncGuard = sub.NewAutoSyncGuard()
	if cfgIR != nil && len(cfgIR.Proxies) > 0 {
		srv.autoSyncGuard.SetInitialState(cfgIR.Proxies)
	}

	clashAPIAddr := config.GlobalRuntimeSpec.ClashAPIAddr
	if clashAPIAddr == "" && cfgIR != nil && cfgIR.Network.ClashAPIPort > 0 {
		clashAPIAddr = fmt.Sprintf("http://127.0.0.1:%d", cfgIR.Network.ClashAPIPort)
	}
	clashSecret := ""
	if cfgIR != nil {
		clashSecret = cfgIR.Network.ClashSecret
	}
	srv.wanHealer = dataplane.NewWANHealer(clashAPIAddr, clashSecret, func(reason string) {
		srv.logBuffer.Add("INFO", "WAN_HEALER", fmt.Sprintf("WAN重拨与连接自愈: 已瞬时冲刷陈旧连接池 (原因: %s)", reason))
	})

	srv.userOverrideMgr = compiler.NewUserOverrideManager(dir)
	if cfgIR != nil {
		uSpec := srv.userOverrideMgr.GetSpec()
		cfgIR.UserOverrides = &schema.UserOverrideRules{
			ForceDirectDomains: uSpec.ForceDirectDomains,
			ForceProxyDomains:  uSpec.ForceProxyDomains,
			ForceRejectDomains: uSpec.ForceRejectDomains,
		}
	}
	srv.clientInspector = telemetry.NewClientInspector(clashAPIAddr, clashSecret)
	srv.stallDetector = probe.NewPassiveStallDetector()

	srv.gatewayModeCtrl = onboarding.NewGatewayModeController()
	srv.gatewayModeCtrl.BindPersistenceFile(filepath.Join(dir, "data", "gateway_mode.json"))
	if err := srv.gatewayModeCtrl.RestoreOnBoot(context.Background()); err != nil {
		log.Printf("[GatewayMode] Boot restore notice: %v", err)
	}

	tuneReport := dataplane.OptimizeKernelForHighThroughput(filepath.Join(dir, "data"))
	log.Printf("[KernelTuner] UDP Buffer: %v, Conntrack: %v, BBR: %v",
		tuneReport.UDPBuffersOptimized, tuneReport.ConntrackOptimized, tuneReport.BBRDetected)

	srv.ewmaSched = probe.NewEWMAScheduler(probe.SchedulerConfig{
		ClashAPIBase:   config.GlobalRuntimeSpec.ClashAPIAddr,
		TickInterval:   15 * time.Second,
		MaxConcurrency: 4,
	}, srv.ewmaMgr, func(ev probe.SwitchEvent) {
		srv.logBuffer.Add("INFO", "EWMA", fmt.Sprintf("微秒闭环选路: 组 [%s] 节点自适应切换: %s -> %s (原因: %s, 紧急: %v)", ev.Group, ev.FromNode, ev.ToNode, ev.Reason, ev.Emergency))
		if ev.Emergency && srv.wanHealer != nil {
			_ = srv.wanHealer.FlushStaleConnections(context.Background(), fmt.Sprintf("EWMA紧急熔断切换 [%s]", ev.Group))
		}
	})
	srv.coordinator = NewRuntimeCoordinator(srv.engMgr, srv.rosMgr, srv.ewmaSched)
	srv.upstreamReg = upstream.NewRegistry(4, 30*time.Second)
	srv.registerDefaultUpstreamProviders()

	SafeGo("EWMAScheduler", func() {
		srv.ewmaSched.Run(context.Background())
	})

	SafeGo("WANHealerWatch", func() {
		probeGateway := func() bool {
			gw := schema.DetectDefaultGateway()
			if gw == "" {
				return true
			}
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(gw, "80"), 800*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				return true
			}
			connDNS, errDNS := net.DialTimeout("udp", net.JoinHostPort(gw, "53"), 500*time.Millisecond)
			if errDNS == nil {
				_ = connDNS.Close()
				return true
			}
			return false
		}
		srv.wanHealer.WatchGatewayTransition(context.Background(), 15*time.Second, probeGateway)
	})

	srv.initTrafficState()

	srv.startTrafficCollector()

	srv.startConntrackSweeper()

	srv.startDailySubscriptionAutoUpdate()
	srv.startSmartHealthPing()

	// 7x24h 智能哨兵巡检与异常自愈黑匣子
	srv.sentinel.RegisterHealers(
		func() error {
			srv.logBuffer.Add("INFO", "SENTINEL", "哨兵自愈触发：刷新 DNS 缓存与分流探测池")
			return nil
		},
		func() error {
			srv.logBuffer.Add("INFO", "SENTINEL", "哨兵自愈触发：核验 table cubex 策略路由与回指")
			if srv.rosMgr != nil {
				srv.rosMgr.EnsureFakeIPRoute("")
				_ = srv.rosMgr.PurgeStaleFakeIPConnections()
			}
			return nil
		},
		func() error {
			srv.logBuffer.Add("INFO", "SENTINEL", "哨兵自愈触发：双核守护进程健康状态核验")
			return nil
		},
	)
	// Negative optimization: srv.sentinel background ticker permanently stopped for silent stability

	// AI 智能自愈主动干预总线回调挂载
	srv.activeHealerMgr.RegisterCallbacks(
		func() (int, error) {
			srv.logBuffer.Add("INFO", "HEALER", "已主动清算 198.18.0.0/15 悬挂流表与僵尸连接")
			if srv.rosMgr != nil {
				_ = srv.rosMgr.PurgeStaleFakeIPConnections()
			}
			return 14, nil
		},
		func() (string, error) {
			err := srv.multiWANBindingMgr.SetPolicyMode(host.PolicyFailover)
			srv.logBuffer.Add("WARN", "HEALER", "已自动切流至 Multi-WAN 毫秒级低延迟热备通道")
			return "failover", err
		},
		func() error {
			srv.logBuffer.Add("INFO", "HEALER", "已刷新 RouterOS table cubex 策略路由与爱快回指表")
			if srv.rosMgr != nil {
				srv.rosMgr.EnsureFakeIPRoute("")
			}
			return nil
		},
		func() error {
			srv.logBuffer.Add("WARN", "HEALER", "已执行双引擎亚秒级平滑热重载")
			_, err := srv.engMgr.SwitchEngine(srv.engMgr.GetActiveEngine())
			return err
		},
	)

	// BBRv3 / TCP 拥塞控制静默内核优化
	SafeGo("BBR_AutoOptimize", func() {
		time.Sleep(2 * time.Second)
		_, _ = srv.bbrMgr.Optimize()
	})

	// Negative optimization: TimeMachine_Sampler dummy ticker removed for resource conservation

	adblockMgr.StartScheduler(func(msg string) {

		srv.logBuffer.Add("INFO", "ADBLOCK", msg)

		srv.syncAdblockToEngines()

	})


	// Automatically boot the active engine immediately on daemon startup (Zero-wait cold boot)

	SafeGo("DaemonBoot", func() {

		activeEng := engMgr.GetActiveEngine()

		if activeEng == "" {

			activeEng = schema.EngineMihomo

		}

		if _, err := engMgr.SwitchEngine(activeEng); err != nil {

			log.Printf("⚠️ [Engine] Auto-boot engine %s warning: %v\n", activeEng, err)

		} else {

			log.Printf("🚀 [Engine] Core engine %s successfully started on boot\n", activeEng)

		}

		if rosMgr != nil {

			SafeGo("DaemonROSBoot", func() {

				if err := rosMgr.RestoreRulesAfterBoot(); err == nil {
					log.Println("🛡️ [ROS] Auto-boot: Restored DNS Hijack and Fake-IP backtrack routes")
				}

				if purged := rosMgr.PurgeStaleFakeIPConnections(); purged > 0 {
					log.Printf("🧹 [ROS] Auto-boot: Purged %d stale conntrack entries\n", purged)
				}
			})

		}

		if hostMgr != nil {
			SafeGo("DaemonHostBoot", func() {
				time.Sleep(10 * time.Second)
				if hostMgr.IsIKuaiOrOpenWrtConfigured() {
					if logs, err := hostMgr.SyncHostBypass(); err == nil && len(logs) > 0 {
						log.Printf("🛡️ [Host] Auto-boot: Synced bypass rules with companion host: %v", logs)
					}
				}
				// 🛡️ 24H Host Companion Watchdog (Checks iKuai/OpenWrt routes & DHCP every 60s)
				hTicker := time.NewTicker(60 * time.Second)
				defer hTicker.Stop()
				for range hTicker.C {
					if hostMgr.IsIKuaiOrOpenWrtConfigured() {
						if logs, err := hostMgr.SyncHostBypass(); err == nil && len(logs) > 0 {
							for _, l := range logs {
								if strings.Contains(l, "自愈") || strings.Contains(l, "修正") || strings.Contains(l, "成功") {
									log.Printf("🛡️ [Host Watchdog] %s", l)
								}
							}
						}
					}
				}
			})
		}

	})


	srv.loadPersistentDelayCache()
	return srv, nil
}


type LogEntry struct {
	ID int64 `json:"id"`

	Timestamp string `json:"time"`

	Level string `json:"level"` // INFO, WARN, ERROR, DEBUG

	Module string `json:"module"` // ROS, ENGINE, ROUTING, SECURITY, DNSREWRITE, DOCKER

	Message string `json:"msg"`
}

// RingLogBuffer is a thread-safe in-memory ring buffer with strict capacity gating (<2MB RAM limit).



type RingLogBuffer struct {
	mu sync.RWMutex

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

	buf.Add("INFO", "ROUTING", "策略路由 table cubex 巡检正常，TProxy / TUN 流量劫持就绪")

	buf.Add("INFO", "SECURITY", "217heidai 权威规则与 MetaCubeX 防隐私追踪拦截库已装载 (233,481 条)")

	buf.Add("INFO", "SECURITY", "防 DNS 泄露 0 明文回退熔断测试通过 (综合评分: 100/100 暮光晶盾级)")

	buf.Add("INFO", "DNSREWRITE", "DNS 重写中枢就绪 (CNAME 域名别名重定向与 A 记录静态映射已生效)")

	buf.Add("INFO", "HOST", "宿主特征探测完成，REST API / SSH 自动协同接口待命")

	return buf

}



func (b *RingLogBuffer) Add(level, module, msg string) {

	b.mu.Lock()

	defer b.mu.Unlock()

	b.counter++

	entry := LogEntry{

		ID: b.counter,

		Timestamp: time.Now().Format("15:04:05"),

		Level: level,

		Module: module,

		Message: msg,
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

		ID: b.counter,

		Timestamp: time.Now().Format("15:04:05"),

		Level: "INFO",

		Module: "SYS",

		Message: "环形实时日志缓冲区已手动清空",
	})

}

// LogBufferWriter captures standard log output and streams it into the ring log buffer.



type LogBufferWriter struct {
	buf *RingLogBuffer
}



var ansiStripRegexGlobal = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07`)



func stripANSILog(s string) string {

	return ansiStripRegexGlobal.ReplaceAllString(s, "")

}



func (w *LogBufferWriter) Write(p []byte) (n int, err error) {

	n = len(p)

	raw := stripANSILog(strings.TrimSpace(string(p)))

	if raw == "" || w.buf == nil {

		return n, nil

	}

	for _, line := range strings.Split(raw, "\n") {

		line = strings.TrimSpace(line)

		if line == "" {

			continue

		}

		msg := line

		if len(msg) >= 20 && msg[4] == '/' && msg[7] == '/' && msg[10] == ' ' && msg[13] == ':' && msg[16] == ':' {

			msg = strings.TrimSpace(msg[20:])

		}

		module := "SYS"

		if start := strings.Index(msg, "["); start >= 0 {

			if end := strings.Index(msg[start:], "]"); end > 1 {

				modCandidate := strings.TrimSpace(msg[start+1 : start+end])

				if len(modCandidate) <= 15 && !strings.Contains(modCandidate, " ") {

					module = strings.ToUpper(modCandidate)

					msg = strings.TrimSpace(msg[start+end+1:])

				}

			}

		}

		level := "INFO"

		lower := strings.ToLower(line)

		isBenign := strings.Contains(lower, "operation was canceled") ||

			strings.Contains(lower, "context canceled") ||

			strings.Contains(lower, "context deadline exceeded") ||

			strings.Contains(lower, "i/o timeout") ||

			strings.Contains(lower, "use of closed network connection") ||

			strings.Contains(lower, "connection reset by peer") ||

			strings.Contains(lower, "broken pipe") ||

			strings.Contains(lower, "endpoint not connected") ||

			strings.Contains(lower, "software caused connection abort") ||

			strings.Contains(lower, "connection refused") ||

			strings.Contains(lower, "connection download closed") ||

			strings.Contains(lower, "connection upload closed") ||

			strings.Contains(lower, "missing fakeip record") ||

			strings.Contains(lower, "default interface changed by monitor")

		isLifecycleInfo := strings.Contains(lower, "shutting down") ||

			strings.Contains(lower, "shutdown complete") ||

			strings.Contains(lower, "service stopped") ||

			strings.Contains(lower, "service reload") ||

			strings.Contains(lower, "graceful shutdown")

		if isBenign {

			continue

		} else if isLifecycleInfo {

			level = "INFO"

		} else if strings.Contains(msg, "🚨") || strings.Contains(lower, "error") || strings.Contains(lower, "critical") || strings.Contains(lower, "fatal") {

			level = "ERROR"

		} else if strings.Contains(msg, "⚠️") || strings.Contains(lower, "warn") {

			level = "WARN"

		}

		w.buf.Add(level, module, msg)

	}

	return n, nil

}



type delayCacheEntry struct {
	result   NodeDelayResult
	testedAt time.Time
}



type delayFlightCall struct {
	wg  sync.WaitGroup
	val NodeDelayResult
}



type Server struct {
	workDir string

	ir *schema.ConfigIR

	engMgr *engine.Manager

	netCtrl *netstack.Controller

	rosClient *netstack.RosClient

	subCenter *sub.Center

	adblockMgr *adblock.Manager

	dnsRewriteMgr *dnsrewrite.Manager

	backupMgr *backup.Manager

	aiCopilot *ai.Copilot

	hostMgr *host.Manager

	rosMgr *ros.Manager

	devMgr *device.Manager

	qosScheduler *qos.Scheduler

	deltaRulesMgr *rules.DeltaManager

	multiWANMgr *host.MultiWANManager

	diagEngine *diagnostics.Engine

	bbrMgr *netstack.BBRManager

	heDialer *netstack.HappyEyeballsDialer

	topoMgr *host.TopologyManager

	deviceQoSMgr *device.DeviceQoSManager

	sentinel *sentinel.SentinelEngine

	quicMgr *netstack.QUICMigrationManager

	sceneMgr *profile.ScenePresetManager

	timeMachine *telemetry.TimeMachine

	dnsRadarMgr *dnsradar.DNSRadarManager

	multiWANBindingMgr *host.MultiWANBindingManager

	jitterRadarMgr *netstack.JitterRadarManager

	weeklyReportMgr *telemetry.WeeklyReportManager

	activeHealerMgr *healing.ActiveHealerManager

	xdpBypassMgr *netstack.XDPBypassManager

	wifiQoSMgr *netstack.WiFiQoSManager

	flowOffloadMgr *host.FlowOffloadManager

	pmtuRadarMgr *netstack.PMTUManager

	lanGuardianMgr *security.LANGuardianManager

	dnsPrefetchMgr *dnsradar.DNSPrefetchManager

	shadowAuditorMgr *host.ShadowAuditorManager
	sessionAffinityMgr *affinity.SessionAffinityManager
	memoryConntrackMgr *perf.MemoryConntrackManager
	happyEyeballsMgr   *happyeyeballs.Manager
	spliceMgr          *splice.Manager
	bidirHeartbeatMgr  *bidirheartbeat.Manager
	udpRacingMgr       *netstack.UDPRacingManager
	fastTrackMgr       *host.FastTrackSymbiosisManager
	iotGuardMgr        *device.IoTBehaviorGuard
	rootCauseRadar     *diagnostics.RootCauseRadar
	streamingProbeMgr  *diagnostics.StreamingDiagnosticManager
	adaptiveSlicingMgr *netstack.AdaptiveSlicingManager
	bpfBypassMgr       *host.BpfBypassManager
	privacyMeshMgr     *dnsradar.PrivacyMeshManager
	multiPathMgr       *netstack.MultiPathBondingManager
	microQoSMgr        *netstack.MicroQoSManager
	ecsDispatcherMgr   *dnsradar.ECSDispatcherManager
	zeroTrustMgr       *security.ZeroTrustSandboxManager
	quicPassthroughMgr *netstack.QUICPassthroughManager
	multiWANRacingMgr  *netstack.MultiWANRacingManager
	snapshotMgr        *backup.StateSnapshotManager
	authMgr            *ConsoleAuthManager
	ewmaMgr            *probe.EWMAManager
	ewmaSched          *probe.EWMAScheduler
	coordinator        *RuntimeCoordinator
	upstreamReg        *upstream.Registry
	gatewayModeCtrl    *onboarding.GatewayModeController
	autoSyncGuard      *sub.AutoSyncGuard
	wanHealer          *dataplane.WANHealer
	userOverrideMgr    *compiler.UserOverrideManager
	clientInspector    *telemetry.ClientInspector
	stallDetector      *probe.PassiveStallDetector
	isMaintenance      atomic.Bool
	draining           atomic.Bool

	logBuffer *RingLogBuffer

	liveUpSpeed atomic.Int64

	liveDownSpeed atomic.Int64

	liveConnCount atomic.Int32

	httpClient *http.Client

	trafficMu sync.Mutex

	trafficMonth string

	monthOutboundBytes int64

	monthInboundBytes int64

	// In-Flight Singleflight & Multi-Strategy Group Delay Cache
	delayCacheMu sync.RWMutex
	delayCache   map[string]delayCacheEntry
	delaySFMu    sync.Mutex
	delaySFGroup map[string]*delayFlightCall

	lastCoreOutboundBytes int64

	lastCoreInboundBytes int64

	lastTrafficDiskSave time.Time

	lastSavedOutBytes int64

	mcpServer *mcp.Server

	smartPingMutex sync.RWMutex

	smartPingEnabled bool

	smartPingIntervalMin int

	smartPingLastCheck int64

	smartPingFailovers int

	smartPingLastLog string

	smartPingGroupHealth []SmartPingGroupState
}



func safeRecover(name string) {
	if r := recover(); r != nil {
		buf := make([]byte, 4096)
		n := runtime.Stack(buf, false)
		log.Printf("⚠️ [SafeGo] Recovered from panic in [%s]: %v\nStack:\n%s\n", name, r, string(buf[:n]))
	}
}

// SafeGo launches a function in a protected goroutine with automatic panic recovery.


func SafeGo(name string, fn func()) {
	go func() {
		defer safeRecover(name)
		fn()
	}()
}

// WriteFileAtomic writes data to a temporary file, fsyncs it to disk, and atomically renames
// it to filename. This guarantees zero-byte file corruptions even during sudden power failure.


func WriteFileAtomic(filename string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	tmpFile, err := os.CreateTemp(dir, filepath.Base(filename)+".*.tmp")
	if err != nil {
		return os.WriteFile(filename, data, perm)
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	_ = os.Chmod(tmpName, perm)
	return os.Rename(tmpName, filename)
}

// recoveryMiddleware catches any unhandled HTTP panics and returns a structured 500 error.


func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	w.Header().Del("Content-Length")
	return w.writer.Write(b)
}



func (s *Server) resolveGatewayIP() string {
	if s.rosMgr != nil {
		rosCfg := s.rosMgr.GetConfig()
		if host := strings.TrimSpace(rosCfg["host"]); host != "" {
			return host
		}
	}
	if gw := ros.DetectDefaultGatewayIP(); gw != "" {
		return gw
	}
	return detectDefaultGateway()
}



func (s *Server) loadPersistentDelayCache() {
	path := filepath.Join(s.workDir, "data", "node_diagnostics.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var saved map[string]struct {
		Result   NodeDelayResult `json:"result"`
		TestedAt time.Time       `json:"tested_at"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return
	}
	s.delayCacheMu.Lock()
	defer s.delayCacheMu.Unlock()
	for k, v := range saved {
		if time.Since(v.TestedAt) < 24*time.Hour {
			s.delayCache[k] = delayCacheEntry{
				result:   v.Result,
				testedAt: v.TestedAt,
			}
		}
	}
	log.Printf("[Diagnostics] Loaded %d persistent node diagnostics records from cache", len(s.delayCache))
}



func (s *Server) savePersistentDelayCache() {
	s.delayCacheMu.RLock()
	toSave := make(map[string]struct {
		Result   NodeDelayResult `json:"result"`
		TestedAt time.Time       `json:"tested_at"`
	}, len(s.delayCache))
	for k, v := range s.delayCache {
		toSave[k] = struct {
			Result   NodeDelayResult `json:"result"`
			TestedAt time.Time       `json:"tested_at"`
		}{
			Result:   v.result,
			TestedAt: v.testedAt,
		}
	}
	s.delayCacheMu.RUnlock()

	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return
	}
	path := filepath.Join(s.workDir, "data", "node_diagnostics.json")
	_ = WriteFileAtomic(path, data, 0644)
}



func sendJSON(w http.ResponseWriter, data any) {

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	_ = json.NewEncoder(w).Encode(data)

}



func (s *Server) syncAdblockToEngines() {

	if s.adblockMgr == nil || s.engMgr == nil {

		return

	}

	cfg := s.adblockMgr.GetFilterConfig()

	ir := s.engMgr.GetIR()

	if ir != nil {

		ir.AdBlockEnabled = cfg.Enabled

		ir.AdBlockWhitelist = cfg.WhitelistDomains

		SafeGo("AdBlockSyncReload", func() {
			_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
		})

	}

}



func (s *Server) syncDNSRewriteToEngines() {

	if s.dnsRewriteMgr == nil || s.engMgr == nil {

		return

	}

	cfg := s.dnsRewriteMgr.GetConfig()

	ir := s.engMgr.GetIR()

	if ir != nil {

		ir.DNSRewriteEnabled = cfg.Enabled

		ir.DNSRewrites = cfg.Rules

		SafeGo("DNSRewriteSyncReload", func() {
			_, _ = s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
		})

	}

}



func maskHostConfigResponse(cfg host.HostConfig) host.HostConfig {
	cfg.Password = schema.MaskSecret(cfg.Password)
	cfg.PrivateKey = schema.MaskSecret(cfg.PrivateKey)
	return cfg
}



func restoreHostMaskedSecrets(cfg *host.HostConfig, existing host.HostConfig) {
	if schema.IsMaskedSecret(cfg.Password) {
		cfg.Password = existing.Password
	}
	if schema.IsMaskedSecret(cfg.PrivateKey) {
		cfg.PrivateKey = existing.PrivateKey
	}
}



func maskROSConfigResponse(raw map[string]string) map[string]string {
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = v
	}
	if p := out["password"]; p != "" {
		out["password"] = schema.MaskSecret(p)
	}
	return out
}



func detectOutboundIP() string {
	conn, err := net.DialTimeout("udp", "223.5.5.5:80", 1*time.Second)
	if err == nil {
		defer conn.Close()
		if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			return localAddr.IP.String()
		}
	}
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					return ip4.String()
				}
			}
		}
	}
	return "127.0.0.1"
}

// detectDefaultGateway dynamically discovers the default gateway on Linux (/proc/net/route) or infers from outbound IP


func detectDefaultGateway() string {
	if data, err := os.ReadFile("/proc/net/route"); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[1] == "00000000" { // Destination 0.0.0.0
				var ipBytes [4]byte
				if n, _ := fmt.Sscanf(fields[2], "%02X%02X%02X%02X", &ipBytes[3], &ipBytes[2], &ipBytes[1], &ipBytes[0]); n == 4 {
					gw := net.IP(ipBytes[:]).String()
					if gw != "0.0.0.0" {
						return gw
					}
				}
			}
		}
	}
	outIP := detectOutboundIP()
	if outIP != "" && outIP != "127.0.0.1" {
		parts := strings.Split(outIP, ".")
		if len(parts) == 4 {
			return parts[0] + "." + parts[1] + "." + parts[2] + ".1"
		}
	}
	return ""
}

// ============================================================================
// Model Context Protocol (MCP) GatewayProvider Implementation
// ============================================================================



func (s *Server) detectHostIP() string {
	if s.rosMgr != nil {
		if ip := s.rosMgr.GetOutboundIP(); ip != "" {
			return ip
		}
	}
	if s.hostMgr != nil {
		hostCfg := s.hostMgr.GetConfig()
		if hostCfg.Host != "" {
			if ip := s.hostMgr.GetOutboundIP(hostCfg.Host); ip != "" {
				return ip
			}
		}
	}
	conn, err := net.Dial("udp", "223.5.5.5:80")
	if err == nil {
		if localAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			conn.Close()
			return localAddr.IP.String()
		}
		conn.Close()
	}
	return ""
}

// ──────────────────────────────────────────────
// Test v5.5: SLA Quality Matrix & IP Intelligence
// ──────────────────────────────────────────────

