package upstream

import (
	"context"
	"time"
)

// Dimension defines the four-dimensional categories for version and probe telemetry.
type Dimension string

const (
	DimControlPlane  Dimension = "control_plane"  // D1: CubeX Control Plane daemon & WebUI assets
	DimEngineCore    Dimension = "engine_core"    // D2: Mihomo / sing-box dual core engines
	DimRuleCatalog   Dimension = "rule_catalog"   // D3: 105 sets of SRS/MRS and GeoData rules
	DimHostSymbiosis Dimension = "host_symbiosis" // D4: RouterOS v7 / PVE / OpenWrt / iKuai host symbiosis
)

// VersionStatus packages a snapshot of a component's four-dimensional version and probe status.
type VersionStatus struct {
	ProviderID     string            `json:"provider_id"`     // e.g. "core.sing-box", "host.routeros"
	Dimension      Dimension         `json:"dimension"`       // Category dimension
	DisplayName    string            `json:"display_name"`    // Human-readable title
	CurrentVersion string            `json:"current_version"` // Local running version
	LatestVersion  string            `json:"latest_version"`  // Remote upstream latest version
	HasUpdate      bool              `json:"has_update"`      // Whether an update is available
	DownloadURL    string            `json:"download_url,omitempty"`
	ExpectedSHA256 string            `json:"expected_sha256,omitempty"`
	AssetSize      int64             `json:"asset_size,omitempty"`
	CheckedAt      time.Time         `json:"checked_at"`
	Metadata       map[string]string `json:"metadata,omitempty"` // Host arch, channel, release notes, etc.
}

// UpgradeResult records an audited execution of an atomic upgrade transaction.
type UpgradeResult struct {
	ProviderID  string `json:"provider_id"`
	FromVersion string `json:"from_version"`
	ToVersion   string `json:"to_version"`
	CostMs      int64  `json:"cost_ms"`
	RolledBack  bool   `json:"rolled_back"`
	Message     string `json:"message"`
}

// UpstreamProvider defines the unified lifecycle contract for engines, rule catalogs, and host platforms.
type UpstreamProvider interface {
	// ID returns the globally unique provider identifier (e.g. "core.mihomo", "rules.217heidai", "host.ros")
	ID() string
	// Dimension returns the four-dimensional category
	Dimension() Dimension
	// Probe performs a lightweight upstream version and metadata check (must not download heavy payloads)
	Probe(ctx context.Context) (*VersionStatus, error)
	// Upgrade performs streaming download, pre-flight verification, POSIX atomic replacement, and auto-rollback on failure
	Upgrade(ctx context.Context, tx *AtomicInstaller) (*UpgradeResult, error)
}
