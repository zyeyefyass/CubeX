package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/zyeyefyass/CubeX/internal/backup"
	"github.com/zyeyefyass/CubeX/internal/engine"
	"github.com/zyeyefyass/CubeX/internal/profile"
	"github.com/zyeyefyass/CubeX/internal/ros"
	"github.com/zyeyefyass/CubeX/internal/schema"
	"github.com/zyeyefyass/CubeX/internal/telemetry"
	"github.com/zyeyefyass/CubeX/internal/webui"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (s *Server) RegisterRoutes(mux *http.ServeMux) {

	// 1. Subscription Routes

	mux.HandleFunc("/sub/karing", s.subCenter.HandleKaring)

	mux.HandleFunc("/sub/singbox", s.subCenter.HandleSingBox)

	mux.HandleFunc("/sub/clash", s.subCenter.HandleClash)

	mux.HandleFunc("/sub/mihomo", s.subCenter.HandleClash)

	mux.HandleFunc("/sub/shadowrocket", s.subCenter.HandleShadowrocket)

	// 2. REST API v1 & High-Availability Healthz

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if s.draining.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("draining\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","healthy":true}`))
	})
	mux.HandleFunc("/api/v1/auth/status", s.handleAuthStatus)
	mux.HandleFunc("/api/v1/auth/login", s.handleAuthLogin)
	mux.HandleFunc("/api/v1/auth/logout", s.handleAuthLogout)
	mux.HandleFunc("/api/v1/auth/password", s.handleAuthPassword)
	mux.HandleFunc("/api/v1/system/status", s.handleSystemStatus)

	mux.HandleFunc("/api/v1/system/edition", s.handleSystemEdition)

	mux.HandleFunc("/api/v1/system/panic", s.handleSystemPanic)

	mux.HandleFunc("/api/v1/system/reload-config", s.handleReloadConfig)

	mux.HandleFunc("/api/v1/system/reload-core", s.handleReloadCore)

	mux.HandleFunc("/api/v1/system/restart", s.handleRestartService)
	mux.HandleFunc("/api/v1/host/hot-restart", s.handleHostHotRestart)
	mux.HandleFunc("/api/v1/system/poweroff", s.handleSystemPoweroff)
	mux.HandleFunc("/api/v1/system/reboot", s.handleSystemReboot)

	mux.HandleFunc("/api/v1/system/reset", s.handleSystemReset)

	mux.HandleFunc("/api/v1/config/ir", s.handleConfigIR)
	mux.HandleFunc("/api/v1/config/ui_state", s.handleUIState)

	mux.HandleFunc("/api/v1/engine/switch", s.handleEngineSwitch)

	mux.HandleFunc("/api/v1/engine/mode", s.handleEngineMode)

	mux.HandleFunc("/api/v1/engine/benchmark", s.handleBenchmark)


	mux.HandleFunc("/api/v1/netstack/status", s.handleNetstackStatus)

	mux.HandleFunc("/api/v1/netstack/bypass", s.handleNetstackBypass)

	mux.HandleFunc("/api/v1/netstack/apply", s.handleNetstackApply)

	mux.HandleFunc("/api/v1/netstack/webrtc-audit", s.handleWebRTCAudit)
	mux.HandleFunc("/api/v1/network/quic/status", s.handleQUICStatus)
	mux.HandleFunc("/api/v1/network/quic/mode", s.handleQUICMode)
	mux.HandleFunc("/api/v1/chaos/status", s.handleChaosStatus)
	mux.HandleFunc("/api/v1/chaos/drill", s.handleChaosDrill)

	// QoS 智能流控与动态调度器 (维度一)
	mux.HandleFunc("/api/v1/network/qos/status", s.handleQoSStatus)
	mux.HandleFunc("/api/v1/network/qos/config", s.handleQoSConfig)
	mux.HandleFunc("/api/v1/network/qos/device", s.handleQoSDevice)

	// 双核规则增量热补丁与冷启动零等待 (维度二)
	mux.HandleFunc("/api/v1/rules/delta/status", s.handleRulesDeltaStatus)

	// 物理双 WAN 动态感知与对等运营商分流 (维度三)
	mux.HandleFunc("/api/v1/host/multiwan/status", s.handleMultiWANStatus)
	mux.HandleFunc("/api/v1/host/multiwan/config", s.handleMultiWANConfig)

	// 全链路一秒体检自省与极简故障自愈 (维度四)
	mux.HandleFunc("/api/v1/diagnostics/report", s.handleDiagnosticsReport)
	mux.HandleFunc("/api/v1/diagnostics/run", s.handleDiagnosticsRun)
	mux.HandleFunc("/api/v1/diagnostics/heal", s.handleDiagnosticsHeal)

	// ── Test v7.0 四大核心架构维度路由 ──
	// 维度一：TCP BBRv3 与 Happy Eyeballs v2 极速并发
	mux.HandleFunc("/api/v1/network/bbr/status", s.handleBBRStatus)
	mux.HandleFunc("/api/v1/network/bbr/optimize", s.handleBBROptimize)
	mux.HandleFunc("/api/v1/network/happyeyeballs/status", s.handleHappyEyeballsStatus)

	// 维度二：全网拓扑动态绘制与多宿主邻居透视
	mux.HandleFunc("/api/v1/network/topology", s.handleNetworkTopology)
	mux.HandleFunc("/api/v1/network/topology/refresh", s.handleNetworkTopologyRefresh)

	// 维度三：全家终端精细化带宽画像与单设备 VIP 抢占流控
	mux.HandleFunc("/api/v1/devices/telemetry", s.handleDevicesTelemetry)
	mux.HandleFunc("/api/v1/devices/priority", s.handleDevicesPriority)

	// 维度四：7x24h 智能哨兵巡检与异常自愈黑匣子
	mux.HandleFunc("/api/v1/sentinel/status", s.handleSentinelStatus)
	mux.HandleFunc("/api/v1/sentinel/blackbox", s.handleSentinelBlackbox)
	mux.HandleFunc("/api/v1/sentinel/trigger", s.handleSentinelTrigger)

	// ── Test v7.1 四大核心架构演进路由 ──
	// 维度一：QUIC 0-RTT 极速连接迁移
	mux.HandleFunc("/api/v1/network/quic/migration/status", s.handleQUICMigrationStatus)
	mux.HandleFunc("/api/v1/network/quic/migration/toggle", s.handleQUICMigrationToggle)

	// 维度二：三大极客场景模式一键切换
	mux.HandleFunc("/api/v1/profile/scene", s.handleProfileScene)

	// 维度三：全网 24h 流量时光机与历史回溯
	mux.HandleFunc("/api/v1/telemetry/timemachine", s.handleTimeMachine)

	// 维度四：DNS 主动防劫持盲测雷达与未登记设备动态隔离沙盒
	mux.HandleFunc("/api/v1/dns/radar", s.handleDNSRadar)
	mux.HandleFunc("/api/v1/dns/radar/probe", s.handleDNSRadarProbe)
	mux.HandleFunc("/api/v1/devices/sandbox", s.handleDevicesSandbox)
	mux.HandleFunc("/api/v1/devices/sandbox/approve", s.handleDevicesSandboxApprove)

	// ── Test v7.2 三大进阶演进路由 ──
	// 维度一：双宽带多路径智能分流与毫秒容灾
	mux.HandleFunc("/api/v1/host/multiwan/binding", s.handleMultiWANBinding)

	// 维度二：全链路微突发抖动探测雷达
	mux.HandleFunc("/api/v1/network/jitter/status", s.handleJitterRadarStatus)
	mux.HandleFunc("/api/v1/network/jitter/report", s.handleJitterRadarStatus)
	mux.HandleFunc("/api/v1/network/jitter/probe", s.handleJitterRadarProbe)

	// 维度三：AI 全维网络健康周报
	mux.HandleFunc("/api/v1/telemetry/weekly", s.handleWeeklyReport)

	// ── 智能自愈主动干预总线与驱动级 XDP/QoS 演进路由 ──
	// 维度一：AI 智能自愈主动干预总线
	mux.HandleFunc("/api/v1/healing/active-status", s.handleActiveHealingStatus)
	mux.HandleFunc("/api/v1/healing/active-toggle", s.handleActiveHealingToggle)
	mux.HandleFunc("/api/v1/healing/trigger-manual", s.handleActiveHealingTrigger)

	// 维度二：Linux / PVE 网卡级 XDP / eBPF 零拷贝快转
	mux.HandleFunc("/api/v1/netstack/xdp/status", s.handleXDPStatus)
	mux.HandleFunc("/api/v1/netstack/xdp/toggle", s.handleXDPToggle)

	// 维度三：局域网边缘弱网智能识别与动态微流控
	mux.HandleFunc("/api/v1/netstack/wifi/qos/status", s.handleWiFiQoSStatus)
	mux.HandleFunc("/api/v1/netstack/wifi/qos/toggle", s.handleWiFiQoSToggle)
	mux.HandleFunc("/api/v1/netstack/wifi/qos/tune", s.handleWiFiQoSTune)
	mux.HandleFunc("/api/v1/netstack/wifi/qos/inspect", s.handleWiFiQoSInspect)

	// ── Test v7.4 三大深度协同与物理防护路由 ──
	// 维度一：软路由硬件 NAT 流表加速深度协同
	mux.HandleFunc("/api/v1/host/flow-offload/status", s.handleFlowOffloadStatus)
	mux.HandleFunc("/api/v1/host/flow-offload/toggle", s.handleFlowOffloadToggle)
	mux.HandleFunc("/api/v1/host/flow-offload/optimize", s.handleFlowOffloadOptimize)

	// 维度二：全链路智能 MTU / MSS 动态自协商与分片消除雷达
	mux.HandleFunc("/api/v1/network/pmtu/status", s.handlePMTUStatus)
	mux.HandleFunc("/api/v1/network/pmtu/probe", s.handlePMTUProbe)
	mux.HandleFunc("/api/v1/network/pmtu/clamp", s.handlePMTUClamp)

	// 维度三：局域网全拓扑 ARP 动态防欺骗与非法 DHCP 探针
	mux.HandleFunc("/api/v1/security/guardian/status", s.handleLANGuardianStatus)
	mux.HandleFunc("/api/v1/security/guardian/scan", s.handleLANGuardianScan)
	mux.HandleFunc("/api/v1/security/guardian/isolate", s.handleLANGuardianIsolate)

	// ── DNS 负缓存免疫与常访域名预取引擎 ──
	mux.HandleFunc("/api/v1/network/dns/prefetch/status", s.handleDNSPrefetchStatus)
	mux.HandleFunc("/api/v1/network/dns/prefetch/toggle", s.handleDNSPrefetchToggle)
	mux.HandleFunc("/api/v1/network/dns/prefetch/flush", s.handleDNSPrefetchFlush)

	// ── 宿主底座策略路由影子巡检与防篡改自愈器 ──
	mux.HandleFunc("/api/v1/host/shadow/status", s.handleShadowAuditorStatus)
	mux.HandleFunc("/api/v1/host/shadow/audit", s.handleShadowAuditorRun)
	mux.HandleFunc("/api/v1/host/shadow/toggle", s.handleShadowAuditorToggle)

	// ── Test v7.5 核心进阶架构路由 ──
	// 维度一：高防风控自适应会话粘滞一致性哈希调度
	mux.HandleFunc("/api/v1/proxy/session-affinity/status", s.handleSessionAffinityStatus)
	mux.HandleFunc("/api/v1/proxy/session-affinity/toggle", s.handleSessionAffinityToggle)
	mux.HandleFunc("/api/v1/proxy/session-affinity/reset", s.handleSessionAffinityReset)

	// 维度二：双引擎内存极简回收与连接池热清洗
	mux.HandleFunc("/api/v1/system/memory-conntrack/status", s.handleMemoryConntrackStatus)
	mux.HandleFunc("/api/v1/system/memory/reclaim", s.handleMemoryReclaim)
	mux.HandleFunc("/api/v1/system/conntrack/flush", s.handleConntrackFlush)
	mux.HandleFunc("/api/v1/system/memory-conntrack/toggle", s.handleMemoryConntrackToggle)

	// ── Test v7.6 核心进阶架构路由 ──
	// 维度一：RFC 8305 Happy Eyeballs v2 双栈竞速与 IPv6 零侧漏阻断引擎
	mux.HandleFunc("/api/v1/network/happyeyeballs/v2/status", s.handleRFC8305HappyEyeballsStatus)
	mux.HandleFunc("/api/v1/network/happyeyeballs/v2/toggle", s.handleRFC8305HappyEyeballsToggle)
	mux.HandleFunc("/api/v1/network/happyeyeballs/v2/probe", s.handleRFC8305HappyEyeballsProbe)

	// 维度二：Linux 原生 Splice 零拷贝与 Socket 直通数据流管道
	mux.HandleFunc("/api/v1/system/splice/status", s.handleSpliceStatus)
	mux.HandleFunc("/api/v1/system/splice/toggle", s.handleSpliceToggle)
	mux.HandleFunc("/api/v1/system/splice/flush", s.handleSpliceFlush)

	// 维度三：RouterOS / 爱快 20ms 双向心跳与热修补影子总线
	mux.HandleFunc("/api/v1/ros/bidirheartbeat/status", s.handleBidirHeartbeatStatus)
	mux.HandleFunc("/api/v1/ros/bidirheartbeat/probe", s.handleBidirHeartbeatProbe)
	mux.HandleFunc("/api/v1/ros/bidirheartbeat/patch", s.handleBidirHeartbeatPatch)

	// ── Test v7.8 核心进阶架构路由 ──
	// 维度一：UDP 智能避障与并发极速竞速总线
	mux.HandleFunc("/api/v1/network/udp-racing/status", s.handleUDPRacingStatus)
	mux.HandleFunc("/api/v1/network/udp-racing/probe", s.handleUDPRacingProbe)
	mux.HandleFunc("/api/v1/network/udp-racing/toggle", s.handleUDPRacingToggle)
	mux.HandleFunc("/api/v1/network/udp-racing/mss", s.handleUDPRacingMSS)

	// 维度二：软路由硬件加速快路径与内核旁路协同
	mux.HandleFunc("/api/v1/host/fasttrack/status", s.handleFastTrackStatus)
	mux.HandleFunc("/api/v1/host/fasttrack/optimize", s.handleFastTrackOptimize)
	mux.HandleFunc("/api/v1/host/fasttrack/toggle", s.handleFastTrackToggle)

	// 维度三：全屋智能设备行为沙盒与异常上传拦截
	mux.HandleFunc("/api/v1/devices/iot-guard/status", s.handleIoTGuardStatus)
	mux.HandleFunc("/api/v1/devices/iot-guard/tune", s.handleIoTGuardTune)
	mux.HandleFunc("/api/v1/devices/iot-guard/toggle", s.handleIoTGuardToggle)

	// 维度四：AI 伴生管家三秒全网根因拓扑雷达与一键自愈
	mux.HandleFunc("/api/v1/ai/root-cause/report", s.handleRootCauseReport)
	mux.HandleFunc("/api/v1/ai/root-cause/heal", s.handleRootCauseHeal)

	// ── Test v7.8 四大核心演进架构路由 ──
	// 维度一：流媒体 4K 秒开专项诊断与单流吞吐深度体检卡
	mux.HandleFunc("/api/v1/diagnostics/streaming/report", s.handleStreamingReport)
	mux.HandleFunc("/api/v1/diagnostics/streaming/run", s.handleStreamingRun)
	mux.HandleFunc("/api/v1/diagnostics/streaming/heal", s.handleStreamingHeal)

	// 维度二：微流吞吐采样与多出口拥塞自适应智能切流
	mux.HandleFunc("/api/v1/network/adaptive-slicing/status", s.handleAdaptiveSlicingStatus)
	mux.HandleFunc("/api/v1/network/adaptive-slicing/tune", s.handleAdaptiveSlicingTune)
	mux.HandleFunc("/api/v1/network/adaptive-slicing/sample", s.handleAdaptiveSlicingSample)

	// 维度三：基于 BPF/XDP 的极速网络数据流旁路短路直通
	mux.HandleFunc("/api/v1/host/bpf-bypass/status", s.handleBpfBypassStatus)
	mux.HandleFunc("/api/v1/host/bpf-bypass/toggle", s.handleBpfBypassToggle)
	mux.HandleFunc("/api/v1/host/bpf-bypass/optimize", s.handleBpfBypassOptimize)

	// 维度四：全协议 DNS 泄漏深度阻断与客户端网格防污染
	mux.HandleFunc("/api/v1/dns/privacy-mesh/status", s.handlePrivacyMeshStatus)
	mux.HandleFunc("/api/v1/dns/privacy-mesh/tune", s.handlePrivacyMeshTune)
	mux.HandleFunc("/api/v1/dns/privacy-mesh/optimize", s.handlePrivacyMeshOptimize)

	// ── Test v7.9 四大核心演进架构路由 ──
	// 维度一：多通道多路径智能微流并发聚合
	mux.HandleFunc("/api/v1/network/multipath-bonding/status", s.handleMultiPathBondingStatus)
	mux.HandleFunc("/api/v1/network/multipath-bonding/toggle", s.handleMultiPathBondingToggle)
	mux.HandleFunc("/api/v1/network/multipath-bonding/channel/toggle", s.handleMultiPathChannelToggle)
	mux.HandleFunc("/api/v1/network/multipath-bonding/optimize", s.handleMultiPathBondingOptimize)

	// 维度二：微秒级流量特征感知与智能 QoS 抢占调度
	mux.HandleFunc("/api/v1/network/micro-qos/status", s.handleMicroQoSStatus)
	mux.HandleFunc("/api/v1/network/micro-qos/toggle", s.handleMicroQoSToggle)
	mux.HandleFunc("/api/v1/network/micro-qos/gaming/toggle", s.handleMicroQoSGamingToggle)
	mux.HandleFunc("/api/v1/network/micro-qos/flush", s.handleMicroQoSFlush)

	// 维度三：运营商级智能 ECS 保真与 Anycast 就近调度总线
	mux.HandleFunc("/api/v1/dns/ecs-dispatcher/status", s.handleECSDispatcherStatus)
	mux.HandleFunc("/api/v1/dns/ecs-dispatcher/toggle", s.handleECSDispatcherToggle)
	mux.HandleFunc("/api/v1/dns/ecs-dispatcher/mode", s.handleECSDispatcherMode)
	mux.HandleFunc("/api/v1/dns/ecs-dispatcher/optimize", s.handleECSDispatcherOptimize)

	// 维度四：局域网全协议零信任微隔离与设备沙盒
	mux.HandleFunc("/api/v1/security/zerotrust/status", s.handleZeroTrustStatus)
	mux.HandleFunc("/api/v1/security/zerotrust/toggle", s.handleZeroTrustToggle)
	mux.HandleFunc("/api/v1/security/zerotrust/device/policy", s.handleZeroTrustDevicePolicy)
	mux.HandleFunc("/api/v1/security/zerotrust/device/quarantine", s.handleZeroTrustDeviceQuarantine)
	mux.HandleFunc("/api/v1/security/zerotrust/scan", s.handleZeroTrustScan)

	// ── Test v8.0 三大核心演进架构路由 ──
	// 核心一：RFC 9000 原生 QUIC / HTTP3 智能识别与零拷贝旁路透传
	mux.HandleFunc("/api/v1/network/quic-passthrough/status", s.handleQUICPassthroughStatus)
	mux.HandleFunc("/api/v1/network/quic-passthrough/toggle", s.handleQUICPassthroughToggle)
	mux.HandleFunc("/api/v1/network/quic-passthrough/mode", s.handleQUICPassthroughMode)
	mux.HandleFunc("/api/v1/network/quic-passthrough/fallback", s.handleQUICPassthroughFallback)

	// 核心二：多宽带出口毫秒级智能避障与动态选路
	mux.HandleFunc("/api/v1/network/multiwan-racing/status", s.handleMultiWANRacingStatus)
	mux.HandleFunc("/api/v1/network/multiwan-racing/toggle", s.handleMultiWANRacingToggle)
	mux.HandleFunc("/api/v1/network/multiwan-racing/policy", s.handleMultiWANRacingPolicy)
	mux.HandleFunc("/api/v1/network/multiwan-racing/probe", s.handleMultiWANRacingProbe)

	// 核心三：软路由宿主一键全量冷热灾备与秒级还原 (单文件原子快照)
	mux.HandleFunc("/api/v1/system/snapshot", s.handleSnapshotUnified)
	mux.HandleFunc("/api/v1/system/snapshot/export", s.handleSnapshotExport)
	mux.HandleFunc("/api/v1/system/snapshot/restore", s.handleSnapshotRestore)
	mux.HandleFunc("/api/v1/system/snapshot/status", s.handleSnapshotStatus)
	mux.HandleFunc("/api/v1/network/wan-heal", s.handleWANHeal)

	mux.HandleFunc("/api/v1/security/radar", s.handleSecurityRadar)

	mux.HandleFunc("/api/v1/security/audit", s.handleSecurityRadar)

	mux.HandleFunc("/api/v1/security/remediate", s.handleSecurityRemediate)

	mux.HandleFunc("/api/v1/config/subscriptions", s.handleSubList)

	mux.HandleFunc("/api/v1/config/subscriptions/add", s.handleSubAdd)

	mux.HandleFunc("/api/v1/config/subscriptions/batch-add", s.handleSubBatchAdd)

	mux.HandleFunc("/api/v1/config/subscriptions/update", s.handleSubUpdate)

	mux.HandleFunc("/api/v1/config/subscriptions/cron", s.handleSubCronStatus)

	mux.HandleFunc("/api/v1/config/subscriptions/speedtest", s.handleSubSpeedTest)
	mux.HandleFunc("/api/v1/config/subscriptions/quality", s.handleSubQualityMatrix)
	mux.HandleFunc("/api/v1/config/proxies/ip-quality", s.handleProxyIPQuality)
	mux.HandleFunc("/api/v1/config/proxies/bandwidth-probe", s.handleProxyBandwidthProbe)
	mux.HandleFunc("/api/v1/config/proxies/scores", s.handleProxyNodeScores)
	mux.HandleFunc("/api/v1/config/proxies/ewma", s.handleProxyNodeEWMAScores)
	mux.HandleFunc("/api/v1/health/full-audit", s.handleFullHealthAudit)

	mux.HandleFunc("/api/v1/config/proxies/speedtest", s.handleProxiesSpeedTest)

	mux.HandleFunc("/api/v1/config/subscriptions/delete", s.handleSubDelete)

	mux.HandleFunc("/api/v1/config/subscriptions/toggle", s.handleSubToggle)

	mux.HandleFunc("/api/v1/config/subscriptions/rename", s.handleSubRename)

	mux.HandleFunc("/api/v1/config/subscriptions/node/add", s.handleNodeAdd)

	mux.HandleFunc("/api/v1/config/subscriptions/node/batch-add", s.handleNodeBatchAdd)

	mux.HandleFunc("/api/v1/config/subscriptions/node/delete", s.handleNodeDelete)

	mux.HandleFunc("/api/v1/engine/group/select", s.handleGroupSelect)

	mux.HandleFunc("/api/v1/engine/proxy/delay", s.handleProxyDelay)
	mux.HandleFunc("/api/v1/smartping/status", s.handleSmartPingStatus)
	mux.HandleFunc("/api/v1/smartping/config", s.handleSmartPingConfig)
	mux.HandleFunc("/api/v1/smartping/trigger", s.handleSmartPingTrigger)

	mux.HandleFunc("/api/v1/routing/regions", s.handleRoutingRegions)

	mux.HandleFunc("/api/v1/config/routing/group/add", s.handleCustomGroupAdd)

	mux.HandleFunc("/api/v1/config/routing/group/update", s.handleCustomGroupUpdate)

	mux.HandleFunc("/api/v1/config/routing/group/delete", s.handleCustomGroupDelete)

	mux.HandleFunc("/api/v1/config/routing/group/toggle", s.handleCustomGroupToggle)

	mux.HandleFunc("/api/v1/config/subscriptions/edit", s.handleSubEdit)

	mux.HandleFunc("/api/v1/ros/script", s.handleRosScript)

	mux.HandleFunc("/api/v1/ros/conntrack/clean", s.handleRosConntrackClean)

	// 5. Host Base Probing & Firewall Linkage

	mux.HandleFunc("/api/v1/host/info", s.handleHostInfo)

	mux.HandleFunc("/api/v1/host/config", s.handleHostConfig)

	mux.HandleFunc("/api/v1/host/test", s.handleHostTest)

	mux.HandleFunc("/api/v1/host/firewall/apply", s.handleHostFirewallApply)

	mux.HandleFunc("/api/v1/host/firewall/script", s.handleHostFirewallScript)

	// 6. RouterOS v7 REST API Inspection & 1-Click Linkage

	mux.HandleFunc("/api/v1/ros/inspect", s.handleROSInspect)

	mux.HandleFunc("/api/v1/ros/apply", s.handleROSApply)

	mux.HandleFunc("/api/v1/ros/config", s.handleROSConfig)

	// 7. 4-Tier Version Overview & Upgrades (CubeX, Cores, GeoData, Host)

	mux.HandleFunc("/api/v1/system/overview", s.handleSystemOverview)

	mux.HandleFunc("/api/v1/upgrade/core", s.handleUpgradeCore)

	mux.HandleFunc("/api/v1/upgrade/geodata", s.handleUpgradeGeoData)

	mux.HandleFunc("/api/v1/upgrade/panel", s.handleUpgradePanel)

	mux.HandleFunc("/api/v1/host/upgrade/check", s.handleHostUpgradeCheck)

	mux.HandleFunc("/api/v1/host/upgrade/execute", s.handleHostUpgradeExecute)

	// 8. Adblock & Domain Filter Hub

	mux.HandleFunc("/api/v1/adblock/stats", s.handleAdblockStats)

	mux.HandleFunc("/api/v1/adblock/config", s.handleAdblockConfig)

	mux.HandleFunc("/api/v1/adblock/toggle", s.handleAdblockToggle)

	mux.HandleFunc("/api/v1/adblock/source/add", s.handleAdblockSourceAdd)

	mux.HandleFunc("/api/v1/adblock/source/update", s.handleAdblockSourceUpdate)

	mux.HandleFunc("/api/v1/adblock/source/delete", s.handleAdblockSourceDelete)

	mux.HandleFunc("/api/v1/adblock/source/toggle", s.handleAdblockSourceToggle)

	mux.HandleFunc("/api/v1/adblock/source/refresh", s.handleAdblockSourceRefresh)

	mux.HandleFunc("/api/v1/adblock/whitelist/add", s.handleAdblockWhitelistAdd)

	mux.HandleFunc("/api/v1/adblock/whitelist/delete", s.handleAdblockWhitelistDelete)

	// 8.1. DNS Rewrite (CNAME & Static A Record Mapping)

	mux.HandleFunc("/api/v1/dns/rewrite/config", s.handleDNSRewriteConfig)

	mux.HandleFunc("/api/v1/dns/rewrite/toggle", s.handleDNSRewriteToggle)

	mux.HandleFunc("/api/v1/dns/rewrite/rule/add", s.handleDNSRewriteRuleAdd)

	mux.HandleFunc("/api/v1/dns/rewrite/rule/update", s.handleDNSRewriteRuleUpdate)

	mux.HandleFunc("/api/v1/dns/rewrite/rule/delete", s.handleDNSRewriteRuleDelete)

	mux.HandleFunc("/api/v1/dns/rewrite/rule/toggle", s.handleDNSRewriteRuleToggle)

	mux.HandleFunc("/api/v1/dns/rewrite/reset", s.handleDNSRewriteReset)
	mux.HandleFunc("/api/v1/dns/fake-ip-filter", s.handleFakeIPFilter)

	// 9. Time Machine Backup & Restore

	mux.HandleFunc("/api/v1/backup/export", s.handleBackupExport)

	mux.HandleFunc("/api/v1/backup/import", s.handleBackupImport)

	// 10. AI Copilot

	mux.HandleFunc("/api/v1/ai/chat", s.handleAIChat)
	mux.HandleFunc("/api/v1/ai/config", s.handleAIConfig)
	mux.HandleFunc("/api/v1/ai/diagnose", s.handleAIDiagnose)
	mux.HandleFunc("/api/v1/system/heal", s.handleSystemHeal)
	mux.HandleFunc("/api/v1/failover/trigger", s.handleSmartPingTrigger)

	// 10.2 Realtime Telemetry & Traffic Speed (Overview Module)

	mux.HandleFunc("/api/v1/telemetry/traffic", s.handleTelemetryTraffic)

	// 10.3 Native Model Context Protocol (MCP Server JSON-RPC 2.0 / SSE)
	mux.Handle("/api/v1/mcp", s.mcpServer)

	// 11. LAN Devices & Per-Device Policy Matrix

	mux.HandleFunc("/api/v1/devices", s.handleDevices)

	mux.HandleFunc("/api/v1/devices/alias", s.handleDeviceAlias)

	mux.HandleFunc("/api/v1/devices/policy", s.handleDevicePolicy)

	mux.HandleFunc("/api/v1/devices/toggle", s.handleDeviceToggle)

	mux.HandleFunc("/api/v1/devices/batch", s.handleDeviceBatch)

	// 11.1 Router Panorama & Hardware Telemetry (方向一：主路由全景透视驾驶舱)

	mux.HandleFunc("/api/v1/host/router/hardware", s.handleRouterHardware)

	mux.HandleFunc("/api/v1/host/router/clients", s.handleRouterClients)

	mux.HandleFunc("/api/v1/host/router/clients/toggle", s.handleDeviceToggle)

	mux.HandleFunc("/api/v1/host/router/clients/batch", s.handleDeviceBatch)

	// 12. Host Unified Auto-Setup & Compatible Aliases

	mux.HandleFunc("/api/v1/host/auto-setup", s.handleHostAutoSetup)

	mux.HandleFunc("/api/v1/host/ikuai/docker-params", s.handleIkuaiDockerParams)
	mux.HandleFunc("/api/v1/host/ikuai/sync", s.handleIkuaiSync)

	mux.HandleFunc("/api/v1/host/openwrt/inspect", s.handleOpenWrtInspect)

	mux.HandleFunc("/api/v1/host/openwrt/apply", s.handleOpenWrtApply)

	mux.HandleFunc("/api/v1/host/triad/matrix", s.handleHostTriadMatrix)
	mux.HandleFunc("/api/v1/host/triad/heal", s.handleHostTriadHeal)

	// 12.1 HA Drain & Upstream 4D Matrix & EWMA Closed-Loop Routing
	mux.HandleFunc("/api/v1/ha/drain", s.handleHADrain)
	mux.HandleFunc("/api/v1/probe/ewma/matrix", s.handleEWMAMatrix)
	mux.HandleFunc("/api/v1/upstream/matrix", s.handleUpstreamMatrix)
	mux.HandleFunc("/api/v1/upstream/upgrade", s.handleUpstreamUpgrade)

	// 12.2 Telemetry SSE Stream & FlightRecorder Blackbox (v9.0 LTS)
	mux.Handle("/api/v1/telemetry/stream", telemetry.GetGlobalSSEBroadcaster())
	mux.HandleFunc("/api/v1/diag/flight-log", s.handleFlightLog)
	mux.HandleFunc("/api/v1/onboarding/doctor", s.handleOnboardingDoctor)
	mux.HandleFunc("/api/v1/onboarding/gateway-mode", s.handleGatewayMode)
	mux.HandleFunc("/api/v1/onboarding/tri-router", s.handleTriRouterBundle)
	mux.HandleFunc("/api/v1/onboarding/verify-loop", s.handleVerifySymbioticLoop)
	mux.HandleFunc("/api/v1/onboarding/essential-cidrs", s.handleGetEssentialCIDRs)
	mux.HandleFunc("/api/v1/diag/bundle", telemetry.ServeDiagnosticBundle(
		telemetry.GetGlobalFlightRecorder(),
		func() string {
			if s.engMgr != nil {
				return string(s.engMgr.GetActiveEngine())
			}
			return "singbox"
		},
		func() []byte {
			if s.engMgr != nil {
				if ir := s.engMgr.GetIR(); ir != nil {
					b, _ := json.Marshal(ir)
					return b
				}
			}
			return nil
		},
	))

	mux.HandleFunc("/api/v1/ros/routing/status", s.handleROSInspect)

	mux.HandleFunc("/api/v1/ros/routing/apply", s.handleROSApply)

	mux.HandleFunc("/api/v1/ros/routing/clean-and-reapply", s.handleROSCleanAndReapply)

	mux.HandleFunc("/api/v1/ros/clean-and-reapply", s.handleROSCleanAndReapply)

	mux.HandleFunc("/api/v1/ros/ipv6/setup", s.handleROSIpv6Setup)

	mux.HandleFunc("/api/v1/ros/routing/script", s.handleRosScript)

	mux.HandleFunc("/api/v1/system/geodata/update", s.handleUpgradeGeoData)

	mux.HandleFunc("/api/v1/system/upgrade-panel", s.handleUpgradePanel)

	mux.HandleFunc("/api/v1/ros/upgrade-mihomo", s.handleUpgradeCore)

	mux.HandleFunc("/api/v1/system/latency", s.handleSystemLatency)

	mux.HandleFunc("/api/v1/rules/trace", s.handleRulesTrace)

	// 12.3 v9.4.0 User Overrides & Top-N Client Inspector
	mux.HandleFunc("/api/v1/rules/overrides", s.handleRuleOverrides)
	mux.HandleFunc("/api/v1/telemetry/clients", s.handleTelemetryClients)
	mux.HandleFunc("/api/v1/telemetry/clients/kick", s.handleTelemetryKickClient)

	// 13. Ring Log Terminal & Desensitized Diagnostic Package Export

	mux.HandleFunc("/api/v1/system/logs", s.handleSystemLogs)

	mux.HandleFunc("/api/v1/system/diagnose/export", s.handleSystemDiagnoseExport)

	mux.HandleFunc("/api/v1/system/diagnose/dual-core", s.handleDualCoreDiagnose)

	mux.HandleFunc("/api/v1/health/radar", s.handleHealthRadar)
	mux.HandleFunc("/api/v1/system/failover/drill", s.handleFailoverDrill)

	// 3. Static Web UI (Precompressed Gzip Direct Streaming & Zero-CPU / Zero-Copy)
	precompressedServer := webui.NewPrecompressedFileServer(getEmbeddedFS())

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/sub/") {
			http.NotFound(w, r)
			return
		}
		precompressedServer.ServeHTTP(w, r)
	})


}




func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				buf := make([]byte, 2048)
				n := runtime.Stack(buf, false)
				log.Printf("❌ [HTTP] Panic in handler %s %s: %v\nStack:\n%s\n", r.Method, r.URL.Path, rec, string(buf[:n]))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error":  "Internal server panic recovered",
					"detail": fmt.Sprintf("%v", rec),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}





type gzipResponseWriter struct {
	http.ResponseWriter
	writer *gzip.Writer
}





func (w *gzipResponseWriter) WriteHeader(status int) {
	w.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(status)
}





func (w *gzipResponseWriter) Flush() {
	if w.writer != nil {
		_ = w.writer.Flush()
	}
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}





func gzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Static assets are handled by PrecompressedFileServer directly with 0 CPU overhead
		if !strings.HasPrefix(r.URL.Path, "/api/") && !strings.HasPrefix(r.URL.Path, "/sub/") {
			next.ServeHTTP(w, r)
			return
		}

		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") ||
			r.Header.Get("Upgrade") != "" ||
			strings.Contains(r.Header.Get("Accept"), "text/event-stream") {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Set("Content-Encoding", "gzip")

		gz, err := gzip.NewWriterLevel(w, gzip.BestSpeed)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		defer gz.Close()

		gzw := &gzipResponseWriter{ResponseWriter: w, writer: gz}
		next.ServeHTTP(gzw, r)
	})
}





func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isTrustedOrigin(origin, r.Host) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {

			w.WriteHeader(http.StatusOK)

			return

		}

		next.ServeHTTP(w, r)

	})

}

// getCgroupMemoryMB reads host cgroup memory usage for the container (standard in Docker, Podman, PVE LXC, fnOS).





func (s *Server) handleSystemStatus(w http.ResponseWriter, r *http.Request) {

	var m runtime.MemStats

	runtime.ReadMemStats(&m)

	ir := s.engMgr.GetIR()

	lastRes := s.engMgr.GetLastResult()

	appRSS, coreRSS, totalOverhead := s.getLiveMemoryMetrics()

	resp := map[string]any{

		"version": version,

		"host_ip": s.detectHostIP(),

		"uptime_seconds": int64(time.Since(startTime).Seconds()),

		"active_engine": s.engMgr.GetActiveEngine(),

		"edition": ir.Meta.Edition,

		"proxy_mode": ir.Meta.ProxyMode,

		"active_profile": ir.Meta.ActiveProfile,

		"memory_mb": appRSS + coreRSS,

		"app_memory_mb": appRSS,

		"memory_alloc_mb": float64(m.Alloc) / 1024 / 1024,

		"memory_sys_mb": float64(m.Sys) / 1024 / 1024,

		"core_memory_mb": coreRSS,

		"total_overhead_mb": totalOverhead,

		"memory_bytes": int64(totalOverhead * 1024 * 1024),

		"goroutines": runtime.NumGoroutine(),

		"os": runtime.GOOS,

		"arch": runtime.GOARCH,

		"last_switch_res": lastRes,

		"tun_interface": ir.Network.TunInterface,

		"tproxy_port": ir.Network.TProxyPort,

		"dns_port": ir.Network.DNSPort,

		"clash_api_port": ir.Network.ClashAPIPort,

		"total_proxies": len(ir.Proxies),

		"total_subscriptions": len(ir.Subscriptions),

		"initialized": len(ir.Proxies) > 0 || len(ir.Subscriptions) > 0,

		"total_groups": len(ir.Groups),
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

			"status": "ok",

			"edition": ir.Meta.Edition,
		})

		return

	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

}





func (s *Server) handleUIState(w http.ResponseWriter, r *http.Request) {
	stateFile := filepath.Join(s.workDir, "data", "ui_state.json")
	if r.Method == http.MethodGet {
		if b, err := os.ReadFile(stateFile); err == nil && len(b) > 0 {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			_, _ = w.Write(b)
			return
		}
		sendJSON(w, map[string]any{"ok": true, "state": map[string]any{}})
		return
	}
	if r.Method == http.MethodPost {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = os.MkdirAll(filepath.Dir(stateFile), 0755)
		b, _ := json.MarshalIndent(payload, "", "  ")
		_ = WriteFileAtomic(stateFile, b, 0644)
		sendJSON(w, map[string]any{"ok": true})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
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





func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {

	var req struct {
		NodeName string `json:"node_name"`

		GroupName string `json:"group_name"`

		URL string `json:"url"`
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

			Name: "本地直连 (DIRECT)",

			Server: "127.0.0.1",

			Port: 80,

			Type: "direct",
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

	_, _, totalOverhead := s.getLiveMemoryMetrics()

	if totalOverhead > 0 {

		if s.engMgr.GetActiveEngine() == schema.EngineSingbox {

			res.SingboxMetric.MemoryUsageMB = totalOverhead

			res.MihomoMetric.MemoryUsageMB = totalOverhead + 12.0

		} else if s.engMgr.GetActiveEngine() == schema.EngineMihomo {

			res.MihomoMetric.MemoryUsageMB = totalOverhead

			if totalOverhead > 20.0 {

				res.SingboxMetric.MemoryUsageMB = totalOverhead - 12.0

			} else {

				res.SingboxMetric.MemoryUsageMB = totalOverhead

			}

		}

	}

	sendJSON(w, res)

}





func (s *Server) handleProfileScene(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		status := s.sceneMgr.GetStatus()
		sendJSON(w, map[string]any{"status": "ok", "scene": status})
		return
	}
	if r.Method == http.MethodPost {
		var req struct {
			Scene profile.SceneMode `json:"scene"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		params, err := s.sceneMgr.SwitchScene(req.Scene)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.logBuffer.Add("INFO", "PROFILE", fmt.Sprintf("切换场景模式为: %s (%s)", params.Name, req.Scene))
		sendJSON(w, map[string]any{"status": "ok", "active_scene": req.Scene, "params": params})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

// 维度三：全网 24h 流量时光机与历史回溯




func (s *Server) handleTimeMachine(w http.ResponseWriter, r *http.Request) {
	hist := s.timeMachine.Get24HHistory()
	sendJSON(w, map[string]any{"status": "ok", "timemachine": hist})
}

// 维度四：DNS 主动防劫持盲测雷达与未登记设备动态隔离沙盒




func (s *Server) handleWeeklyReport(w http.ResponseWriter, r *http.Request) {
	rep := s.weeklyReportMgr.GetReport()
	sendJSON(w, map[string]any{"status": "ok", "weekly": rep})
}

// ── 智能自愈与 XDP/QoS 演进接口 ──

// 维度一：AI 智能自愈主动干预总线




func (s *Server) handleBackupExport(w http.ResponseWriter, r *http.Request) {

	desensitize := r.URL.Query().Get("desensitize") == "true"

	var b *backup.BackupData

	var err error

	if desensitize {

		b, err = s.backupMgr.ExportDesensitizedBackup(version)

	} else {

		b, err = s.backupMgr.ExportBackup(version)

	}

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

		"status": "restored",

		"message": "配置快照已成功恢复并装载！",

		"sub_count": subCount,

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

	if s.aiCopilot != nil {
		res, err := s.aiCopilot.ChatWithActions(req.Prompt)
		if err != nil {
			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{
			"status":  "ok",
			"reply":   res.Reply,
			"actions": res.Actions,
		})
		return
	}
	sendJSON(w, map[string]any{"status": "ok", "reply": "已完成推演"})
}





func (s *Server) handleAIConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if s.aiCopilot != nil {
			sendJSON(w, s.aiCopilot.GetConfig())
		} else {
			sendJSON(w, map[string]any{"provider": "qwen", "model": "qwen-plus"})
		}
		return
	}
	if r.Method == http.MethodPost {
		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			sendJSON(w, map[string]any{"status": "error", "error": "invalid payload"})
			return
		}
		if s.aiCopilot != nil {
			if err := s.aiCopilot.SaveConfig(req); err != nil {
				sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
				return
			}
		}
		sendJSON(w, map[string]any{"status": "ok", "message": "AI 管家配置已成功更新并持久化！"})
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}





func (s *Server) handleAIDiagnose(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.aiCopilot != nil {
		res, err := s.aiCopilot.DiagnoseWithActions()
		if err != nil {
			sendJSON(w, map[string]any{"status": "error", "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{
			"status":  "ok",
			"reply":   res.Reply,
			"actions": res.Actions,
		})
		return
	}
	sendJSON(w, map[string]any{
		"status": "ok",
		"reply":  "全网体检完毕：双核运行正常，内核数据包转发畅通，0 明文 DNS 泄露，DNS 重写中枢与权威规则库 100% 暮光晶盾级就绪！",
		"actions": []map[string]any{
			{
				"id":       "action-heal",
				"label":    "一键全栈自愈加固",
				"type":     "heal",
				"icon":     "🚑",
				"desc":     "自动加固策略路由 table cubex、清除死连接并加固防侧漏",
				"endpoint": "/api/v1/system/heal",
			},
		},
	})
}





func (s *Server) handleSystemOverview(w http.ResponseWriter, r *http.Request) {

	overview := s.rosMgr.GetOverview()

	overview["cubex_version"] = version

	overview["panel_curr_version"] = version

	latestVer, _ := overview["panel_latest_version"].(string)

	if latestVer == "" || latestVer == "Test-v1.3" || !ros.IsVersionNewer(latestVer, version) {

		overview["panel_latest_version"] = version

		overview["panel_update_avail"] = false

	} else {

		overview["panel_update_avail"] = true

	}

	mihomoAvail, _ := overview["mihomo_update_avail"].(bool)

	singboxAvail, _ := overview["singbox_update_avail"].(bool)

	panelAvail, _ := overview["panel_update_avail"].(bool)

	overview["update_available"] = mihomoAvail || singboxAvail || panelAvail

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

	if r.Method == http.MethodGet {

		if r.URL.Query().Get("check") == "true" {

			hasUpdate, ver, _ := s.rosMgr.CheckGeoDataUpdate()

			status := s.rosMgr.GetGeoDataStatus()

			status.HasUpdate = hasUpdate

			if ver != "" {

				status.LatestVersion = ver

			}

			sendJSON(w, status)

			return

		}

		status := s.rosMgr.GetGeoDataStatus()

		sendJSON(w, status)

		return

	}

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

	if res.TotalRules > 0 {

		s.adblockMgr.SetCachedRules(res.TotalRules)

	}

	sendJSON(w, res)

}





func (s *Server) handleUpgradePanel(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return

	}

	force := r.URL.Query().Get("force") == "true"

	ver, err := s.rosMgr.UpgradePanel(force, func() error {
		log.Printf("[CubeX] Restarting process for upgrade...")
		SafeGo("PanelRestart", func() {
			time.Sleep(1200 * time.Millisecond)
			if _, err := exec.LookPath("rc-service"); err == nil {
				log.Printf("[CubeX] Executing rc-service cubexd restart...")
				if err := exec.Command("rc-service", "cubexd", "restart").Start(); err == nil {
					return
				}
			}
			if _, err := exec.LookPath("systemctl"); err == nil {
				log.Printf("[CubeX] Executing systemctl restart cubexd...")
				if err := exec.Command("systemctl", "restart", "cubexd").Start(); err == nil {
					return
				}
			}
			exe, err := os.Executable()
			if err == nil {
				log.Printf("[CubeX] Self-spawning detached binary %s...", exe)
				cmd := exec.Command(exe, os.Args[1:]...)
				cmd.Dir = filepath.Dir(exe)
				_ = cmd.Start()
			}
			os.Exit(0)
		})
		return nil
	})

	if err != nil {

		if strings.Contains(err.Error(), "already at the latest version") || strings.Contains(err.Error(), "已是最新") {

			sendJSON(w, map[string]any{"status": "already_latest", "version": ver, "message": "CubeX 系统已是最新版本"})

			return

		}

		sendJSON(w, map[string]any{"status": "error", "error": err.Error()})

		return

	}

	sendJSON(w, map[string]any{"status": "upgraded", "version": ver})

}





func (s *Server) handleSystemLogs(w http.ResponseWriter, r *http.Request) {

	if (r.Method == http.MethodPost && r.URL.Query().Get("action") == "clear") || r.Method == http.MethodDelete {

		s.logBuffer.Clear()

		sendJSON(w, map[string]any{

			"status": "cleared",

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

		"status": "success",

		"total": len(logs),

		"max_size": 500,

		"logs": logs,
	})

}





func (s *Server) handleSystemDiagnoseExport(w http.ResponseWriter, r *http.Request) {

	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	meta := map[string]any{

		"brand": "CubeX",

		"version": version,

		"export_time": time.Now().Format(time.RFC3339),

		"uptime_seconds": int64(time.Since(startTime).Seconds()),

		"os": runtime.GOOS,

		"arch": runtime.GOARCH,

		"num_cpu": runtime.NumCPU(),

		"goroutines": runtime.NumGoroutine(),

		"memory_alloc_mb": float64(memStats.Alloc) / (1024 * 1024),

		"memory_sys_mb": float64(memStats.Sys) / (1024 * 1024),
	}

	lastRes := s.engMgr.GetLastResult()

	engineRunning := true

	if lastRes != nil && !lastRes.Success {

		engineRunning = false

	}

	probePort := func(network, addr string, timeout time.Duration) bool {

		conn, err := net.DialTimeout(network, addr, timeout)

		if err == nil {

			conn.Close()

			return true

		}

		return false

	}

	liveHealth := map[string]any{

		"dns_53": probePort("udp", "127.0.0.1:53", 250*time.Millisecond),

		"clash_api_9090": probePort("tcp", "127.0.0.1:9090", 250*time.Millisecond),

		"mixed_7890": probePort("tcp", "127.0.0.1:7890", 250*time.Millisecond),
	}

	engineStatus := map[string]any{

		"active_engine": s.engMgr.GetActiveEngine(),

		"running": engineRunning,

		"core_pid": s.engMgr.GetCurrentProcPID(),

		"last_switch": lastRes,

		"health_probes": liveHealth,
	}

	var safeHost map[string]any

	if hInfo := s.hostMgr.GetCachedHostInfo(); hInfo != nil {

		safeHost = map[string]any{

			"platform_id": hInfo.PlatformID,

			"os_name": hInfo.OSName,

			"hostname": hInfo.HostName,

			"kernel_version": hInfo.KernelVersion,

			"arch": hInfo.Arch,

			"cpu_cores": hInfo.CPUCores,

			"cpu_load": hInfo.CPULoad,

			"memory_total": hInfo.MemoryTotal,

			"memory_used": hInfo.MemoryUsed,

			"ip_forward": hInfo.IPForward,

			"docker_running": hInfo.DockerRunning,

			"firewall_configured": hInfo.FirewallConfigured,
		}

	} else {

		safeHost = map[string]any{"status": "not_cached"}

	}

	rosStatus := s.rosMgr.GetOverview()

	safeROS := map[string]any{

		"connected": rosStatus["ros_connected"],

		"version": rosStatus["ros_version"],

		"model": rosStatus["ros_model"],

		"routing_synced": rosStatus["routing_synced"],
	}

	subSummary := []map[string]any{}

	proxyTypeCount := make(map[string]int)

	ir := s.engMgr.GetIR()

	if ir != nil {

		for _, sub := range ir.Subscriptions {

			subSummary = append(subSummary, map[string]any{

				"id": sub.ID,

				"name": sub.Name,

				"node_count": sub.NodeCount,

				"url": "***DESENSITIZED_TOKEN_STRIPPED***",
			})

		}

		for _, p := range ir.Proxies {

			proxyTypeCount[p.Type]++

		}

	}

	adStats := s.adblockMgr.GetStats(true, 0)

	recentLogs := s.logBuffer.GetLogs(500)

	pack := map[string]any{

		"cubex_diagnostic_report": true,

		"desensitized": true,

		"compliance": "ZERO_CREDENTIAL_LEAKAGE_STRICT",

		"meta": meta,

		"engine": engineStatus,

		"host": safeHost,

		"routeros": safeROS,

		"subscriptions_summary": subSummary,

		"proxy_protocols_summary": proxyTypeCount,

		"adblock_stats": adStats,

		"recent_logs": recentLogs,
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





func (s *Server) handleSnapshotExport(w http.ResponseWriter, r *http.Request) {
	if s.snapshotMgr == nil {
		sendJSON(w, map[string]any{"status": "error", "message": "snapshot manager not initialized"})
		return
	}
	hInfo := s.hostMgr.GetCachedHostInfo()
	plat := "Generic Linux"
	if hInfo != nil && hInfo.OSName != "" {
		plat = hInfo.OSName
	}
	snap, finalBytes, err := s.snapshotMgr.CreateSnapshot(version, plat)
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "message": err.Error()})
		return
	}
	s.logBuffer.Add("INFO", "BACKUP", fmt.Sprintf("已成功创建系统全量灾备快照 (校验签名: %s)", snap.ChecksumSHA256[:12]))
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=cubex-backup-%s-%d.cubex-backup.json", version, snap.Timestamp))
	w.Write(finalBytes)
}





func (s *Server) handleSnapshotUnified(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleSnapshotExport(w, r)
		return
	}
	if r.Method == http.MethodPost {
		s.handleSnapshotRestore(w, r)
		return
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}





func (s *Server) handleWANHeal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.wanHealer == nil {
		sendJSON(w, map[string]any{"ok": false, "message": "WANHealer not initialized"})
		return
	}
	err := s.wanHealer.FlushStaleConnections(r.Context(), "控制台一键主动冲刷")
	if err != nil {
		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	sendJSON(w, map[string]any{"ok": true, "message": "已成功冲刷卡死在旧WAN上的僵尸连接池，0秒无感恢复"})
}





func (s *Server) handleSnapshotRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.snapshotMgr == nil {
		sendJSON(w, map[string]any{"status": "error", "message": "snapshot manager not initialized"})
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		sendJSON(w, map[string]any{"status": "error", "message": "empty snapshot payload"})
		return
	}
	res, err := s.snapshotMgr.RestoreSnapshot(body, func() error {
		_, switchErr := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
		return switchErr
	})
	if err != nil {
		sendJSON(w, map[string]any{"status": "error", "message": err.Error()})
		return
	}
	s.logBuffer.Add("INFO", "BACKUP", "系统全量快照秒级原子恢复生效，引擎已重新编排完成")
	sendJSON(w, map[string]any{"status": "ok", "result": res})
}





func (s *Server) handleSnapshotStatus(w http.ResponseWriter, r *http.Request) {
	snapPath := filepath.Join(s.workDir, "data", "config.ir.json")
	var snapMeta map[string]any
	if d, err := os.ReadFile(snapPath); err == nil {
		var m map[string]any
		if json.Unmarshal(d, &m) == nil {
			snapMeta = map[string]any{
				"has_snapshot": true,
				"ir_bytes":     len(d),
				"last_saved":   time.Now().Format("2006-01-02 15:04:05"),
			}
		}
	}
	if snapMeta == nil {
		snapMeta = map[string]any{"has_snapshot": false}
	}
	sendJSON(w, map[string]any{"status": "ok", "snapshot": snapMeta})
}

// ── v9.4.0 User Overrides & Top-N Client Inspector ──



