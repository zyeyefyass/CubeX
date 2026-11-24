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

func TestMonthlyLedgerMonthRollover(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cubex-rollover-test-*")
	if err != nil {
		t.Fatalf("create temp dir failed: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fakeTime := time.Date(2026, time.November, 24, 11, 0, 0, 0, time.Local)
	l := NewMonthlyLedger(tempDir)
	defer l.Close()

	// 注入可控时间函数
	l.timeNow = func() time.Time {
		return fakeTime
	}
	// 对齐初始状态
	l.monthMu.Lock()
	l.currMonth = fakeTime.Format("2006-01")
	l.monthMu.Unlock()

	// 记录一批流量
	conns := []ConnectionMeta{
		{
			ID:       "c1",
			Upload:   1000,
			Download: 2000,
			Chains:   []string{"HK-Node"},
		},
	}
	l.ProcessConnections(conns)

	s1 := l.GetMonthlySummary()
	if s1.Total != 3000 {
		t.Fatalf("expected 3000, got %d", s1.Total)
	}

	// 模拟自然时间跨月 (跨入每月 1 日 00:00:01，例如从 2026-11 跨入 2026-12)
	fakeTime = time.Date(2026, time.December, 1, 0, 0, 1, 0, time.Local)

	// 此时任何查询或连接处理，均会自动感知新月份并清零
	s2 := l.GetMonthlySummary()
	if s2.YearMonth != "2026-12" {
		t.Fatalf("expected year_month 2026-12, got %s", s2.YearMonth)
	}
	if s2.Uplink != 0 || s2.Downlink != 0 || s2.Total != 0 {
		t.Fatalf("expected 0 traffic after rollover, got %+v", s2)
	}

	// 检查旧月份 (2026-11) 的历史归档文件是否存在并准确
	historyFile := filepath.Join(tempDir, "data", "traffic_history_2026-11.json")
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		t.Fatalf("expected history archive to be created at %s", historyFile)
	}

	// 模拟再次跨年 (跨入 2027 年 1 月 1 日 00:00:01)
	fakeTime = time.Date(2027, time.January, 1, 0, 0, 1, 0, time.Local)
	s3 := l.GetMonthlySummary()
	if s3.YearMonth != "2027-01" {
		t.Fatalf("expected year_month 2027-01, got %s", s3.YearMonth)
	}
	if s3.Total != 0 {
		t.Fatalf("expected 0 traffic for new year, got %d", s3.Total)
	}
}

