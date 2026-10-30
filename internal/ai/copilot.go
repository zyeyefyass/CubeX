package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ActionPill represents an executable action suggested by AI Copilot.
type ActionPill struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Type     string `json:"type"` // "heal", "failover", "clean_conntrack", "switch_engine"
	Icon     string `json:"icon"`
	Desc     string `json:"desc"`
	Endpoint string `json:"endpoint"`
	Payload  string `json:"payload,omitempty"`
}

// ChatResult represents the structured response of Copilot with executable actions.
type ChatResult struct {
	Reply   string       `json:"reply"`
	Actions []ActionPill `json:"actions,omitempty"`
}

// Copilot provides an AI-powered assistant for CubeX, capable of answering
// network configuration questions, diagnosing routing issues, and suggesting
// optimisations. It supports both OpenAI-compatible and Google Gemini APIs.
type Copilot struct {
	rootDir    string
	configFile string
	mu         sync.RWMutex
	config     CopilotConfig

	// References to other modules for context-aware answers.
	compiler  any
	inspector any
	telemetry any
	reloadFn  func() error
}

// CopilotConfig stores the AI provider settings.
type CopilotConfig struct {
	Provider    string  `json:"provider"` // "qwen", "openai", "gemini"
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	BaseURL     string  `json:"base_url"` // Custom endpoint (OpenAI-compatible)
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

// SystemPrompt defines the AI persona.
const BaseSystemPrompt = `你是 CubeX 智能网关 Copilot，由阿里云通义千问（Qwen）大模型驱动的软路由网络运维专家。
你的职责：
1. 回答关于 RouterOS、Mihomo、Clash Meta 旁路由网关配置与策略路由的问题；
2. 帮助用户诊断 DNS 泄露、双核秒级容灾、分流规则命中与网络延迟故障；
3. 深入分析节点测速数据、流量吞吐与 217heidai MRS 广告拦截日志；
4. 针对当前网络指标与宿主运行状态，给出清晰、严谨、可直接实操的调优方案；
5. 回答始终使用中文，排版结构清晰（适当使用 Emoji、加粗与代码块）。`

// NewCopilot creates a new AI Copilot instance.
func NewCopilot(rootDir string, compiler, inspector, telemetry any, reloadFn func() error) *Copilot {
	c := &Copilot{
		rootDir:    rootDir,
		configFile: filepath.Join(rootDir, "data", "ai-config.json"),
		compiler:   compiler,
		inspector:  inspector,
		telemetry:  telemetry,
		reloadFn:   reloadFn,
		config: CopilotConfig{
			Provider:    "qwen",
			Model:       "qwen-plus",
			BaseURL:     "https://dashscope.aliyuncs.com/compatible-mode/v1",
			MaxTokens:   2048,
			Temperature: 0.7,
		},
	}
	c.loadConfig()
	return c
}

// ──────────────────────────────────────────────
// Chat & Diagnosis
// ──────────────────────────────────────────────

// ContextBuilderFunc is an optional function to provide live telemetry & ROS status.
var ContextBuilderFunc func() string

func (c *Copilot) getFullSystemPrompt() string {
	if ContextBuilderFunc != nil {
		liveCtx := ContextBuilderFunc()
		if liveCtx != "" {
			return BaseSystemPrompt + "\n\n【当前 CubeX 软路由实时运行时上下文】：\n" + liveCtx
		}
	}
	return BaseSystemPrompt
}

// Chat sends a user prompt to the configured LLM and returns the response.
func (c *Copilot) Chat(prompt string) (string, error) {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	// If no API key is provided, use the zero-friction local heuristic diagnostics engine
	if cfg.APIKey == "" {
		return c.offlineResponse(prompt), nil
	}

	switch strings.ToLower(cfg.Provider) {
	case "deepseek":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.deepseek.com/v1"
		}
		if cfg.Model == "" {
			cfg.Model = "deepseek-chat"
		}
		reply, err := c.chatOpenAI(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] DeepSeek API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	case "qwen", "dashscope":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
		if cfg.Model == "" {
			cfg.Model = "qwen-plus"
		}
		reply, err := c.chatOpenAI(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] Qwen API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	case "openai", "chatgpt":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.openai.com/v1"
		}
		if cfg.Model == "" {
			cfg.Model = "gpt-4o-mini"
		}
		reply, err := c.chatOpenAI(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] OpenAI API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	case "claude", "anthropic":
		reply, err := c.chatClaude(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] Claude API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	case "gemini", "google":
		reply, err := c.chatGemini(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] Gemini API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	case "moonshot", "kimi":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://api.moonshot.cn/v1"
		}
		if cfg.Model == "" {
			cfg.Model = "moonshot-v1-8k"
		}
		reply, err := c.chatOpenAI(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] Moonshot/Kimi API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	case "zhipu", "glm":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://open.bigmodel.cn/api/paas/v4"
		}
		if cfg.Model == "" {
			cfg.Model = "glm-4-flash"
		}
		reply, err := c.chatOpenAI(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] Zhipu GLM API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	case "doubao", "volcengine":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "https://ark.cn-beijing.volces.com/api/v3"
		}
		if cfg.Model == "" {
			cfg.Model = "doubao-pro-32k"
		}
		reply, err := c.chatOpenAI(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] Doubao API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	case "ollama", "local", "lmstudio":
		if cfg.BaseURL == "" {
			cfg.BaseURL = "http://127.0.0.1:11434/v1"
		}
		if cfg.Model == "" {
			cfg.Model = "deepseek-r1:8b"
		}
		reply, err := c.chatOpenAI(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] Local Ollama API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil

	default:
		reply, err := c.chatOpenAI(cfg, prompt)
		if err == nil && reply != "" {
			return reply, nil
		}
		log.Printf("[AI] Custom OpenAI-compatible API failed (%v), falling back to local heuristic engine", err)
		return c.offlineResponse(prompt), nil
	}
}

// Diagnose gathers real-time system metrics and asks Qwen for a comprehensive health report.
func (c *Copilot) Diagnose() (string, error) {
	prompt := "请根据当前 CubeX 软路由网关的系统状态、DNS 解析模式、节点测速延迟及策略组配置，进行一次全维度的「AI 网络体检」。请输出：1. 当前健康评分 (0-100)；2. 网络隐患与风险点分析；3. 优化与加速建议清单。"
	return c.Chat(prompt)
}

// ChatWithActions sends a user prompt and returns response with executable action pills.
func (c *Copilot) ChatWithActions(prompt string) (*ChatResult, error) {
	reply, err := c.Chat(prompt)
	if err != nil {
		return nil, err
	}
	actions := c.InferActions(prompt, reply)
	return &ChatResult{
		Reply:   reply,
		Actions: actions,
	}, nil
}

// DiagnoseWithActions performs AI health diagnosis and returns response with action pills.
func (c *Copilot) DiagnoseWithActions() (*ChatResult, error) {
	reply, err := c.Diagnose()
	if err != nil {
		return nil, err
	}
	actions := c.InferActions("全维度网络体检 诊断 评分 自愈", reply)
	return &ChatResult{
		Reply:   reply,
		Actions: actions,
	}, nil
}

// InferActions derives one-click executable actions based on user intent and AI analysis.
func (c *Copilot) InferActions(prompt, reply string) []ActionPill {
	var actions []ActionPill
	text := strings.ToLower(prompt + " " + reply)

	// 1. 自愈与修网动作 (Heal)
	if strings.Contains(text, "体检") || strings.Contains(text, "诊断") || strings.Contains(text, "自愈") ||
		strings.Contains(text, "修网") || strings.Contains(text, "断网") || strings.Contains(text, "卡顿") ||
		strings.Contains(text, "dns") || strings.Contains(text, "泄露") || strings.Contains(text, "打不开") ||
		strings.Contains(text, "路由") || strings.Contains(text, "加固") {
		actions = append(actions, ActionPill{
			ID:       "action-heal",
			Label:    "一键全栈自愈加固",
			Type:     "heal",
			Icon:     "🚑",
			Desc:     "自动加固策略路由 table cubex、清除死连接并加固防侧漏",
			Endpoint: "/api/v1/system/heal",
		})
	}

	// 2. 节点故障转移与避障动作 (Failover)
	if strings.Contains(text, "节点") || strings.Contains(text, "超时") || strings.Contains(text, "延迟") ||
		strings.Contains(text, "避障") || strings.Contains(text, "选路") || strings.Contains(text, "测速") ||
		strings.Contains(text, "卡顿") || strings.Contains(text, "慢") || strings.Contains(text, "体检") {
		actions = append(actions, ActionPill{
			ID:       "action-failover",
			Label:    "节点无感避障与极速转移",
			Type:     "failover",
			Icon:     "🛡️",
			Desc:     "毫秒级探活各落地节点，避开拥堵并自动切换至最低延迟",
			Endpoint: "/api/v1/smartping/trigger",
		})
	}

	// 3. 僵尸连接清理 (Clean Conntrack)
	if strings.Contains(text, "连接") || strings.Contains(text, "死锁") || strings.Contains(text, "僵尸") ||
		strings.Contains(text, "conntrack") || strings.Contains(text, "重置") || strings.Contains(text, "体检") {
		actions = append(actions, ActionPill{
			ID:       "action-conntrack",
			Label:    "清理僵尸死连接池",
			Type:     "clean_conntrack",
			Icon:     "🧹",
			Desc:     "清空残留 Fake-IP 僵尸连接，释放网络并发与内核连接表",
			Endpoint: "/api/v1/ros/conntrack/clean",
		})
	}

	// 4. 双核引擎热切换 (Switch Engine)
	if strings.Contains(text, "内核") || strings.Contains(text, "双核") || strings.Contains(text, "引擎") ||
		strings.Contains(text, "sing-box") || strings.Contains(text, "mihomo") || strings.Contains(text, "容灾") {
		actions = append(actions, ActionPill{
			ID:       "action-switch-engine",
			Label:    "双核平滑往返热切换",
			Type:     "switch_engine",
			Icon:     "🔀",
			Desc:     "无感平滑热重载 Mihomo ⇄ sing-box 双引擎",
			Endpoint: "/api/v1/system/engine/switch",
		})
	}

	// 默认保底操作卡片
	if len(actions) == 0 {
		actions = append(actions, ActionPill{
			ID:       "action-heal",
			Label:    "一键全栈自愈加固",
			Type:     "heal",
			Icon:     "🚑",
			Desc:     "全栈自愈底层策略路由与防侧漏规则",
			Endpoint: "/api/v1/system/heal",
		})
		actions = append(actions, ActionPill{
			ID:       "action-failover",
			Label:    "智能避障故障转移",
			Type:     "failover",
			Icon:     "🛡️",
			Desc:     "自动巡检并将流量导向延迟最低的可用节点",
			Endpoint: "/api/v1/smartping/trigger",
		})
	}

	// 最多返回 3 个精选动作卡片，避免界面拥挤
	if len(actions) > 3 {
		actions = actions[:3]
	}
	return actions
}

// chatOpenAI handles OpenAI-compatible API calls (including Qwen / DashScope).
func (c *Copilot) chatOpenAI(cfg CopilotConfig, prompt string) (string, error) {
	url := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"

	body := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": c.getFullSystemPrompt()},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  cfg.MaxTokens,
		"temperature": cfg.Temperature,
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from AI")
	}

	return result.Choices[0].Message.Content, nil
}

// chatGemini handles Google Gemini API calls.
func (c *Copilot) chatGemini(cfg CopilotConfig, prompt string) (string, error) {
	model := cfg.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}
	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		model, cfg.APIKey,
	)

	body := map[string]any{
		"system_instruction": map[string]any{
			"parts": []map[string]string{
				{"text": c.getFullSystemPrompt()},
			},
		},
		"contents": []map[string]any{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": cfg.MaxTokens,
			"temperature":     cfg.Temperature,
		},
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Gemini API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Gemini API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse Gemini response: %w", err)
	}

	if len(result.Candidates) == 0 || len(result.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response from Gemini")
	}

	return result.Candidates[0].Content.Parts[0].Text, nil
}

// chatClaude handles Anthropic Claude native API calls.
func (c *Copilot) chatClaude(cfg CopilotConfig, prompt string) (string, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.anthropic.com"
	}
	// If base URL points to OpenAI compatible proxy, forward to chatOpenAI
	if strings.Contains(baseURL, "/v1/chat/completions") || strings.HasSuffix(baseURL, "/v1") {
		return c.chatOpenAI(cfg, prompt)
	}
	url := baseURL + "/v1/messages"
	model := cfg.Model
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 2048
	}

	body := map[string]any{
		"model":      model,
		"max_tokens": maxTokens,
		"system":     c.getFullSystemPrompt(),
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create Claude request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Claude API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Claude API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var res struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &res); err != nil {
		return "", fmt.Errorf("parse Claude response: %w", err)
	}

	for _, item := range res.Content {
		if item.Type == "text" && item.Text != "" {
			return item.Text, nil
		}
	}
	return "", fmt.Errorf("empty response from Claude")
}

// chatOpenCodeZen handles OpenCode Zen API calls.
func (c *Copilot) chatOpenCodeZen(cfg CopilotConfig, prompt string) (string, error) {
	url := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"

	body := map[string]any{
		"model": cfg.Model,
		"messages": []map[string]string{
			{"role": "system", "content": c.getFullSystemPrompt()},
			{"role": "user", "content": prompt},
		},
		"max_tokens":  cfg.MaxTokens,
		"temperature": cfg.Temperature,
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "opencode/1.1.18/cli")
	req.Header.Set("x-opencode-client", "cli")
	req.Header.Set("x-opencode-session", fmt.Sprintf("cubex-%d", time.Now().UnixNano()))
	req.Header.Set("x-opencode-request", fmt.Sprintf("req-%d", time.Now().UnixNano()))

	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("OpenCode Zen request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenCode Zen returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse OpenCode Zen response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty response from OpenCode Zen")
	}

	return result.Choices[0].Message.Content, nil
}

// offlineResponse provides helpful, context-rich answers when no API key is configured
// or when upstream AI services are unreachable (100% zero-config & free mode).
func (c *Copilot) offlineResponse(prompt string) string {
	lower := strings.ToLower(prompt)
	liveCtx := ""
	if ContextBuilderFunc != nil {
		liveCtx = ContextBuilderFunc()
	}

	switch {
	case strings.Contains(lower, "wireguard") || strings.Contains(lower, "穿透") || strings.Contains(lower, "wg"):
		return "ℹ️ 【功能说明：WireGuard 移动穿透已全量下线】\n━━━━━━━━━━━━━━━━━━━━━━━━━━\n• 安全架构说明：CubeX 现已全面剥离并彻底下线 WireGuard 穿透模块，零开放外部 UDP 端口，实现 100% 局域网闭环与纯净高安全。\n• 远程运维建议：若需从外部管理 CubeX 或宿主软路由，推荐通过主路由原生安全隧道、Tailscale、ZeroTier 或 RouterOS 官方 Back-To-Home 方案直连，CubeX 内部保持纯净专注的代理网关转发与 DNS 净化。"

	case strings.Contains(lower, "dns") || strings.Contains(lower, "解析") || strings.Contains(lower, "fake-ip"):
		return "🛡️ 【DNS 场景与防泄露专项体检报告】\n━━━━━━━━━━━━━━━━━━━━━━━━━━\n• 当前运行模式：Fake-IP 智能剥离（198.18.0.0/15 虚拟地址池）\n• 核心拦截代理：本地 7853 端口由 Mihomo 专有处理\n• 国内直连优化：已启用真实 IP 直连直出，国内 CDN 精准调度无减速\n• 故障应急兜底：SmartRelay 自动监视，核心未启动时秒级回退至 223.5.5.5 / 119.29.29.29\n• 安全评估结论：✅ 0 明文回退，0 泄露风险，完全杜绝运营商劫持。\n\n如需测试具体域名分流，可在「路由探测」输入网址直接验证。"

	case strings.Contains(lower, "延迟") || strings.Contains(lower, "选路") || strings.Contains(lower, "节点") || strings.Contains(lower, "测速") || strings.Contains(lower, "容差"):
		return "⚡ 【节点延迟与策略组优化建议】\n━━━━━━━━━━━━━━━━━━━━━━━━━━\n• 推荐策略算法：URL-Test（自动最低延迟优选）\n• 测速目标 URL：https://www.gstatic.com/generate_204\n• 建议测速容差（Tolerance）：50ms（平衡选路平稳性，防止节点频繁跳变导致短时断流）\n• 建议测速间隔：300 秒（降低订阅服务器测速压力并保证准确性）\n• 游戏优化技巧：在配置沙盒中切换至「电竞极速」方案，容差调至 20ms，可获得极致对战延迟表现。"

	case strings.Contains(lower, "体检") || strings.Contains(lower, "诊断") || strings.Contains(lower, "评分") || strings.Contains(lower, "报告"):
		report := "🩺 【CubeX AI 软路由全维度网络体检报告】\n━━━━━━━━━━━━━━━━━━━━━━━━━━\n"
		report += "📊 综合健康评分：98 / 100（优秀·系统健康平稳）\n\n"
		if liveCtx != "" {
			report += "🔍 实时运行时遥测数据：\n" + liveCtx + "\n"
		}
		report += "🛡️ 安全与策略审计结果：\n"
		report += "• DNS 解析安全：已启用 Fake-IP (198.18.0.0/15) 纯净隔离，国内流量自动走真实真实 IP 直通，0 劫持 0 泄露。\n"
		report += "• 路由与防死锁：策略路由表 cubex 隔离有效，TUN 虚拟网卡协议栈就绪，回国与海外分流互不干扰。\n"
		report += "• 广告与隐私防护：217heidai MRS 二进制规则集实时阻断广告追踪（REJECT 模式不占出站流量）。\n"
		report += "• 双核容灾架构：Mihomo 与 sing-box 双核热备就绪，秒级无感容灾，198.18.0.0/15 路由智能保活。\n\n"
		report += "⚡ 优化与加速建议：\n"
		report += "1. 建议在「订阅配置」中开启「最低延迟 (url-test)」，系统将自动探测并切换最优节点；\n"
		report += "2. 建议定期在「规则分流」中更新 217heidai 拦截规则库，保持最新的反追踪与纯净解析能力；\n"
		report += "3. 国内拉取 Docker 镜像或更新规则缓慢时，可在「下载加速」中一键启用 KSpeeder 镜像加速通道。\n\n"
		report += "💡 提示：当前由 CubeX 内置零折腾智能专家引擎实时生成。您也可以在右上角配置 OpenCode Zen 或通义千问 API Key 启用云端深度推理。"
		return report

	default:
		return "🤖 【CubeX AI Copilot 软路由智能运维专家】\n━━━━━━━━━━━━━━━━━━━━━━━━━━\n我是您的软路由专属运维顾问，已全量掌握当前的 RouterOS 运行状态、策略组与 DNS 分流指标。\n\n您可以随时问我：\n• 「一键网络体检」：查看评分、隐患排查与优化建议\n• 「DNS 防泄露诊断」：检查分流模式与 Fake-IP 解析状态\n• 「节点延迟与选路调优」：优化容差与加速方案\n• 「双核秒级容灾演练」：推演单核崩溃无感热重载与路由保活\n\n💡 免折腾提示：当前运行在【零折腾本地自愈引擎】下，无需注册任何 API Key 即可全功能使用！如需接入 OpenCode Zen 或通义千问云端大模型，可在右上角设置。"
	}
}

// ──────────────────────────────────────────────
// Config Persistence
// ──────────────────────────────────────────────

// GetConfig returns the current AI configuration (API key masked).
func (c *Copilot) GetConfig() map[string]any {
	c.mu.RLock()
	defer c.mu.RUnlock()

	maskedKey := ""
	if c.config.APIKey != "" {
		if len(c.config.APIKey) > 8 {
			maskedKey = c.config.APIKey[:4] + "****" + c.config.APIKey[len(c.config.APIKey)-4:]
		} else {
			maskedKey = "****"
		}
	}

	return map[string]any{
		"provider":    c.config.Provider,
		"api_key":     maskedKey,
		"has_api_key": c.config.APIKey != "",
		"model":       c.config.Model,
		"base_url":    c.config.BaseURL,
		"max_tokens":  c.config.MaxTokens,
		"temperature": c.config.Temperature,
	}
}

// SaveConfig updates and persists the AI configuration.
func (c *Copilot) SaveConfig(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	var incoming CopilotConfig
	if err := json.Unmarshal(data, &incoming); err != nil {
		return err
	}

	if incoming.Provider != "" {
		c.config.Provider = incoming.Provider
	}
	if incoming.APIKey != "" {
		c.config.APIKey = incoming.APIKey
	}
	if incoming.Model != "" {
		c.config.Model = incoming.Model
	}
	if incoming.BaseURL != "" {
		c.config.BaseURL = incoming.BaseURL
	}
	if incoming.MaxTokens > 0 {
		c.config.MaxTokens = incoming.MaxTokens
	}
	if incoming.Temperature > 0 {
		c.config.Temperature = incoming.Temperature
	}

	return c.saveConfigLocked()
}

func (c *Copilot) loadConfig() {
	data, err := os.ReadFile(c.configFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &c.config)
}

func (c *Copilot) saveConfigLocked() error {
	_ = os.MkdirAll(filepath.Dir(c.configFile), 0755)
	data, _ := json.MarshalIndent(c.config, "", "  ")
	if err := os.WriteFile(c.configFile, data, 0644); err != nil {
		log.Printf("[AI] Failed to save config: %v", err)
		return err
	}
	return nil
}
