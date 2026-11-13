package upstream

import (
	"testing"
)

func TestVersionCompare(t *testing.T) {
	// 1. 常规同通道迭代：Test-v9.0 -> Test-v9.1 应为更新
	if !IsNewerVersion("Test-v9.0", "Test-v9.1") {
		t.Errorf("expected Test-v9.1 to be newer than Test-v9.0")
	}

	// 2. 同版本号转正：同为 9.1，正式版 v9.1.0 权重优先于测试版 Test-v9.1
	if !IsNewerVersion("Test-v9.1", "v9.1.0") {
		t.Errorf("expected v9.1.0 (formal) to be newer than Test-v9.1 (test)")
	}

	// 3. 跨代摘帽升级：Test-v9.1 -> v9.2.0 正式版应顺畅通过
	if !IsNewerVersion("Test-v9.1", "v9.2.0") {
		t.Errorf("expected v9.2.0 to be newer than Test-v9.1")
	}

	// 4. 旧版本检测：v9.0 不应新于 v9.1
	if IsNewerVersion("Test-v9.1", "Test-v9.0") {
		t.Errorf("expected Test-v9.0 NOT to be newer than Test-v9.1")
	}

	// 5. 相同版本检测
	if CompareVersions("Test-v9.1", "Test-v9.1") != 0 {
		t.Errorf("expected CompareVersions of identical tags to be 0")
	}
}
