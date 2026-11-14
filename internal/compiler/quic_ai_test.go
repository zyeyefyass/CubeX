package compiler

import (
	"strings"
	"testing"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

func TestInjectQUICDegradationRule(t *testing.T) {
	mihomoBase := []string{"MATCH,DIRECT"}
	sbBase := []map[string]any{{"action": "route", "outbound": "direct"}}

	// When disabled
	mOut, sbOut := InjectQUICDegradationRule(false, mihomoBase, sbBase)
	if len(mOut) != 1 || len(sbOut) != 1 {
		t.Fatalf("Expected unchanged rules when disabled")
	}

	// When enabled
	mOut, sbOut = InjectQUICDegradationRule(true, mihomoBase, sbBase)
	if len(mOut) != 2 || mOut[0] != "AND,((NETWORK,UDP),(DST-PORT,443)),REJECT" {
		t.Errorf("Unexpected Mihomo QUIC rule: %v", mOut)
	}
	if len(sbOut) != 2 || sbOut[0]["action"] != "reject" || sbOut[0]["method"] != "default" {
		t.Errorf("Unexpected sing-box QUIC rule: %v", sbOut)
	}
}

func TestSelectAICompliantNodes(t *testing.T) {
	input := []string{
		"HK-01-香港专线",
		"US-01-美国BGP",
		"JP-Tokyo-原生",
		"SG-Singapore-01",
		"MO-Macau-01",
		"CN-Bypass-直连",
		"UK-London-01",
	}

	compliant := SelectAICompliantNodes(input)
	for _, n := range compliant {
		if n == "HK-01-香港专线" || n == "MO-Macau-01" || n == "CN-Bypass-直连" {
			t.Fatalf("Restricted node found in AI pool: %s", n)
		}
	}
	if len(compliant) != 4 {
		t.Fatalf("Expected 4 compliant nodes (US, JP, SG, UK), got: %d (%v)", len(compliant), compliant)
	}
}

func TestUpsertAICompliantGroup(t *testing.T) {
	groups := []schema.StrategyGroup{
		{Name: "PROXY", Type: "select", Proxies: []string{"US-01", "HK-01"}},
	}
	nodes := []string{"US-01-美国", "JP-01-日本", "HK-01-香港"}

	res := UpsertAICompliantGroup(groups, nodes)
	if len(res) != 2 {
		t.Fatalf("Expected 2 groups, got: %d", len(res))
	}
	if res[0].Name != "AI 专线" {
		t.Errorf("Expected AI 专线 group at index 0, got: %s", res[0].Name)
	}
	if len(res[0].Proxies) != 2 {
		t.Errorf("Expected 2 compliant nodes in AI 专线, got: %v", res[0].Proxies)
	}
}

func TestAIRulesTargetAlignment(t *testing.T) {
	// Scenario 1: Fresh config without existing AI group -> dynamically creates "AI 专线" and targets it
	irFresh := schema.NewDefaultIR()
	irFresh.Groups = nil // clear predefined groups
	irFresh.Proxies = []schema.ProxyNode{
		{Name: "US-01-美国", Type: "ss", Server: "1.1.1.1", Port: 8388, Cipher: "aes-128-gcm", Password: "pwd"},
		{Name: "HK-01-香港", Type: "ss", Server: "2.2.2.2", Port: 8388, Cipher: "aes-128-gcm", Password: "pwd"},
	}

	mBytes, err := BuildMihomo(irFresh)
	if err != nil {
		t.Fatalf("BuildMihomo failed: %v", err)
	}
	mStr := string(mBytes)
	if !strings.Contains(mStr, "name: AI 专线") {
		t.Errorf("Mihomo expected group 'AI 专线' generated")
	}
	if !strings.Contains(mStr, "DOMAIN-SUFFIX,openai.com,AI 专线") {
		t.Errorf("Mihomo expected AI rule targeting 'AI 专线'")
	}

	sbBytes, err := BuildSingBox(irFresh)
	if err != nil {
		t.Fatalf("BuildSingBox failed: %v", err)
	}
	sbStr := string(sbBytes)
	if !strings.Contains(sbStr, `"tag": "AI 专线"`) {
		t.Errorf("sing-box expected group 'AI 专线' generated")
	}
	if !strings.Contains(sbStr, `"outbound": "AI 专线"`) {
		t.Errorf("sing-box expected route rule targeting 'AI 专线'")
	}

	// Scenario 2: Config with existing "AI" group -> updates it and targets "AI"
	irExisting := schema.NewDefaultIR()
	irExisting.Proxies = []schema.ProxyNode{
		{Name: "US-02-美国专线", Type: "ss", Server: "3.3.3.3", Port: 8388, Cipher: "aes-128-gcm", Password: "pwd"},
	}
	mBytes2, err := BuildMihomo(irExisting)
	if err != nil {
		t.Fatalf("BuildMihomo with existing AI failed: %v", err)
	}
	if !strings.Contains(string(mBytes2), "DOMAIN-SUFFIX,openai.com,AI") {
		t.Errorf("Mihomo expected AI rule targeting existing group 'AI'")
	}
}
