package compiler

import (
	"testing"
)

func TestCompactDomainTrie(t *testing.T) {
	trie := NewCompactDomainTrie()

	// 1. Insert suffix rule
	if !trie.Insert("google.com", true, "Google") {
		t.Errorf("expected google.com suffix to insert successfully")
	}

	// 2. Child domain under existing suffix with same target should be pruned as redundant
	if trie.Insert("mail.google.com", false, "Google") {
		t.Errorf("mail.google.com should be detected as redundant under google.com")
	}
	if trie.Insert("www.google.com", true, "Google") {
		t.Errorf("www.google.com suffix should be detected as redundant under google.com")
	}

	// 3. Child domain under existing suffix with DIFFERENT target should NOT be pruned
	if !trie.Insert("special.google.com", false, "SpecialGroup") {
		t.Errorf("special.google.com with different target should be allowed")
	}

	// 4. Distinct domain should be allowed
	if !trie.Insert("baidu.com", true, "Direct") {
		t.Errorf("baidu.com should be allowed")
	}
}

func TestCompactCIDRTree(t *testing.T) {
	trie := NewCompactCIDRTree()

	// 1. Insert broad CIDR
	if !trie.InsertCIDR("10.0.0.0/8", "DIRECT") {
		t.Errorf("expected 10.0.0.0/8 to insert successfully")
	}

	// 2. Sub-CIDR with identical target should be pruned
	if trie.InsertCIDR("10.1.2.0/24", "DIRECT") {
		t.Errorf("10.1.2.0/24 should be pruned as redundant under 10.0.0.0/8")
	}
	if trie.InsertCIDR("10.100.0.0/16", "DIRECT") {
		t.Errorf("10.100.0.0/16 should be pruned as redundant under 10.0.0.0/8")
	}

	// 3. Sub-CIDR with DIFFERENT target should NOT be pruned
	if !trie.InsertCIDR("10.5.0.0/16", "PROXY") {
		t.Errorf("10.5.0.0/16 with different target PROXY should be accepted")
	}

	// 4. Distinct CIDR should be allowed
	if !trie.InsertCIDR("192.168.1.0/24", "DIRECT") {
		t.Errorf("192.168.1.0/24 should be accepted")
	}
}

func TestOptimizeMihomoRules(t *testing.T) {
	rawRules := []string{
		"DOMAIN-SUFFIX,google.com,Google",
		"DOMAIN,mail.google.com,Google", // redundant
		"DOMAIN,www.google.com,Google",  // redundant
		"DOMAIN,special.google.com,AI",  // different target, keep
		"IP-CIDR,10.0.0.0/8,DIRECT",
		"IP-CIDR,10.0.0.0/8,DIRECT",    // exact dup, redundant
		"IP-CIDR,10.1.0.0/16,DIRECT",   // sub-CIDR, redundant
		"IP-CIDR,192.168.0.0/16,DIRECT",
		"RULE-SET,cn_domain,DIRECT",
		"MATCH,DIRECT",
	}

	opt, stats := OptimizeMihomoRules(rawRules)
	if stats.PrunedCount < 4 {
		t.Errorf("expected at least 4 pruned rules, got %d", stats.PrunedCount)
	}
	if len(opt) >= len(rawRules) {
		t.Errorf("expected optimized count (%d) to be smaller than original (%d)", len(opt), len(rawRules))
	}
}

func BenchmarkCompactDomainTrie(b *testing.B) {
	trie := NewCompactDomainTrie()
	trie.Insert("google.com", true, "Google")
	trie.Insert("apple.com", true, "Apple")
	trie.Insert("microsoft.com", true, "Microsoft")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Insert("mail.google.com", false, "Google")
		trie.Insert("cloud.microsoft.com", false, "Microsoft")
		trie.Insert("store.apple.com", false, "Apple")
		trie.Insert("unique-domain.org", false, "Other")
	}
}
