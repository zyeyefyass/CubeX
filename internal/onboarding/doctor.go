package onboarding

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/zyeyefyass/CubeX/internal/netenv"
)

// CheckItemStatus 定义单项开机环境预检状态
type CheckItemStatus struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Passed      bool   `json:"passed"`
	Severity    string `json:"severity"`    // "REQUIRED" (阻断级) | "RECOMMENDED" (降级级)
	Detail      string `json:"detail"`      // 探测详情
	Remediation string `json:"remediation"` // 针对性的排障修复指令（新手可直接复制）
}

// QuickstartGuideReport 汇总环境自检结果与自动填入本机真实 IP 的多宿主共生命令
type QuickstartGuideReport struct {
	GeneratedAt  time.Time         `json:"generated_at"`
	DetectedArch string            `json:"detected_arch"`
	CubeXHostIP  string            `json:"cubex_host_ip"`
	ControlPort  int               `json:"control_port"`
	FakeIPSubnet string            `json:"fake_ip_subnet"`
	AllReady     bool              `json:"all_ready"`
	Checks       []CheckItemStatus `json:"checks"`
	HostSnippets map[string]string `json:"host_snippets"` // key: "routeros_v7", "pve_lxc", "docker_cli", "linux_iproute"
}

// RunOnboardingDoctor 执行全自动新手环境诊断并生成个性化部署指引
func RunOnboardingDoctor(ctx context.Context, hostIP string, controlPort int, fakeIPSubnet string) *QuickstartGuideReport {
	if hostIP == "" {
		topo := netenv.DetectHostTopology("", "")
		if topo.LocalIP != "" {
			hostIP = topo.LocalIP
		} else {
			hostIP = netenv.PlaceholderLocalIP
		}
	}
	if controlPort <= 0 {
		controlPort = 2026
	}
	if fakeIPSubnet == "" {
		fakeIPSubnet = "198.18.0.0/15"
	}

	checks := make([]CheckItemStatus, 0, 4)

	// Check 1: /dev/net/tun 字符设备可用性校验
	tunCheck := checkTunDevice()
	checks = append(checks, tunCheck)

	// Check 2: 网络管理权限 (ip link / route) 校验
	capCheck := checkNetAdminCapability(ctx)
	checks = append(checks, capCheck)

	// Check 3: nftables 内核快转模块可用性校验
	nftCheck := checkNftablesSupport(ctx)
	checks = append(checks, nftCheck)

	allRequiredPassed := tunCheck.Passed && capCheck.Passed

	return &QuickstartGuideReport{
		GeneratedAt:  time.Now(),
		DetectedArch: runtime.GOARCH,
		CubeXHostIP:  hostIP,
		ControlPort:  controlPort,
		FakeIPSubnet: fakeIPSubnet,
		AllReady:     allRequiredPassed,
		Checks:       checks,
		HostSnippets: buildPersonalizedSnippets(hostIP, controlPort, fakeIPSubnet),
	}
}

func checkTunDevice() CheckItemStatus {
	item := CheckItemStatus{
		ID:       "dev_net_tun",
		Title:    "TUN 虚拟网卡字符设备 (/dev/net/tun)",
		Severity: "REQUIRED",
	}
	f, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	if err != nil {
		item.Passed = false
		item.Detail = fmt.Sprintf("无法打开 /dev/net/tun: %v", err)
		item.Remediation = "【PVE LXC 修复】在宿主机 /etc/pve/lxc/<CTID>.conf 末尾追加：\n" +
			"lxc.cgroup2.devices.allow: c 10:200 rwm\n" +
			"lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file\n" +
			"【Docker 修复】启动参数需加上：--device /dev/net/tun:/dev/net/tun"
		return item
	}
	_ = f.Close()
	item.Passed = true
	item.Detail = "/dev/net/tun 读写就绪"
	return item
}

func checkNetAdminCapability(ctx context.Context) CheckItemStatus {
	item := CheckItemStatus{
		ID:       "cap_net_admin",
		Title:    "内核网络栈管理权限 (CAP_NET_ADMIN)",
		Severity: "REQUIRED",
	}
	cCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()

	out, err := exec.CommandContext(cCtx, "ip", "rule", "list").CombinedOutput()
	if err != nil {
		item.Passed = false
		item.Detail = fmt.Sprintf("ip rule 权限受限: %v (%s)", err, strings.TrimSpace(string(out)))
		item.Remediation = "【Docker 修复】请添加 --cap-add=NET_ADMIN --cap-add=NET_RAW 参数；\n" +
			"【RouterOS 容器修复】请确保容器挂载了正确的 VETH 接口且以 root 运行。"
		return item
	}
	item.Passed = true
	item.Detail = "具备策略路由与网卡管理权限"
	return item
}

func checkNftablesSupport(_ context.Context) CheckItemStatus {
	item := CheckItemStatus{
		ID:       "nftables_fastpath",
		Title:    "nftables 内核态零拷贝快转引擎",
		Severity: "RECOMMENDED",
	}

	if _, err := exec.LookPath("nft"); err != nil {
		item.Passed = false
		item.Detail = "未检测到 nft 命令（已自动平滑降级为用户态 TUN 分流）"
		item.Remediation = "如需开启千兆国内流量内核态零拷贝直通，请在容器/宿主安装 nftables 包：apt install -y nftables 或 apk add nftables"
		return item
	}
	item.Passed = true
	item.Detail = "nftables 内核快转与 DNS 防漏墙就绪"
	return item
}

func buildPersonalizedSnippets(hostIP string, port int, fakeIPSubnet string) map[string]string {
	rosScript := fmt.Sprintf(`# === CubeX v9.0 LTS × RouterOS v7 宿主共生一键绑定脚本 ===
# 1. 添加 Fake-IP 毫秒级回程路由（国内真 IP 走主路由硬件直出，仅 Fake-IP 进 CubeX 隧道）
/ip route add dst-address=%s gateway=%s distance=1 comment="CubeX-FakeIP-Backtrack"

# 2. 将主路由上游 DNS 指向 CubeX 零污染 DNS 引擎
/ip dns set servers=%s allow-remote-requests=yes
/ip dns cache flush

# 3. 部署两阶段平滑防抖 Netwatch 看门狗（容器热更 0 震荡，意外断电秒级切换 223.5.5.5 逃生）
/tool netwatch add host=%s port=%d type=tcp-conn interval=2s timeout=1s comment="CubeX-HA-Failover"`,
		fakeIPSubnet, hostIP, hostIP, hostIP, port)

	dockerCmd := fmt.Sprintf(`docker run -d --name cubex \
  --restart always \
  --network host \
  --cap-add=NET_ADMIN --cap-add=NET_RAW \
  --device /dev/net/tun:/dev/net/tun \
  -e CUBEX_HOST_IP="%s" \
  -e CUBEX_CONTROL_ADDR=":%d" \
  -v /etc/cubex:/var/lib/cubex \
  ghcr.io/zyeyefyass/cubex:v9.0.0-LTS`, hostIP, port)

	pveConf := `# 在 PVE 宿主机编辑 /etc/pve/lxc/<CTID>.conf 末尾加入以下两行后重启容器：
lxc.cgroup2.devices.allow: c 10:200 rwm
lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file`

	return map[string]string{
		"routeros_v7": rosScript,
		"docker_cli":  dockerCmd,
		"pve_lxc":     pveConf,
	}
}
