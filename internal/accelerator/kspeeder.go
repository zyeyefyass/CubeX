package accelerator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// KSpeeder provides Docker registry mirrors and GitHub CDN acceleration
// for high-speed downloads on RouterOS, NAS, and Linux environments.
type KSpeeder struct {
	rootDir    string
	configFile string
	mu         sync.RWMutex
	config     Config
	httpClient *http.Client
}

// Config stores KSpeeder runtime settings and mirror definitions.
type Config struct {
	Enabled               bool           `json:"enabled"`
	ActiveGitHubProxy     string         `json:"active_github_proxy"`
	ActiveDockerMirror    string         `json:"active_docker_mirror"`
	GitHubProxies         []ProxyNode    `json:"github_proxies"`
	DockerMirrors         []ProxyNode    `json:"docker_mirrors"`
	AutoSelectBest        bool           `json:"auto_select_best"`
	LastTestedAt          string         `json:"last_tested_at"`
}

// ProxyNode represents a single acceleration mirror or proxy endpoint.
type ProxyNode struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	LatencyMs int    `json:"latency_ms"`
	Status    string `json:"status"` // "ok", "slow", "error", "untested"
	IsCustom  bool   `json:"is_custom"`
}

// Default GitHub acceleration mirrors
var defaultGitHubProxies = []ProxyNode{
	{Name: "GHProxy 全球 CDN (推荐)", URL: "https://ghproxy.net/", Status: "untested"},
	{Name: "GHFast 高速镜像源", URL: "https://ghfast.top/", Status: "untested"},
	{Name: "Mirror GHProxy 高速节点", URL: "https://mirror.ghproxy.com/", Status: "untested"},
	{Name: "GHP.CI 开源镜像", URL: "https://ghp.ci/", Status: "untested"},
	{Name: "官方直连 (Direct)", URL: "", Status: "untested"},
}

// Default Docker Registry acceleration mirrors
var defaultDockerMirrors = []ProxyNode{
	{Name: "DaoCloud 镜像聚合源", URL: "https://docker.m.daocloud.io", Status: "untested"},
	{Name: "DockerProxy 高速代理", URL: "https://dockerproxy.net", Status: "untested"},
	{Name: "AtomHub 开放原子镜像", URL: "https://atomhub.openatom.cn", Status: "untested"},
	{Name: "1Panel 社区开源镜像", URL: "https://docker.1panel.live", Status: "untested"},
	{Name: "RatDev 公共镜像", URL: "https://hub.rat.dev", Status: "untested"},
}

// NewKSpeeder creates an acceleration engine instance.
func NewKSpeeder(rootDir string) *KSpeeder {
	ks := &KSpeeder{
		rootDir:    rootDir,
		configFile: filepath.Join(rootDir, "data", "kspeeder-config.json"),
		httpClient: &http.Client{Timeout: 4 * time.Second},
		config: Config{
			Enabled:            true,
			ActiveGitHubProxy:  "https://ghproxy.net/",
			ActiveDockerMirror: "https://docker.m.daocloud.io",
			GitHubProxies:      defaultGitHubProxies,
			DockerMirrors:      defaultDockerMirrors,
			AutoSelectBest:     true,
		},
	}
	ks.loadConfig()
	return ks
}

// GetStatus returns the current acceleration state.
func (ks *KSpeeder) GetStatus() Config {
	ks.mu.RLock()
	defer ks.mu.RUnlock()
	return ks.config
}

// UpdateConfig updates acceleration settings.
func (ks *KSpeeder) UpdateConfig(cfg Config) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.config.Enabled = cfg.Enabled
	if cfg.ActiveGitHubProxy != "" {
		ks.config.ActiveGitHubProxy = strings.TrimSpace(cfg.ActiveGitHubProxy)
	}
	if cfg.ActiveDockerMirror != "" {
		ks.config.ActiveDockerMirror = strings.TrimSpace(cfg.ActiveDockerMirror)
	}
	ks.config.AutoSelectBest = cfg.AutoSelectBest
	if len(cfg.GitHubProxies) > 0 {
		ks.config.GitHubProxies = cfg.GitHubProxies
	}
	if len(cfg.DockerMirrors) > 0 {
		ks.config.DockerMirrors = cfg.DockerMirrors
	}

	return ks.saveConfigLocked()
}

// OptimizeURL rewrites a raw GitHub URL using the active acceleration proxy.
func (ks *KSpeeder) OptimizeURL(rawURL string) string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	if !ks.config.Enabled || ks.config.ActiveGitHubProxy == "" {
		return rawURL
	}

	// Only accelerate GitHub URLs
	if !strings.Contains(rawURL, "github.com") &&
		!strings.Contains(rawURL, "raw.githubusercontent.com") &&
		!strings.Contains(rawURL, "github.io") {
		return rawURL
	}

	proxy := strings.TrimRight(ks.config.ActiveGitHubProxy, "/") + "/"
	cleanURL := strings.TrimSpace(rawURL)

	// If already prefixed, return as is
	if strings.HasPrefix(cleanURL, proxy) {
		return cleanURL
	}

	return proxy + cleanURL
}

// TestGitHubProxies tests latency for all GitHub proxy nodes concurrently.
func (ks *KSpeeder) TestGitHubProxies() []ProxyNode {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	var wg sync.WaitGroup
	var testMu sync.Mutex

	testResource := "https://raw.githubusercontent.com/MetaCubeX/meta-rules-dat/meta/version.txt"

	for i := range ks.config.GitHubProxies {
		node := &ks.config.GitHubProxies[i]
		wg.Add(1)
		go func(n *ProxyNode) {
			defer wg.Done()
			var targetURL string
			if n.URL == "" {
				targetURL = testResource
			} else {
				targetURL = strings.TrimRight(n.URL, "/") + "/" + testResource
			}

			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 3500*time.Millisecond)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodHead, targetURL, nil)
			if err != nil {
				testMu.Lock()
				n.LatencyMs = -1
				n.Status = "error"
				testMu.Unlock()
				return
			}
			req.Header.Set("User-Agent", "MiCube-KSpeeder/1.0")

			resp, err := ks.httpClient.Do(req)
			latency := int(time.Since(start).Milliseconds())

			testMu.Lock()
			defer testMu.Unlock()
			if err != nil || (resp != nil && resp.StatusCode >= 500) {
				n.LatencyMs = -1
				n.Status = "error"
			} else {
				if resp != nil {
					resp.Body.Close()
				}
				n.LatencyMs = latency
				if latency < 350 {
					n.Status = "ok"
				} else {
					n.Status = "slow"
				}
			}
		}(node)
	}

	wg.Wait()
	ks.config.LastTestedAt = time.Now().Format("2006-01-02 15:04:05")

	// If AutoSelectBest is enabled, pick the fastest available proxy
	if ks.config.AutoSelectBest {
		var bestURL string
		lowestLatency := 999999
		for _, n := range ks.config.GitHubProxies {
			if n.Status == "ok" && n.LatencyMs > 0 && n.LatencyMs < lowestLatency {
				lowestLatency = n.LatencyMs
				bestURL = n.URL
			}
		}
		if bestURL != "" {
			ks.config.ActiveGitHubProxy = bestURL
		}
	}

	_ = ks.saveConfigLocked()
	return ks.config.GitHubProxies
}

// TestDockerMirrors tests latency for all Docker registry mirrors concurrently.
func (ks *KSpeeder) TestDockerMirrors() []ProxyNode {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	var wg sync.WaitGroup
	var testMu sync.Mutex

	for i := range ks.config.DockerMirrors {
		node := &ks.config.DockerMirrors[i]
		wg.Add(1)
		go func(n *ProxyNode) {
			defer wg.Done()
			probeURL := strings.TrimRight(n.URL, "/") + "/v2/"

			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 3500*time.Millisecond)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
			if err != nil {
				testMu.Lock()
				n.LatencyMs = -1
				n.Status = "error"
				testMu.Unlock()
				return
			}
			req.Header.Set("User-Agent", "MiCube-KSpeeder/1.0")

			resp, err := ks.httpClient.Do(req)
			latency := int(time.Since(start).Milliseconds())

			testMu.Lock()
			defer testMu.Unlock()
			// /v2/ usually returns 200 or 401 Unauthorized for valid registry
			if err != nil || (resp != nil && resp.StatusCode >= 500) {
				n.LatencyMs = -1
				n.Status = "error"
			} else {
				if resp != nil {
					resp.Body.Close()
				}
				n.LatencyMs = latency
				if latency < 400 {
					n.Status = "ok"
				} else {
					n.Status = "slow"
				}
			}
		}(node)
	}

	wg.Wait()
	ks.config.LastTestedAt = time.Now().Format("2006-01-02 15:04:05")

	if ks.config.AutoSelectBest {
		var bestURL string
		lowestLatency := 999999
		for _, n := range ks.config.DockerMirrors {
			if n.Status == "ok" && n.LatencyMs > 0 && n.LatencyMs < lowestLatency {
				lowestLatency = n.LatencyMs
				bestURL = n.URL
			}
		}
		if bestURL != "" {
			ks.config.ActiveDockerMirror = bestURL
		}
	}

	_ = ks.saveConfigLocked()
	return ks.config.DockerMirrors
}

// GenerateROSRegistryScript generates copy-pasteable RouterOS command to set registry URL.
func (ks *KSpeeder) GenerateROSRegistryScript() string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	mirror := ks.config.ActiveDockerMirror
	if mirror == "" {
		mirror = "https://docker.m.daocloud.io"
	}

	return fmt.Sprintf(`# 🚀 MiCube KSpeeder RouterOS 容器镜像加速配置
# WinBox 点击 New Terminal 粘贴执行即可：
/container/config/set registry-url="%s"
/container/config/print
`, mirror)
}

// GenerateDockerDaemonJSON generates /etc/docker/daemon.json content.
func (ks *KSpeeder) GenerateDockerDaemonJSON() string {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	var mirrors []string
	if ks.config.ActiveDockerMirror != "" {
		mirrors = append(mirrors, ks.config.ActiveDockerMirror)
	}
	for _, m := range ks.config.DockerMirrors {
		if m.URL != "" && m.URL != ks.config.ActiveDockerMirror {
			mirrors = append(mirrors, m.URL)
		}
	}

	daemon := map[string]any{
		"registry-mirrors": mirrors,
	}
	data, _ := json.MarshalIndent(daemon, "", "  ")
	return string(data)
}

func (ks *KSpeeder) loadConfig() {
	data, err := os.ReadFile(ks.configFile)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &ks.config)
	if len(ks.config.GitHubProxies) == 0 {
		ks.config.GitHubProxies = defaultGitHubProxies
	}
	if len(ks.config.DockerMirrors) == 0 {
		ks.config.DockerMirrors = defaultDockerMirrors
	}
}

func (ks *KSpeeder) saveConfigLocked() error {
	_ = os.MkdirAll(filepath.Dir(ks.configFile), 0755)
	data, err := json.MarshalIndent(ks.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ks.configFile, data, 0644)
}
