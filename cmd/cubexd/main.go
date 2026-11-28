package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zyeyefyass/CubeX/internal/config"
	"github.com/zyeyefyass/CubeX/internal/onboarding"
	"github.com/zyeyefyass/CubeX/internal/perf"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

var (
	version = "v9.991"

	startTime = time.Now()
)

func configureAdaptiveMemoryLimit() {
	// Base fallback: 24MiB default ceiling for cubexd control plane
	limit := int64(24 * 1024 * 1024)
	gcPercent := 40
	totalMb := perf.DetectTotalRAMMb()

	if totalMb <= 256 {
		limit = 12 * 1024 * 1024 // 256M container: 12MiB
		gcPercent = 20
	} else if totalMb <= 512 {
		limit = 16 * 1024 * 1024 // 512M CHR: 16MiB
		gcPercent = 25
	} else if totalMb <= 1024 {
		limit = 20 * 1024 * 1024 // 1GB router: 20MiB
		gcPercent = 25
	} else {
		limit = 24 * 1024 * 1024 // 2GB+ host: 24MiB keeps control plane minimal
		gcPercent = 30
	}

	debug.SetMemoryLimit(limit)
	debug.SetGCPercent(gcPercent)
	log.Printf("🛡️ [Supervisor] Adaptive GOMEMLIMIT configured: %d MiB (GCPercent=%d, DetectedRAM=%dMB)\n", limit/(1024*1024), gcPercent, totalMb)
}

// safeRecover provides panic interception and stack tracing for all asynchronous goroutines.


func bindWithFallback(initialAddr string) (net.Listener, string, error) {
	ln, err := net.Listen("tcp", initialAddr)
	if err == nil {
		return ln, initialAddr, nil
	}

	errStr := strings.ToLower(err.Error())
	if !strings.Contains(errStr, "address already in use") && !strings.Contains(errStr, "only one usage of each socket address") && !strings.Contains(errStr, "bind") {
		return nil, initialAddr, err
	}

	// 重启瞬态退避窗口：等待旧进程释放套接字（最长 2s），避免 supervise-daemon / systemd 快速拉起时误判占用并漂移端口
	for attempt := 0; attempt < 8; attempt++ {
		time.Sleep(250 * time.Millisecond)
		if retryLn, retryErr := net.Listen("tcp", initialAddr); retryErr == nil {
			return retryLn, initialAddr, nil
		} else {
			err = retryErr
		}
	}

	host, portStr, splitErr := net.SplitHostPort(initialAddr)
	if splitErr != nil {
		if strings.HasPrefix(initialAddr, ":") {
			portStr = strings.TrimPrefix(initialAddr, ":")
		} else {
			return nil, initialAddr, err
		}
	}
	basePort, parseErr := strconv.Atoi(portStr)
	if parseErr != nil {
		return nil, initialAddr, err
	}

	log.Printf("⚠️ [端口占用自愈] 默认服务端口 %s 已被宿主其它应用占用，启动智能空闲端口探查...\n", initialAddr)
	for nextPort := basePort + 1; nextPort <= basePort+10; nextPort++ {
		candidateAddr := fmt.Sprintf(":%d", nextPort)
		if host != "" {
			candidateAddr = net.JoinHostPort(host, strconv.Itoa(nextPort))
		}
		candidateLn, candidateErr := net.Listen("tcp", candidateAddr)
		if candidateErr == nil {
			log.Printf("✅ [端口占用自愈] 成功绑定备用空闲端口 %s！Web驾驶舱访问入口: http://<宿主IP>:%d\n", candidateAddr, nextPort)
			return candidateLn, candidateAddr, nil
		}
	}

	return nil, initialAddr, fmt.Errorf("端口 %s 及其后 10 个端口均已被占用: %w", initialAddr, err)
}



func main() {
	configureAdaptiveMemoryLimit()

	listenAddr := flag.String("listen", ":2026", "Listen address (e.g. :2026 or 0.0.0.0:2026)")
	workDir := flag.String("dir", "", "Working directory (defaults to current dir, /app, or /opt/cubex)")
	runDoctor := flag.Bool("doctor", false, "Run onboarding doctor environment preflight and print diagnostic report")
	flag.Parse()

	if *runDoctor {
		rep := onboarding.RunOnboardingDoctor(context.Background(), detectOutboundIP(), 2026, "198.18.0.0/15")
		b, _ := json.MarshalIndent(rep, "", "  ")
		fmt.Println(string(b))
		os.Exit(0)
	}

	if envPort := os.Getenv("CUBEX_PORT"); envPort != "" && *listenAddr == ":2026" {
		*listenAddr = ":" + envPort
	} else if envPort := os.Getenv("PORT"); envPort != "" && *listenAddr == ":2026" {
		*listenAddr = ":" + envPort
	}

	dir := *workDir
	if dir == "" {
		if runtime.GOOS == "windows" {
			dir = "."
		} else {
			dir = config.ResolveDefaultWorkDir()
		}
	}

	srv, err := NewServer(dir, *listenAddr)
	if err != nil {
		log.Fatalf("Fatal server initialization error: %v", err)
	}

	if err := srv.Run(*listenAddr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server exited with error: %v", err)
	}
}
