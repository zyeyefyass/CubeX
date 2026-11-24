package config

// Global System Defaults - Single Source of Truth
const (
	// DefaultPort is the standard web management cockpit port for CubeX
	DefaultPort = 2026
	// DefaultPortStr string representation of DefaultPort
	DefaultPortStr = "2026"
	// DefaultInstallDir is the standard host directory for binaries and config
	DefaultInstallDir = "/opt/cubex"
	// DefaultFakeIPRange is the RFC 2544 benchmark subnet reserved for Fake-IP
	DefaultFakeIPRange = "198.18.0.0/15"
	// DefaultFakeIPv6Range is the IPv6 benchmark subnet reserved for Fake-IP IPv6
	DefaultFakeIPv6Range = "fc00::/18"
)
