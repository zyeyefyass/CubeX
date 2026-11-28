package schema

import "testing"

func TestConfigIR_NormalizeDefaults(t *testing.T) {
	ir := &ConfigIR{
		Meta: MetaConfig{
			ActiveEngine: EngineType("unknown"),
		},
		Network: NetworkConfig{
			DNSPort: 1053,
			TunMTU:  1400, // custom user setting must be preserved
		},
		Subscriptions: []Subscription{{Name: "AirportA", URL: "https://example.com/sub"}},
	}
	ir.NormalizeDefaults()

	if ir.Meta.ActiveEngine != EngineMihomo || ir.Meta.ProxyMode != ModeRule || ir.Meta.LogLevel != "info" {
		t.Fatalf("unexpected normalized Meta: %+v", ir.Meta)
	}
	if ir.Network.DNSPort != 53 || ir.Network.ClashAPIPort != 9090 || ir.Network.TunMTU != 1400 || ir.Network.TunInterface != "cubex-tun" {
		t.Fatalf("unexpected normalized Network: %+v", ir.Network)
	}
	if ir.DNS.FakeIPRange != "198.18.0.0/15" || len(ir.DNS.DirectDNS) == 0 || len(ir.DNS.RemoteDNS) == 0 {
		t.Fatalf("unexpected normalized DNS: %+v", ir.DNS)
	}
	if len(ir.Subscriptions) != 1 || ir.Subscriptions[0].Name != "AirportA" {
		t.Fatalf("expected user Subscriptions preserved, got %+v", ir.Subscriptions)
	}
	if ir.Proxies == nil || ir.Rules == nil {
		t.Fatalf("expected nil slices initialized")
	}
}
