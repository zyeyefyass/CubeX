package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/zyeyefyass/CubeX/internal/schema"
)

const (
	consoleAuthCookieName = "cubex_session"
	sessionTTL            = 7 * 24 * time.Hour
	maxActiveSessions     = 64
)

type consoleAuthConfig struct {
	Enabled      bool   `json:"enabled"`
	Salt         string `json:"salt"`
	PasswordHash string `json:"password_hash"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

// ConsoleAuthManager manages optional Web console access password and session tokens.
type ConsoleAuthManager struct {
	mu       sync.RWMutex
	filePath string
	cfg      consoleAuthConfig
	sessions map[string]time.Time
}

// NewConsoleAuthManager initializes or loads persisted console lock state.
func NewConsoleAuthManager(workDir string) *ConsoleAuthManager {
	mgr := &ConsoleAuthManager{
		filePath: filepath.Join(workDir, "data", "console-auth.json"),
		sessions: make(map[string]time.Time),
	}
	mgr.load()
	return mgr
}

func (m *ConsoleAuthManager) load() {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return
	}
	var cfg consoleAuthConfig
	if json.Unmarshal(data, &cfg) == nil {
		m.cfg = cfg
	}
}

func (m *ConsoleAuthManager) saveLocked() error {
	data, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return err
	}
	return schema.WriteFileAtomicSecure(m.filePath, data)
}

func hashConsolePassword(salt, password string) string {
	sum := sha256.Sum256([]byte(salt + ":" + password))
	return hex.EncodeToString(sum[:])
}

func randomHex(nBytes int) (string, error) {
	b := make([]byte, nBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// IsEnabled returns whether Web console password protection is currently active.
func (m *ConsoleAuthManager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg.Enabled && m.cfg.PasswordHash != ""
}

// VerifyPassword checks password in constant time.
func (m *ConsoleAuthManager) VerifyPassword(password string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.cfg.Enabled || m.cfg.PasswordHash == "" {
		return true
	}
	candidate := hashConsolePassword(m.cfg.Salt, password)
	return subtle.ConstantTimeCompare([]byte(candidate), []byte(m.cfg.PasswordHash)) == 1
}

// SetPassword enables, updates, or disables console lock.
// If currently enabled, currentPassword must match before modifying or disabling.
func (m *ConsoleAuthManager) SetPassword(currentPassword, newPassword string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cfg.Enabled && m.cfg.PasswordHash != "" {
		candidate := hashConsolePassword(m.cfg.Salt, currentPassword)
		if subtle.ConstantTimeCompare([]byte(candidate), []byte(m.cfg.PasswordHash)) != 1 {
			return fmt.Errorf("当前控制台密码不正确")
		}
	}

	newPassword = strings.TrimSpace(newPassword)
	if newPassword == "" {
		m.cfg = consoleAuthConfig{
			Enabled:   false,
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
		m.sessions = make(map[string]time.Time)
		return m.saveLocked()
	}

	if len(newPassword) < 4 {
		return fmt.Errorf("控制台访问密码长度至少为 4 位")
	}

	salt, err := randomHex(16)
	if err != nil {
		return err
	}
	m.cfg = consoleAuthConfig{
		Enabled:      true,
		Salt:         salt,
		PasswordHash: hashConsolePassword(salt, newPassword),
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}
	m.sessions = make(map[string]time.Time)
	return m.saveLocked()
}

// CreateSession issues a new cryptographic session token.
func (m *ConsoleAuthManager) CreateSession() (string, error) {
	token, err := randomHex(32)
	if err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for k, exp := range m.sessions {
		if now.After(exp) {
			delete(m.sessions, k)
		}
	}
	if len(m.sessions) >= maxActiveSessions {
		for k := range m.sessions {
			delete(m.sessions, k)
			if len(m.sessions) < maxActiveSessions {
				break
			}
		}
	}
	m.sessions[token] = now.Add(sessionTTL)
	return token, nil
}

func extractSessionToken(r *http.Request) string {
	if c, err := r.Cookie(consoleAuthCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		return strings.TrimSpace(authz[7:])
	}
	return ""
}

// ValidateRequest checks if the HTTP request is authorized when lock is enabled.
func (m *ConsoleAuthManager) ValidateRequest(r *http.Request) bool {
	if !m.IsEnabled() {
		return true
	}
	token := extractSessionToken(r)
	if token == "" {
		return false
	}
	m.mu.RLock()
	exp, ok := m.sessions[token]
	m.mu.RUnlock()
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		m.mu.Lock()
		delete(m.sessions, token)
		m.mu.Unlock()
		return false
	}
	return true
}

// RevokeRequest invalidates the session token carried on the request.
func (m *ConsoleAuthManager) RevokeRequest(r *http.Request) {
	token := extractSessionToken(r)
	if token == "" {
		return
	}
	m.mu.Lock()
	delete(m.sessions, token)
	m.mu.Unlock()
}

// isTrustedOrigin checks whether an Origin header belongs to localhost, a private RFC1918 LAN IP, or matches Host.
func isTrustedOrigin(origin, reqHost string) bool {
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	if reqHost != "" {
		rh := reqHost
		if h, _, splitErr := net.SplitHostPort(reqHost); splitErr == nil {
			rh = h
		}
		if strings.EqualFold(host, rh) {
			return true
		}
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast()
	}
	return strings.HasSuffix(host, ".lan") || strings.HasSuffix(host, ".local")
}

func authMiddleware(authMgr *ConsoleAuthManager, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if authMgr == nil || !authMgr.IsEnabled() {
			next.ServeHTTP(w, r)
			return
		}
		path := r.URL.Path
		if !strings.HasPrefix(path, "/api/") ||
			path == "/api/health" ||
			path == "/api/v1/auth/status" ||
			path == "/api/v1/auth/login" {
			next.ServeHTTP(w, r)
			return
		}
		if !authMgr.ValidateRequest(r) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error":         "控制台访问已锁定，请先登录验证",
				"auth_required": true,
			})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleAuthStatus(w http.ResponseWriter, r *http.Request) {
	if s.authMgr == nil {
		sendJSON(w, map[string]any{"enabled": false, "authenticated": true})
		return
	}
	enabled := s.authMgr.IsEnabled()
	authenticated := !enabled || s.authMgr.ValidateRequest(r)
	sendJSON(w, map[string]any{
		"enabled":       enabled,
		"authenticated": authenticated,
	})
}

func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.authMgr == nil || !s.authMgr.IsEnabled() {
		sendJSON(w, map[string]any{"ok": true, "enabled": false})
		return
	}
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !s.authMgr.VerifyPassword(req.Password) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": "控制台访问密码错误"})
		return
	}
	token, err := s.authMgr.CreateSession()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     consoleAuthCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	sendJSON(w, map[string]any{"ok": true, "token": token})
}

func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.authMgr != nil {
		s.authMgr.RevokeRequest(r)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     consoleAuthCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	sendJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleAuthPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.authMgr == nil {
		http.Error(w, "auth manager unavailable", http.StatusInternalServerError)
		return
	}
	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.authMgr.SetPassword(req.CurrentPassword, req.NewPassword); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	if s.authMgr.IsEnabled() {
		if token, err := s.authMgr.CreateSession(); err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:     consoleAuthCookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   int(sessionTTL.Seconds()),
			})
		}
	}
	sendJSON(w, map[string]any{
		"ok":      true,
		"enabled": s.authMgr.IsEnabled(),
	})
}
