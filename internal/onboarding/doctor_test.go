package onboarding

import (
	"context"
	"strings"
	"testing"
)

func TestBuildPersonalizedSnippets(t *testing.T) {
	customIP := "192.168.50.8"
	customPort := 2028
	fakeIP := "198.18.0.0/15"

	snippets := buildPersonalizedSnippets(customIP, customPort, fakeIP)

	// 1. 验证 RouterOS 脚本
	ros, ok := snippets["routeros_v7"]
	if !ok {
		t.Fatalf("missing routeros_v7 snippet")
	}
	if !strings.Contains(ros, "gateway=192.168.50.8") {
		t.Errorf("expected ROS script to contain gateway=%s, got:\n%s", customIP, ros)
	}
	if !strings.Contains(ros, "servers=192.168.50.8") {
		t.Errorf("expected ROS script to contain servers=%s, got:\n%s", customIP, ros)
	}
	if !strings.Contains(ros, "port=2028") {
		t.Errorf("expected ROS script to contain port=%d, got:\n%s", customPort, ros)
	}

	// 2. 验证 Docker 启动指令
	docker, ok := snippets["docker_cli"]
	if !ok {
		t.Fatalf("missing docker_cli snippet")
	}
	if !strings.Contains(docker, `CUBEX_HOST_IP="192.168.50.8"`) {
		t.Errorf("expected Docker command to contain CUBEX_HOST_IP, got:\n%s", docker)
	}
	if !strings.Contains(docker, `CUBEX_CONTROL_ADDR=":2028"`) {
		t.Errorf("expected Docker command to contain CUBEX_CONTROL_ADDR, got:\n%s", docker)
	}

	// 3. 验证 PVE LXC 配置
	pve, ok := snippets["pve_lxc"]
	if !ok {
		t.Fatalf("missing pve_lxc snippet")
	}
	if !strings.Contains(pve, "lxc.cgroup2.devices.allow: c 10:200 rwm") {
		t.Errorf("expected PVE config to contain cgroup rule, got:\n%s", pve)
	}
}

func TestRunOnboardingDoctor(t *testing.T) {
	ctx := context.Background()
	report := RunOnboardingDoctor(ctx, "10.0.0.99", 2026, "198.18.0.0/15")

	if report == nil {
		t.Fatalf("expected non-nil QuickstartGuideReport")
	}
	if report.CubeXHostIP != "10.0.0.99" {
		t.Errorf("expected host IP 10.0.0.99, got: %s", report.CubeXHostIP)
	}
	if report.ControlPort != 2026 {
		t.Errorf("expected port 2026, got: %d", report.ControlPort)
	}
	if len(report.Checks) < 3 {
		t.Errorf("expected at least 3 checks, got: %d", len(report.Checks))
	}
	if len(report.HostSnippets) < 3 {
		t.Errorf("expected at least 3 host snippets, got: %d", len(report.HostSnippets))
	}
}
