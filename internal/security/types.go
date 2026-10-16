package security

import "time"

// RiskLevel indicates severity of a security exposure.
type RiskLevel string

const (
	RiskCritical RiskLevel = "CRITICAL"
	RiskHigh     RiskLevel = "HIGH"
	RiskMedium   RiskLevel = "MEDIUM"
	RiskLow      RiskLevel = "LOW"
)

// AuditStatus represents the pass/fail state of an audit item.
type AuditStatus string

const (
	StatusPass AuditStatus = "PASS"
	StatusWarn AuditStatus = "WARN"
	StatusFail AuditStatus = "FAIL"
)

// SecurityTestItem represents an individual hardcore security audit probe.
type SecurityTestItem struct {
	ID             string      `json:"id"`              // ipv6_leak, apple_type65, webrtc_stun, ecs_privacy, dns_fallback, quic_degrade
	Name           string      `json:"name"`            // 中文测试名称
	Category       string      `json:"category"`        // 分类: DNS安全, 协议栈防御, 隐私溯源
	RiskLevel      RiskLevel   `json:"risk_level"`      // CRITICAL / HIGH / MEDIUM
	Status         AuditStatus `json:"status"`          // PASS / WARN / FAIL
	Score          int         `json:"score"`           // 单项得分 (0~100)
	Details        string      `json:"details"`         // 详细检测结果说明
	TechnicalSpecs string      `json:"technical_specs"` // 涉及的技术规范 (RFC 9460, RFC 7871, RFC 8489 等)
	Recommendation string      `json:"recommendation"`  // 加固建议
	Remediated     bool        `json:"remediated"`      // 是否已执行自愈加固
}

// SecurityAuditReport aggregates the entire hardcore security radar report.
type SecurityAuditReport struct {
	TotalScore       int                `json:"total_score"`        // 综合安全分 (0~100)
	SecurityRating   string             `json:"security_rating"`    // 评级名称 (如: 极光盾级 100% 零泄露)
	ActiveEngine     string             `json:"active_engine"`      // 当前活跃内核 (sing-box / mihomo)
	TestedAt         time.Time          `json:"tested_at"`          // 检测时间
	Summary          string             `json:"summary"`            // 总体诊断结论
	Items            []SecurityTestItem `json:"items"`              // 6 大核心安全检测项
	AuditLogs        []string           `json:"audit_logs"`         // 渗透测试日志流
	CanAutoRemediate bool               `json:"can_auto_remediate"` // 是否支持一键自愈加固
}

// RemediateResult represents the response of an auto-remediation action.
type RemediateResult struct {
	Success         bool      `json:"success"`
	RemediatedItems []string  `json:"remediated_items"`
	NewTotalScore   int       `json:"new_total_score"`
	RestartedEngine string    `json:"restarted_engine"`
	Message         string    `json:"message"`
	RemediatedAt    time.Time `json:"remediated_at"`
}
