package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SchedulerConfig defines parameters for closed-loop EWMA route arbitration.
type SchedulerConfig struct {
	ClashAPIBase      string        // e.g. "http://127.0.0.1:9090"
	ClashAPISecret    string        // optional Bearer Token
	ProbeTargetURL    string        // default "https://www.gstatic.com/generate_204"
	TickInterval      time.Duration // scheduling cycle, default 12s
	ProbeTimeout      time.Duration // per-node probe timeout, default 2500ms
	MaxConcurrency    int           // maximum concurrent active probes, default 4 (keeps memory ~35MB)
	HotStandbyCount   int           // T1 hot-standby top candidates count, default 3
	ColdBatchSize     int           // T2 cold-pool sweep count per tick, default 2
	SwitchMarginMs    int           // proactive absolute improvement margin, default 25ms
	SwitchMarginRatio float64       // proactive relative improvement ratio, default 0.18 (18%)
	SwitchCooldown    time.Duration // per-group proactive switch cooldown, default 60s
	ManagedGroups     []string      // explicit groups to manage (empty = auto-detect all non-builtin selector groups)
}

// SwitchEvent records an audited closed-loop proxy routing transition.
type SwitchEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Group     string    `json:"group"`
	FromNode  string    `json:"from_node"`
	ToNode    string    `json:"to_node"`
	Reason    string    `json:"reason"`
	Emergency bool      `json:"emergency"`
}

// EWMAScheduler provides microsecond-precision adaptive closed-loop route optimization.
type EWMAScheduler struct {
	cfg           SchedulerConfig
	ewma          *EWMAManager
	httpClient    *http.Client
	sem           chan struct{}
	mu            sync.Mutex
	coldCursors   map[string]int       // tracks ring cursor position per policy group
	groupSwitchAt map[string]time.Time // tracks last switch timestamp per policy group
	penalizedUntil map[string]time.Time // tracks temporary blackhole penalty expiration per node (default 300s)
	pausedUntil   atomic.Int64         // kernel switchover silence barrier timestamp (UnixNano)
	onSwitch      func(ev SwitchEvent)
}

// NewEWMAScheduler constructs an initialized EWMAScheduler.
func NewEWMAScheduler(cfg SchedulerConfig, ewma *EWMAManager, onSwitch func(SwitchEvent)) *EWMAScheduler {
	if cfg.ProbeTargetURL == "" {
		cfg.ProbeTargetURL = "https://www.gstatic.com/generate_204"
	}
	if cfg.TickInterval <= 0 {
		cfg.TickInterval = 12 * time.Second
	}
	if cfg.ProbeTimeout <= 0 {
		cfg.ProbeTimeout = 2500 * time.Millisecond
	}
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 4
	}
	if cfg.HotStandbyCount <= 0 {
		cfg.HotStandbyCount = 3
	}
	if cfg.ColdBatchSize <= 0 {
		cfg.ColdBatchSize = 2
	}
	if cfg.SwitchMarginMs <= 0 {
		cfg.SwitchMarginMs = 25
	}
	if cfg.SwitchMarginRatio <= 0 {
		cfg.SwitchMarginRatio = 0.18
	}
	if cfg.SwitchCooldown <= 0 {
		cfg.SwitchCooldown = 60 * time.Second
	}

	tr := &http.Transport{
		MaxIdleConns:        8,
		MaxIdleConnsPerHost: 4,
		IdleConnTimeout:     30 * time.Second,
	}

	return &EWMAScheduler{
		cfg:            cfg,
		ewma:           ewma,
		httpClient:     &http.Client{Transport: tr, Timeout: cfg.ProbeTimeout + 1*time.Second},
		sem:            make(chan struct{}, cfg.MaxConcurrency),
		coldCursors:    make(map[string]int),
		groupSwitchAt:  make(map[string]time.Time),
		penalizedUntil: make(map[string]time.Time),
		onSwitch:       onSwitch,
	}
}

// NotifyEngineSwitched pauses probing for 5 seconds during dual-engine hot-swap to avoid false alarms.
func (s *EWMAScheduler) NotifyEngineSwitched() {
	s.pausedUntil.Store(time.Now().Add(5 * time.Second).UnixNano())
}

// Run executes the continuous background closed-loop scheduling loop.
func (s *EWMAScheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.TickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if time.Now().UnixNano() < s.pausedUntil.Load() {
				continue
			}
			s.runSingleCycle(ctx)
		}
	}
}

type clashProxiesResp struct {
	Proxies map[string]clashProxyItem `json:"proxies"`
}

type clashProxyItem struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Now     string   `json:"now"`
	All     []string `json:"all"`
	History []struct {
		Delay int `json:"delay"`
	} `json:"history"`
}

// RunSingleCycle executes a single round of tiered sampling and closed-loop routing evaluation (public for testing).
func (s *EWMAScheduler) RunSingleCycle(ctx context.Context) {
	s.runSingleCycle(ctx)
}

func (s *EWMAScheduler) runSingleCycle(ctx context.Context) {
	s.mu.Lock()
	s.sweepPenaltiesLocked(time.Now())
	s.mu.Unlock()

	proxies, err := s.fetchProxiesSnapshot(ctx)
	if err != nil || len(proxies) == 0 {
		return
	}

	// Step 1: Collect managed groups and build deduplicated nodes to probe
	targetGroups := s.filterManagedGroups(proxies)
	if len(targetGroups) == 0 {
		return
	}

	nodesToProbe := make(map[string]struct{}, 16)
	allValidNodes := make(map[string]struct{}, len(proxies))

	for name, item := range proxies {
		if !isPolicyGroupType(item.Type) && !isBuiltInNode(name) {
			allValidNodes[name] = struct{}{}
		}
	}
	s.ewma.PruneStaleNodes(allValidNodes)

	for _, grp := range targetGroups {
		candidates := filterRealNodes(grp.All, proxies)
		if len(candidates) <= 1 {
			continue
		}
		batch := s.SelectTieredBatch(grp.Name, grp.Now, candidates)
		for _, nodeName := range batch {
			nodesToProbe[nodeName] = struct{}{}
		}
	}

	// Step 2: Rate-limited concurrent probe execution via semaphore
	var wg sync.WaitGroup
	for nodeName := range nodesToProbe {
		if ctx.Err() != nil || time.Now().UnixNano() < s.pausedUntil.Load() {
			break
		}
		wg.Add(1)
		go func(node string) {
			defer wg.Done()
			select {
			case s.sem <- struct{}{}:
				defer func() { <-s.sem }()
			case <-ctx.Done():
				return
			}

			delayMs, ok := s.probeNodeDelay(ctx, node)
			s.ewma.RecordSample(node, delayMs, ok)
		}(nodeName)
	}
	wg.Wait()

	// Step 3: Evaluate dual-mode failover and optimization per group
	for _, grp := range targetGroups {
		if time.Now().UnixNano() < s.pausedUntil.Load() {
			return
		}
		candidates := filterRealNodes(grp.All, proxies)
		if len(candidates) <= 1 {
			continue
		}
		s.EvaluateAndSwitchGroup(ctx, grp.Name, grp.Now, candidates)
	}
}

// SelectTieredBatch builds a T0(Active) + T1(Hot Standby Top-K) + T2(Cold Ring Sweep) batch (public for testing).
func (s *EWMAScheduler) SelectTieredBatch(groupName, activeNode string, candidates []string) []string {
	selected := make([]string, 0, 1+s.cfg.HotStandbyCount+s.cfg.ColdBatchSize)
	seen := make(map[string]struct{}, 8)

	addNode := func(n string) {
		if n == "" {
			return
		}
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			selected = append(selected, n)
		}
	}

	// [T0 Active Sentinel] Always probe
	addNode(activeNode)

	// [T1 Hot Standby Layer] Top-K candidates by EWMA score
	ranked := s.ewma.RankNodes(candidates)
	for _, r := range ranked {
		if len(selected) >= 1+s.cfg.HotStandbyCount {
			break
		}
		addNode(r.NodeName)
	}

	// [T2 Cold Sweep Layer] Ring-cursor sharding
	s.mu.Lock()
	cursor := s.coldCursors[groupName]
	n := len(candidates)
	addedCold := 0
	if n > 0 {
		for i := 0; i < n && addedCold < s.cfg.ColdBatchSize; i++ {
			idx := (cursor + i) % n
			cand := candidates[idx]
			if _, exists := seen[cand]; !exists {
				addNode(cand)
				addedCold++
			}
		}
		s.coldCursors[groupName] = (cursor + s.cfg.ColdBatchSize) % n
	}
	s.mu.Unlock()

	return selected
}

// EvaluateAndSwitchGroup executes dual-mode switching logic (public for testing).
func (s *EWMAScheduler) EvaluateAndSwitchGroup(ctx context.Context, groupName, currentActive string, candidates []string) {
	ranked := s.ewma.RankNodes(candidates)
	if len(ranked) == 0 || !ranked[0].HasSample || ranked[0].ConsecutiveFails > 0 {
		return
	}

	best := ranked[0]
	if best.NodeName == currentActive {
		return
	}

	curStat, hasCur := s.ewma.GetNodeSnapshot(currentActive)

	// [Mode A: Fast-Failover Emergency] Active node failed >= 2 times, bypass cooldown immediately
	if !hasCur || curStat.ConsecutiveFails >= 2 {
		reason := fmt.Sprintf("fast-failover: active node %s consecutive_fails=%d, switch to %s (eff=%dms)",
			currentActive, curStat.ConsecutiveFails, best.NodeName, best.EffectiveDelay)
		if err := s.switchClashGroup(ctx, groupName, best.NodeName); err == nil {
			s.recordSwitch(groupName, currentActive, best.NodeName, reason, true)
		}
		return
	}

	// [Mode B: Proactive Hysteresis Optimization] Check cooldown and 3-stage threshold
	s.mu.Lock()
	lastSwitched := s.groupSwitchAt[groupName]
	s.mu.Unlock()
	if !lastSwitched.IsZero() && time.Since(lastSwitched) < s.cfg.SwitchCooldown {
		return
	}

	should, reason := s.ewma.ShouldSwitch(
		currentActive,
		best.NodeName,
		s.cfg.SwitchMarginMs,
		s.cfg.SwitchMarginRatio,
		s.cfg.SwitchCooldown,
	)
	if !should {
		return
	}

	if err := s.switchClashGroup(ctx, groupName, best.NodeName); err == nil {
		s.recordSwitch(groupName, currentActive, best.NodeName, reason, false)
	}
}

func (s *EWMAScheduler) recordSwitch(group, from, to, reason string, emergency bool) {
	s.mu.Lock()
	s.groupSwitchAt[group] = time.Now()
	s.mu.Unlock()

	s.ewma.MarkGroupSwitched(to)
	if s.onSwitch != nil {
		s.onSwitch(SwitchEvent{
			Timestamp: time.Now(),
			Group:     group,
			FromNode:  from,
			ToNode:    to,
			Reason:    reason,
			Emergency: emergency,
		})
	}
}

func (s *EWMAScheduler) probeNodeDelay(ctx context.Context, nodeName string) (int, bool) {
	u := fmt.Sprintf("%s/proxies/%s/delay?url=%s&timeout=%d",
		strings.TrimRight(s.cfg.ClashAPIBase, "/"),
		url.PathEscape(nodeName),
		url.QueryEscape(s.cfg.ProbeTargetURL),
		s.cfg.ProbeTimeout.Milliseconds(),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, false
	}
	if s.cfg.ClashAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.ClashAPISecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, false
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return 0, false
	}

	var data struct {
		Delay int `json:"delay"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil || data.Delay <= 0 {
		return 0, false
	}
	return data.Delay, true
}

func (s *EWMAScheduler) switchClashGroup(ctx context.Context, groupName, targetNode string) error {
	if s.cfg.ClashAPIBase == "" {
		return nil
	}
	u := fmt.Sprintf("%s/proxies/%s",
		strings.TrimRight(s.cfg.ClashAPIBase, "/"),
		url.PathEscape(groupName),
	)
	bodyBytes, _ := json.Marshal(map[string]string{"name": targetNode})

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, u, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.ClashAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.ClashAPISecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("clash api switch group %s returned status %d", groupName, resp.StatusCode)
	}
	return nil
}

func (s *EWMAScheduler) fetchProxiesSnapshot(ctx context.Context) (map[string]clashProxyItem, error) {
	u := strings.TrimRight(s.cfg.ClashAPIBase, "/") + "/proxies"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if s.cfg.ClashAPISecret != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.ClashAPISecret)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var parsed clashProxiesResp
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}
	return parsed.Proxies, nil
}

func (s *EWMAScheduler) filterManagedGroups(proxies map[string]clashProxyItem) []clashProxyItem {
	managedSet := make(map[string]struct{}, len(s.cfg.ManagedGroups))
	for _, g := range s.cfg.ManagedGroups {
		managedSet[g] = struct{}{}
	}

	out := make([]clashProxyItem, 0, 4)
	for name, item := range proxies {
		if !strings.EqualFold(item.Type, "Selector") && !strings.EqualFold(item.Type, "URLTest") && !strings.EqualFold(item.Type, "Fallback") {
			continue
		}
		if name == "GLOBAL" {
			continue
		}
		if len(managedSet) > 0 {
			if _, ok := managedSet[name]; !ok {
				continue
			}
		}
		out = append(out, item)
	}
	return out
}

func filterRealNodes(all []string, proxies map[string]clashProxyItem) []string {
	out := make([]string, 0, len(all))
	for _, name := range all {
		if isBuiltInNode(name) {
			continue
		}
		if p, ok := proxies[name]; ok && !isPolicyGroupType(p.Type) {
			out = append(out, name)
		}
	}
	return out
}

func isPolicyGroupType(t string) bool {
	switch strings.ToLower(t) {
	case "selector", "urltest", "fallback", "loadbalance", "relay":
		return true
	default:
		return false
	}
}

func isBuiltInNode(name string) bool {
	switch strings.ToUpper(name) {
	case "DIRECT", "REJECT", "REJECT-DROP", "PASS", "COMPATIBLE", "GLOBAL":
		return true
	default:
		return false
	}
}

// sweepPenaltiesLocked clears expired node penalties and restores their candidate eligibility in EWMA.
func (s *EWMAScheduler) sweepPenaltiesLocked(now time.Time) {
	if s.penalizedUntil == nil {
		return
	}
	for node, until := range s.penalizedUntil {
		if now.After(until) {
			delete(s.penalizedUntil, node)
			if s.ewma != nil {
				s.ewma.ResetPenalty(node)
			}
		}
	}
}

// IsNodePenalized checks if a node is currently under temporary blackhole quarantine.
func (s *EWMAScheduler) IsNodePenalized(nodeName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sweepPenaltiesLocked(time.Now())
	if s.penalizedUntil == nil {
		return false
	}
	_, penalized := s.penalizedUntil[nodeName]
	return penalized
}

// SetNodePenalizedForTest configures a custom penalty expiration timestamp for testing.
func (s *EWMAScheduler) SetNodePenalizedForTest(nodeName string, until time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.penalizedUntil == nil {
		s.penalizedUntil = make(map[string]time.Time)
	}
	s.penalizedUntil[nodeName] = until
}

// PenalizeAndRotateActive applies an immediate failure penalty to the active node in managed groups
// and rotates to the best backup node to break out of a silent blackhole.
// The penalized node is placed under a 300-second cooling window after which candidate eligibility is restored.
func (s *EWMAScheduler) PenalizeAndRotateActive(reason string) (prevNode, nextNode string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	proxies, err := s.fetchProxiesSnapshot(ctx)
	if err != nil {
		return "", ""
	}
	groups := s.filterManagedGroups(proxies)
	if len(groups) == 0 {
		return "", ""
	}

	now := time.Now()
	s.mu.Lock()
	s.sweepPenaltiesLocked(now)
	s.mu.Unlock()

	for _, g := range groups {
		if g.Now == "" || len(g.All) <= 1 {
			continue
		}
		realCandidates := filterRealNodes(g.All, proxies)
		if len(realCandidates) <= 1 {
			continue
		}

		prevNode = g.Now
		// Penalize current active node heavily (consecutive fails and high penalty delay)
		if s.ewma != nil {
			s.ewma.RecordSample(g.Now, 9999, false)
			s.ewma.RecordSample(g.Now, 9999, false)
		}

		// Register 300-second automatic decay cooling window to prevent infinite blacklisting
		s.mu.Lock()
		if s.penalizedUntil == nil {
			s.penalizedUntil = make(map[string]time.Time)
		}
		s.penalizedUntil[g.Now] = time.Now().Add(300 * time.Second)
		s.mu.Unlock()

		var ranked []CandidateRank
		if s.ewma != nil {
			ranked = s.ewma.RankNodes(realCandidates)
		}

		s.mu.Lock()
		for _, cand := range ranked {
			if cand.NodeName != g.Now {
				if until, ok := s.penalizedUntil[cand.NodeName]; ok && time.Now().Before(until) {
					continue // skip candidate in penalty cooling window
				}
				nextNode = cand.NodeName
				if err := s.switchClashGroup(ctx, g.Name, cand.NodeName); err == nil {
					s.recordSwitch(g.Name, g.Now, cand.NodeName, reason, true)
				}
				break
			}
		}

		// Fallback to first alternate node if EWMA ranking returned nothing
		if nextNode == "" {
			for _, cand := range realCandidates {
				if cand != g.Now {
					if until, ok := s.penalizedUntil[cand]; ok && time.Now().Before(until) {
						continue
					}
					nextNode = cand
					if err := s.switchClashGroup(ctx, g.Name, cand); err == nil {
						s.recordSwitch(g.Name, g.Now, cand, reason, true)
					}
					break
				}
			}
		}
		s.mu.Unlock()

		if nextNode != "" {
			break
		}
	}
	return prevNode, nextNode
}

