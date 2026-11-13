package onboarding

import (
	"context"
	"strings"
	"testing"
)

func TestGatewayMode_SelfLoopDetection(t *testing.T) {
	ctrl := NewGatewayModeController()
	ctx := context.Background()

	// 1. 同 IP 自环死锁应立即被拦截并返回明确中文报错
	_, err := ctrl.ApplyMode(ctx, GatewayModeSpec{
		Mode:              ModeSingleNICBypass,
		PrimaryInterface:  "eth0",
		CubeXLocalIP:      "192.168.1.50",
		UpstreamGatewayIP: "192.168.1.50",
	})
	if err == nil {
		t.Fatalf("expected error for gateway self-loop, got nil")
	}
	if !strings.Contains(err.Error(), "不能与 CubeX 本机 IP 相同") {
		t.Errorf("unexpected error message: %v", err)
	}

	// 2. 无效 IP 格式报错拦截
	_, err = ctrl.ApplyMode(ctx, GatewayModeSpec{
		Mode:              ModeSingleNICBypass,
		PrimaryInterface:  "eth0",
		CubeXLocalIP:      "192.168.1.50",
		UpstreamGatewayIP: "invalid-ip-address",
	})
	if err == nil {
		t.Fatalf("expected error for invalid gateway IP, got nil")
	}
}

func TestGatewayMode_PlainChineseSteps(t *testing.T) {
	// 模式一：单网口旁路由
	specBypass := GatewayModeSpec{
		Mode:              ModeSingleNICBypass,
		CubeXLocalIP:      "192.168.31.200",
		UpstreamGatewayIP: "192.168.31.1",
	}
	stepsBypass := buildPlainChineseSteps(specBypass)
	if len(stepsBypass) != 3 {
		t.Fatalf("expected 3 plain steps for bypass mode, got %d", len(stepsBypass))
	}
	if !strings.Contains(stepsBypass[0], "192.168.31.200") {
		t.Errorf("step 1 should contain local IP 192.168.31.200, got: %s", stepsBypass[0])
	}
	if !strings.Contains(stepsBypass[1], "192.168.31.1") {
		t.Errorf("step 2 should contain gateway IP 192.168.31.1, got: %s", stepsBypass[1])
	}
	// 校验无生僻黑话
	for _, s := range stepsBypass {
		if strings.Contains(s, "EWMA") || strings.Contains(s, "L2") || strings.Contains(s, "朱迪") || strings.Contains(s, "尼克") {
			t.Errorf("steps should not contain jargon, found in: %s", s)
		}
	}

	// 模式二：宿主共生分流
	specSymbiotic := GatewayModeSpec{
		Mode:              ModeSymbioticRoute,
		CubeXLocalIP:      "10.0.0.7",
		UpstreamGatewayIP: "10.0.0.1",
	}
	stepsSymbiotic := buildPlainChineseSteps(specSymbiotic)
	if len(stepsSymbiotic) != 3 {
		t.Fatalf("expected 3 plain steps for symbiotic mode, got %d", len(stepsSymbiotic))
	}
	if !strings.Contains(stepsSymbiotic[0], "10.0.0.1") {
		t.Errorf("step 1 should contain gateway IP 10.0.0.1, got: %s", stepsSymbiotic[0])
	}
	if !strings.Contains(stepsSymbiotic[1], "198.18.0.0/15") {
		t.Errorf("step 2 should contain 198.18.0.0/15, got: %s", stepsSymbiotic[1])
	}
	if !strings.Contains(stepsSymbiotic[2], "10.0.0.7") {
		t.Errorf("step 3 should contain local IP 10.0.0.7, got: %s", stepsSymbiotic[2])
	}
}

func TestGatewayMode_InspectStatus(t *testing.T) {
	ctrl := NewGatewayModeController()
	ctx := context.Background()

	status := ctrl.InspectStatus(ctx)
	if status == nil {
		t.Fatal("expected non-nil status")
	}
	if len(status.PlainUserSteps) != 3 {
		t.Errorf("expected 3 user steps, got %d", len(status.PlainUserSteps))
	}
}

func TestGatewayMode_PersistenceAndRestore(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := tmpDir + "/gateway_mode.json"

	ctrl := NewGatewayModeController()
	ctrl.BindPersistenceFile(stateFile)
	ctx := context.Background()

	spec := GatewayModeSpec{
		Mode:              ModeSingleNICBypass,
		PrimaryInterface:  "eth0",
		CubeXLocalIP:      "192.168.1.100",
		UpstreamGatewayIP: "192.168.1.1",
		UpstreamDNS:       "223.5.5.5",
		EnableMasquerade:  true,
	}

	_, err := ctrl.ApplyMode(ctx, spec)
	if err != nil {
		t.Fatalf("ApplyMode failed: %v", err)
	}

	// 模拟进程重启
	ctrl2 := NewGatewayModeController()
	ctrl2.BindPersistenceFile(stateFile)
	if err := ctrl2.RestoreOnBoot(ctx); err != nil {
		t.Fatalf("RestoreOnBoot failed: %v", err)
	}

	st := ctrl2.InspectStatus(ctx)
	if st.CurrentSpec.CubeXLocalIP != "192.168.1.100" {
		t.Errorf("expected restored IP 192.168.1.100, got: %s", st.CurrentSpec.CubeXLocalIP)
	}
	if st.CurrentSpec.Mode != ModeSingleNICBypass {
		t.Errorf("expected restored mode bypass_single_nic, got: %s", st.CurrentSpec.Mode)
	}
}
