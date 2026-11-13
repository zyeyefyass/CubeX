package upstream

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// ProbeMatrixReport summarizes four-dimensional health and version probe telemetry.
type ProbeMatrixReport struct {
	GeneratedAt time.Time                      `json:"generated_at"`
	Dimensions  map[Dimension][]*VersionStatus `json:"dimensions"`
	Errors      map[string]string              `json:"errors,omitempty"`
}

// Registry manages concurrency-controlled four-dimensional probes and atomic upgrades.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]UpstreamProvider
	installer *AtomicInstaller
	probeSem  chan struct{}
	upgradeMu sync.Mutex
}

// NewRegistry constructs an initialized Registry.
func NewRegistry(maxConcurrentProbes int, downloadTimeout time.Duration) *Registry {
	if maxConcurrentProbes <= 0 {
		maxConcurrentProbes = 4
	}
	return &Registry{
		providers: make(map[string]UpstreamProvider),
		installer: NewAtomicInstaller(downloadTimeout),
		probeSem:  make(chan struct{}, maxConcurrentProbes),
	}
}

// Register registers an upstream provider.
func (r *Registry) Register(p UpstreamProvider) {
	if p == nil || p.ID() == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.ID()] = p
}

// GetProvider retrieves a registered provider by ID.
func (r *Registry) GetProvider(id string) (UpstreamProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[id]
	return p, ok
}

// ProbeAll executes concurrent non-blocking probes across all dimensions with per-provider timeout isolation.
func (r *Registry) ProbeAll(ctx context.Context, perProviderTimeout time.Duration) *ProbeMatrixReport {
	if perProviderTimeout <= 0 {
		perProviderTimeout = 5 * time.Second
	}

	r.mu.RLock()
	list := make([]UpstreamProvider, 0, len(r.providers))
	for _, p := range r.providers {
		list = append(list, p)
	}
	r.mu.RUnlock()

	report := &ProbeMatrixReport{
		GeneratedAt: time.Now(),
		Dimensions:  make(map[Dimension][]*VersionStatus, 4),
		Errors:      make(map[string]string),
	}

	var resMu sync.Mutex
	var wg sync.WaitGroup

	for _, prov := range list {
		wg.Add(1)
		go func(p UpstreamProvider) {
			defer wg.Done()
			select {
			case r.probeSem <- struct{}{}:
				defer func() { <-r.probeSem }()
			case <-ctx.Done():
				return
			}

			pCtx, cancel := context.WithTimeout(ctx, perProviderTimeout)
			defer cancel()

			status, err := p.Probe(pCtx)
			resMu.Lock()
			defer resMu.Unlock()
			if err != nil {
				report.Errors[p.ID()] = err.Error()
				return
			}
			if status != nil {
				status.ProviderID = p.ID()
				status.Dimension = p.Dimension()
				if status.CheckedAt.IsZero() {
					status.CheckedAt = time.Now()
				}
				report.Dimensions[p.Dimension()] = append(report.Dimensions[p.Dimension()], status)
			}
		}(prov)
	}
	wg.Wait()

	// Sort deterministic output for UI and test assertions
	for dim := range report.Dimensions {
		items := report.Dimensions[dim]
		sort.Slice(items, func(i, j int) bool {
			return items[i].ProviderID < items[j].ProviderID
		})
	}

	return report
}

// UpgradeByID triggers an atomic upgrade transaction for the specified provider ID.
func (r *Registry) UpgradeByID(ctx context.Context, providerID string) (*UpgradeResult, error) {
	if !r.upgradeMu.TryLock() {
		return nil, fmt.Errorf("another component upgrade is already in progress")
	}
	defer r.upgradeMu.Unlock()

	r.mu.RLock()
	prov, ok := r.providers[providerID]
	r.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("upstream provider %q not registered", providerID)
	}

	return prov.Upgrade(ctx, r.installer)
}
