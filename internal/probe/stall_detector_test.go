package probe

import (
	"testing"
)

func TestPassiveStallDetector(t *testing.T) {
	d := NewPassiveStallDetector()

	// 正常流量：不触发
	for i := 0; i < 10; i++ {
		if d.ObserveSecond(5, 5000, 20000) {
			t.Fatalf("normal traffic should not trigger stall, tick %d", i)
		}
	}

	// 连续 5 秒假死：尚未达到阈值 6 秒，不应触发
	for i := 0; i < 5; i++ {
		if d.ObserveSecond(5, 2048, 0) {
			t.Fatalf("5 ticks should not trigger stall yet, tick %d", i)
		}
	}

	// 第 6 秒假死：必须触发！
	if !d.ObserveSecond(5, 2048, 0) {
		t.Fatalf("6th tick of stall MUST trigger failover!")
	}

	// 触发后立即进入 30s 冷却期，下一秒即使仍死链也不重复触发（防抖风暴）
	if d.ObserveSecond(5, 2048, 0) {
		t.Fatalf("subsequent tick in cooldown period should be suppressed")
	}

	// 恢复下行流量后自动清零计数
	d.ObserveSecond(5, 2048, 10240)
	d.ObserveSecond(5, 2048, 0) // 重新开始从 1 计数
}
