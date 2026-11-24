package traffic

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMonthlyLedgerBasic(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex-traffic-test-*")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	l := NewMonthlyLedger(tempDir)
	defer l.Close()

	// 初始状态应为 0
	summary := l.GetMonthlySummary()
	if summary.Uplink != 0 || summary.Downlink != 0 || summary.Total != 0 {
		t.Fatalf("expected 0 initial traffic, got %+v", summary)
	}

	// 模拟一批连接数据：包含 DIRECT 连接与真实代理连接
	conns := []ConnectionMeta{
		{
			ID:       "conn-direct-1",
			Upload:   1000,
			Download: 5000,
			Chains:   []string{"DIRECT"},
		},
		{
			ID:       "conn-reject-1",
			Upload:   200,
			Download: 0,
			Chains:   []string{"REJECT"},
		},
		{
			ID:       "conn-proxy-1",
			Upload:   2048,
			Download: 8192,
			Chains:   []string{"🚀 节点选择", "HK-Premium-01"},
		},
	}

	// 第一次采样
	l.ProcessConnections(conns)

	summary = l.GetMonthlySummary()
	// 期望只计入 conn-proxy-1: Up=2048, Down=8192, Total=10240
	if summary.Uplink != 2048 || summary.Downlink != 8192 || summary.Total != 10240 {
		t.Fatalf("unexpected summary after first sample: %+v", summary)
	}

	// 第二次采样：conn-proxy-1 流量增长，新来一个 conn-proxy-2，conn-direct-1 也增长
	conns2 := []ConnectionMeta{
		{
			ID:       "conn-direct-1",
			Upload:   2000,
			Download: 10000,
			Chains:   []string{"DIRECT"},
		},
		{
			ID:       "conn-proxy-1",
			Upload:   3072, // 增加了 1024
			Download: 9216, // 增加了 1024
			Chains:   []string{"🚀 节点选择", "HK-Premium-01"},
		},
		{
			ID:       "conn-proxy-2",
			Upload:   512,
			Download: 1024,
			Chains:   []string{"US-Node-1"},
		},
	}

	l.ProcessConnections(conns2)

	summary = l.GetMonthlySummary()
	// conn-proxy-1 增量: Up 1024, Down 1024
	// conn-proxy-2 首次: Up 512, Down 1024
	// 总增量: Up 1536 (2048+1536=3584), Down 2048 (8192+2048=10240), Total=13824
	if summary.Uplink != 3584 || summary.Downlink != 10240 || summary.Total != 13824 {
		t.Fatalf("unexpected summary after second sample: %+v", summary)
	}

	// 测试落盘与重新加载
	if err := l.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	// 检查磁盘文件是否存在
	dataFile := filepath.Join(tempDir, "data", "traffic_monthly.json")
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		t.Fatalf("traffic_monthly.json does not exist")
	}

	// 创建新的 ledger 读取该目录
	l2 := NewMonthlyLedger(tempDir)
	defer l2.Close()

	summary2 := l2.GetMonthlySummary()
	if summary2.Uplink != 3584 || summary2.Downlink != 10240 || summary2.Total != 13824 {
		t.Fatalf("l2 failed to reload persisted traffic, got %+v", summary2)
	}
	if summary2.YearMonth != time.Now().Format("2006-01") {
		t.Fatalf("expected year_month %s, got %s", time.Now().Format("2006-01"), summary2.YearMonth)
	}
}
