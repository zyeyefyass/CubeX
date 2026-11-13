package compiler

import (
	"os"
	"testing"
)

func TestUserOverrideManager_NormalizationAndCompilation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cubex-override-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewUserOverrideManager(tmpDir)

	inputSpec := UserOverrideSpec{
		ForceDirectDomains: []string{
			"https://*.Bilibili.com/video",
			"HTTP://Sub.Aliyun.com:8080/api",
			"*.taobao.com",
			"+.jd.com",
		},
		ForceProxyDomains: []string{
			"https://github.com/torvalds",
			"*.openai.com",
		},
		ForceRejectDomains: []string{
			"ad.doubleclick.net",
			"http://telemetry.badtracker.org/ping",
		},
	}

	err = mgr.UpdateSpec(inputSpec)
	if err != nil {
		t.Fatalf("UpdateSpec failed: %v", err)
	}

	savedSpec := mgr.GetSpec()
	if len(savedSpec.ForceDirectDomains) != 4 {
		t.Errorf("expected 4 direct domains, got %d", len(savedSpec.ForceDirectDomains))
	}
	if savedSpec.ForceDirectDomains[0] != "bilibili.com" {
		t.Errorf("expected bilibili.com, got %s", savedSpec.ForceDirectDomains[0])
	}
	if savedSpec.ForceDirectDomains[1] != "sub.aliyun.com" {
		t.Errorf("expected sub.aliyun.com, got %s", savedSpec.ForceDirectDomains[1])
	}

	// 编译补丁验证
	patch := mgr.CompilePatch("Proxy")

	// 1. 验证 Fake-IP 免劫持白名单
	foundFakeIPBypass := false
	for _, f := range patch.ExtraFakeIPBypass {
		if f == "+.bilibili.com" {
			foundFakeIPBypass = true
			break
		}
	}
	if !foundFakeIPBypass {
		t.Errorf("expected +.bilibili.com in ExtraFakeIPBypass, got %v", patch.ExtraFakeIPBypass)
	}

	// 2. 验证 Mihomo 前置规则
	foundMihomoDirect := false
	foundMihomoProxy := false
	foundMihomoReject := false
	for _, r := range patch.MihomoPrependRules {
		if r == "DOMAIN-SUFFIX,bilibili.com,DIRECT" {
			foundMihomoDirect = true
		}
		if r == "DOMAIN-SUFFIX,github.com,Proxy" {
			foundMihomoProxy = true
		}
		if r == "DOMAIN-SUFFIX,ad.doubleclick.net,REJECT" {
			foundMihomoReject = true
		}
	}
	if !foundMihomoDirect || !foundMihomoProxy || !foundMihomoReject {
		t.Errorf("missing rules in MihomoPrependRules: direct=%v, proxy=%v, reject=%v",
			foundMihomoDirect, foundMihomoProxy, foundMihomoReject)
	}

	// 3. 验证持久化重新加载
	mgr2 := NewUserOverrideManager(tmpDir)
	loadedSpec := mgr2.GetSpec()
	if len(loadedSpec.ForceDirectDomains) != 4 || loadedSpec.ForceDirectDomains[0] != "bilibili.com" {
		t.Errorf("reloaded spec mismatch: %+v", loadedSpec)
	}
}
