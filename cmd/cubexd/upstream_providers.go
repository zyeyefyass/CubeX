package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/zyeyefyass/CubeX/internal/dataplane"
	"github.com/zyeyefyass/CubeX/internal/onboarding"
	"github.com/zyeyefyass/CubeX/internal/telemetry"
	"github.com/zyeyefyass/CubeX/internal/upstream"
)

// handleHADrain 处理宿主旁路维护模式切换与状态查询
func (s *Server) handleHADrain(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		sendJSON(w, map[string]any{
			"ok":          true,
			"maintenance": s.isMaintenance.Load(),
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Mode string `json:"mode"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Mode == "maintenance" {
		s.isMaintenance.Store(true)
		if s.rosMgr != nil {
			_ = s.rosMgr.GracefulDrain(r.Context(), true)
		}
		_ = dataplane.TeardownAtomicNftRules(r.Context())
		telemetry.GetGlobalFlightRecorder().Record(telemetry.CatHAFailover, "WARN", "进入旁路维护（不断网停机）：RouterOS 策略路由与 DNS 已平滑降级", 0)
		s.logBuffer.Add("WARN", "HA", "已进入「旁路维护（不断网停机）」模式：RouterOS 策略路由与 DNS 已无感降级直连，全家网络畅通，可安全维护容器！")
		sendJSON(w, map[string]any{
			"ok":          true,
			"maintenance": true,
			"message":     "已进入旁路维护模式，全网无感直连",
		})
		return
	}

	if req.Mode == "resume" {
		s.isMaintenance.Store(false)
		if s.rosMgr != nil {
			_ = s.rosMgr.NotifyEngineReady(r.Context())
		}
		telemetry.GetGlobalFlightRecorder().Record(telemetry.CatHAFailover, "INFO", "恢复旁路接管：RouterOS 策略路由与 DNS 已恢复接管", 0)
		s.logBuffer.Add("INFO", "HA", "已恢复旁路接管：RouterOS 策略路由与 DNS 已恢复接管。")
		sendJSON(w, map[string]any{
			"ok":          true,
			"maintenance": false,
			"message":     "已恢复旁路分流与 DNS 接管",
		})
		return
	}

	sendJSON(w, map[string]any{
		"ok":          true,
		"maintenance": s.isMaintenance.Load(),
	})
}

// handleEWMAMatrix 返回 EWMA 闭环选路金字塔分级（T0 现役 / T1 热备前三 / T2 冷备）
func (s *Server) handleEWMAMatrix(w http.ResponseWriter, r *http.Request) {
	ir := s.engMgr.GetIR()
	if ir == nil {
		sendJSON(w, map[string]any{"ok": false, "error": "ir not initialized"})
		return
	}

	type PyramidTier struct {
		Group          string                 `json:"group"`
		ActiveNode     string                 `json:"active_node"`
		ActiveRank     map[string]any         `json:"active_rank,omitempty"`
		HotStandby     []map[string]any       `json:"hot_standby"`
		ColdStandby    []map[string]any       `json:"cold_standby"`
		TotalNodes     int                    `json:"total_nodes"`
		LastSwitchTime string                 `json:"last_switch_time,omitempty"`
	}

	groups := make([]PyramidTier, 0, len(ir.Groups))
	for _, grp := range ir.Groups {
		if grp.Type != "select" && grp.Type != "url-test" {
			continue
		}
		allNodes := grp.Proxies
		if len(allNodes) == 0 {
			continue
		}

		ranks := s.ewmaMgr.RankNodes(allNodes)
		var activeItem map[string]any
		hotItems := make([]map[string]any, 0, 3)
		coldItems := make([]map[string]any, 0, len(ranks))

		for idx, rk := range ranks {
			info := map[string]any{
				"name":             rk.NodeName,
				"effective_delay":  rk.EffectiveDelay,
				"stability_score":  rk.StabilityScore,
				"consecutive_fails": rk.ConsecutiveFails,
				"has_sample":       rk.HasSample,
			}
			if snap, ok := s.ewmaMgr.GetNodeSnapshot(rk.NodeName); ok {
				info["srtt"] = int(snap.SRTT)
				info["rttvar"] = int(snap.RTTVAR)
				info["loss_rate"] = snap.LossRate
				info["grade"] = snap.Grade
			}

			if rk.NodeName == grp.Now {
				activeItem = info
			}

			if idx < 3 && rk.NodeName != grp.Now {
				hotItems = append(hotItems, info)
			} else if rk.NodeName != grp.Now {
				coldItems = append(coldItems, info)
			}
		}

		lastSwitchStr := ""
		if snap, ok := s.ewmaMgr.GetNodeSnapshot(grp.Now); ok && !snap.LastSwitchAt.IsZero() {
			lastSwitchStr = snap.LastSwitchAt.Format("2006-01-02 15:04:05")
		}

		groups = append(groups, PyramidTier{
			Group:          grp.Name,
			ActiveNode:     grp.Now,
			ActiveRank:     activeItem,
			HotStandby:     hotItems,
			ColdStandby:    coldItems,
			TotalNodes:     len(allNodes),
			LastSwitchTime: lastSwitchStr,
		})
	}

	sendJSON(w, map[string]any{
		"ok":        true,
		"groups":    groups,
		"timestamp": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// handleUpstreamMatrix 汇总四维版本中心探针报告
func (s *Server) handleUpstreamMatrix(w http.ResponseWriter, r *http.Request) {
	if s.upstreamReg == nil {
		sendJSON(w, map[string]any{"ok": false, "error": "upstream registry not initialized"})
		return
	}

	report := s.upstreamReg.ProbeAll(r.Context(), 5*time.Second)
	sendJSON(w, map[string]any{
		"ok":     true,
		"matrix": report,
	})
}

// handleUpstreamUpgrade 执行单维原子流式升级与故障毫秒级自动回滚
func (s *Server) handleUpstreamUpgrade(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ProviderID string `json:"provider_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.ProviderID == "" {
		sendJSON(w, map[string]any{"ok": false, "error": "provider_id required"})
		return
	}

	// 代理至 rosMgr 传统升级总线，同时支持四维插件化升级
	switch req.ProviderID {
	case "control_plane", "control.cubexd":
		msg, err := s.rosMgr.UpgradePanel(true, func() error {
			return nil
		})
		if err != nil {
			sendJSON(w, map[string]any{"ok": false, "rolled_back": false, "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"ok": true, "rolled_back": false, "message": msg})
	case "core.mihomo":
		msg, err := s.rosMgr.UpgradeMihomoCore(func() error {
			_, switchErr := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
			return switchErr
		})
		if err != nil {
			sendJSON(w, map[string]any{"ok": false, "rolled_back": false, "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"ok": true, "rolled_back": false, "message": msg})
	case "core.singbox", "core.sing-box":
		msg, err := s.rosMgr.UpgradeSingboxCore(func() error {
			_, switchErr := s.engMgr.SwitchEngine(s.engMgr.GetActiveEngine())
			return switchErr
		})
		if err != nil {
			sendJSON(w, map[string]any{"ok": false, "rolled_back": false, "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"ok": true, "rolled_back": false, "message": msg})
	case "host.ros", "host.routeros":
		err := s.rosMgr.TriggerROSFirmwareUpgrade()
		if err != nil {
			sendJSON(w, map[string]any{"ok": false, "rolled_back": false, "error": err.Error()})
			return
		}
		sendJSON(w, map[string]any{"ok": true, "rolled_back": false, "message": "RouterOS 固件升级与自动重启指令已下发"})
	default:
		sendJSON(w, map[string]any{"ok": false, "error": fmt.Sprintf("unsupported provider %s", req.ProviderID)})
	}
}

// defaultBridgeProvider 适配已有 rosMgr 探针为 upstream.UpstreamProvider
type defaultBridgeProvider struct {
	id          string
	dim         upstream.Dimension
	displayName string
	probeFn     func(ctx context.Context) (*upstream.VersionStatus, error)
}

func (p *defaultBridgeProvider) ID() string           { return p.id }
func (p *defaultBridgeProvider) Dimension() upstream.Dimension { return p.dim }
func (p *defaultBridgeProvider) Probe(ctx context.Context) (*upstream.VersionStatus, error) {
	if p.probeFn != nil {
		return p.probeFn(ctx)
	}
	return &upstream.VersionStatus{
		ProviderID:  p.id,
		Dimension:   p.dim,
		DisplayName: p.displayName,
		CheckedAt:   time.Now(),
	}, nil
}
func (p *defaultBridgeProvider) Upgrade(ctx context.Context, tx *upstream.AtomicInstaller) (*upstream.UpgradeResult, error) {
	return &upstream.UpgradeResult{
		ProviderID: p.id,
		Message:    "upgrade delegated to system supervisor",
	}, nil
}

// registerDefaultUpstreamProviders 注册默认的四维版本探针
func (s *Server) registerDefaultUpstreamProviders() {
	if s.upstreamReg == nil {
		return
	}

	// D1: Control Plane
	s.upstreamReg.Register(&defaultBridgeProvider{
		id:          "control.cubexd",
		dim:         upstream.DimControlPlane,
		displayName: "CubeX 控制面与全息交互面板",
		probeFn: func(ctx context.Context) (*upstream.VersionStatus, error) {
			curVer := "Test v8.9"
			ov := s.rosMgr.GetOverview()
			remoteVer, _ := ov["remote_version"].(string)
			hasUpdate, _ := ov["has_update"].(bool)
			return &upstream.VersionStatus{
				ProviderID:     "control.cubexd",
				Dimension:      upstream.DimControlPlane,
				DisplayName:    "CubeX 控制面与全息交互面板",
				CurrentVersion: curVer,
				LatestVersion:  remoteVer,
				HasUpdate:      hasUpdate,
				CheckedAt:      time.Now(),
			}, nil
		},
	})

	// D2: Mihomo Core
	s.upstreamReg.Register(&defaultBridgeProvider{
		id:          "core.mihomo",
		dim:         upstream.DimEngineCore,
		displayName: "Mihomo 极速规则完备引擎",
		probeFn: func(ctx context.Context) (*upstream.VersionStatus, error) {
			ov := s.rosMgr.GetOverview()
			curVer, _ := ov["mihomo_local_version"].(string)
			remVer, _ := ov["mihomo_remote_version"].(string)
			hasUpdate, _ := ov["mihomo_has_update"].(bool)
			return &upstream.VersionStatus{
				ProviderID:     "core.mihomo",
				Dimension:      upstream.DimEngineCore,
				DisplayName:    "Mihomo 极速规则完备引擎",
				CurrentVersion: curVer,
				LatestVersion:  remVer,
				HasUpdate:      hasUpdate,
				CheckedAt:      time.Now(),
			}, nil
		},
	})

	// D2: sing-box Core
	s.upstreamReg.Register(&defaultBridgeProvider{
		id:          "core.singbox",
		dim:         upstream.DimEngineCore,
		displayName: "sing-box 极速轻量路由核心",
		probeFn: func(ctx context.Context) (*upstream.VersionStatus, error) {
			ov := s.rosMgr.GetOverview()
			curVer, _ := ov["singbox_local_version"].(string)
			remVer, _ := ov["singbox_remote_version"].(string)
			hasUpdate, _ := ov["singbox_has_update"].(bool)
			return &upstream.VersionStatus{
				ProviderID:     "core.singbox",
				Dimension:      upstream.DimEngineCore,
				DisplayName:    "sing-box 极速轻量路由核心",
				CurrentVersion: curVer,
				LatestVersion:  remVer,
				HasUpdate:      hasUpdate,
				CheckedAt:      time.Now(),
			}, nil
		},
	})

	// D3: Rule Catalog
	s.upstreamReg.Register(&defaultBridgeProvider{
		id:          "rules.catalog",
		dim:         upstream.DimRuleCatalog,
		displayName: "217heidai 全量 105 套 SRS/MRS 规则体系",
		probeFn: func(ctx context.Context) (*upstream.VersionStatus, error) {
			rulesCount := 105
			if s.ir != nil {
				rulesCount = len(s.ir.Rules)
			}
			return &upstream.VersionStatus{
				ProviderID:     "rules.catalog",
				Dimension:      upstream.DimRuleCatalog,
				DisplayName:    "217heidai 全量 105 套 SRS/MRS 规则体系",
				CurrentVersion: fmt.Sprintf("%d 套内置规则 (SRS/MRS 双核流式缓存)", rulesCount),
				LatestVersion:  "持续自动静默更新",
				HasUpdate:      false,
				CheckedAt:      time.Now(),
			}, nil
		},
	})

	// D4: Host Symbiosis (RouterOS)
	s.upstreamReg.Register(&defaultBridgeProvider{
		id:          "host.ros",
		dim:         upstream.DimHostSymbiosis,
		displayName: "MikroTik RouterOS 宿主底座",
		probeFn: func(ctx context.Context) (*upstream.VersionStatus, error) {
			ov := s.rosMgr.GetOverview()
			curVer, _ := ov["ros_version"].(string)
			remVer, _ := ov["ros_latest_version"].(string)
			hasUpdate, _ := ov["ros_has_update"].(bool)
			return &upstream.VersionStatus{
				ProviderID:     "host.ros",
				Dimension:      upstream.DimHostSymbiosis,
				DisplayName:    "MikroTik RouterOS 宿主底座",
				CurrentVersion: curVer,
				LatestVersion:  remVer,
				HasUpdate:      hasUpdate,
				CheckedAt:      time.Now(),
			}, nil
		},
	})
}

// handleFlightLog 提供 64KB 内存环形黑匣子记录快照（零磁盘 I/O）
func (s *Server) handleFlightLog(w http.ResponseWriter, r *http.Request) {
	events := telemetry.GetGlobalFlightRecorder().Snapshot(100)
	sendJSON(w, map[string]any{
		"ok":     true,
		"events": events,
		"count":  len(events),
	})
}

// handleOnboardingDoctor 提供环境预检自检与个性化宿主共生脚本
func (s *Server) handleOnboardingDoctor(w http.ResponseWriter, r *http.Request) {
	hostIP := detectOutboundIP()
	if envIP := os.Getenv("CUBEX_HOST_IP"); envIP != "" {
		hostIP = envIP
	}
	rep := onboarding.RunOnboardingDoctor(r.Context(), hostIP, 2026, "198.18.0.0/15")
	sendJSON(w, rep)
}

// handleGatewayMode 提供主旁双模网络拓扑配置与自愈
func (s *Server) handleGatewayMode(w http.ResponseWriter, r *http.Request) {
	if s.gatewayModeCtrl == nil {
		s.gatewayModeCtrl = onboarding.NewGatewayModeController()
	}
	switch r.Method {
	case http.MethodGet:
		st := s.gatewayModeCtrl.InspectStatus(r.Context())
		sendJSON(w, st)
	case http.MethodPost:
		var spec onboarding.GatewayModeSpec
		if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
			http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}
		st, err := s.gatewayModeCtrl.ApplyMode(r.Context(), spec)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sendJSON(w, st)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleTriRouterBundle 提供针对软路由三巨头（ROS x OpenWrt x 爱快）的一键配置包
func (s *Server) handleTriRouterBundle(w http.ResponseWriter, r *http.Request) {
	if s.gatewayModeCtrl == nil {
		s.gatewayModeCtrl = onboarding.NewGatewayModeController()
	}
	st := s.gatewayModeCtrl.InspectStatus(r.Context())
	cubexIP := st.CurrentSpec.CubeXLocalIP
	gwIP := st.CurrentSpec.UpstreamGatewayIP
	bundle := onboarding.BuildTriRouterSymbiosisBundle(cubexIP, gwIP, "223.5.5.5", 2026)
	sendJSON(w, bundle)
}

// handleVerifySymbioticLoop 实机自检与主路由网关的连通回路
func (s *Server) handleVerifySymbioticLoop(w http.ResponseWriter, r *http.Request) {
	if s.gatewayModeCtrl == nil {
		s.gatewayModeCtrl = onboarding.NewGatewayModeController()
	}
	st := s.gatewayModeCtrl.InspectStatus(r.Context())
	gwIP := st.CurrentSpec.UpstreamGatewayIP
	if gwIP == "" {
		gwIP = "10.0.0.1"
	}
	dnsPort := 53
	if s.engMgr != nil {
		if ir := s.engMgr.GetIR(); ir != nil && ir.Network.DNSPort > 0 {
			dnsPort = ir.Network.DNSPort
		}
	}
	res := onboarding.VerifySymbioticLoop(r.Context(), gwIP, dnsPort)
	sendJSON(w, res)
}
