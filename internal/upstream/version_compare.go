package upstream

import (
	"regexp"
	"strconv"
	"strings"
)

var tagRegex = regexp.MustCompile(`(?i)^(?:test[-_.\s]?)?v?(\d+)\.(\d+)(?:\.(\d+))?(?:[-_.](.+))?$`)

// CubeXVersion 表示可按语义权重严格比对的 CubeX 版本结构
type CubeXVersion struct {
	Major         int
	Minor         int
	Patch         int
	ChannelWeight int // 2 为正式版，1 为测试版 (Test/RC/Beta/Alpha)
	Raw           string
}

// ParseCubeXVersion 将 'Test-v9.1', 'Test-v9.0-LTS', 'v9.1.0', 'v9.2.0' 统一解析为语义版本
func ParseCubeXVersion(raw string) CubeXVersion {
	s := strings.TrimSpace(raw)
	m := tagRegex.FindStringSubmatch(s)
	if len(m) == 0 {
		return CubeXVersion{Raw: s}
	}

	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch := 0
	if len(m) > 3 && m[3] != "" {
		patch, _ = strconv.Atoi(m[3])
	}

	suffix := ""
	if len(m) > 4 {
		suffix = strings.ToLower(m[4])
	}

	lower := strings.ToLower(s)
	isTest := strings.HasPrefix(lower, "test") ||
		strings.Contains(suffix, "test") ||
		strings.Contains(suffix, "rc") ||
		strings.Contains(suffix, "beta") ||
		strings.Contains(suffix, "alpha")

	channelWeight := 2
	if isTest {
		channelWeight = 1
	}

	return CubeXVersion{
		Major:         major,
		Minor:         minor,
		Patch:         patch,
		ChannelWeight: channelWeight,
		Raw:           s,
	}
}

// CompareVersions 比较两个版本字符串大小
// 若 v1 > v2 返回 1; 若 v1 < v2 返回 -1; 若相等返回 0
func CompareVersions(v1, v2 string) int {
	p1 := ParseCubeXVersion(v1)
	p2 := ParseCubeXVersion(v2)

	if p1.Major != p2.Major {
		if p1.Major > p2.Major {
			return 1
		}
		return -1
	}
	if p1.Minor != p2.Minor {
		if p1.Minor > p2.Minor {
			return 1
		}
		return -1
	}
	if p1.Patch != p2.Patch {
		if p1.Patch > p2.Patch {
			return 1
		}
		return -1
	}
	if p1.ChannelWeight != p2.ChannelWeight {
		if p1.ChannelWeight > p2.ChannelWeight {
			return 1
		}
		return -1
	}
	return 0
}

// IsNewerVersion 判断 targetVersion 是否严格新于 currentVersion
func IsNewerVersion(currentVersion, targetVersion string) bool {
	return CompareVersions(targetVersion, currentVersion) > 0
}
