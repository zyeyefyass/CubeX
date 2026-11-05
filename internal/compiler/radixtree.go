package compiler

import (
	"net"
	"strings"
)

// CompactDomainNode represents a node in the reverse domain trie.
type CompactDomainNode struct {
	children map[string]*CompactDomainNode
	isSuffix bool
	target   string
}

// CompactDomainTrie optimizes domain and domain-suffix rules by pruning redundant subdomains.
type CompactDomainTrie struct {
	root *CompactDomainNode
}

// NewCompactDomainTrie creates a new domain trie.
func NewCompactDomainTrie() *CompactDomainTrie {
	return &CompactDomainTrie{
		root: &CompactDomainNode{
			children: make(map[string]*CompactDomainNode),
		},
	}
}

// Insert adds a domain rule. If a broader suffix already covers this domain with the same target,
// it returns false (redundant). Otherwise returns true.
func (t *CompactDomainTrie) Insert(domain string, isSuffix bool, target string) bool {
	domain = strings.TrimSpace(strings.ToLower(domain))
	domain = strings.TrimPrefix(domain, ".")
	if domain == "" {
		return false
	}

	parts := strings.Split(domain, ".")
	// Reverse parts for suffix matching (e.g. ["google", "com"] -> ["com", "google"])
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}

	curr := t.root
	for _, part := range parts {
		// If an existing ancestor node is already a suffix with the exact same target,
		// the new child is strictly redundant!
		if curr.isSuffix && curr.target == target {
			return false
		}

		child, exists := curr.children[part]
		if !exists {
			child = &CompactDomainNode{
				children: make(map[string]*CompactDomainNode),
			}
			curr.children[part] = child
		}
		curr = child
	}

	// If current node is already marked as a suffix with identical target, return false
	if curr.isSuffix && curr.target == target {
		return false
	}

	// If we are now inserting a SUFFIX, any existing children with the same target become redundant
	// and should be pruned to shrink memory.
	if isSuffix {
		curr.isSuffix = true
		curr.target = target
		// Prune redundant descendants with same target
		pruneDescendants(curr, target)
	} else if !curr.isSuffix {
		curr.target = target
	}

	return true
}

func pruneDescendants(node *CompactDomainNode, target string) {
	for k, child := range node.children {
		if child.target == target {
			delete(node.children, k)
		} else {
			pruneDescendants(child, target)
		}
	}
}

// CIDRNode represents a bitwise node in IPv4 prefix tree.
type CIDRNode struct {
	children [2]*CIDRNode
	isCover  bool
	target   string
}

// CompactCIDRTree performs bitwise radix tree pruning for IP CIDR rules.
type CompactCIDRTree struct {
	root *CIDRNode
}

// NewCompactCIDRTree creates a new bitwise CIDR tree.
func NewCompactCIDRTree() *CompactCIDRTree {
	return &CompactCIDRTree{
		root: &CIDRNode{},
	}
}

// InsertCIDR inserts an IPv4 CIDR. Returns false if already covered by an ancestor with same target.
func (t *CompactCIDRTree) InsertCIDR(cidrStr, target string) bool {
	cidrStr = strings.TrimSpace(cidrStr)
	if !strings.Contains(cidrStr, "/") {
		cidrStr += "/32"
	}
	_, ipnet, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return false
	}
	ip4 := ipnet.IP.To4()
	if ip4 == nil {
		return true // Skip IPv6 bitwise tree for now, keep as-is
	}

	ones, bits := ipnet.Mask.Size()
	if bits != 32 {
		return false
	}

	curr := t.root
	for i := 0; i < ones; i++ {
		// If ancestor already covers this range with the same target, this CIDR is redundant
		if curr.isCover && curr.target == target {
			return false
		}

		byteIdx := i / 8
		bitIdx := 7 - (i % 8)
		bit := (ip4[byteIdx] >> bitIdx) & 1

		if curr.children[bit] == nil {
			curr.children[bit] = &CIDRNode{}
		}
		curr = curr.children[bit]
	}

	if curr.isCover && curr.target == target {
		return false
	}

	curr.isCover = true
	curr.target = target
	// Prune any sub-CIDRs underneath with identical target
	if curr.children[0] != nil && curr.children[0].target == target {
		curr.children[0] = nil
	}
	if curr.children[1] != nil && curr.children[1].target == target {
		curr.children[1] = nil
	}

	return true
}

// RuleOptimizationStats tracks metrics of rule compression.
type RuleOptimizationStats struct {
	OriginalCount  int
	OptimizedCount int
	PrunedCount    int
	CompressionPct float64
}

// OptimizeMihomoRules performs compact radix trie optimization on a raw Mihomo rules list.
// It deduplicates and eliminates redundant subdomain and sub-CIDR rules while preserving strict evaluation order.
func OptimizeMihomoRules(rules []string) ([]string, RuleOptimizationStats) {
	domainTrie := NewCompactDomainTrie()
	cidrTrie := NewCompactCIDRTree()
	seenExact := make(map[string]bool)

	var optimized []string
	stats := RuleOptimizationStats{
		OriginalCount: len(rules),
	}

	for _, line := range rules {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Fast exact duplicate suppression
		if seenExact[trimmed] {
			stats.PrunedCount++
			continue
		}

		parts := strings.Split(trimmed, ",")
		if len(parts) < 2 {
			optimized = append(optimized, trimmed)
			seenExact[trimmed] = true
			continue
		}

		ruleType := strings.ToUpper(strings.TrimSpace(parts[0]))
		ruleVal := strings.TrimSpace(parts[1])
		target := ""
		if len(parts) >= 3 {
			target = strings.TrimSpace(parts[2])
		}

		switch ruleType {
		case "DOMAIN-SUFFIX":
			if !domainTrie.Insert(ruleVal, true, target) {
				stats.PrunedCount++
				continue
			}
		case "DOMAIN":
			if !domainTrie.Insert(ruleVal, false, target) {
				stats.PrunedCount++
				continue
			}
		case "IP-CIDR", "IP-CIDR6":
			if strings.Contains(ruleVal, ".") {
				if !cidrTrie.InsertCIDR(ruleVal, target) {
					stats.PrunedCount++
					continue
				}
			}
		}

		seenExact[trimmed] = true
		optimized = append(optimized, trimmed)
	}

	stats.OptimizedCount = len(optimized)
	if stats.OriginalCount > 0 {
		stats.CompressionPct = float64(stats.PrunedCount) / float64(stats.OriginalCount) * 100.0
	}

	return optimized, stats
}

// OptimizeSingboxRules performs compact trie pruning on sing-box route.rules definitions.
func OptimizeSingboxRules(rules []map[string]any) ([]map[string]any, RuleOptimizationStats) {
	domainTrie := NewCompactDomainTrie()
	cidrTrie := NewCompactCIDRTree()

	var optimized []map[string]any
	stats := RuleOptimizationStats{
		OriginalCount: len(rules),
	}

	for _, rule := range rules {
		outbound, _ := rule["outbound"].(string)

		// 1. Optimize domain and domain_suffix lists in rule
		if domains, ok := rule["domain"].([]string); ok && len(domains) > 0 {
			var prunedDomains []string
			for _, d := range domains {
				if domainTrie.Insert(d, false, outbound) {
					prunedDomains = append(prunedDomains, d)
				} else {
					stats.PrunedCount++
				}
			}
			if len(prunedDomains) == 0 {
				delete(rule, "domain")
			} else {
				rule["domain"] = prunedDomains
			}
		}

		if suffixes, ok := rule["domain_suffix"].([]string); ok && len(suffixes) > 0 {
			var prunedSuffixes []string
			for _, s := range suffixes {
				if domainTrie.Insert(s, true, outbound) {
					prunedSuffixes = append(prunedSuffixes, s)
				} else {
					stats.PrunedCount++
				}
			}
			if len(prunedSuffixes) == 0 {
				delete(rule, "domain_suffix")
			} else {
				rule["domain_suffix"] = prunedSuffixes
			}
		}

		// 2. Optimize ip_cidr lists in rule
		if cidrs, ok := rule["ip_cidr"].([]string); ok && len(cidrs) > 0 {
			var prunedCIDRs []string
			for _, c := range cidrs {
				if strings.Contains(c, ".") {
					if cidrTrie.InsertCIDR(c, outbound) {
						prunedCIDRs = append(prunedCIDRs, c)
					} else {
						stats.PrunedCount++
					}
				} else {
					prunedCIDRs = append(prunedCIDRs, c)
				}
			}
			if len(prunedCIDRs) == 0 {
				delete(rule, "ip_cidr")
			} else {
				rule["ip_cidr"] = prunedCIDRs
			}
		}

		// If a rule had criteria but now has nothing left after pruning, skip it
		hasCriteria := false
		for k := range rule {
			if k != "outbound" && k != "invert" {
				hasCriteria = true
				break
			}
		}

		if hasCriteria {
			optimized = append(optimized, rule)
		} else if outbound != "" {
			// Rule with no criteria left is equivalent to a match-all rule; only retain if intentional
			optimized = append(optimized, rule)
		}
	}

	stats.OptimizedCount = len(optimized)
	if stats.OriginalCount > 0 {
		stats.CompressionPct = float64(stats.PrunedCount) / float64(stats.OriginalCount) * 100.0
	}

	return optimized, stats
}
