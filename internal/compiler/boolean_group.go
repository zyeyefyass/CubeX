package compiler

import (
	"strings"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

// FilterPreviewResult provides immediate node match/exclude statistics for UI preview.
type FilterPreviewResult struct {
	Mode         string             `json:"mode"`          // "all", "or", "and"
	Keywords     []string           `json:"keywords"`      // List of search tokens
	MatchedNodes []string           `json:"matched_nodes"` // Names of matched nodes
	ExcludedNodes []string          `json:"excluded_nodes"`
	TotalCount   int                `json:"total_count"`
	MatchedCount int                `json:"matched_count"`
}

// FilterNodesByBooleanKeywords filters a proxy list based on ALL / OR / AND boolean keyword logic.
func FilterNodesByBooleanKeywords(nodes []schema.ProxyNode, mode string, keywords []string) []string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		mode = "all"
	}

	// Clean keywords
	cleanedKws := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		k := strings.ToLower(strings.TrimSpace(kw))
		if k != "" {
			cleanedKws = append(cleanedKws, k)
		}
	}

	if mode == "all" || len(cleanedKws) == 0 {
		res := make([]string, 0, len(nodes))
		for _, n := range nodes {
			name := strings.TrimSpace(n.Name)
			if name != "" && name != "DIRECT" && name != "REJECT" && name != "GLOBAL" {
				res = append(res, name)
			}
		}
		return res
	}

	matched := make([]string, 0, len(nodes))
	for _, n := range nodes {
		name := strings.TrimSpace(n.Name)
		if name == "" || name == "DIRECT" || name == "REJECT" || name == "GLOBAL" {
			continue
		}
		nameLower := strings.ToLower(name)

		if mode == "and" {
			// Must match all keywords
			allMatch := true
			for _, kw := range cleanedKws {
				if !strings.Contains(nameLower, kw) {
					allMatch = false
					break
				}
			}
			if allMatch {
				matched = append(matched, name)
			}
		} else {
			// Mode "or": match any keyword
			for _, kw := range cleanedKws {
				if strings.Contains(nameLower, kw) {
					matched = append(matched, name)
					break
				}
			}
		}
	}
	return matched
}

// PreviewBooleanFilter generates a complete diagnostic preview for group modal editing.
func PreviewBooleanFilter(nodes []schema.ProxyNode, mode string, keywords []string) FilterPreviewResult {
	matchedNames := FilterNodesByBooleanKeywords(nodes, mode, keywords)
	matchedMap := make(map[string]bool, len(matchedNames))
	for _, name := range matchedNames {
		matchedMap[name] = true
	}

	excluded := make([]string, 0, len(nodes))
	for _, n := range nodes {
		name := strings.TrimSpace(n.Name)
		if name == "" || name == "DIRECT" || name == "REJECT" || name == "GLOBAL" {
			continue
		}
		if !matchedMap[name] {
			excluded = append(excluded, name)
		}
	}

	return FilterPreviewResult{
		Mode:          mode,
		Keywords:      keywords,
		MatchedNodes:  matchedNames,
		ExcludedNodes: excluded,
		TotalCount:    len(matchedNames) + len(excluded),
		MatchedCount:  len(matchedNames),
	}
}
