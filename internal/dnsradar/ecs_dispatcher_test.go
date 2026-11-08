package dnsradar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestECSDispatcherManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ecs_dispatcher_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewECSDispatcherManager(tempDir)
	status := mgr.GetStatus()

	if !status.Enabled {
		t.Errorf("预期初始为开启状态，实际为: %v", status.Enabled)
	}
	if len(status.Profiles) < 3 {
		t.Errorf("预期至少包含 3 大运营商配置，实际为: %d", len(status.Profiles))
	}

	// 1. 测试切换模式
	updated, err := mgr.SetPreservationMode("strict_domestic_only")
	if err != nil {
		t.Fatalf("设置有效模式失败: %v", err)
	}
	if updated.CarrierPreservationMode != "strict_domestic_only" {
		t.Errorf("预期模式为 strict_domestic_only，实际为: %s", updated.CarrierPreservationMode)
	}

	// 无效模式
	_, err = mgr.SetPreservationMode("invalid_mode")
	if err == nil {
		t.Errorf("预期无效模式返回错误")
	}

	// 2. 测试更新运营商子网
	updated, err = mgr.UpdateCarrierPrefix("telecom_primary", "101.226.0.0/24")
	if err != nil {
		t.Fatalf("更新子网失败: %v", err)
	}
	var telecomFound bool
	for _, p := range updated.Profiles {
		if p.ID == "telecom_primary" {
			telecomFound = true
			if p.SubnetPrefix != "101.226.0.0/24" {
				t.Errorf("预期子网为 101.226.0.0/24，实际为: %s", p.SubnetPrefix)
			}
		}
	}
	if !telecomFound {
		t.Errorf("未找到 telecom_primary 配置")
	}

	// 非法 CIDR 格式
	_, err = mgr.UpdateCarrierPrefix("telecom_primary", "invalid_cidr")
	if err == nil {
		t.Errorf("预期非法 CIDR 格式抛出错误")
	}

	// 不存在的运营商
	_, err = mgr.UpdateCarrierPrefix("non_existent", "1.1.1.0/24")
	if err == nil {
		t.Errorf("预期不存在的配置抛出错误")
	}

	// 3. 测试探针优化
	optimized := mgr.OptimizeECSProfiles()
	if optimized.CarrierMatchingPct < 99.0 {
		t.Errorf("预期匹配率大于 99%%，实际为: %f", optimized.CarrierMatchingPct)
	}

	// 4. 测试开关总控
	off := mgr.ToggleEnabled(false)
	if off.Enabled {
		t.Errorf("预期已关闭 ECS 保真，实际仍为开启")
	}

	// 5. 验证持久化与重新加载
	mgr2 := NewECSDispatcherManager(tempDir)
	status2 := mgr2.GetStatus()
	if status2.Enabled != false {
		t.Errorf("重新加载后预期 Enabled=false，实际为: %v", status2.Enabled)
	}
	if status2.CarrierPreservationMode != "strict_domestic_only" {
		t.Errorf("重新加载后预期 strict_domestic_only，实际为: %s", status2.CarrierPreservationMode)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "ecs_dispatcher.json")); os.IsNotExist(err) {
		t.Errorf("预期持久化 JSON 文件存在，实际缺失")
	}
}
