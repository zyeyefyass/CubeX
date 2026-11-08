package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestZeroTrustSandboxManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "zerotrust_test_*")
	if err != nil {
		t.Fatalf("创建临时测试目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewZeroTrustSandboxManager(tempDir)
	status := mgr.GetStatus()

	if !status.Enabled {
		t.Errorf("预期初始为开启状态，实际为: %v", status.Enabled)
	}
	if len(status.Devices) == 0 {
		t.Errorf("预期初始默认设备沙盒不为空")
	}

	// 1. 测试设置设备策略
	testMAC := "AA:BB:CC:11:22:33"
	updated, err := mgr.SetDevicePolicy(testMAC, "TRUSTED")
	if err != nil {
		t.Fatalf("设置设备策略失败: %v", err)
	}
	var found bool
	for _, d := range updated.Devices {
		if d.MAC == testMAC {
			found = true
			if d.Policy != "TRUSTED" {
				t.Errorf("预期策略为 TRUSTED，实际为: %s", d.Policy)
			}
			if d.TrustScore != 100 {
				t.Errorf("预期信任评分为 100，实际为: %d", d.TrustScore)
			}
		}
	}
	if !found {
		t.Errorf("未找到测试 MAC: %s", testMAC)
	}

	// 无效策略测试
	_, err = mgr.SetDevicePolicy(testMAC, "INVALID_POLICY")
	if err == nil {
		t.Errorf("预期无效策略返回错误")
	}

	// 不存在的设备
	_, err = mgr.SetDevicePolicy("00:00:00:00:00:00", "TRUSTED")
	if err == nil {
		t.Errorf("预期不存在的 MAC 返回错误")
	}

	// 2. 测试一键全面气隙隔离
	updated, err = mgr.QuarantineDevice(testMAC, "发现未授权横向扫描")
	if err != nil {
		t.Fatalf("一键气隙隔离失败: %v", err)
	}
	for _, d := range updated.Devices {
		if d.MAC == testMAC {
			if d.Policy != "AIR_GAP_QUARANTINE" {
				t.Errorf("预期策略为 AIR_GAP_QUARANTINE，实际为: %s", d.Policy)
			}
		}
	}
	if len(updated.RecentAnomalies) == 0 {
		t.Errorf("预期生成隔离审计事件")
	}

	// 3. 测试解除隔离
	released, err := mgr.ReleaseDevice(testMAC)
	if err != nil {
		t.Fatalf("解除隔离失败: %v", err)
	}
	for _, d := range released.Devices {
		if d.MAC == testMAC {
			if d.Policy != "TRUSTED" {
				t.Errorf("解除后预期策略为 TRUSTED")
			}
		}
	}

	// 4. 测试深度巡检加固
	healed := mgr.HealAndScan()
	if healed.LateralIntrusionsSevered <= 0 {
		t.Errorf("预期拦截次数大于 0")
	}

	// 5. 测试全局开关
	off := mgr.ToggleEnabled(false)
	if off.Enabled {
		t.Errorf("预期零信任沙盒已关闭")
	}

	// 6. 验证重启持久化
	mgr2 := NewZeroTrustSandboxManager(tempDir)
	status2 := mgr2.GetStatus()
	if status2.Enabled != false {
		t.Errorf("重新加载后预期 Enabled=false，实际为: %v", status2.Enabled)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "zerotrust_sandbox.json")); os.IsNotExist(err) {
		t.Errorf("预期持久化 JSON 文件存在，实际缺失")
	}
}
