package onboarding

import (
	"strings"
	"testing"
)

func TestEssentialCIDRs(t *testing.T) {
	if len(EssentialNonDNSCIDRs) != 10 {
		t.Fatalf("expected 10 essential CIDRs, got %d", len(EssentialNonDNSCIDRs))
	}

	// 验证 Fake-IP 核心
	if EssentialNonDNSCIDRs[0].CIDR != "198.18.0.0/15" {
		t.Errorf("expected first CIDR to be 198.18.0.0/15, got %s", EssentialNonDNSCIDRs[0].CIDR)
	}

	// 验证 RouterOS 生成
	rosScript := BuildRouterOSFullRoutes("10.0.0.8")
	if !strings.Contains(rosScript, "gateway=10.0.0.8") {
		t.Errorf("rosScript should contain custom gateway 10.0.0.8")
	}
	if !strings.Contains(rosScript, "149.154.160.0/20") {
		t.Errorf("rosScript missing Telegram DC CIDR")
	}

	// 验证 OpenWrt 生成
	owrtScript := BuildOpenWrtFullRoutes("10.0.0.8")
	if !strings.Contains(owrtScript, "gateway='10.0.0.8'") {
		t.Errorf("owrtScript should contain custom gateway 10.0.0.8")
	}
	if !strings.Contains(owrtScript, "uci commit network") {
		t.Errorf("owrtScript should commit network")
	}
}
