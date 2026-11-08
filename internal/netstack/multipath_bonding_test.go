package netstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMultiPathBondingManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "multipath_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewMultiPathBondingManager(tempDir)
	status := mgr.GetStatus()

	if !status.Enabled {
		t.Errorf("预期初始为开启状态，实际为: %v", status.Enabled)
	}
	if len(status.Channels) == 0 {
		t.Errorf("预期存在默认物理通道，实际数量为 0")
	}

	// 1. 测试切换模式
	updated, err := mgr.SetBondingMode("rtt_priority")
	if err != nil {
		t.Fatalf("设置有效模式失败: %v", err)
	}
	if updated.BondingMode != "rtt_priority" {
		t.Errorf("预期模式为 rtt_priority，实际为: %s", updated.BondingMode)
	}

	// 无效模式
	_, err = mgr.SetBondingMode("invalid_mode")
	if err == nil {
		t.Errorf("预期无效模式返回错误，但实际没有")
	}

	// 2. 测试启闭通道
	updated, err = mgr.ToggleChannel("wan3_5g_cpe", true)
	if err != nil {
		t.Fatalf("激活 5G CPE 失败: %v", err)
	}
	if updated.ActiveChannelsCount < 3 {
		t.Errorf("预期激活 3 条通道，实际为: %d", updated.ActiveChannelsCount)
	}

	// 不存在的通道
	_, err = mgr.ToggleChannel("non_existent", true)
	if err == nil {
		t.Errorf("预期无效通道返回错误")
	}

	// 3. 测试通道探针优化与智能再平衡
	optimized := mgr.OptimizeAndRebalance()
	if optimized.PeakSpeedupRatio < 1.0 {
		t.Errorf("预期加速比大于等于 1.0，实际为: %f", optimized.PeakSpeedupRatio)
	}

	// 4. 测试开关总控
	off := mgr.ToggleEnabled(false)
	if off.Enabled {
		t.Errorf("预期已关闭多路径聚合，实际为开启")
	}

	// 5. 验证持久化与重新加载
	mgr2 := NewMultiPathBondingManager(tempDir)
	status2 := mgr2.GetStatus()
	if status2.Enabled != false {
		t.Errorf("重新加载后预期 Enabled=false，实际为: %v", status2.Enabled)
	}
	if status2.BondingMode != "rtt_priority" {
		t.Errorf("重新加载后预期 BondingMode=rtt_priority，实际为: %s", status2.BondingMode)
	}

	// 检查落盘文件
	if _, err := os.Stat(filepath.Join(tempDir, "multipath_bonding.json")); os.IsNotExist(err) {
		t.Errorf("预期生成持久化 JSON 文件，但未找到")
	}
}
