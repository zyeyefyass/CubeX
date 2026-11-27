package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/config"
	"github.com/zyeyefyass/CubeX/internal/netutil"
)

func resolveAPIBase() string {
	if raw := strings.TrimSpace(os.Getenv("CUBEX_API")); raw != "" {
		return strings.TrimRight(raw, "/")
	}
	port := strings.TrimSpace(os.Getenv("CUBEX_PORT"))
	if port == "" {
		port = config.DefaultPortStr
	}
	return fmt.Sprintf("http://127.0.0.1:%s/api/v1", port)
}

var cliHTTPClient = netutil.NewBoundedHTTPClient(8 * time.Second)

func dieDaemonError(err error) {
	fmt.Fprintf(os.Stderr, "❌ 无法连接 CubeX 控制面 (%s): %v\n请确认 cubexd 守护进程已启动，或通过 CUBEX_API / CUBEX_PORT 指定监听地址。\n", resolveAPIBase(), err)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		printBanner()
		printMenu()
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "switch":
		if len(os.Args) < 3 {
			fmt.Println("用法: cubex switch <sing|mihomo>")
			os.Exit(2)
		}
		handleSwitch(os.Args[2])
	case "mode":
		if len(os.Args) < 3 {
			fmt.Println("用法: cubex mode <rule|global|direct|hybrid>")
			os.Exit(2)
		}
		handleMode(os.Args[2])
	case "doctor":
		handleDoctor()
	case "trace":
		if len(os.Args) < 3 {
			fmt.Println("用法: cubex trace <domain|ip> (例: cubex trace openai.com)")
			os.Exit(2)
		}
		handleTrace(os.Args[2])
	case "host":
		handleHost(os.Args[2:])
	case "bypass":
		handleBypass()
	case "status":
		handleStatus()
	case "help", "-h", "--help":
		printBanner()
		printHelp()
	default:
		fmt.Printf("未知指令: %s\n输入 'cubex help' 查看支持指令。\n", cmd)
		os.Exit(2)
	}
}

func printBanner() {
	fmt.Println("\033[38;2;0;245;212m")
	fmt.Println("   ██████╗██╗   ██╗██████╗ ███████╗██╗  ██╗")
	fmt.Println("  ██╔════╝██║   ██║██╔══██╗██╔════╝╚██╗██╔╝")
	fmt.Println("  ██║     ██║   ██║██████╔╝█████╗   ╚███╔╝ ")
	fmt.Println("  ██║     ██║   ██║██╔══██╗██╔══╝   ██╔██╗ ")
	fmt.Println("  ╚██████╗╚██████╔╝██████╔╝███████╗██╔╝ ██╗")
	fmt.Println("   ╚═════╝ ╚═════╝ ╚═════╝ ╚══════╝╚═╝  ╚═╝")
	fmt.Println("\033[0m\033[38;2;123;44;191m  🌌 CubeX · 双擎旁路网关运维工具 (Mihomo × sing-box)\033[0m")
	fmt.Println()
}

func printMenu() {
	fmt.Println("\033[1;37m快捷指令集 (Quick Commands):\033[0m")
	fmt.Println("  \033[36mcubex switch sing\033[0m       - 秒级切换至 sing-box 内核")
	fmt.Println("  \033[36mcubex switch mihomo\033[0m     - 秒级切换至 Mihomo 内核")
	fmt.Println("  \033[36mcubex mode <mode>\033[0m       - 切换路由模式 (rule/global/direct/hybrid)")
	fmt.Println("  \033[36mcubex doctor\033[0m            - 调用守护进程执行系统全链路自检")
	fmt.Println("  \033[36mcubex trace <domain>\033[0m    - 查询真实规则匹配链")
	fmt.Println("  \033[36mcubex host <status|sync|ikuai|ros>\033[0m - 宿主共生联动巡检与自愈")
	fmt.Println("  \033[36mcubex bypass\033[0m            - 紧急直连逃生模式")
	fmt.Println("  \033[36mcubex status\033[0m            - 实时查看网关运行指标与内存开销")
	fmt.Println()
}

func printHelp() {
	printMenu()
}

func doJSONPost(path string, body []byte) ([]byte, int, error) {
	var reader io.Reader
	if len(body) > 0 {
		reader = bytes.NewReader(body)
	}
	resp, err := cliHTTPClient.Post(resolveAPIBase()+path, "application/json", reader)
	if err != nil {
		return nil, 0, err
	}
	defer netutil.DrainAndCloseBody(resp)
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return raw, resp.StatusCode, nil
}

func doJSONGet(path string) ([]byte, int, error) {
	resp, err := cliHTTPClient.Get(resolveAPIBase() + path)
	if err != nil {
		return nil, 0, err
	}
	defer netutil.DrainAndCloseBody(resp)
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return raw, resp.StatusCode, nil
}

func handleSwitch(target string) {
	t := strings.ToLower(target)
	engineName := "singbox"
	if t == "mihomo" || t == "mi" {
		engineName = "mihomo"
	}

	fmt.Printf("⚡ 正在执行热切换 -> [\033[1;32m%s\033[0m]...\n", engineName)
	start := time.Now()

	payload, _ := json.Marshal(map[string]string{"engine": engineName})
	raw, code, err := doJSONPost("/engine/switch", payload)
	if err != nil {
		dieDaemonError(err)
	}
	if code >= 400 {
		fmt.Fprintf(os.Stderr, "❌ 切换请求返回 HTTP %d: %s\n", code, strings.TrimSpace(string(raw)))
		os.Exit(1)
	}

	var envelope struct {
		Status string `json:"status"`
		Error  string `json:"error"`
		Result struct {
			ElapsedMs      int64  `json:"elapsed_ms"`
			RestoredGroups int    `json:"restored_groups"`
			Success        bool   `json:"success"`
			ErrorMessage   string `json:"error_message"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if envelope.Status == "switched" || envelope.Result.Success {
		ms := envelope.Result.ElapsedMs
		if ms == 0 {
			ms = time.Since(start).Milliseconds()
		}
		fmt.Printf("✅ 切换成功！耗时: \033[1;36m%d ms\033[0m | 恢复策略组: \033[1;32m%d\033[0m\n", ms, envelope.Result.RestoredGroups)
		return
	}
	errStr := envelope.Error
	if errStr == "" {
		errStr = envelope.Result.ErrorMessage
	}
	fmt.Fprintf(os.Stderr, "❌ 切换失败: %s\n", errStr)
	os.Exit(1)
}

func handleMode(mode string) {
	fmt.Printf("🔄 正在切换模式 -> [\033[1;35m%s\033[0m]...\n", mode)
	payload, _ := json.Marshal(map[string]string{"mode": mode})
	raw, code, err := doJSONPost("/engine/mode", payload)
	if err != nil {
		dieDaemonError(err)
	}
	if code >= 400 {
		fmt.Fprintf(os.Stderr, "❌ 切换模式失败 (HTTP %d): %s\n", code, strings.TrimSpace(string(raw)))
		os.Exit(1)
	}
	fmt.Println("✅ 路由模式已生效。")
}

func handleDoctor() {
	fmt.Println("🩺 正在调用守护进程执行全链路深度体检...")
	raw, code, err := doJSONGet("/health/full-audit")
	if err != nil {
		dieDaemonError(err)
	}
	if code >= 400 {
		fmt.Fprintf(os.Stderr, "❌ 自检接口返回 HTTP %d: %s\n", code, strings.TrimSpace(string(raw)))
		os.Exit(1)
	}
	fmt.Println(strings.TrimSpace(string(raw)))
}

func handleTrace(target string) {
	fmt.Printf("🧬 正在查询真实分流链路: [\033[1;36m%s\033[0m]\n", target)
	raw, code, err := doJSONGet("/rules/trace?target=" + url.QueryEscape(target))
	if err != nil {
		dieDaemonError(err)
	}
	if code >= 400 {
		fmt.Fprintf(os.Stderr, "❌ 链路探测返回 HTTP %d: %s\n", code, strings.TrimSpace(string(raw)))
		os.Exit(1)
	}
	fmt.Println(strings.TrimSpace(string(raw)))
}

func handleBypass() {
	fmt.Println("🚨 正在触发紧急直连逃生模式 (Emergency Bypass)...")
	raw, code, err := doJSONPost("/netstack/bypass", nil)
	if err != nil {
		dieDaemonError(err)
	}
	if code >= 400 {
		fmt.Fprintf(os.Stderr, "❌ 逃生接口返回 HTTP %d: %s\n", code, strings.TrimSpace(string(raw)))
		os.Exit(1)
	}
	fmt.Println("✅ 逃生成功！所有透明代理劫持已解除，全网恢复纯直连。")
}

func handleStatus() {
	raw, code, err := doJSONGet("/system/status")
	if err != nil {
		dieDaemonError(err)
	}
	if code >= 400 {
		fmt.Fprintf(os.Stderr, "❌ 状态查询失败 (HTTP %d): %s\n", code, strings.TrimSpace(string(raw)))
		os.Exit(1)
	}
	var st map[string]any
	if err := json.Unmarshal(raw, &st); err != nil {
		fmt.Fprintf(os.Stderr, "❌ 解析状态响应失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("🌌 CubeX 运行概览:")
	fmt.Printf("  系统版本: \033[1;36m%v\033[0m\n", st["version"])
	fmt.Printf("  激活内核: \033[1;32m%v\033[0m\n", st["active_engine"])
	fmt.Printf("  路由模式: \033[1;36m%v\033[0m\n", st["proxy_mode"])
	coreMem, _ := st["core_memory_mb"].(float64)
	appMem, _ := st["app_memory_mb"].(float64)
	totMem, _ := st["total_overhead_mb"].(float64)
	fmt.Printf("  专属常驻: \033[1;33m%.1f MB\033[0m (内核: %.1fM + 主控: %.1fM)\n", totMem, coreMem, appMem)
	fmt.Printf("  代理节点: %v 个 | 策略组: %v 组\n", st["total_proxies"], st["total_groups"])
	fmt.Printf("  TUN 网卡: %v | 运行时间: %v 秒\n", st["tun_interface"], st["uptime_seconds"])
}

func handleHost(args []string) {
	subCmd := "status"
	if len(args) > 0 {
		subCmd = strings.ToLower(args[0])
	}

	switch subCmd {
	case "status":
		fmt.Println("🖥️  正在探查宿主运行环境...")
		raw, code, err := doJSONGet("/host/info")
		if err != nil {
			dieDaemonError(err)
		}
		if code >= 400 {
			fmt.Fprintf(os.Stderr, "❌ 宿主信息查询失败 (HTTP %d): %s\n", code, strings.TrimSpace(string(raw)))
			os.Exit(1)
		}
		var info map[string]any
		_ = json.Unmarshal(raw, &info)
		fmt.Printf("  宿主平台: \033[1;36m%v\033[0m\n", info["platform_name"])
		fmt.Printf("  内核版本: %v\n", info["kernel_version"])
		fmt.Printf("  宿主状态: 连通 (%v)\n", info["connected"])

	case "sync", "heal":
		fmt.Println("⚡ 正在执行全宿主多底座同步与自愈...")
		raw, code, err := doJSONPost("/host/firewall/apply", nil)
		if err != nil {
			dieDaemonError(err)
		}
		if code >= 400 {
			fmt.Fprintf(os.Stderr, "❌ 宿主同步失败 (HTTP %d): %s\n", code, strings.TrimSpace(string(raw)))
			os.Exit(1)
		}
		fmt.Println("  ✓ 宿主防火墙与路由策略同步成功！")

	case "ikuai":
		fmt.Println("🩺 正在巡检与加固爱快 (iKuai) Web-RPC 联动与静态回指路由...")
		raw, code, err := doJSONPost("/host/ikuai/sync", nil)
		if err != nil {
			dieDaemonError(err)
		}
		if code >= 400 {
			fmt.Fprintf(os.Stderr, "❌ 爱快同步失败 (HTTP %d): %s\n", code, strings.TrimSpace(string(raw)))
			os.Exit(1)
		}
		var res map[string]any
		if json.Unmarshal(raw, &res) == nil {
			if msg, ok := res["message"].(string); ok && msg != "" {
				fmt.Printf("  ✓ %s\n", msg)
			}
			if logs, ok := res["logs"].([]any); ok {
				for _, l := range logs {
					fmt.Printf("    • %v\n", l)
				}
			}
		}

	case "ros":
		fmt.Println("🩺 正在深度巡检 RouterOS 策略路由与 Netwatch 看门狗...")
		raw, code, err := doJSONPost("/ros/routing/clean-and-reapply", nil)
		if err != nil {
			dieDaemonError(err)
		}
		if code >= 400 {
			fmt.Fprintf(os.Stderr, "❌ RouterOS 同步失败 (HTTP %d): %s\n", code, strings.TrimSpace(string(raw)))
			os.Exit(1)
		}
		fmt.Println("  ✓ RouterOS 策略路由及 Netwatch 刷新完成！")

	default:
		fmt.Println("用法: cubex host <status|sync|ikuai|ros>")
		os.Exit(2)
	}
}
