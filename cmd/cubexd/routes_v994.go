package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/backup"
	"github.com/zyeyefyass/CubeX/internal/clashapi"
	"github.com/zyeyefyass/CubeX/internal/config"
	"github.com/zyeyefyass/CubeX/internal/netutil"
	"github.com/zyeyefyass/CubeX/internal/schema"
)

type threeLightProbeConfig struct {
	DomesticTargets  []string
	ProxyAddr        string
	ProxyProbeURL    string
	DNSAddr          string
	FakeIPTestDomain string
	ProbeTimeout     time.Duration
}

type threeLightConnResult struct {
	OK        bool   `json:"ok"`
	LatencyMs int    `json:"latency_ms"`
	Detail    string `json:"detail"`
}

type threeLightDNSResult struct {
	OK         bool   `json:"ok"`
	ResolvedIP string `json:"resolved_ip"`
	Detail     string `json:"detail"`
}

// ActiveNodeRate holds real-time upload/download throughput in bytes/sec for a single outbound node.
type ActiveNodeRate struct {
	UpBps   int64 `json:"up_bps"`
	DownBps int64 `json:"down_bps"`
}

type connRateSample struct {
	Upload   int64
	Download int64
}

type activeRateTracker struct {
	mu           sync.Mutex
	prevConns    map[string]connRateSample
	lastSampleAt time.Time
}

func newActiveRateTracker() *activeRateTracker {
	return &activeRateTracker{
		prevConns: make(map[string]connRateSample),
	}
}

func (s *Server) registerV994Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/v1/config/fakeip-filter", s.handleConfigFakeIPFilter)
	mux.HandleFunc("/api/v1/diagnostics/three-light-check", s.handleThreeLightCheck)
	mux.HandleFunc("/api/v1/system/backup/export", s.handleSystemBackupExport)
	mux.HandleFunc("/api/v1/system/backup/import", s.handleSystemBackupImport)
	mux.HandleFunc("/api/v1/proxy/active-rates", s.handleProxyActiveRates)
}

// --------------------------------------------------------------------------
// API 1: Fake-IP Compatible Whitelist Management
// --------------------------------------------------------------------------

func (s *Server) handleConfigFakeIPFilter(w http.ResponseWriter, r *http.Request) {
	if s.engMgr == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		sendJSON(w, map[string]any{"error": "engine manager not initialized"})
		return
	}
	ir := s.engMgr.GetIR()
	if ir == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		sendJSON(w, map[string]any{"error": "IR not initialized"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		custom := make([]string, len(ir.DNS.FakeIPFilter))
		copy(custom, ir.DNS.FakeIPFilter)
		sendJSON(w, map[string]any{
			"builtin_enabled": ir.DNS.IsFakeIPBuiltinEnabled(),
			"builtin_count":   schema.BuiltinFakeIPFilterCount(),
			"custom_domains":  custom,
		})
		return

	case http.MethodPost:
		var req struct {
			BuiltinEnabled *bool    `json:"builtin_enabled"`
			CustomDomains  []string `json:"custom_domains"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			sendJSON(w, map[string]any{"error": "invalid json: " + err.Error()})
			return
		}

		normalized, err := schema.NormalizeFakeIPDomains(req.CustomDomains)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			sendJSON(w, map[string]any{"error": err.Error()})
			return
		}

		builtinEnabled := ir.DNS.IsFakeIPBuiltinEnabled()
		if req.BuiltinEnabled != nil {
			builtinEnabled = *req.BuiltinEnabled
		}
		ir.DNS.SetFakeIPBuiltinEnabled(builtinEnabled)
		ir.DNS.FakeIPFilter = normalized
		ir.UpdatedAt = time.Now()

		if err := s.engMgr.UpdateIRWithErr(ir); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			sendJSON(w, map[string]any{"error": "failed to persist IR: " + err.Error()})
			return
		}
		s.ir = s.engMgr.GetIR()
		s.UpdateIRAndReload(s.ir, 50*time.Millisecond)

		if s.logBuffer != nil {
			s.logBuffer.Add("INFO", "DNS", fmt.Sprintf("Fake-IP 兼容白名单已更新 (builtin=%v, builtin_count=%d, custom=%d)",
				builtinEnabled, schema.BuiltinFakeIPFilterCount(), len(normalized)))
		}

		sendJSON(w, map[string]any{
			"builtin_enabled": builtinEnabled,
			"builtin_count":   schema.BuiltinFakeIPFilterCount(),
			"custom_domains":  normalized,
		})
		return

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

// --------------------------------------------------------------------------
// API 2: Three-Light One-Click Connectivity Check
// --------------------------------------------------------------------------

func (s *Server) handleThreeLightCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var ir *schema.ConfigIR
	if s.engMgr != nil {
		ir = s.engMgr.GetIR()
	}

	cfg := s.resolveThreeLightConfig(ir)

	var (
		domesticRes threeLightConnResult
		proxyRes    threeLightConnResult
		dnsRes      threeLightDNSResult
		wg          sync.WaitGroup
	)

	wg.Add(3)
	SafeGo("ThreeLightDomesticProbe", func() {
		defer wg.Done()
		domesticRes = probeDomesticReachability(cfg.DomesticTargets, cfg.ProbeTimeout)
	})
	SafeGo("ThreeLightProxyProbe", func() {
		defer wg.Done()
		proxyRes = probeProxyReachability(cfg.ProxyAddr, cfg.ProxyProbeURL, cfg.ProbeTimeout)
	})
	SafeGo("ThreeLightFakeIPDNSProbe", func() {
		defer wg.Done()
		fakeIPCIDR := config.DefaultFakeIPRange
		if ir != nil && strings.TrimSpace(ir.DNS.FakeIPRange) != "" {
			fakeIPCIDR = strings.TrimSpace(ir.DNS.FakeIPRange)
		}
		dnsRes = probeFakeIPDNSResolution(cfg.DNSAddr, cfg.FakeIPTestDomain, fakeIPCIDR, cfg.ProbeTimeout)
	})
	wg.Wait()

	dualProxyHint := detectDualProxyHint(r)

	sendJSON(w, map[string]any{
		"domestic":        domesticRes,
		"proxy":           proxyRes,
		"fakeip_dns":      dnsRes,
		"dual_proxy_hint": dualProxyHint,
	})
}

func (s *Server) resolveThreeLightConfig(ir *schema.ConfigIR) threeLightProbeConfig {
	mixedPort := 7890
	dnsPort := 53
	if ir != nil {
		if ir.Network.MixedPort > 0 {
			mixedPort = ir.Network.MixedPort
		}
		if ir.Network.DNSPort > 0 {
			dnsPort = ir.Network.DNSPort
		}
	}

	cfg := threeLightProbeConfig{
		DomesticTargets:  []string{"223.5.5.5:53", "119.29.29.29:53"},
		ProxyAddr:        fmt.Sprintf("127.0.0.1:%d", mixedPort),
		ProxyProbeURL:    "http://cp.cloudflare.com/generate_204",
		DNSAddr:          fmt.Sprintf("127.0.0.1:%d", dnsPort),
		FakeIPTestDomain: "www.google.com",
		ProbeTimeout:     2 * time.Second,
	}

	if s.threeLightCfg != nil {
		if len(s.threeLightCfg.DomesticTargets) > 0 {
			cfg.DomesticTargets = append([]string(nil), s.threeLightCfg.DomesticTargets...)
		}
		if s.threeLightCfg.ProxyAddr != "" {
			cfg.ProxyAddr = s.threeLightCfg.ProxyAddr
		}
		if s.threeLightCfg.ProxyProbeURL != "" {
			cfg.ProxyProbeURL = s.threeLightCfg.ProxyProbeURL
		}
		if s.threeLightCfg.DNSAddr != "" {
			cfg.DNSAddr = s.threeLightCfg.DNSAddr
		}
		if s.threeLightCfg.FakeIPTestDomain != "" {
			cfg.FakeIPTestDomain = s.threeLightCfg.FakeIPTestDomain
		}
		if s.threeLightCfg.ProbeTimeout > 0 {
			cfg.ProbeTimeout = s.threeLightCfg.ProbeTimeout
		}
	}
	return cfg
}

func probeDomesticReachability(targets []string, timeout time.Duration) threeLightConnResult {
	if len(targets) == 0 {
		return threeLightConnResult{OK: false, Detail: "未配置国内直连探针目标"}
	}
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	var lastErr error
	for _, target := range targets {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", target, timeout)
		elapsed := int(time.Since(start).Milliseconds())
		if err == nil {
			_ = conn.Close()
			if elapsed <= 0 {
				elapsed = 1
			}
			return threeLightConnResult{
				OK:        true,
				LatencyMs: elapsed,
				Detail:    fmt.Sprintf("国内直连正常 (%s, %dms)", target, elapsed),
			}
		}
		lastErr = err
	}
	detail := "国内直连超时或不可达"
	if lastErr != nil {
		detail = fmt.Sprintf("国内直连不可达: %v", lastErr)
	}
	return threeLightConnResult{OK: false, LatencyMs: 0, Detail: detail}
}

func probeProxyReachability(proxyAddr, probeURL string, timeout time.Duration) threeLightConnResult {
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}
	proxyURL, err := url.Parse("http://" + strings.TrimPrefix(strings.TrimPrefix(proxyAddr, "http://"), "https://"))
	if err != nil {
		return threeLightConnResult{OK: false, Detail: "本地代理地址无效: " + err.Error()}
	}

	tr := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DialContext:           (&net.Dialer{Timeout: timeout}).DialContext,
		ResponseHeaderTimeout: timeout,
		DisableKeepAlives:     true,
	}
	defer tr.CloseIdleConnections()

	client := &http.Client{
		Transport: tr,
		Timeout:   timeout,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return threeLightConnResult{OK: false, Detail: "构造代理探针失败: " + err.Error()}
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := int(time.Since(start).Milliseconds())
	if err != nil {
		return threeLightConnResult{
			OK:        false,
			LatencyMs: 0,
			Detail:    fmt.Sprintf("代理出口探针失败 (%s): %v", proxyAddr, err),
		}
	}
	netutil.DrainAndCloseBody(resp)

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		if elapsed <= 0 {
			elapsed = 1
		}
		return threeLightConnResult{
			OK:        true,
			LatencyMs: elapsed,
			Detail:    fmt.Sprintf("代理链路连通正常 (%s, HTTP %d, %dms)", proxyAddr, resp.StatusCode, elapsed),
		}
	}

	return threeLightConnResult{
		OK:        false,
		LatencyMs: elapsed,
		Detail:    fmt.Sprintf("代理节点返回异常状态 HTTP %d", resp.StatusCode),
	}
}

func probeFakeIPDNSResolution(dnsAddr, domain, fakeIPCIDR string, timeout time.Duration) threeLightDNSResult {
	if timeout <= 0 {
		timeout = 1500 * time.Millisecond
	}
	_, fakeNet, err := net.ParseCIDR(fakeIPCIDR)
	if err != nil {
		_, fakeNet, _ = net.ParseCIDR(config.DefaultFakeIPRange)
	}

	ipStr, err := queryDNSARecord(dnsAddr, domain, timeout)
	if err != nil {
		return threeLightDNSResult{
			OK:         false,
			ResolvedIP: "",
			Detail:     fmt.Sprintf("本地 DNS 解析失败 (%s): %v", dnsAddr, err),
		}
	}

	parsed := net.ParseIP(ipStr)
	if parsed != nil && fakeNet != nil && fakeNet.Contains(parsed) {
		return threeLightDNSResult{
			OK:         true,
			ResolvedIP: ipStr,
			Detail:     fmt.Sprintf("Fake-IP DNS 映射正常 (%s -> %s)", domain, ipStr),
		}
	}

	return threeLightDNSResult{
		OK:         false,
		ResolvedIP: ipStr,
		Detail:     fmt.Sprintf("DNS 解析未命中 Fake-IP 网段 %s (%s -> %s)", fakeIPCIDR, domain, ipStr),
	}
}

func buildDNSAQuery(domain string) ([]byte, error) {
	domain = strings.TrimSuffix(strings.TrimSpace(domain), ".")
	if domain == "" {
		return nil, fmt.Errorf("empty dns domain")
	}
	buf := make([]byte, 0, 64)
	// Header: ID=0x4358, Flags=0x0100 (RD=1), QDCOUNT=1
	buf = append(buf, 0x43, 0x58, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00)
	for _, label := range strings.Split(domain, ".") {
		if len(label) == 0 || len(label) > 63 {
			return nil, fmt.Errorf("invalid dns label %q", label)
		}
		buf = append(buf, byte(len(label)))
		buf = append(buf, label...)
	}
	buf = append(buf, 0x00, 0x00, 0x01, 0x00, 0x01) // QTYPE=A, QCLASS=IN
	return buf, nil
}

func skipDNSName(pkt []byte, offset int) (int, bool) {
	for offset < len(pkt) {
		length := int(pkt[offset])
		if length == 0 {
			return offset + 1, true
		}
		if length&0xC0 == 0xC0 {
			if offset+1 >= len(pkt) {
				return 0, false
			}
			return offset + 2, true
		}
		offset += 1 + length
	}
	return 0, false
}

func parseDNSARecordResponse(pkt []byte) (string, error) {
	if len(pkt) < 12 {
		return "", fmt.Errorf("truncated dns header")
	}
	qdCount := int(binary.BigEndian.Uint16(pkt[4:6]))
	anCount := int(binary.BigEndian.Uint16(pkt[6:8]))
	if anCount == 0 {
		return "", fmt.Errorf("no answer records in dns response")
	}

	offset := 12
	for i := 0; i < qdCount; i++ {
		next, ok := skipDNSName(pkt, offset)
		if !ok || next+4 > len(pkt) {
			return "", fmt.Errorf("malformed dns question")
		}
		offset = next + 4
	}

	for i := 0; i < anCount; i++ {
		next, ok := skipDNSName(pkt, offset)
		if !ok || next+10 > len(pkt) {
			return "", fmt.Errorf("malformed dns answer")
		}
		rtype := binary.BigEndian.Uint16(pkt[next : next+2])
		rdLen := int(binary.BigEndian.Uint16(pkt[next+8 : next+10]))
		rdataStart := next + 10
		if rdataStart+rdLen > len(pkt) {
			return "", fmt.Errorf("truncated dns rdata")
		}
		if rtype == 1 && rdLen == 4 {
			return net.IPv4(pkt[rdataStart], pkt[rdataStart+1], pkt[rdataStart+2], pkt[rdataStart+3]).String(), nil
		}
		offset = rdataStart + rdLen
	}
	return "", fmt.Errorf("no A record found in dns answer")
}

func queryDNSARecord(dnsAddr, domain string, timeout time.Duration) (string, error) {
	query, err := buildDNSAQuery(domain)
	if err != nil {
		return "", err
	}
	conn, err := net.DialTimeout("udp", dnsAddr, timeout)
	if err != nil {
		return "", err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write(query); err != nil {
		return "", err
	}
	resp := make([]byte, 1024)
	n, err := conn.Read(resp)
	if err != nil {
		return "", err
	}
	return parseDNSARecordResponse(resp[:n])
}

func detectDualProxyHint(r *http.Request) bool {
	if r != nil {
		for _, hdr := range []string{"Via", "Proxy-Connection", "Forwarded", "X-Forwarded-For"} {
			if strings.TrimSpace(r.Header.Get(hdr)) != "" {
				return true
			}
		}
	}
	for _, envKey := range []string{"HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "http_proxy", "https_proxy", "all_proxy"} {
		if strings.TrimSpace(os.Getenv(envKey)) != "" {
			return true
		}
	}
	return false
}

// --------------------------------------------------------------------------
// API 3: Configuration Backup & Restore (.tar.gz Migration Bundle)
// --------------------------------------------------------------------------

func (s *Server) handleSystemBackupExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var buf bytes.Buffer
	if err := backup.ExportConfigTarGz(s.workDir, &buf); err != nil {
		http.Error(w, "failed to export backup archive: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="cubex-config-backup.tar.gz"`)
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write(buf.Bytes())
	}
}

func (s *Server) handleSystemBackupImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, backup.MaxBackupUploadBytes)

	var stream io.Reader
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(backup.MaxBackupUploadBytes); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			sendJSON(w, map[string]any{"ok": false, "error": "invalid multipart upload: " + err.Error()})
			return
		}
		var file io.ReadCloser
		var err error
		for _, field := range []string{"file", "backup", "package"} {
			file, _, err = r.FormFile(field)
			if err == nil {
				break
			}
		}
		if err != nil && r.MultipartForm != nil {
			for _, files := range r.MultipartForm.File {
				if len(files) > 0 {
					file, err = files[0].Open()
					break
				}
			}
		}
		if err != nil || file == nil {
			w.WriteHeader(http.StatusBadRequest)
			sendJSON(w, map[string]any{"ok": false, "error": "missing backup file in multipart form"})
			return
		}
		defer file.Close()
		stream = file
	} else {
		stream = r.Body
	}

	restoredItems, err := backup.ImportConfigTarGz(s.workDir, stream, backup.MaxBackupUploadBytes)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		sendJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	// Reload IR and subsystems if config.ir.json was restored
	hasIR := false
	for _, item := range restoredItems {
		if item == "config.ir.json" {
			hasIR = true
			break
		}
	}
	if hasIR && s.engMgr != nil {
		irPath := filepath.Join(s.workDir, "data", "config.ir.json")
		if raw, rErr := os.ReadFile(irPath); rErr == nil {
			var restoredIR schema.ConfigIR
			if json.Unmarshal(raw, &restoredIR) == nil {
				restoredIR.NormalizeDefaults()
				schema.SyncAllStrategyGroups(&restoredIR)
				if uErr := s.engMgr.UpdateIRWithErr(&restoredIR); uErr == nil {
					s.ir = s.engMgr.GetIR()
					s.UpdateIRAndReload(s.ir, 50*time.Millisecond)
				}
			}
		}
	}
	if s.devMgr != nil {
		s.devMgr.Reload()
	}
	if s.logBuffer != nil {
		s.logBuffer.Add("INFO", "BACKUP", fmt.Sprintf("轻量迁移包原子恢复完成，共还原 %d 个核心配置文件", len(restoredItems)))
	}

	sendJSON(w, map[string]any{
		"ok":             true,
		"restored_items": restoredItems,
	})
}

// --------------------------------------------------------------------------
// API 4: Active Outbound Node Real-Time Rates Aggregation
// --------------------------------------------------------------------------

func (s *Server) handleProxyActiveRates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	emptyResp := func() {
		sendJSON(w, map[string]any{
			"active_node_rates": map[string]ActiveNodeRate{},
		})
	}

	if s.engMgr == nil {
		emptyResp()
		return
	}

	ir := s.engMgr.GetIR()
	clashPort := 9090
	clashSecret := ""
	if ir != nil {
		if ir.Network.ClashAPIPort > 0 {
			clashPort = ir.Network.ClashAPIPort
		}
		clashSecret = ir.Network.ClashSecret
	}

	clashBase := config.ResolveClashAPIBase(clashPort)
	apiClient := clashapi.NewClient(clashBase, clashSecret, netutil.NewBoundedHTTPClient(800*time.Millisecond))

	ctx, cancel := context.WithTimeout(r.Context(), 800*time.Millisecond)
	defer cancel()

	connsResp, err := apiClient.GetConnections(ctx)
	if err != nil || connsResp == nil {
		emptyResp()
		return
	}

	s.activeRateOnce.Do(func() {
		if s.activeRateTracker == nil {
			s.activeRateTracker = newActiveRateTracker()
		}
	})
	rates := s.activeRateTracker.Compute(connsResp.Connections, ir, time.Now())
	sendJSON(w, map[string]any{
		"active_node_rates": rates,
	})
}

func isBuiltInOutboundName(name string) bool {
	switch strings.ToUpper(strings.TrimSpace(name)) {
	case "", "DIRECT", "REJECT", "REJECT-DROP", "PASS", "COMPATIBLE", "GLOBAL", "BLOCK", "DNS-OUT":
		return true
	}
	return false
}

func resolveConnectionOutboundNode(chains []string, proxySet, groupSet map[string]bool) string {
	if len(chains) == 0 {
		return ""
	}
	// First pass: exact match against known physical proxy nodes in IR
	for _, c := range chains {
		c = strings.TrimSpace(c)
		if isBuiltInOutboundName(c) {
			continue
		}
		if proxySet[c] {
			return c
		}
	}
	// Second pass: pick first chain element that is neither a built-in nor a strategy group
	for _, c := range chains {
		c = strings.TrimSpace(c)
		if isBuiltInOutboundName(c) || groupSet[c] {
			continue
		}
		return c
	}
	return ""
}

func (t *activeRateTracker) Compute(conns []clashapi.ConnectionEntry, ir *schema.ConfigIR, now time.Time) map[string]ActiveNodeRate {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.prevConns == nil {
		t.prevConns = make(map[string]connRateSample)
	}

	proxySet := make(map[string]bool)
	groupSet := make(map[string]bool)
	if ir != nil {
		for _, p := range ir.Proxies {
			if !isBuiltInOutboundName(p.Name) {
				proxySet[p.Name] = true
			}
		}
		for _, g := range ir.Groups {
			if g.Name != "" {
				groupSet[g.Name] = true
			}
		}
	}

	hasPrevTick := !t.lastSampleAt.IsZero()
	dt := now.Sub(t.lastSampleAt).Seconds()
	if !hasPrevTick || dt < 0.05 {
		dt = 1.0
	}

	nextPrev := make(map[string]connRateSample, len(conns))
	rates := make(map[string]ActiveNodeRate)

	for _, c := range conns {
		if c.ID == "" {
			continue
		}
		node := resolveConnectionOutboundNode(c.Chains, proxySet, groupSet)
		if node == "" {
			continue
		}

		nextPrev[c.ID] = connRateSample{
			Upload:   c.Upload,
			Download: c.Download,
		}

		var dUp, dDown int64
		if prev, ok := t.prevConns[c.ID]; ok {
			if c.Upload > prev.Upload {
				dUp = c.Upload - prev.Upload
			}
			if c.Download > prev.Download {
				dDown = c.Download - prev.Download
			}
		} else if hasPrevTick {
			if c.Upload > 0 {
				dUp = c.Upload
			}
			if c.Download > 0 {
				dDown = c.Download
			}
		} else if !c.Start.IsZero() {
			if age := now.Sub(c.Start); age >= 100*time.Millisecond && age <= 120*time.Second {
				cur := rates[node]
				cur.UpBps += int64(float64(c.Upload) / age.Seconds())
				cur.DownBps += int64(float64(c.Download) / age.Seconds())
				rates[node] = cur
				continue
			}
		}

		cur := rates[node]
		cur.UpBps += int64(float64(dUp) / dt)
		cur.DownBps += int64(float64(dDown) / dt)
		rates[node] = cur
	}

	t.prevConns = nextPrev
	t.lastSampleAt = now
	return rates
}
