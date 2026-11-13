package onboarding

import (
	"context"
	"strings"
	"testing"
)

func TestBuildTriRouterSymbiosisBundle(t *testing.T) {
	bundle := BuildTriRouterSymbiosisBundle("192.168.1.100", "192.168.1.1", "223.5.5.5", 2026)
	if bundle == nil {
		t.Fatal("BuildTriRouterSymbiosisBundle returned nil")
	}

	if !strings.Contains(bundle.RouterOSOneLiner, "192.168.1.100") {
		t.Errorf("RouterOS script missing CubeX IP: %s", bundle.RouterOSOneLiner)
	}
	if !strings.Contains(bundle.OpenWrtShellSetup, "cubex-watchdog.sh") {
		t.Errorf("OpenWrt script missing watchdog: %s", bundle.OpenWrtShellSetup)
	}
	if !strings.Contains(bundle.IKuaiStepMatrix, "198.18.0.0") {
		t.Errorf("iKuai matrix missing 198.18.0.0: %s", bundle.IKuaiStepMatrix)
	}
}

func TestVerifySymbioticLoop(t *testing.T) {
	ctx := context.Background()
	// Test with local loopback
	res := VerifySymbioticLoop(ctx, "127.0.0.1", 53)
	if res == nil {
		t.Fatal("VerifySymbioticLoop returned nil")
	}
	if !res.FakeIPRangeValid {
		t.Error("expected FakeIPRangeValid to be true")
	}
}
