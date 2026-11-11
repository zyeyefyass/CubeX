package schema

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// SecretMaskPlaceholder is returned by read APIs when a secret/password is stored.
const SecretMaskPlaceholder = "******"

// MaskSecret returns a fixed mask placeholder if secret is non-empty.
func MaskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	return SecretMaskPlaceholder
}

// IsMaskedSecret reports whether v is the mask placeholder sent back unchanged by UI forms.
func IsMaskedSecret(v string) bool {
	return v == SecretMaskPlaceholder
}

// DetectOutboundIP probes the primary non-loopback LAN IPv4 address used for outbound traffic.
// Returns "" if no active network interface is found.
func DetectOutboundIP(targetHost string) string {
	if targetHost == "" {
		targetHost = "223.5.5.5"
	}
	conn, err := net.DialTimeout("udp", net.JoinHostPort(targetHost, "80"), 800*time.Millisecond)
	if err == nil {
		defer conn.Close()
		if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
			if ip4 := udpAddr.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
				return ip4.String()
			}
		}
	}

	ifaddrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, a := range ifaddrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					return ip4.String()
				}
			}
		}
	}
	return ""
}

// DetectDefaultGateway inspects /proc/net/route on Linux to find the active IPv4 default gateway.
// Returns "" when running outside Linux or when no default route is configured.
func DetectDefaultGateway() string {
	f, err := os.Open("/proc/net/route")
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return ""
	}
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 3 {
			continue
		}
		if fields[1] != "00000000" {
			continue
		}
		val, parseErr := strconv.ParseUint(fields[2], 16, 32)
		if parseErr != nil || val == 0 {
			continue
		}
		ip := net.IPv4(byte(val), byte(val>>8), byte(val>>16), byte(val>>24))
		return ip.String()
	}
	return ""
}

// DetectLANSubnet derives the /24 CIDR for a given IPv4 address, or "" if invalid.
func DetectLANSubnet(ipStr string) string {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return ""
	}
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	return net.IPv4(ip4[0], ip4[1], ip4[2], 0).String() + "/24"
}
