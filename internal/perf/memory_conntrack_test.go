package perf

import (
	"path/filepath"
	"testing"
)

func TestMemoryConntrackManager(t *testing.T) {
	tmpDir := t.TempDir()
	persistPath := filepath.Join(tmpDir, "memory_conntrack_test.json")

	mgr := NewMemoryConntrackManager(persistPath)
	if !mgr.autoReclaimEnabled {
		t.Errorf("expected default autoReclaimEnabled=true")
	}

	// 1. 测试内存强制退还
	freed, currentHeap := mgr.ReclaimMemory()
	if freed <= 0 {
		t.Errorf("expected freed memory > 0, got %f", freed)
	}
	if currentHeap < 0 {
		t.Errorf("expected currentHeap >= 0, got %f", currentHeap)
	}

	// 2. 测试连接池热清洗
	flushed, err := mgr.FlushConntrack()
	if err != nil {
		t.Errorf("expected no error during conntrack flush, got %v", err)
	}
	if flushed < 0 {
		t.Errorf("expected flushed count >= 0, got %d", flushed)
	}

	// 3. 验证遥测状态
	status := mgr.GetStatus()
	if status.TotalReclaims != 1 {
		t.Errorf("expected TotalReclaims=1, got %d", status.TotalReclaims)
	}
	if status.TotalConntrackFlushed != uint64(flushed) {
		t.Errorf("expected TotalConntrackFlushed=%d, got %d", flushed, status.TotalConntrackFlushed)
	}

	// 4. 开关切换
	mgr.SetAutoReclaim(false)
	if mgr.GetStatus().AutoReclaimEnabled {
		t.Errorf("expected AutoReclaimEnabled=false")
	}
}
