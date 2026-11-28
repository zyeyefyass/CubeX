package config

import (
	"net/http"
	"testing"
)

func TestNormalizeBaseURL(t *testing.T) {
	cases := map[string]string{
		"":                        "",
		"  ":                      "",
		"127.0.0.1:9090":          "http://127.0.0.1:9090",
		"http://127.0.0.1:9090/":  "http://127.0.0.1:9090",
		"https://10.1.2.3:9443//": "https://10.1.2.3:9443",
	}
	for in, want := range cases {
		if got := NormalizeBaseURL(in); got != want {
			t.Errorf("NormalizeBaseURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveClashAPIBaseBuildsValidRequest(t *testing.T) {
	saved := GlobalRuntimeSpec.ClashAPIAddr
	defer func() { GlobalRuntimeSpec.ClashAPIAddr = saved }()

	GlobalRuntimeSpec.ClashAPIAddr = ""
	if got := ResolveClashAPIBase(0); got != "http://127.0.0.1:9090" {
		t.Fatalf("default base = %q", got)
	}
	if got := ResolveClashAPIBase(19090); got != "http://127.0.0.1:19090" {
		t.Fatalf("ir port base = %q", got)
	}

	GlobalRuntimeSpec.ClashAPIAddr = NormalizeBaseURL("127.0.0.1:29090")
	base := ResolveClashAPIBase(9090)
	if base != "http://127.0.0.1:29090" {
		t.Fatalf("env override base = %q", base)
	}
	// 回归防护：规范化后的地址必须能被 net/http 正常构造请求
	if _, err := http.NewRequest(http.MethodGet, base+"/connections", nil); err != nil {
		t.Fatalf("base url must be requestable: %v", err)
	}
}
