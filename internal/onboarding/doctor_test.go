package onboarding

import (
	"context"
	"path/filepath"
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

func TestCheckPersistMountFrom_ContainerMountsAndWritable(t *testing.T) {
	tmp := t.TempDir()
	// 1. Bare-metal / non-container writable directories pass as REQUIRED
	res := checkPersistMountFrom(tmp+"/etc", tmp+"/data", "", false)
	if !res.Passed || res.ID != "persist_mount" {
		t.Fatalf("expected non-container writable check to pass, got %+v", res)
	}

	// 2. Container with explicit mountpoints in mountinfo passes
	etcDir := tmp + "/etc"
	dataDir := tmp + "/data"
	normEtc := filepath.ToSlash(filepath.Clean(etcDir))
	normData := filepath.ToSlash(filepath.Clean(dataDir))
	mountOK := "101 80 8:1 /vol1/cubex/etc " + normEtc + " rw,relatime - ext4 /dev/sda1 rw\n" +
		"102 80 8:1 /vol1/cubex/data " + normData + " rw,relatime - ext4 /dev/sda1 rw\n"
	resOK := checkPersistMountFrom(etcDir, dataDir, mountOK, true)
	if !resOK.Passed {
		t.Fatalf("expected mounted container dirs to pass, got %+v", resOK)
	}
	mounts := parseMountPoints("101 80 8:1 /vol1/cubex/etc /app/etc rw,relatime - ext4 /dev/sda1 rw\n" +
		"102 80 8:1 /vol1/cubex/data /app/data rw,relatime - ext4 /dev/sda1 rw\n")
	if !mounts["/app/etc"] || !mounts["/app/data"] {
		t.Fatalf("parseMountPoints failed to extract /app/etc and /app/data: %+v", mounts)
	}

	// 3. Container missing /app/data mount flags clear warning & remediation
	mountMissing := "101 80 8:1 /vol1/cubex/etc " + normEtc + " rw - ext4 /dev/sda1 rw\n"
	resMissing := checkPersistMountFrom(etcDir, dataDir, mountMissing, true)
	if resMissing.Passed || resMissing.Severity != "RECOMMENDED" || !strings.Contains(resMissing.Detail, normData) {
		t.Fatalf("expected missing mount warning for %s, got %+v", normData, resMissing)
	}
}

func TestGatewayModeSpec_NormalizeDefaults(t *testing.T) {
	spec := GatewayModeSpec{
		Mode:              GatewayTopologyMode("invalid_legacy"),
		PrimaryInterface:  " eth0 ",
		CubeXLocalIP:      " 192.168.10.2 ",
		UpstreamGatewayIP: " 192.168.10.1 ",
		UpstreamDNS:       "  ",
	}
	spec.NormalizeDefaults()
	if spec.Mode != ModeSingleNICBypass || spec.UpstreamDNS != "223.5.5.5" ||
		spec.PrimaryInterface != "eth0" || spec.CubeXLocalIP != "192.168.10.2" {
		t.Fatalf("unexpected normalized GatewayModeSpec: %+v", spec)
	}
}

