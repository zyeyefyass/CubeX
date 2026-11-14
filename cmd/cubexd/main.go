package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zyeyefyass/CubeX/internal/onboarding"
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
	version = "v9.6.0"

	startTime = time.Now()
)

func configureAdaptiveMemoryLimit() {

	// Base fallback: 24MiB default ceiling for cubexd control plane

	limit := int64(24 * 1024 * 1024)

	gcPercent := 40

	totalMb := int64(1024)

	// 1. Check cgroups v2 memory.max
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.max"); err == nil {
		str := strings.TrimSpace(string(data))
		if str != "" && str != "max" {
			if bytesVal, err := strconv.ParseInt(str, 10, 64); err == nil && bytesVal > 0 {
				mb := bytesVal / (1024 * 1024)
				if mb > 0 && mb < totalMb {
					totalMb = mb
				}
			}
		}
	}

	// 2. Check cgroups v1 memory.limit_in_bytes
	if data, err := os.ReadFile("/sys/fs/cgroup/memory/memory.limit_in_bytes"); err == nil {
		str := strings.TrimSpace(string(data))
		if str != "" {
			if bytesVal, err := strconv.ParseInt(str, 10, 64); err == nil && bytesVal > 0 && bytesVal < (1<<60) {
				mb := bytesVal / (1024 * 1024)
				if mb > 0 && mb < totalMb {
					totalMb = mb
				}
			}
		}
	}

	// 3. Check total system RAM on Linux via /proc/meminfo

	if data, err := os.ReadFile("/proc/meminfo"); err == nil {

		for _, line := range strings.Split(string(data), "\n") {

			if strings.HasPrefix(line, "MemTotal:") {

				fields := strings.Fields(line)

				if len(fields) >= 2 {

					if totalKb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {

						mb := totalKb / 1024

						if mb > 0 && (totalMb == 1024 || mb < totalMb) {

							totalMb = mb

						}

					}

				}

				break

			}

		}

	}

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

	SafeGo("MemoryScavenger", func() {
		time.Sleep(15 * time.Second)
		runtime.GC()
		debug.FreeOSMemory()

		scavengeInterval := 3 * time.Minute
		ticker := time.NewTicker(scavengeInterval)
		defer ticker.Stop()
		for range ticker.C {
			runtime.GC()
			debug.FreeOSMemory()
		}
	})

	listenAddr := flag.String("listen", ":2026", "Listen address (e.g. :2026 or 0.0.0.0:2026)")
	workDir := flag.String("dir", "", "Working directory (defaults to current dir or /opt/CubeX)")
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
			if _, err := os.Stat("/app"); err == nil {
				dir = "/app"
			} else {
				dir = "/opt/CubeX"
			}
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
