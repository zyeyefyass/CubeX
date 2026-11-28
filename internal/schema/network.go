package schema

import (
	"net"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/netenv"
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
	if targetHost != "" {
		conn, err := net.DialTimeout("udp", net.JoinHostPort(targetHost, "80"), 800*time.Millisecond)
		if err == nil {
			defer conn.Close()
			if udpAddr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
				if ip4 := udpAddr.IP.To4(); ip4 != nil && !ip4.IsLoopback() {
					return ip4.String()
				}
			}
		}
	}
	return netenv.DetectHostTopology("", "").LocalIP
}

// DetectDefaultGateway inspects /proc/net/route on Linux via netenv to find the active IPv4 default gateway.
// Returns "" when running outside Linux or when no default route is configured.
func DetectDefaultGateway() string {
	return netenv.DetectHostTopology("", "").GatewayIP
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
