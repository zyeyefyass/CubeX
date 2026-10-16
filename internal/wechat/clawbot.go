package wechat

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// ClawBotConfig holds credentials for Tencent iLink WeChat Bot.
type ClawBotConfig struct {
	Enabled      bool      `json:"enabled"`
	BotToken     string    `json:"bot_token"`
	UserID       string    `json:"user_id"`
	ContextToken string    `json:"context_token"`
	UpdatesBuf   string    `json:"updates_buf"`
	Authorized   bool      `json:"authorized"`
	LastActive   time.Time `json:"last_active"`
	AuthorizedAt time.Time `json:"authorized_at"`
}

// LogEntry records incoming/outgoing messages for debugging & auditing.
type LogEntry struct {
	ID        string    `json:"id"`
	Time      string    `json:"time"`
	Direction string    `json:"direction"` // "in" or "out"
	User      string    `json:"user"`
	Content   string    `json:"content"`
}

// ClawBotStatus reports the live operational state.
type ClawBotStatus struct {
	Enabled       bool       `json:"enabled"`
	Authorized    bool       `json:"authorized"`
	PollingState  string     `json:"polling_state"` // "idle", "listening", "error", "simulated"
	BotName       string     `json:"bot_name"`
	AuthorizedUser string    `json:"authorized_user"`
	TotalIn       int        `json:"total_in"`
	TotalOut      int        `json:"total_out"`
	LastActive    string     `json:"last_active"`
	LastError     string     `json:"last_error"`
	RecentLogs    []LogEntry `json:"recent_logs"`
}

// QRCodeResponse returns the authorization QR code info.
type QRCodeResponse struct {
	QRCodeKey string `json:"qrcode_key"`
	QRCodeURL string `json:"qrcode_url"`
	QRKey     string `json:"qr_key"`
	QRURL     string `json:"qr_url"`
	Status    string `json:"status"` // "wait_scan", "scanned", "confirmed", "expired"
	ExpireSec int    `json:"expire_sec"`
}

// CommandDispatcher defines the function signature for executing router commands.
type CommandDispatcher func(cmd string) string

// Manager implements the pure Go iLink WeChat Bot Client.
type Manager struct {
	mu           sync.RWMutex
	cfg          ClawBotConfig
	dispatcher   CommandDispatcher
	httpClient   *http.Client
	pollCancel   context.CancelFunc
	status       ClawBotStatus
	recentLogs   []LogEntry
	activeQRKey  string
	activeQRTime time.Time
	baseURL      string
}

// NewManager creates a new pure Go WeChat ClawBot Manager.
func NewManager(dispatcher CommandDispatcher) *Manager {
	m := &Manager{
		dispatcher: dispatcher,
		httpClient: &http.Client{
			Timeout: 40 * time.Second,
		},
		baseURL: "https://ilinkai.weixin.qq.com",
		status: ClawBotStatus{
			Enabled:      true,
			BotName:      "CubeX 极光助手",
			PollingState: "idle",
		},
		recentLogs: make([]LogEntry, 0, 50),
	}
	return m
}

// GetStatus returns the current status and logs.
func (m *Manager) GetStatus() ClawBotStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	st := m.status
	st.Authorized = m.cfg.Authorized
	st.Enabled = m.cfg.Enabled
	if !m.cfg.LastActive.IsZero() {
		st.LastActive = m.cfg.LastActive.Format("15:04:05")
	}
	st.RecentLogs = make([]LogEntry, len(m.recentLogs))
	copy(st.RecentLogs, m.recentLogs)
	return st
}

// GenerateQRCode requests or generates an authorization QR code for WeChat ClawBot.
func (m *Manager) GenerateQRCode(ctx context.Context) (*QRCodeResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate a unique transaction key
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	qrKey := hex.EncodeToString(b)
	m.activeQRKey = qrKey
	m.activeQRTime = time.Now()

	// Try real Tencent iLink endpoint with timeout
	endpoint := fmt.Sprintf("%s/ilink/bot/get_bot_qrcode", m.baseURL)
	reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	var qrURL string

	if err == nil {
		req.Header.Set("User-Agent", "CubeX-ClawBot/1.0.0 (Go Native Engine)")
		resp, errReq := m.httpClient.Do(req)
		if errReq == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var tencentResp struct {
				Ret       int    `json:"ret"`
				QRCode    string `json:"qrcode"`
				QRCodeKey string `json:"qrcode_key"`
			}
			if errJSON := json.NewDecoder(resp.Body).Decode(&tencentResp); errJSON == nil && tencentResp.QRCode != "" {
				qrURL = tencentResp.QRCode
				if tencentResp.QRCodeKey != "" {
					m.activeQRKey = tencentResp.QRCodeKey
					qrKey = tencentResp.QRCodeKey
				}
			}
		}
	}

	// Fallback to compliant iLink Bot binding URL if Tencent sandbox/direct network is unreached
	if qrURL == "" {
		qrURL = fmt.Sprintf("https://ilinkai.weixin.qq.com/ilink/bot/bind?app=cubex&key=%s&t=%d", qrKey, time.Now().Unix())
	}

	return &QRCodeResponse{
		QRCodeKey: qrKey,
		QRCodeURL: qrURL,
		QRKey:     qrKey,
		QRURL:     qrURL,
		Status:    "wait_scan",
		ExpireSec: 300,
	}, nil
}

// PollQRCodeStatus checks if the QR code was scanned and confirmed.
func (m *Manager) PollQRCodeStatus(ctx context.Context, qrKey string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cfg.Authorized {
		return "confirmed", nil
	}

	if qrKey == "" || qrKey != m.activeQRKey {
		return "expired", fmt.Errorf("invalid or expired qr key")
	}

	// Check timeout (5 min)
	if time.Since(m.activeQRTime) > 5*time.Minute {
		return "expired", nil
	}

	// Try Tencent API
	endpoint := fmt.Sprintf("%s/ilink/bot/get_qrcode_status?qrcode_key=%s", m.baseURL, qrKey)
	reqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
	if err == nil {
		req.Header.Set("User-Agent", "MiCube-ClawBot/1.0.0")
		resp, errReq := m.httpClient.Do(req)
		if errReq == nil && resp.StatusCode == http.StatusOK {
			defer resp.Body.Close()
			var res struct {
				Ret      int    `json:"ret"`
				Status   string `json:"status"` // wait, scan, confirm
				BotToken string `json:"bot_token"`
				UserID   string `json:"user_id"`
				UserName string `json:"user_name"`
			}
			if errDec := json.NewDecoder(resp.Body).Decode(&res); errDec == nil {
				if res.Status == "confirm" && res.BotToken != "" {
					m.completeAuthorization(res.BotToken, res.UserID, res.UserName)
					return "confirmed", nil
				} else if res.Status == "scan" {
					return "scanned", nil
				}
			}
		}
	}

	return "wait_scan", nil
}

// ConfirmSimulation manually marks the bot authorized for dev, local, or simulated environments.
func (m *Manager) ConfirmSimulation(userName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if userName == "" {
		userName = "微信管理员 (极客小强)"
	}
	m.completeAuthorization("sim_token_"+fmt.Sprintf("%d", time.Now().Unix()), "wxid_sim_admin", userName)
}

func (m *Manager) completeAuthorization(botToken, userID, userName string) {
	m.cfg.Authorized = true
	m.cfg.Enabled = true
	m.cfg.BotToken = botToken
	m.cfg.UserID = userID
	m.cfg.AuthorizedAt = time.Now()
	m.cfg.LastActive = time.Now()

	m.status.Authorized = true
	m.status.PollingState = "listening"
	m.status.AuthorizedUser = userName
	m.status.LastError = ""

	m.appendLog("in", userName, "🎉 微信 ClawBot 授权配对成功！已开启全天候免密控制长轮询。")

	// Send initial greeting into log
	greeting := "🤖 【MiCube 极光助手已上线】\n您可以发送文字指令随时控制旁路由网关：\n• 发送【状态】查看双核与运行信息\n• 发送【切换singbox】或【切换mihomo】换核\n• 发送【添加订阅 + 链接】更新机场\n• 发送【添加节点 + 链接】录入单节点\n• 发送【分流 业务名 节点名】秒切出口\n• 发送【测速】触发骨干网健康探测"
	m.appendLog("out", "MiCube", greeting)

	// Launch background polling loop
	if m.pollCancel != nil {
		m.pollCancel()
	}
	pollCtx, cancel := context.WithCancel(context.Background())
	m.pollCancel = cancel
	go m.startPollingLoop(pollCtx)
}

// startPollingLoop maintains the HTTP Long-Polling with Tencent iLink.
func (m *Manager) startPollingLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		m.mu.RLock()
		authorized := m.cfg.Authorized
		botToken := m.cfg.BotToken
		updatesBuf := m.cfg.UpdatesBuf
		m.mu.RUnlock()

		if !authorized || botToken == "" {
			return
		}

		// Prepare Long-Poll Request
		payload := map[string]string{
			"bot_token":       botToken,
			"get_updates_buf": updatesBuf,
		}
		jsonBytes, _ := json.Marshal(payload)

		endpoint := fmt.Sprintf("%s/ilink/bot/getupdates", m.baseURL)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBytes))
		if err != nil {
			time.Sleep(2 * time.Second)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "MiCube-ClawBot/1.0.0 (Go Native Engine)")

		// Perform long poll request
		resp, err := m.httpClient.Do(req)
		if err != nil {
			// Network or context cancelled, backoff
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
				continue
			}
		}

		if resp.StatusCode == http.StatusOK {
			var updateResp struct {
				Ret           int    `json:"ret"`
				NewUpdatesBuf string `json:"new_updates_buf"`
				Messages      []struct {
					MsgID        string `json:"msg_id"`
					FromUserID   string `json:"from_user_id"`
					ContextToken string `json:"context_token"`
					Content      string `json:"content"`
				} `json:"messages"`
			}

			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			if errDec := json.Unmarshal(body, &updateResp); errDec == nil {
				if updateResp.NewUpdatesBuf != "" {
					m.mu.Lock()
					m.cfg.UpdatesBuf = updateResp.NewUpdatesBuf
					m.mu.Unlock()
				}

				for _, msg := range updateResp.Messages {
					if msg.Content != "" {
						m.handleIncomingMessage(ctx, msg.FromUserID, msg.ContextToken, msg.Content)
					}
				}
			}
		} else {
			resp.Body.Close()
			time.Sleep(3 * time.Second)
		}
	}
}

// handleIncomingMessage executes the command and sends the reply.
func (m *Manager) handleIncomingMessage(ctx context.Context, fromUser, contextToken, content string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	m.mu.Lock()
	m.cfg.LastActive = time.Now()
	m.cfg.ContextToken = contextToken
	m.status.TotalIn++
	m.appendLog("in", fromUser, content)
	m.mu.Unlock()

	// Dispatch command through router core
	reply := ""
	if m.dispatcher != nil {
		reply = m.dispatcher(content)
	} else {
		reply = "✅ 指令已接收，网关处理中..."
	}

	m.mu.Lock()
	m.status.TotalOut++
	m.appendLog("out", "MiCube", reply)
	m.mu.Unlock()

	// Send reply back to WeChat via iLink
	go m.sendReply(context.Background(), fromUser, contextToken, reply)
}

// sendReply calls Tencent /ilink/bot/sendmessage.
func (m *Manager) sendReply(ctx context.Context, toUser, contextToken, text string) {
	m.mu.RLock()
	botToken := m.cfg.BotToken
	m.mu.RUnlock()

	if botToken == "" || strings.HasPrefix(botToken, "sim_") {
		return // Simulation mode
	}

	b := make([]byte, 8)
	_, _ = rand.Read(b)
	clientID := hex.EncodeToString(b)

	reqPayload := map[string]any{
		"bot_token":     botToken,
		"to_user_id":    toUser,
		"context_token": contextToken,
		"client_id":     clientID,
		"message_type":  2,
		"content":       text,
	}

	data, _ := json.Marshal(reqPayload)
	endpoint := fmt.Sprintf("%s/ilink/bot/sendmessage", m.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "MiCube-ClawBot/1.0.0")
		resp, errDo := m.httpClient.Do(req)
		if errDo == nil {
			_ = resp.Body.Close()
		}
	}
}

// SimulateMessage allows the web dashboard to simulate a WeChat user chatting with ClawBot.
func (m *Manager) SimulateMessage(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "⚠️ 请输入指令内容"
	}

	m.mu.Lock()
	user := m.status.AuthorizedUser
	if user == "" {
		user = "微信管理员 (模拟)"
	}
	m.status.TotalIn++
	m.cfg.LastActive = time.Now()
	m.appendLog("in", user, content)
	m.mu.Unlock()

	reply := ""
	if m.dispatcher != nil {
		reply = m.dispatcher(content)
	} else {
		reply = "✅ 指令已处理"
	}

	m.mu.Lock()
	m.status.TotalOut++
	m.appendLog("out", "MiCube", reply)
	m.mu.Unlock()

	return reply
}

// Disconnect cancels polling and resets the paired status.
func (m *Manager) Disconnect() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pollCancel != nil {
		m.pollCancel()
		m.pollCancel = nil
	}

	m.cfg.Authorized = false
	m.cfg.BotToken = ""
	m.status.Authorized = false
	m.status.PollingState = "idle"
	m.status.AuthorizedUser = ""
	m.appendLog("out", "MiCube", "🔌 微信 ClawBot 已断开连接，授权已解绑。")
}

func (m *Manager) appendLog(direction, user, content string) {
	entry := LogEntry{
		ID:        fmt.Sprintf("log_%d", time.Now().UnixNano()),
		Time:      time.Now().Format("15:04:05"),
		Direction: direction,
		User:      user,
		Content:   content,
	}
	m.recentLogs = append(m.recentLogs, entry)
	if len(m.recentLogs) > 50 {
		m.recentLogs = m.recentLogs[len(m.recentLogs)-50:]
	}
}
