package telemetry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlightRecorderRingBuffer(t *testing.T) {
	rec := NewFlightRecorder()

	// 写入超过容量（256）的条目，验证定长环形缓冲区的无溢出循环覆盖
	totalWrites := 300
	for i := 1; i <= totalWrites; i++ {
		rec.Record(CatEngineSwitch, "INFO", "switch test", int64(i))
	}

	snaps := rec.Snapshot(10)
	if len(snaps) != 10 {
		t.Fatalf("expected 10 items, got %d", len(snaps))
	}

	// 倒序检查：最新的应该在最前（seq 应该是 300）
	if snaps[0].Seq != uint64(totalWrites) {
		t.Errorf("expected latest seq %d, got %d", totalWrites, snaps[0].Seq)
	}
	if snaps[0].CostMs != int64(totalWrites) {
		t.Errorf("expected cost_ms %d, got %d", totalWrites, snaps[0].CostMs)
	}

	// 导出全量快照，最多应该只有 RingBufferCapacity (256)
	allSnaps := rec.Snapshot(0)
	if len(allSnaps) != RingBufferCapacity {
		t.Errorf("expected capacity %d, got %d", RingBufferCapacity, len(allSnaps))
	}
}

func TestSSEBroadcasterZeroClients(t *testing.T) {
	b := NewSSEBroadcaster()
	// 无客户端连接时应立即安全返回，0 序列化开销
	b.BroadcastPayload("test_event", map[string]string{"foo": "bar"})
}

func TestLKGConfigGuard(t *testing.T) {
	dir := t.TempDir()
	activePath := filepath.Join(dir, "config.active.json")
	lkgPath := filepath.Join(dir, "config.lkg.json")
	rec := NewFlightRecorder()

	guard := NewLKGConfigGuard(activePath, lkgPath, rec)

	goodData := []byte(`{"version":"good"}`)
	if err := guard.PromoteAfterStable(goodData); err != nil {
		t.Fatalf("promote failed: %v", err)
	}

	// 验证 LKG 文件落盘
	readLKG, err := os.ReadFile(lkgPath)
	if err != nil || string(readLKG) != string(goodData) {
		t.Fatalf("expected LKG data %s, got %s (err: %v)", string(goodData), string(readLKG), err)
	}

	// 模拟 active 文件损坏并触发回退
	badData := []byte(`{"corrupted"}`)
	_ = os.WriteFile(activePath, badData, 0644)

	restored, err := guard.FallbackToLKG()
	if err != nil {
		t.Fatalf("fallback failed: %v", err)
	}
	if string(restored) != string(goodData) {
		t.Errorf("expected restored data %s, got %s", string(goodData), string(restored))
	}

	// 检查黑匣子是否记录了 CRIT 事件
	snaps := rec.Snapshot(5)
	if len(snaps) < 2 {
		t.Fatalf("expected at least 2 flight events, got %d", len(snaps))
	}
	if snaps[0].Level != "CRIT" || snaps[0].Category != CatConfigLKG {
		t.Errorf("expected CRIT CONFIG_LKG event, got %+v", snaps[0])
	}
}
