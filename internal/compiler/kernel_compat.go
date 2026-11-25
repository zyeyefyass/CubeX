package compiler

import (
	"os"
	"runtime"
	"strconv"
	"strings"
)

// DetectLinuxKernelMajorMinor parses the running Linux kernel major/minor version
// (or CUBEX_KERNEL_RELEASE override for cross-platform/unit-test verification).
func DetectLinuxKernelMajorMinor() (major, minor int, raw string, ok bool) {
	raw = strings.TrimSpace(os.Getenv("CUBEX_KERNEL_RELEASE"))
	if raw == "" {
		if runtime.GOOS != "linux" {
			return 0, 0, "", false
		}
		data, err := os.ReadFile("/proc/sys/kernel/osrelease")
		if err != nil {
			return 0, 0, "", false
		}
		raw = strings.TrimSpace(string(data))
	}
	// Strip suffix like "4.4.60+" or "6.12.112-amd64"
	corePart := strings.SplitN(raw, "-", 2)[0]
	corePart = strings.TrimRight(corePart, "+")
	parts := strings.Split(corePart, ".")
	if len(parts) < 2 {
		return 0, 0, raw, false
	}
	maj, err1 := strconv.Atoi(parts[0])
	min, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return 0, 0, raw, false
	}
	return maj, min, raw, true
}

// IsLegacyLinuxKernel returns true when running on Linux kernels older than 4.19
// (such as Synology DSM 7.x Linux 4.4.60), where modern TUN options (gso/mixed/ipv6 route) may crash cores.
func IsLegacyLinuxKernel() (bool, string) {
	maj, min, raw, ok := DetectLinuxKernelMajorMinor()
	if !ok {
		return false, ""
	}
	return maj < 4 || (maj == 4 && min < 19), raw
}

// HasKernelIPv6Stack checks whether the Linux host kernel exposes active IPv6 interfaces.
func HasKernelIPv6Stack() bool {
	if strings.TrimSpace(os.Getenv("CUBEX_DISABLE_IPV6_DETECT")) == "1" {
		return false
	}
	if runtime.GOOS != "linux" && strings.TrimSpace(os.Getenv("CUBEX_KERNEL_RELEASE")) == "" {
		return true
	}
	data, err := os.ReadFile("/proc/net/if_inet6")
	return err == nil && len(strings.TrimSpace(string(data))) > 0
}
