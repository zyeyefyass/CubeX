package netstack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMicroQoSManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "micro_qos_test_*")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mgr := NewMicroQoSManager(tempDir)
	status := mgr.GetStatus()

	if !status.Enabled {
		t.Errorf("预期初始为启用状态，实际为: %v", status.Enabled)
	}
	if !status.GamingPreemption {
		t.Errorf("预期初始电竞抢占通道已激活")
	}
	if len(status.Queues) != 3 {
		t.Errorf("预期 3 个分级队列，实际为: %d", len(status.Queues))
	}

	// 1. 测试设置缓冲冗余
	updated, err := mgr.SetBufferHeadroomPct(25)
	if err != nil {
		t.Fatalf("设置有效缓冲冗余失败: %v", err)
	}
	if updated.BufferHeadroomPct != 25 {
		t.Errorf("预期缓冲冗余 25%%，实际为: %d", updated.BufferHeadroomPct)
	}

	// 越界值测试
	_, err = mgr.SetBufferHeadroomPct(80)
	if err == nil {
		t.Errorf("预期超出 50%% 冗余抛出错误，实际未报错")
	}

	// 2. 测试电竞抢占开关
	updated = mgr.SetGamingPreemption(false)
	if updated.GamingPreemption {
		t.Errorf("预期电竞抢占已关闭，实际仍为开启")
	}

	// 3. 测试应急缓冲区清算
	flushed := mgr.TriggerEmergencyFlush()
	if flushed.AverageLatencyJitterMs > 2.0 {
		t.Errorf("预期清算后抖动低于 2.0ms，实际为: %f", flushed.AverageLatencyJitterMs)
	}

	// 4. 测试全局总控
	off := mgr.ToggleEnabled(false)
	if off.Enabled {
		t.Errorf("预期 QoS 调度已关闭")
	}

	// 5. 验证重启持久化
	mgr2 := NewMicroQoSManager(tempDir)
	status2 := mgr2.GetStatus()
	if status2.Enabled != false {
		t.Errorf("重启后预期 Enabled=false，实际为: %v", status2.Enabled)
	}
	if status2.BufferHeadroomPct != 25 {
		t.Errorf("重启后预期 BufferHeadroomPct=25，实际为: %d", status2.BufferHeadroomPct)
	}

	if _, err := os.Stat(filepath.Join(tempDir, "micro_qos.json")); os.IsNotExist(err) {
		t.Errorf("预期持久化 JSON 文件存在，实际缺失")
	}
}
