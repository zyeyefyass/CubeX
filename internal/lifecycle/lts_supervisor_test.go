package lifecycle

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zyeyefyass/CubeX/internal/telemetry"
)

// 测试 A（ABA 代数防污染测试）：模拟第 0ms 加载配置 V1，第 10ms 立即加载坏配置 V2（触发回退），验证 V1 的定时器不会覆盖或污染 LKG 快照
func TestLTSDaemonSupervisor_ABA_AntiPollution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cubex_lts_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	activePath := filepath.Join(tmpDir, "active_config.json")
	lkgPath := filepath.Join(tmpDir, "lkg_snapshot.json")
	initialLKG := []byte(`{"version":"golden_v0"}`)
	if err := os.WriteFile(lkgPath, initialLKG, 0644); err != nil {
		t.Fatalf("failed to write initial lkg: %v", err)
	}

	recorder := telemetry.NewFlightRecorder()
	guard := telemetry.NewLKGConfigGuard(activePath, lkgPath, recorder)
	broadcaster := telemetry.NewSSEBroadcaster()

	// 晋升观察窗口设置为 50ms（快速单测）
	sup := NewLTSDaemonSupervisor(guard, recorder, broadcaster, 50*time.Millisecond)

	ctx := context.Background()
	var currentActiveConfig string

	applyMock := func(_ context.Context, raw []byte) error {
		str := string(raw)
		if str == `{"version":"bad_v2"}` {
			return errors.New("simulated bad config crash")
		}
		currentActiveConfig = str
		return nil
	}

	// 1. 0ms: 加载配置 V1 (正常)，启动 50ms 晋升定时器
	configV1 := []byte(`{"version":"good_v1"}`)
	if err := sup.BootOrReloadConfig(ctx, configV1, applyMock); err != nil {
		t.Fatalf("failed to boot config V1: %v", err)
	}
	if currentActiveConfig != string(configV1) {
		t.Fatalf("expected active config to be V1, got: %s", currentActiveConfig)
	}

	// 2. 15ms: 立即加载坏配置 V2，应当触发自动回退到初始 golden_v0，同时递增代数、取消 V1 晋升
	time.Sleep(15 * time.Millisecond)
	configV2 := []byte(`{"version":"bad_v2"}`)
	if err := sup.BootOrReloadConfig(ctx, configV2, applyMock); err != nil {
		t.Fatalf("failed during V2 reload with fallback: %v", err)
	}
	if currentActiveConfig != string(initialLKG) {
		t.Fatalf("expected active config to rollback to golden_v0, got: %s", currentActiveConfig)
	}

	// 3. 等待 80ms（超过 V1 的 50ms 晋升窗口）
	time.Sleep(80 * time.Millisecond)

	// 4. 验证 LKG 文件绝不能被 V1 覆盖，依然严格保持为 golden_v0
	persistedLKG, err := os.ReadFile(lkgPath)
	if err != nil {
		t.Fatalf("failed to read persisted lkg: %v", err)
	}
	if string(persistedLKG) != string(initialLKG) {
		t.Fatalf("ABA pollution detected! Expected LKG to remain golden_v0, but got: %s", string(persistedLKG))
	}
}

// 测试 B（SSE 长连接毫秒级断开测试）：启动一个挂载 SSEBroadcaster 的 httptest.Server 并建立活跃客户端流，调用 CloseAll()，验证客户端流在 20ms 内收到 EOF 正常退出
func TestSSEBroadcaster_MillisecondDisconnect(t *testing.T) {
	broadcaster := telemetry.NewSSEBroadcaster()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		broadcaster.ServeHTTP(w, r)
	}))
	defer server.Close()

	// 启动客户端建立 SSE 长连接
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("failed to connect to SSE stream: %v", err)
	}
	defer resp.Body.Close()

	// 确认第一帧已到达 (connected)
	buf := make([]byte, 256)
	n, err := resp.Body.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("failed to read first frame: %v", err)
	}
	if n == 0 {
		t.Fatalf("empty first frame")
	}

	// 调用 CloseAll() 并计时断开耗时
	disconnectDone := make(chan time.Duration, 1)
	go func() {
		drainBuf := make([]byte, 256)
		start := time.Now()
		for {
			_, readErr := resp.Body.Read(drainBuf)
			if readErr != nil {
				disconnectDone <- time.Since(start)
				return
			}
		}
	}()

	broadcaster.CloseAll()

	select {
	case elapsed := <-disconnectDone:
		t.Logf("SSE client disconnected in %v after CloseAll()", elapsed)
		if elapsed > 100*time.Millisecond {
			t.Fatalf("SSE disconnect took too long (%v), expected < 100ms", elapsed)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("SSE client failed to disconnect within 1s after CloseAll()!")
	}
}
