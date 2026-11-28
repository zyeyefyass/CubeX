package perf

import (
	"bytes"
	"context"
	"runtime/debug"
	"testing"
	"time"
)

func TestBufferPool_GetPutAndCapGuard(t *testing.T) {
	buf := GetBuffer()
	if buf == nil {
		t.Fatal("expected non-nil buffer from pool")
	}

	buf.WriteString("hello cubex autopilot")
	PutBuffer(buf)

	// 测试超过 16KB 丢弃保护
	bigBuf := bytes.NewBuffer(make([]byte, 20*1024))
	PutBuffer(bigBuf) // 应直接丢弃，不报错
}

func TestMemoryGuard_ScavengeAndIdleCheck(t *testing.T) {
	isWebActive := false
	mg := NewMemoryGuard(24, func() bool {
		return isWebActive
	})

	mg.ScavengeNow()
	count, lastTime := mg.GetScavengeStats()
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
	if lastTime <= 0 {
		t.Fatal("expected positive last scavenge time")
	}

	// 快速巡检测试
	ctx, cancel := context.WithCancel(context.Background())
	go mg.StartIdleScavenger(ctx, 50*time.Millisecond)

	time.Sleep(120 * time.Millisecond)
	cancel()
}

func TestMemoryGuard_PreservesPreconfiguredLimitWhenZero(t *testing.T) {
	prev := debug.SetMemoryLimit(16 * 1024 * 1024)
	defer debug.SetMemoryLimit(prev)

	mg := NewMemoryGuard(0, nil)
	if got := mg.LimitMiB(); got != 16 {
		t.Fatalf("expected preserved LimitMiB=16, got %d", got)
	}
	if cur := debug.SetMemoryLimit(-1); cur != 16*1024*1024 {
		t.Fatalf("expected runtime memory limit unchanged at 16MiB, got %d", cur)
	}
}
