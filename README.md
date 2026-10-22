<div align="center">

  <img src="logo-cubex.svg" width="140" height="140" alt="CubeX 3D Cyber Aurora Crystal Logo" />

  # 🌌 CubeX · 宿主共生零感网络系统 (Host-Symbiotic Zero-Perception Gateway)
  ### *单二进制纯自研 · 告别传统重型虚机旁路 · 极致轻量低开销 · 300ms 冷启 · 宛如物理肉身直连全球互联网的终极共生体*

  [![Version](https://img.shields.io/badge/Release-Test_v2.5-00F5D4?style=for-the-badge&logo=github)](https://github.com/zyeyefyass/CubeX/releases)
  [![License](https://img.shields.io/badge/License-Proprietary_EULA-7B2CBF?style=for-the-badge&logo=shield)](COPYRIGHT.md)
  [![Dual-Engine](https://img.shields.io/badge/Dual--Engine-Mihomo_%C3%97_sing--box_v1.14+-0077B6?style=for-the-badge&logo=go)](https://github.com/zyeyefyass/CubeX)
  [![Host Base](https://img.shields.io/badge/Host-RouterOS_v7_%7C_fnOS_%7C_PVE_%7C_Docker-3A86FF?style=for-the-badge)](#)
  [![Arch](https://img.shields.io/badge/Arch-amd64_%7C_arm64-FF007F?style=for-the-badge)](#)

  <p align="center">
    <b>纯 Go 独立自研单二进制架构 · 宿主机实时物理开销仅 ~38MB~49MB (同类最优) · 0 侵入 0 依赖 · 0 明文 DNS 泄露 · 217heidai 权威去广告</b><br>
    <i>让小白零门槛免误操作，让老手彻底省心变懒人！立志做软路由性能、安全与颜值的天花板！</i>
  </p>

</div>

---

> [!IMPORTANT]
> ### 📜 郑重声明与合规免责条款 (Legal & Compliance Statement)
> 1. **学术与技术探索**：本项目属于网络协议栈架构、双引擎动态容灾调度、旁路透明路由与自动化运维的**纯技术探索、工程实践与学术交流**。
> 2. **合法用途界定**：本系统仅面向**合规跨境电商进出口贸易、跨国协同办公、前沿 AI 学术研究与大模型接入、高校与科研院所国际技术交流**等正当合法场景。
> 3. **严格遵规守法**：使用者须严格遵守所在国家与地区的法律法规及行业监管准则。**严禁任何个人或组织将本系统用于任何违反国家法律法规、侵害网络安全利益、散播违规信息或触碰敏感话题与政治红线的非法活动**。
> 4. **免责与自律约定**：软件作者及维护者不对任何第三方非法违规使用本系统所导致的直接或间接法律责任承担连带责任。获取、编译、部署或运行本系统即代表完全知悉并承诺自觉遵守本条款。

---

<a id="announcement"></a>
> [!TIP]
> ### 📢 小伙伴们请立即一键平滑升级 Test v2.5（代理内核升级自主决策与冻结锁定 · 全息控制台四卡片版面极致对称 · Geo 规则体系动态感知与静默自动更新守护架构 · RouterOS 固件并发静默探查 · 极客纯净版）！
> **致所有追求极致软路由稳定性、高并发吞吐与视觉体验的极客老铁们：**
> 
> 在经过多环境深度实机排查与业务分流架构演进后，**CubeX Test v2.5 正式重磅发布**！  
> 本版本重点针对双代理内核自主升级与冻结防护、全息控制台四卡片极致对称重构、Geo 规则体系动态感知与静默自动更新守护进行了深度闭环与硬核优化：
> 
> * 🔒 **双代理内核升级自主决策与冻结锁定中枢 (`Core Upgrade Freeze Protection & Urge`)**：
>   * 支持用户在控制台自主冻结内核升级（`🔒 冻结升级`），锁定经过严格对齐测试的生产内核，防止上游官方语法与特性破坏性变更导致运行异常；
>   * 解锁状态下若探针检测到新版内核，界面展示醒目风险提示与自主决策说明，并提供一键自动生成预填 Issue 模板直达 GitHub（`📮 催更 Issue`），全自动带入当前架构、CubeX 版本与宿主底座，催促作者对齐升级。
> * 📐 **全息控制台四卡片版面极致对称与交互重构 (`Four-Card Grid Geometric Symmetry`)**：
>   * 严格规范控制台顶部四大核心卡片（主控守护进程、双代理内核、Geo 规则体系、RouterOS/宿主固件版本）高度与布局契合；
>   * 操作按钮统一采用 50% / 50% 完美对称双列设计（`grid grid-cols-2 gap-2`），视觉更开阔、现代且符合工控美学。
> * 📚 **Geo 规则体系（105 套规则）动态感知与静默自动更新守护架构 (`Geo Rule Dynamic Update & Scheduled Sync`)**：
>   * 彻底剥离去广告与基础分流规则混淆概念，右上角状态标识动态自适应：基础库与 105 套分流策略集生效 ➔ 发现新规则即刻脉冲提醒（`🚀 发现规则更新 (待同步)`）➔ 一键秒级同步；
>   * 底部状态区新增智能定时自动更新守护状态（`每日 04:00 静默同步 · 内存热重载无需重启`），并支持一键快速开启/暂停，实现主动即时同步与后台定时守护完美共生。
> * 📡 **RouterOS 固件通道智能并发静默探查 (`Silent Firmware Upgrade Probe`)**：
>   * 宿主固件卡片加载时后台静默探查 RouterOS 官方升级通道，精准呈现 `↻ 固件通道核对中...` ➔ `🚀 发现新固件` 或 `✓ 已是最新稳定版`，消除用户认知滞后。
> * 🚀 **双核秒级热切换全链路极速压测闭环验证 (`Dual-Engine Live Benchmark`)**：
>   * 在实机 RouterOS CHR 网关上完成 sing-box ➔ Mihomo ➔ sing-box 双核来回热切换与 4K DASH 大带宽下行吞吐量压测（~80 Mbps 极速），直连与出海代理双通道毫秒级热更无感顺滑，0 明文 DNS 侧漏。
> 
> <a id="release-assets"></a>
> ### 📦 资产下载与校验清单 (Release Assets)
> 
> | 文件名 | 平台 / 底座 | 文件大小 | 说明 |
> | :--- | :--- | :--- | :--- |
> | `CubeX-Test-v2.5-ros-amd64.tar` | MikroTik RouterOS v7 (x86_64) | ~90.0 MB | ROS 专有未压缩容器镜像（支持 WinBox `/container/add` 一键导入） |
> | `CubeX-Test-v2.5-ros-arm64.tar` | MikroTik RouterOS v7 (ARM64 / RB5009) | ~95.0 MB | ROS 专有未压缩容器镜像 |
> | `CubeX-Test-v2.5-linux-amd64.zip` | Linux amd64 (通用宿主 / PVE / 飞牛) | ~68.8 MB | 包含双内核、规则集、配置与控制面的完整生产归档 |
> | `CubeX-Test-v2.5-linux-arm64.zip` | Linux arm64 (通用宿主 / 树莓派 / 软路由) | ~67.5 MB | 包含双内核、规则集、配置与控制面的完整生产归档 |
> | `cubexd-linux-amd64` | Linux amd64 单二进制 | ~10.7 MB | 用于现有容器或系统的一键在线平滑热更执行文件 |
> | `cubexd-linux-arm64` | Linux arm64 单二进制 | ~10.0 MB | 用于现有容器或系统的一键在线平滑热更执行文件 |
> | `setup-routeros-v7.rsc` | RouterOS 终端编排脚本 | ~26.9 KB | RouterOS v7 WinBox 终端复制即用一键自动编排脚本 |
> 
---

<details>
<summary><b>📚 快速直达目录 (点击展开 / 折叠 · 全维度架构与部署导航)</b></summary>

<br>

- [📢 最新版本升级通知](#announcement)
- [🧭 架构决胜：为什么坚定采用「主网关 + 旁路 DNS」黄金解耦与创新？](#architecture-golden)
  - [❌ 传统“全包型”旁路由（网关指旁路）的三大死穴](#pain-points-traditional)
  - [✅ CubeX「主网关 + 旁路 DNS」的降维打击机制与数据流向拓扑](#topology-flow)
  - [🥊 传统旁路由的“五大死穴”与 CubeX 降维打击对照表](#table-five-lethal)
- [⚡ 权威实测：CubeX vs 某测评 全维度基准对决 (同带宽 · 同宿主 · 同节点压测)](#benchmark)
- [🧹 一键自愈与深度巡检重器：拯救网络的降压药](#auto-healing)
  - [1. 轨道 A（针对 RouterOS 宿主 / 主路由）的四层救赎](#healing-track-ros)
  - [2. 轨道 B（针对 飞牛 fnOS / Linux 宿主机）的底层调优](#healing-track-linux)
  - [3. 🌐 原生 IPv6 双栈自愈与加固深度剖析（防火墙如何运作）](#healing-ipv6-dualstack)
  - [4. 🔍 爱快 iKuai 作为宿主时的联动与回指剖析](#healing-ikuai-synergy)
- [🚀 极速部署指南 (RouterOS / 爱快 iKuai / 飞牛 fnOS / PVE / Linux)](#deployment-guides)
  - [方式 A：MikroTik RouterOS v7 容器一键极速安装指南 (官方强烈推荐 · 工业级共生)](#deploy-ros)
  - [方式 B：爱快 iKuai 64位 主路由 Docker 原生轻量部署 (保姆级教程)](#deploy-ikuai)
  - [方式 C：飞牛 fnOS 与通用 Linux Docker 部署](#deploy-fnos-linux)
  - [方式 D：Proxmox VE (PVE) CT 容器 (LXC) 极速部署](#deploy-pve)
  - [方式 E：通用 Linux 终端一键全自动安装](#deploy-linux-cli)
- [💬 极客畅所欲言：意见征集与需求大厅](#community-feedback)
- [📜 净室合规与免责声明](#compliance-statement)

<br>

</details>

---

<a id="architecture-golden"></a>
## 🧭 架构决胜：为什么坚定采用「主网关 + 旁路 DNS」黄金解耦与创新？

这是所有追求极致性能与高可用软路由玩家必须认清的最关键网络命题：**为什么 CubeX 绝不推荐粗暴地把终端网关指向旁路由，而是坚决采用「主网关 + 旁路 DNS」的黄金解耦架构？**

<a id="pain-points-traditional"></a>
### ❌ 传统“全包型”旁路由（网关指旁路）的三大死穴：
1. **网卡双向折返跑，千兆/万兆带宽直接腰斩**：  
   在传统的全包旁路拓扑中，终端哪怕看个国内腾讯视频或访问局域网 NAS，所有数据包也必须经历：`终端 -> 交换机 -> 旁路由网卡 -> 交换机 -> 主路由 -> 光猫`。单网口全双工带宽被双向重叠流量挤爆，软路由 CPU 100% 满载烤鸡蛋，千兆宽带实测往往不到 500Mbps！
2. **单点暴毙，全家瞬间失联“升仙”**：  
   大订阅加载突发内存尖刺导致容器崩溃，或者手残改错一个出站规则导致代理重启，全家所有设备瞬间断网。不仅上不了网，甚至连主路由的管理后台都进不去，智能家居设备集体失联！
3. **ICMP Send-Redirects 三角重定向风暴**：  
   终端、主路由与旁路由同在二层局域网，主路由默认发送 ICMP 重定向，导致 Windows / iOS 频繁出现 TCP 握手停顿、无故掉线与重传。

---

<a id="topology-flow"></a>
### ✅ CubeX「主网关 + 旁路 DNS」的降维打击机制与数据流向拓扑：

在 CubeX 体系中，局域网终端的 **默认网关始终保持指向硬件主路由（如 10.0.0.1）**，而 **DNS 服务器单向指向 CubeX（如 10.0.0.11）**：

```text
【终端发起网络请求】
         │
         ▼
[1. DNS 查询] ───────────────────────────► 【CubeX 旁路 DNS (10.0.0.11)】
                                                      │
                       ┌──────────────────────────────┴──────────────────────────────┐
                       ▼                                                             ▼
              [🇨🇳 国内域名 (如百度/微信/B站)]                               [🌐 国外/加速域名 (如 Google/GitHub/YouTube)]
                       │                                                             │
             返回真实公网 IP (直连)                                          返回 Fake-IP (198.18.0.0/15)
                       │                                                             │
                       ▼                                                             ▼
         【主路由网关 (10.0.0.1)】                                      【主路由网关 (10.0.0.1)】
                       │                                                             │
         主路由 FastTrack 硬件加速                                     命中静态路由: 198.18.0.0/15 -> 10.0.0.11
                       │                                                             │
                       ▼                                                             ▼
              【⚡ 满速直出光猫】                                           【🚀 转入 CubeX 容器双引擎代理出海】
         (跑满千兆/万兆，主路由 CPU 仅 ~1%)                                (Mihomo × sing-box 秒级混动出海)
```

- ⚡ **国内大流量 100% 物理线速直出**：国内流量与局域网内网传输**根本不走 CubeX 容器网卡**，直接被主路由器芯片硬件加速（FastTrack）线速转发，主路由 CPU 仅 ~1%，无论局域网跑 2.5G 还是 10G 满带宽，毫无一丝衰减与发热！
- 🎯 **国外流量 Fake-IP 精准截获**：只有命中 `198.18.0.0/15` 的真正出海流量，才经由主路由一条轻量静态路由递交给 CubeX，零浪费、零折返！
- 🛡️ **宕机零感逃生机制 (Zero-Downtime Escape)**：就算 CubeX 容器宕机或重启，主网关依然健在，国内访问、微信与智能家居完全无感；配套 3 秒 Netwatch 探针自动切换备用 DNS，全家绝不断网！

---

<a id="table-five-lethal"></a>
### 🥊 传统旁路由的“五大死穴”与 CubeX 降维打击对照表

| 传统旁路由痛点 | 发生原理与后果 | CubeX 降维打击破局解法 |
| :--- | :--- | :--- |
| **死穴 ①：直连测速不到带宽一半** | 终端网关强指旁路由，导致去往百度的国内大流量在单网口上双向折返跑，软路由 CPU 爆满掉速。 | **架构解耦**：终端网关指主路由，DNS 指 CubeX。国内真实 IP 直接由主路由硬件 FastTrack 满速直通，主路由 CPU 仅 ~1%！ |
| **死穴 ②：莫名其妙卡顿重传** | 旁路与主路由同处于一个网桥（如 `bridge1`），主路由默认发出 ICMP 重定向包，引发 Windows/iOS 频繁 TCP 停顿。 | **自动封穴**：CubeX 巡检引擎自动设置 `/ip/settings set send-redirects=no`，彻底掐死三角重定向丢包！ |
| **死穴 ③：假 IP 漏往公网证书报错** | 传统教程只写 `198.18.0.0/16`，漏掉了 `198.19.x.x`，被分配到该网段的 Gemini/Google 流量漏到外网光猫报证书错误。 | **完整 /15 回指**：全面升级为 RFC 2544 标准 `198.18.0.0/15` 回指路由，并开启双引擎强域名逆向恢复与 IPv6 ULA 映射！ |
| **死穴 ④：内核崩溃拉崩全家网** | 大订阅加载时突发内存尖刺，Linux 内核 OOM Killer 强杀代理进程，全家断网，电话被打爆。 | **双保险熔断**：为双引擎注入软上限守护与 cgroup 内存墙，并配备 3 秒 Netwatch 宕机零感直连逃生！ |
| **死穴 ⑤：VLESS Reality 协议暴毙** | 很多旧工具解析 URL 时丢掉了 `pbk`、`sid` 或 `fingerprint`，导致握手直接报 `unexpected response version`。 | **全协议纯自研解析器**：完整提取并编排 Reality/gRPC/H2 参数，双核握手 100% 成功，延迟降低 30%。 |

---

<a id="benchmark"></a>
## ⚡ 权威实测：CubeX Test v2.5 vs 某测评 全维度基准对决 (同带宽 · 同宿主 · 同节点压测)

> [!NOTE]
> ### 📌 严格测试条件与控制变量声明 (Benchmark Prerequisites & Controls)
> 为保证评测的客观性、可复现性与工业级严谨度，本次对决严格遵循科学控制变量准则：
> 1. **同带宽网络环境**：完全相同的家庭千兆物理光纤宽带（同一运营商 PPPoE 接入，测试期间无局域网外生带宽干扰）；
> 2. **同物理机/宿主平台**：均部署并运行于同一宿主机底座（RouterOS CHR 4核 CPU 分配，完全一致的硬件芯片、物理网卡与虚拟总线）；
> 3. **同一代理出站节点**：测试流量严格锁定于完全相同的落地专线 VPS（同一目标服务器、完全一致的加密传输协议配置）；
> 4. **持续压力测试统计平均值**：所有延迟、吞吐与首包数据均经由**单流多轮交叉压测、4 线程高并发冲顶、YouTube 4K DASH 切片连续拉取**取多次实测的**严格统计平均值**，拒绝任何单次偶然误差与幸存者偏差。

| 对比实测维度 | 🥊 某测评 (10.0.0.2)<br>某 Box 传统镜像 | 🌌 CubeX Test v2.5 (10.0.0.11)<br>深度优化威武版 | 🏆 优势与性能跃升 |
| :--- | :--- | :--- | :--- |
| **🎬 YouTube 4K 首帧耗时** | `0.92 s` | **`0.37 s`** | **首帧加载快 59.8%（秒开起播）** |
| **🎬 YouTube 4K 缓冲吞吐** | `123.88 Mbps` | **`248.06 Mbps`** | **视频持续缓冲速率翻倍 (+100.2%)** |
| **🎬 YouTube 4K 首包延迟 (TTFB)** | `528.4 ms` | **`180.2 ms`** | **首包响应延迟降低 65.9%** |
| **🚀 单流线速下载平均带宽** | `235.80 Mbps` | **`294.97 Mbps`** | **单连接吞吐提升 +25.1%** |
| **⚡ 4 并发多流冲顶峰值** | `57.22 Mbps` | **`163.01 Mbps`** | **高并发冲顶吞吐近 3 倍 (+184.9%)** |
| **💾 宿主机实时物理内存开销** | `~72.8 MB` (仅几千条基础规则) | **基础冷启 `~35 MB` / 百级节点 23.3万条规则生产实战 `~109 MB` (内核 93M + 主控 16M)** | **百级节点全量实战依然比虚拟机节省 90%+ 资源，物理常驻仅约百兆** |
| **🛡️ 境外 QUIC (UDP 443) 策略** | 宽松放行（国内易遭恶性丢包） | **即时 0ms RST 熔断 (强制平滑降级 TCP+BBR)** | **彻底杜绝 4K 播放途中跳降 1080p** |
| **🔒 0 明文 DNS 泄露防御** | 存在明文回退泄露隐患 | **SERVFAIL 绝对物理熔断 (极光盾 100/100)** | **零侧漏、零假 IP 污染** |
| **🧹 广告与隐私追踪拦截库** | 仅数千条基础规则 | **233,481 条 (217heidai 权威规则)** | **拦截规则规模高出 50 余倍** |
| **🔄 核心容灾与高可用架构** | 单一 sing-box 进程，崩溃即断网 | **Mihomo × sing-box v1.14+ 混动秒级热备** | **双核无感切换，毫秒级故障逃生** |
| **👑 物理主路由深度联动** | 仅基础脚本，无商业主路由联动 | **RouterOS REST API + 爱快 Web-RPC 原生直通** | **全自动注入回指路由与 DHCP DNS** |
| **💾 内存控制与自愈机制** | 虚机占用 1-4GB，无内存上限易 OOM | **独创双核自适应限额 + cgroup 256M 内存墙 + 毫秒级自愈拉起** | **基础冷启 ~35MB，百级节点生产实战稳定控制在 ~109MB，彻底杜绝页缓存膨胀与意外停机** |

---

<a id="auto-healing"></a>
## 🧹 一键自愈与深度巡检重器：拯救网络的降压药

很多朋友疑惑：**“一键深度巡检与全量重配功能是仅针对 RouterOS 吗？在飞牛 fnOS 或 Linux Docker 上这个功能是干嘛的？”**

**答案：CubeX 采用智能底座感知技术，控制台【宿主共生】页面的巡检按钮会根据当前宿主环境自适应呈现与执行！**

```text
┌────────────────────────────────────────────────────────────────────────┐
│               🧹 CubeX 智能双轨自愈巡检总线 (Adaptive Auto-Healing)     │
├────────────────────────────────────────────────────────────────────────┤
│  🟢 轨道 A：当运行于 RouterOS 容器 (或外置主路由为 ROS)                │
│     ↳ 激活按钮：【🧹 一键深度巡检 RouterOS 策略路由及全量重配】        │
│     ↳ 作用目标：MikroTik RouterOS 主路由器                             │
│     ↳ 救赎动作：自动清扫旧 Mangle 冲突、重铸 198.18.0.0/15 回指路由、   │
│                关闭 send-redirects 掐死三角丢包、注入 Netwatch 逃生探针│
├────────────────────────────────────────────────────────────────────────┤
│  🟣 轨道 B：当运行于 飞牛 fnOS / PVE / 通用 Linux Docker               │
│     ↳ 激活按钮：【⚡ 一键深度巡检并配置 宿主防火墙与内核转发】         │
│     ↳ 作用目标：飞牛 fnOS / PVE / Linux 宿主机操作系统                 │
│     ↳ 救赎动作：自动开启 Linux 内核 net.ipv4.ip_forward=1 转发、       │
│                放行 iptables FORWARD、注入 TCP MSS Clamping 消灭分片卡顿│
├────────────────────────────────────────────────────────────────────────┤
│  🔄 宿主联动：RouterOS / 爱快双主路由深度协同联动                     │
│     ↳ 场景：CubeX 跑在飞牛 Docker 上，但家里主路由恰好也是 RouterOS    │
│     ↳ 动作：在飞牛控制台直接点击【📶 联动 RouterOS 主路由】，跨机自动配齐 ROS! │
└────────────────────────────────────────────────────────────────────────┘
```

<a id="healing-track-ros"></a>
### 1. 轨道 A（针对 RouterOS 宿主 / 主路由）的四层救赎：
1. **清道夫扫描**：自动拔除所有带有暴力劫持倾向的旧版 Mangle 抢流规则与 DNS 53 强制 NAT 劫持；
2. **全网段路由重铸**：自动写入 RFC 2544 完整网段 `198.18.0.0/15 -> CubeX IP`，清除残留冲突的旧网关条目；
3. **关闭二层重定向**：自动执行 `/ip/settings/set send-redirects=no`，彻底消除局域网 TCP 握手停顿；
4. **注入安全哨兵**：自动为 FastTrack 添加 `!CUBEX_BYPASS` 白名单、配置 Forward 丢弃前放行规则，锁定 `advertise-dns=no` 掐死运营商 IPv6 DNS 广播，并注入 3 秒极速心跳 Netwatch 逃生探针！

<a id="healing-track-linux"></a>
### 2. 轨道 B（针对 飞牛 fnOS / Linux 宿主机）的底层调优：
1. **内核全开**：自动写入 `net.ipv4.ip_forward=1` 与 `net.ipv4.conf.all.forwarding=1`，打通旁路软网关数据包跨网卡穿透；
2. **防火墙放行**：自动置顶注入 `iptables -I FORWARD 1 -j ACCEPT`，放行 Web 面板 TCP 2026 与 DNS 53 端口；
3. **消除分片黑洞**：注入 `TCP MSS Clamping (--clamp-mss-to-pmtu)`，彻底解决跨 MTU 网络大包丢包、网页只加载文字不加载图片的问题；
4. **TCP 握手加速**：开启 `net.ipv4.tcp_fastopen=3` 并启用 `FQ` 队列调度，降低首包握手延迟与排队抖动。

---

<a id="healing-ipv6-dualstack"></a>
### 3. 【🌐 原生 IPv6 双栈自愈与加固】深度剖析（到底配置了什么？防火墙如何运作？）

在控制台【系统升级与联动】中，点击 **`【🌐 一键自愈并加固 RouterOS 原生 IPv6 双栈 (防 DNS 泄露 · 毫秒级海外快速回退)】`**，很多极客朋友非常关心：**“它在 RouterOS 内部到底帮我配了哪些项目？是不是也启用了 IPv6 并配置了 IPv6 防火墙？”**

**答案：是的！CubeX 通过 RouterOS REST API 执行了 6 大维度的原子级双栈加固与防火墙闭环编排：**

```text
┌───────────────────────────────────────────────────────────────────────────────┐
│           🌐 CubeX RouterOS 原生 IPv6 双栈自动化编排与防火墙防御架构           │
├───────────────────────────────────────────────────────────────────────────────┤
│  1. 全局启用与转发: /ipv6/settings ➔ disable-ipv6=false, forward=true         │
│  2. 外网探查与PD: 自动探查 WAN 口 (PPPoE/ether1), 创建 /ipv6/dhcp-client 获取前缀│
│  3. 内网桥接广播: /ipv6/address ➔ 从地址池 CubeX-IPv6-Pool 为内网 bridge 分配    │
│  4. 防DNS泄露死穴: 锁定所有 /ipv6/nd ➔ advertise-dns=false (掐灭运营商DNS污染)   │
│  5. 防火墙极速拒答: /ipv6/firewall/filter ➔ 2000::/3 reject (毫秒级回退IPv4代理)│
│  6. 国内流量直连: 百度、腾讯、B站等国内 IPv6 流量不受限制，物理线速满带宽狂飙  │
└───────────────────────────────────────────────────────────────────────────────┘
```

#### 🛠️ 6 大关键配置与防火墙技术细节：

1. **全面启用 IPv6 协议栈与内核数据包转发 (`/ipv6/settings`)**：
   - 自动下发 `disable-ipv6=false` 与 `forward=true`。
   - 确保主路由器底座具备完整的 IPv6 三层路由与跨网卡线速转发能力，消除部分系统因默认禁用 IPv6 导致的链路死锁。
2. **自适应探查外网 WAN 出接口**：
   - 算法自动检索 `0.0.0.0/0` 默认路由出接口（智能匹配 `pppoe-out1` 拨号口、`ether1` 或 `wan` 物理口），无需用户在命令行苦苦核对接口名。
3. **自动化编排外网 DHCPv6-PD 前缀获取客户端 (`/ipv6/dhcp-client`)**：
   - 在识别出的 WAN 口自动创建 DHCP-PD 客户端，配置 `request=prefix` 与 `pool-prefix-length=64`，将运营商动态下发的公网 IPv6 前缀汇聚至专用地址池 `CubeX-IPv6-Pool`。
4. **内网局域网 IPv6 地址无感下发与通告 (`/ipv6/address`)**：
   - 自动绑定 `from-pool=CubeX-IPv6-Pool` 至内网主网桥 `bridge`，开启 `advertise=true`。
   - 家中所有连接主路由的手机、电脑、NAS 及智能家居设备，即插即用自动获取到合法独立的全球单播 IPv6 地址。
5. **铁血扼杀运营商 IPv6 DNS 泄露 (`/ipv6/nd`)【最关键防线】**：
   - 遍历主路由所有邻居发现（Neighbor Discovery）接口，**强制锁定 `advertise-dns=false`**！
   - **痛点原理**：99% 的传统 IPv6 漏球灾难，皆因路由器通过 IPv6 RA 广播向内网终端下发了运营商（电信/联通/移动）的 IPv6 本地 DNS 地址。终端在发起网络请求时，往往优先使用运营商 IPv6 DNS 解析外网域名，不仅导致严重的 DNS 污染，更直接让旁路由分流形同虚设！CubeX 一键彻底掐死运营商 DNS 广播，内网所有 DNS 查询必须闭环由 CubeX 安全防泄露内核接管。
6. **防火墙 2000::/3 极速拒答与海外快速回退 (`/ipv6/firewall/filter`)【黑科技】**：
   - 在 RouterOS 防火墙 `forward` 链顶端自动注入规则：
     ```routeros
     /ipv6/firewall/filter
     add chain=forward dst-address=2000::/3 action=reject reject-with=icmp-admin-prohibited comment="CubeX Anti-Block IPv6 Fast-Fallback & Anti-Leak"
     ```
   - **为什么采用 `reject` 而不是 `drop` 或直接放行？**
     - 全球绝大部分海外 VPS 和机场代理节点在 IPv4 下具备最优 BGP/CN2 优化，而直连海外 IPv6（`2000::/3`）极易遭遇 GFW 丢包黑洞或跨国路由拥塞，导致长达数十秒的握手停顿；
     - 若配置为 `drop`，客户端会持续等待 TCP 重传超时（卡顿 10~30 秒）；
     - **CubeX 采用 `reject-with=icmp-admin-prohibited`（管理性阻断）**：当设备试图向海外发起原生 IPv6 直连时，主路由在 **0 毫秒内** 回复拒绝报文，现代浏览器和操作系统的 **Happy Eyeballs 双栈快速竞争算法** 会在**毫秒级瞬间回退至 IPv4 极速代理链路**！国内各大网站与直连服务（通过国内白名单和真实公网 IP）仍旧享受满血 IPv6 直通，彻底兼顾了国内 IPv6 高速体验与海外科学浏览的丝滑秒开！

> [!IMPORTANT]
> **⚠️ 关键澄清（关于封闭家用主路由）：**  
> 如果 CubeX 运行在飞牛 fnOS / Linux 上，而局域网外置主路由是**小米、TP-Link、华硕原厂等封闭家用路由器**：  
> - CubeX 点击一键配置会 100% 全自动完成飞牛宿主自身的内核与防火墙调优（**轨道 B**）；  
> - 但由于家用封闭路由器没有开放 API，**无法远程自动侵入修改主路由后台**。若想体验「主路由网关 + 旁路 DNS」黄金解耦，需用户在小米/TP-Link 后台手动加一条 `198.18.0.0/15` 静态路由（或采用开箱免配置的传统旁路由网关模式）。

---

<a id="healing-ikuai-synergy"></a>
### 4. 【🔍 极客深究：爱快 iKuai 作为宿主时，能否实现同 RouterOS 一样的一键巡检自愈与全量重配？】

许多同时玩爱快与软路由的极客朋友发问：**“如果我家里的主路由/宿主是爱快（iKuai），CubeX 能否像在 RouterOS 里那样，直接在 Web 控制台点击「一键巡检自愈与全量重配」？”**

#### 💡 核心结论速览：
> **结论**：**网络拓扑层的关键联动（静态路由自动重铸、DHCP 旁路 DNS 自动下发/还原、全网在线终端与实时流量穿透监控）完全能够通过爱快 Web JSON-RPC 接口实现！**  
> 但受限于爱快系统的商业闭源黑盒属性，在**微秒级内核防火墙规则精准注入（如 IPv6 `2000::/3 reject`）** 与 **Netwatch 脚本引擎秒级自动切断逃生** 两个深度底层机制上，与 RouterOS 存在系统架构层面的客观差异。

```text
┌─────────────────────────────────────────────────────────────────────────────────┐
│         ⚖️ CubeX 一键自愈巡检能力对比：RouterOS (极客底座) vs 爱快 iKuai (企业底座)      │
├───────────────────────────────────────────────────┬──────────────┬──────────────┤
│ 自动化与自愈巡检维度                               │ RouterOS v7  │ 爱快 iKuai   │
├───────────────────────────────────────────────────┼──────────────┼──────────────┤
│ 1. 自动写入 198.18.0.0/15 静态回指路由             │ ✅ 毫秒级原生│ ✅ Web-RPC可配│
│ 2. 自动切换/恢复 DHCP 首选 DNS (指向 CubeX)        │ ✅ 毫秒级原生│ ✅ Web-RPC可配│
│ 3. 冲突路由/历史残留规则自动扫描与清理             │ ✅ 深度清道夫│ ✅ 静态路由自愈│
│ 4. 局域网在线设备、MAC 与流量速率监控穿透联动      │ ✅ 原生抓取  │ ✅ 原生抓取  │
│ 5. 一键回滚（秒级恢复主路由纯净默认状态）          │ ✅ 一键还原  │ ✅ 一键还原  │
│ 6. 底层自定义防火墙链条精准注入 (Filter/Mangle)    │ ✅ 100% 开放 │ ❌ 商业闭源  │
│ 7. 原生 IPv6 2000::/3 管理性拒答快速回退           │ ✅ 0ms 拒答  │ ⚠️ 需域名规则│
│ 8. Netwatch 3秒心跳脚本自动切断旁路无感逃生        │ ✅ 脚本引擎  │ ⚠️ 架构天然防断│
└───────────────────────────────────────────────────┴──────────────┴──────────────┘
```

#### 🔬 深度技术拆解：

1. **爱快可以 100% 完美做到的核心能力（CubeX 架构能力）**：
   - **自动化静态路由重铸**：爱快管理后台提供了基于 HTTP JSON-RPC 的 Web 管理接口（`/Action/call`，`route_static` 模块）。CubeX 可以通过填入的爱快管理账号密码，全自动检测并一键下发 `198.18.0.0/15`（或双 `/16`）静态路由，彻底杜绝小白用户手动填错掩码或接口的烦恼；
   - **自动化 DHCP 旁路 DNS 赋能与一键回滚**：通过 `dhcp` 模块，一键将爱快当前 LAN 的首选 DNS 设为 CubeX IP，并保留爱快默认网关。出故障或维护时随时支持一键回滚还原回 `114.114.114.114` 或阿里等公共 DNS；
   - **系统资源与全网拓扑感知**：通过 `monitor_lanip` 与 `sysstat` 接口，CubeX 仪表盘可直接穿透读取爱快宿主的 CPU、内存负载与全家终端的实时速率。

2. **爱快与 RouterOS 的客观系统架构边界**：
   - **底层防火墙开放度差异**：RouterOS 是一套完整开放的“工业级网络操作系统”，开放了对 Linux 内核 Netfilter（Raw/Mangle/Filter/NAT）的完全编排权限；而爱快是封闭式商业流控固件，不提供 Root Shell 终端，且防火墙为固定功能模块，**不支持由外部程序动态注入自定义 iptables 规则链**；
   - **IPv6 毫秒级拒答**：RouterOS 支持精准配置 `/ipv6/firewall/filter dst-address=2000::/3 action=reject`；而在爱快中，无法通过 API 注入这样一条精确的底座 reject 规则，防海外 IPv6 卡顿主要由 CubeX 容器内核的 Fake-IP 与 IPv4-First 机制完成；
   - **逃生机制差异**：RouterOS 拥有内置任务计划与 Netwatch 探针脚本，容器一旦假死，RouterOS 会在 3 秒内自动执行脚本剔除路由；而爱快没有可编程脚本引擎。**但幸运的是，由于 CubeX 采用「主网关 + 旁路 DNS」黄金解耦架构，终端网关始终是爱快物理路由，即使 CubeX 容器停止，爱快物理拨号与全家国内网络也永远 100% 畅通，绝不会像传统旁路由那样直接拉崩全家！**

---


<a id="dns-rewrite-hub"></a>
## 🔀 DNS 重写中枢：域名别名 CNAME 重定向与 静态 IP 映射深度剖析

CubeX 在 Test v2.5 中深度整合了划时代的 **DNS 重写中枢 (DNS Rewrite Hub)**，支持两类关键场景的秒级重定向：

```
                    ┌──────────────────────────────────────────────┐
                    │          CubeX DNS 重写中枢 (双核驱动)        │
                    └──────────────────────┬───────────────────────┘
                                           │
                ┌──────────────────────────┴──────────────────────────┐
                ▼                                                     ▼
    【模式 A：CNAME 域名别名重定向】                     【模式 B：A 记录静态 IP 映射】
 ┌──────────────────────────────────────┐          ┌──────────────────────────────────────┐
 │ services.googleapis.cn               │          │ nas.lan   ───────► 10.0.0.100        │
 │            │                         │          │ router.box ──────► 10.0.0.1          │
 │            ▼ (重写为)                │          │ pve.home  ───────► 10.0.0.2          │
 │ services.googleapis.com (海外加速域)  │          ├──────────────────────────────────────┤
 ├──────────────────────────────────────┤          │ 价值：全屋免配 hosts 静态局域网访问  │
 │ 价值：彻底根除 Google Play 0% 死锁  │          │ 双核直出，局域网极速响应 <1ms       │
 └──────────────────────────────────────┘          └──────────────────────────────────────┘
```

### 1. CNAME 域名别名重定向（彻底解决 Google Play 0% 卡死）
* **痛点根源**：在国内网络环境中，Android 设备使用 Google Play 商店下载或更新 App 时，设备默认会查询 `services.googleapis.cn`。由于该域名解析出的国内 IP 无法直连 Google 官方下载 CDN，导致下载进度永久卡死在 0%！
* **CubeX 解决之道**：DNS 重写中枢将 `services.googleapis.cn` 抢先重写为 `services.googleapis.com`。随后请求直接命中海外规则分流，经由高速加密代理节点直通全球 CDN，下载速度瞬间拉满！
* **双核底层实现**：
  * **Mihomo 引擎**：自动注入顶层 `hosts` 映射表，`services.googleapis.cn: services.googleapis.com` 原生 CNAME 递归解析；
  * **sing-box v1.14+ 引擎**：在 `dns.rules` 顶层注入 RFC 1035 标准 `action: "predefined"`，返回 `services.googleapis.cn. 300 IN CNAME services.googleapis.com.`，毫秒级响应。

### 2. A 记录静态 IP 映射（全屋局域网设备免 hosts 静态域名解析）
* **应用场景**：在家庭或办公网络中，私有服务如 NAS、PVE、主路由 Web 管理端通常采用固定内网 IP。
* **CubeX 解决之道**：用户可直接在 Web 面板添加 `nas.lan -> 10.0.0.100` 或 `pve.home -> 10.0.0.2`。无论手机、平板、电视或 PC，只要连接 CubeX 网关，即可通过自定义域名直接访问，免除在每台设备上繁琐修改 `/etc/hosts` 或 Windows hosts 文件的痛苦！

---

<a id="deployment-guides"></a>
## 🚀 极速部署指南 (RouterOS / 爱快 iKuai / 飞牛 fnOS / PVE / Linux)

CubeX 严格遵循专有闭源 EULA 合规准则与脱敏规范，发布包内**绝不包含任何作者私钥、生产 Token 或个人订阅**。

---

<a id="deploy-ros"></a>
### 方式 A：MikroTik RouterOS v7 容器一键极速安装指南 (官方强烈推荐 · 工业级共生)

> [!TIP]
> **👑 为什么强烈推荐将 CubeX 部署在 RouterOS 内部？**  
> 1. **零额外硬件开销**：告别额外购买小主机或在 ESXi/PVE 中运行 4GB 内存重型 OpenWrt 虚拟机的电费与资源浪费，实测 CubeX 专属常驻物理内存仅 **~35MB（基础冷启）/ ~109MB（百级节点全量实战）**！  
> 2. **FastTrack 物理芯片线速狂飙**：「主网关 + 旁路 DNS」黄金解耦架构，全家国内千兆/万兆流量走硬件芯片直通，主路由 CPU 占用仅 1%；  
> 3. **全自动秒级自愈与逃生**：配备 cgroup 256M 内存墙、`restart-policy=on-failure` 内核自愈及 `check-gateway=ping` 毫秒级逃生探针，容器意外升级或重启，**局域网物理网络 0 损伤 0 抖动**！

---

#### 📋 一、 兼容硬件与运行环境要求
* **RouterOS 固件版本**：RouterOS v7.12 及以上（强烈推荐 **v7.14+ / v7.16+** 正式版）；
* **CPU 架构全面兼容**：
  * **x86_64**：PVE / ESXi / Hyper-V / UNRAID 下的 RouterOS CHR 虚拟机，或工控软路由（如 J4125 / N100 / N5105 / 8505 等）；
  * **ARM64**：MikroTik 官方硬件旗舰机型（如 **RB5009**、CCR2004、CCR2116 等）；
  * **ARM32**：如部分 hEX 机型（需外插存储扩展）；
* **存储要求**：
  * **RB5009 等物理路由**：因机身自带闪存较小（约 1GB 且系统占用部分），**强烈建议插入一个高速外置 U 盘或移动 SSD**（RouterOS 会自动识别挂载为 `usb1` 或 `disk1`）；
  * **x86 / CHR 虚拟机**：在虚拟机设置中分配 2GB~4GB 以上虚拟磁盘即可。

---

#### 🚀 二、 极速一键离线安装（国内网络强烈推荐 · 0 失败率）

这是官方推荐的最快、最稳妥安装方式，完全免除在线拉取 Docker Hub 镜像受国内网络波动的困扰：

##### 步骤 1：获取发布资产
前往 [GitHub Releases Test v2.5](https://github.com/zyeyefyass/CubeX/releases) 页面，下载两份文件：
1. **全自动配置脚本**：[`setup-routeros-v7.rsc`](https://github.com/zyeyefyass/CubeX/releases/download/Test-v2.5/setup-routeros-v7.rsc)；
2. **对应架构的离线容器镜像包**：
   * **x86_64 / CHR 虚拟机**：下载 `CubeX-Test-v2.5-ros-amd64.tar`；
   * **RB5009 / CCR2004 (ARM64)**：下载 `CubeX-Test-v2.5-ros-arm64.tar`。

##### 步骤 2：鼠标拖拽上传至 WinBox
1. 打开 WinBox，输入主路由 IP 与管理员账号密码登录；
2. 点击 WinBox 左侧菜单 **`Files`** 弹出文件管理器窗口；
3. 将下载的 **`.tar` 镜像包** 与 **`setup-routeros-v7.rsc`** 脚本文件，**直接拖入 Files 窗口的根目录**（*注意：请勿解压 `.tar` 文件！*）。

##### 步骤 3：在 Terminal 终端一行命令执行导入
1. 点击 WinBox 左侧菜单 **`Terminal`**（终端）；
2. 粘贴并回车执行以下命令：
   ```routeros
   /import setup-routeros-v7.rsc
   ```

##### 步骤 4：自动化流水线执行与输出确认
脚本会自动执行以下五大步骤并在终端打印青色极光进度日志：
```text
🌌 开始执行 CubeX 宿主共生零感网络系统全自动初始化安装流程...
✅ [1/5 硬件架构识别] 识别到当前平台: x86_64 ➔ 匹配容器与包架构: amd64
✅ [2/5 容器检测] RouterOS 容器功能已就绪！
✅ [3/5 网桥探测] 成功锁定局域网主网桥: bridge
✅ [3/5 网段与IP分配] 主路由IP: 10.0.0.1 | 局域网段: 10.0.0.0/24 | 分配 CubeX 独立IP: 10.0.0.11
✅ [4/5 VETH虚拟通道] 已创建 veth-cubex ➔ 10.0.0.11/24
✅ [4/5 网桥挂载] 已将 veth-cubex 挂载至主网桥 bridge
✅ [5/5 策略路由] 已建立 table-cubex-8228 隔离策略路由表
✅ [5/5 默认下一跳] 已下发默认策略路由指向 10.0.0.11 (支持 ping 毫秒自愈逃生)
✅ [5/5 Mangle防火墙] 已配置防死锁豁免与内网直连放行
📦 [CubeX] 发现本地离线包: CubeX-Test-v2.5-ros-amd64.tar，正在快速部署...
⏳ 容器部署指令已下发，正在等待 RouterOS 解压并自动启动 (约 5~10 秒)...
   ...正在解压部署中 (2s)
   ...正在解压部署中 (4s)
🎉 恭喜！CubeX 容器已成功解压并处于 RUNNING 极速运行状态！
=========================================================================
🎉 恭喜！CubeX 宿主共生零感网络系统全自动初始化已全部圆满完成！
👉 控制台 Web 驾驶舱访问地址: http://10.0.0.11:2026 (根据您的真实网段自动生成)
=========================================================================
`

> [!TIP]
> **💡 关于网段与 IP 自动分配机制说明**：  
> 上方终端日志中的 10.0.0.1 与 10.0.0.11 仅为**测试环境实机日志示例**。  
> 安装脚本完全是**根据您 RouterOS 当前主网桥（Bridge）的真实网段动态提取计算**的，绝对没有写死：  
> * 若您的主路由是 MikroTik 默认的 192.168.88.1，脚本会自动计算并分配为 192.168.88.11；  
> * 若您的主路由是常见的 192.168.1.1 或 192.168.31.1，则自动对应分配为 192.168.1.11 或 192.168.31.11；  
> * 若您的主路由自身恰好占用了 .11，脚本会自动检测并顺延分配为 .228，全自动适配，100% 杜绝 IP 冲突！`

---

#### 🌐 三、 终端一行命令在线拉取安装（免下载离线包）

##### 场景 A：网络环境通畅（已具备外网/科学环境）
如果您的主路由已经具备海外直连或临时网络环境，可直接在 WinBox Terminal 终端粘贴官方源一键拉取并安装：
```routeros
/tool fetch url="https://raw.githubusercontent.com/zyeyefyass/CubeX/main/scripts/ros/setup-routeros-v7.rsc" dst-path="setup-routeros-v7.rsc"; /import setup-routeros-v7.rsc
```

---

##### 场景 B：纯国内网络环境（无科学上网、无代理）—— 终极 3 招免翻部署指南！

在纯国内直连网络下，通常面临两大痛点：
1. `raw.githubusercontent.com` 无法直接连接（被 DNS 污染或阻断，报 connection timeout）；
2. 官方 Docker Hub（`registry-1.docker.io`）无法直接拉取镜像（国内主流加速站失效或限流，报 failed to download image）。

针对无代理网络环境，推荐以下 3 种极简解决路径：

* **💡 解法 1（最推荐）：走「二、极速一键离线安装」—— 100% 零外网依赖！**
  * **为什么最推荐**：在电脑或手机上（无论通过手机流量、浏览器下载、迅雷、夸克/百度网盘等方式），只要把 Release 资产包里的 `CubeX-Test-v2.5-ros-amd64.tar`（或 arm64）和 `setup-routeros-v7.rsc` 下载到本地并拖入 WinBox Files，终端执行 `/import setup-routeros-v7.rsc`；
  * **零联网消耗**：脚本已做智能泛匹配，会自动优先探测本地 `.tar` 离线包并完成解压拉起，**全程完全不需要访问任何外网，30 秒 100% 必成！**

* **⚡ 解法 2：使用国内 GitHub 免翻加速直链 + 国内可用 Docker 镜像源**
  1. **国内高速拉取脚本**：
     通过国内稳定加速镜像源（如 `ghfast.top` / `ghproxy.net`）拉取脚本：
     ```routeros
     /tool fetch url="https://ghfast.top/https://raw.githubusercontent.com/zyeyefyass/CubeX/main/scripts/ros/setup-routeros-v7.rsc" dst-path="setup-routeros-v7.rsc"; /import setup-routeros-v7.rsc
     ```
  2. **切换国内可用 Docker Registry 镜像加速站**：
     如果未拖入离线包，RouterOS 在拉取镜像时需要访问镜像站。可先在 WinBox 终端将镜像源指定为国内可用源（例如 DaoCloud 镜像加速站）：
     ```routeros
     /container/config/set registry-url="https://docker.m.daocloud.io"
     ```
     然后再执行导入脚本，即可无翻墙高速拉取官方镜像！

* **💻 解法 3：借助局域网内任意一台电脑/手机的临时代理**
  如果局域网内有某台电脑或手机临时开启了代理软件并勾选了“允许局域网连接”（假设其 IP 为 `192.168.88.50`，端口为 `7890`）：
  WinBox Terminal 终端拉取时只需指定 `http-proxy` 参数：
  ```routeros
  /tool fetch url="https://raw.githubusercontent.com/zyeyefyass/CubeX/main/scripts/ros/setup-routeros-v7.rsc" dst-path="setup-routeros-v7.rsc" http-proxy=192.168.88.50:7890; /import setup-routeros-v7.rsc
  ```
  这样 RouterOS 就会借助这台电脑的临时通道拉取文件，CubeX 容器一旦拉起并配置好节点，全家便拥有了永久原生的智能分流，电脑上的临时代理即可功成身退！

---

#### 🛡️ 四、 首次安装的特殊场景与授权处理 (FAQ)

##### 1. 提示未检测到 container 扩展包怎么办？
* **全自动下载机制**：脚本会全自动探测您当前的 RouterOS 精确版本（如 7.16.1），并自动从 MikroTik 官网拉取匹配架构的 `container-7.16.1-x86.npk` 保存到 Files 根目录；
* **生效操作**：终端输入 `/system reboot` 回车软重启主路由；重启完成后再次在终端执行 `/import setup-routeros-v7.rsc` 即可无缝继续！

##### 2. 提示 Device Mode 授权确认（RouterOS v7.4+ 安全机制）怎么办？
RouterOS 为了设备安全，首次开启容器功能时要求进行物理所有权确认：
* **物理路由器（RB5009 / CCR2004 等）**：在执行完脚本 5 分钟内，长按机身上的 **【Reset / Mode】按键** 直至指示灯闪烁松开，或直接**拔掉电源插头再插上**（硬件冷重启）；
* **虚拟机宿主（CHR / PVE / ESXi / Hyper-V）**：请前往 PVE/ESXi 宿主管理后台，对 RouterOS CHR 虚拟机执行 **【关机/停止 (Stop)】，然后再【开机/启动 (Start)】**！（*重要提示：在 RouterOS 终端执行 `/system reboot` 属于系统软重启，无法触发底座硬件确认*）；
* 宿主冷重启开机后，重新执行 `/import setup-routeros-v7.rsc` 即可！

---

#### 🎯 五、 进入 Web 驾驶舱体验与健康巡检

1. **访问控制台**：在局域网内任意手机、平板或 PC 浏览器打开：`http://<CubeX_IP>:2026`（例如 `http://10.0.0.11:2026`，请以安装脚本终端输出的 IP 为准）；
2. **首次配置向导**：根据 Cyber Aurora 全息向导填入机场订阅或私有节点，一键生成分流策略并启动透明代理；
3. **一键巡检与加固**：控制台顶部常驻 **【一键深度巡检 RouterOS 策略路由及全量重配 (含 Netwatch 0秒逃生)】** 按钮，点击即可对 FastTrack 白名单、Mangle 防死锁规则、VETH 链路进行全自动自愈与体检！

---

<a id="deploy-ikuai"></a>
### 方式 B：爱快 iKuai 64位 主路由 Docker 原生轻量部署 (保姆级教程)

> [!NOTE]
> **适用环境**：iKuai OS 3.6.0 及以上 64 位版本（x86_64 企业版/免费版均可）。  
> **前置条件**：已在【系统设置】➔【磁盘管理】➔【磁盘分区】格式化并挂载磁盘分区（绑定为普通存储分区，例如挂载路径为 `/docker` 或 `/boot`）。

爱快是优秀的国产企业级软路由系统，以强悍的多 WAN 叠加、DPI 七层流控与连接监控著称。在爱快 64 位系统上，官方提供了原生的 Docker 容器扩展环境。**将 CubeX 直接作为轻量微服务部署在爱快原生 Docker 中，是爱快用户运行代理网关的最高效、最稳定形态！**

---

#### 步骤 1：在爱快后台开启 Docker 插件
1. 登录爱快 Web 管理后台，进入【高级应用】➔【插件营销】➔【Docker】；
2. 勾选 **【启用】Docker 服务**；
3. 【存储库分区】下拉选择已挂载的磁盘存储分区（如 `disk1` / `/docker`），点击【保存】；
4. 等待 Docker 服务状态显示为绿色的 **“已运行”**。

---

#### 步骤 2：创建独立 IP (Macvlan) 网络接口
1. 进入【Docker】➔【网络接口】➔ 点击右上角【添加】；
2. 填写网络参数：
   - **接口名称**：`cubex_macvlan`
   - **接口类型**：选择 **`macvlan`**
   - **物理网卡**：选择内网 LAN 口绑定对应的物理网卡（如 `lan1` 或 `eth0`）
   - **IPv4 地址段**：填入局域网网段（例如 `192.168.1.0/24`）
   - **IPv4 网关**：填入爱快自身 LAN IP（例如 `192.168.1.1`）
   - **IPv4 子网掩码**：`255.255.255.0`
3. 点击【保存】完成 Macvlan 独立网络创建。

---

#### 步骤 3：导入或拉取 CubeX 容器镜像
提供以下两种方式：
* **方式 1：离线镜像快速导入（国内网络推荐，免连外网 Docker Hub，0 失败率）**
  1. 在 [GitHub Releases 最新发布页](https://github.com/zyeyefyass/CubeX/releases/latest) 下载 `CubeX-docker-amd64.tar.gz`；
  2. 在爱快【Docker】➔【镜像管理】➔ 点击【导入镜像】；
  3. 上传下载的 `.tar.gz` 文件，系统将在数秒内导入成功并标记为 `fyasshole/cubex:latest`。
* **方式 2：在线搜索下载**
  1. 在爱快【Docker】➔【镜像管理】➔【镜像搜索】；
  2. 搜索 `fyasshole/cubex`，点击【下载镜像】，标签选择 `latest`。

---

#### 步骤 4：创建并启动 CubeX 容器
1. 进入【Docker】➔【容器列表】➔ 点击右上角【添加】；
2. 基础配置：
   - **容器名称**：`cubex`
   - **镜像**：选择 `fyasshole/cubex:latest`
   - **网络类型**：选择刚刚创建的 **`cubex_macvlan`**
   - **IPv4 地址**：填入分配给 CubeX 的局域网专属独立静态 IP（例如 **`192.168.1.200`**，注意请避开爱快 DHCP 动态分配地址池）
   - **开机自启**：勾选【是】
   - **特权模式 (Privileged)**：**必须勾选【是】**（以赋予 TUN 虚拟网卡与网络栈调优能力）
3. 挂载目录 (数据持久化)：
   - 点击【添加】挂载卷：
     - 宿主目录：`/docker/cubex/data` ➔ 容器目录：`/app/data`
     - 宿主目录：`/docker/cubex/etc` ➔ 容器目录：`/app/etc`
4. 环境变量 (ENV)：
   - `TZ` = `Asia/Shanghai`
   - `CUBEX_PORT` = `2026`
   - `CUBEX_LISTEN` = `:2026`
5. 点击【保存并启动】。容器将在 1 秒内平稳拉起，状态显示为绿色“运行中”。

---

#### 步骤 5：爱快主路由后台黄金解耦配置（两步搞定，全网封神）

此时 CubeX 已经在局域网中以独立设备（`192.168.1.200`）就绪。只需在爱快后台配置两项，即可达成「国内全流量爱快物理线速直通 + 出海流量毫秒级自动由 CubeX 接管」的完美体验！

##### ① 配置 Fake-IP 静态路由（出海流量定向引流）
1. 进入【网络设置】➔【静态路由】➔ 点击【添加】；
2. 填写参数：
   - **目的地址**：`198.18.0.0`
   - **子网掩码**：`255.254.0.0`（即 `/15` 专用保留网段；*注：若您使用的爱快版本校验较严提示掩码不合法，可添加两条 `/16`：一条目的地址 `198.18.0.0` 掩码 `255.255.0.0`，另一条目的地址 `198.19.0.0` 掩码 `255.255.0.0`*）
   - **网关 (下一跳)**：填入 CubeX 容器 IP（例如 **`192.168.1.200`**）
   - **接口**：选择 `LAN`（或内网网卡 `lan1`）
   - **备注**：`CubeX-FakeIP-Bypass`
3. 保存生效。

##### ② 配置 DHCP 下发旁路 DNS（国内流量物理线速直通）
1. 进入【网络设置】➔【DHCP服务端】➔【DHCP设置】；
2. 找到局域网对应的 DHCP 规则（通常为 `192.168.1.0/24`），点击【编辑】：
   - **网关 (Gateway)**：**绝对保持原样 `192.168.1.1`（爱快自身 IP，切勿改成 CubeX！）**
   - **首选 DNS**：填入 **`192.168.1.200`**（CubeX 独立 IP）
   - **备选 DNS**：建议填入 `192.168.1.1` 或 `223.5.5.5`（作为安全兜底）
3. 点击【保存】。局域网终端重新断开重连 Wi-Fi 或插拔网线后立即生效！

---

#### 步骤 6：验证访问与极光控制台 (爱快宿主共生深度联动)

<div align="center">
  <img src="docs/assets/screenshots/cubex-live-host-symbiosis.png" alt="CubeX 爱快与 RouterOS 多宿主共生全自动巡检" width="95%" />
</div>

* 打开浏览器访问：`http://192.168.1.200:2026`，即可进入 Cyber Aurora 全息控制台导入订阅；
* 进入【宿主共生】设置，填入爱快管理后台账号密码并点击【保存并下发联动】；
* CubeX 将通过自研纯 Go 原生 Web-RPC 驱动，自动对齐爱快当前活跃网卡与回指路由，并在控制台实时呈现静态路由与 DHCP 联动健康度！
* 此时局域网内任意手机或 PC 访问海外网站即刻丝滑秒开，国内所有大文件下载与测速全部由爱快硬件加速满血狂飙！

---

<a id="deploy-fnos-linux"></a>
### 方式 C：飞牛 fnOS 与通用 Linux Docker 部署

CubeX 容器镜像原生支持飞牛 fnOS、群晖 Synology、QNAP 威联通及各类通用 Linux（Ubuntu/Debian/Alpine/CentOS）底座。**默认持久化存储路径严格对齐飞牛官方规范 `/vol1/1000/cubex`，通用 Linux 默认推荐 `/etc/cubex`**。

---

#### 🌟【强烈推荐 · 最佳实践】途径 ①：独立 IP (Macvlan) 模式部署

> [!TIP]
> **为什么独立 IP 模式是最佳体验？**
> 1. **彻底杜绝端口冲突**：CubeX 独占专属 IP（如 `10.0.0.11`），坐拥 65535 个独立端口，53 端口（DNS）绝不与飞牛自身系统服务或其它 Docker 容器（AdGuard Home 等）争抢；
> 2. **拓扑逻辑极简解耦**：主路由设备列表中清晰呈现一台名为 `CubeX (10.0.0.11)` 的独立虚拟网关，概念不混淆；
> 3. **主路由网关黄金共生**：终端网关填 `10.0.0.1`（国内流量物理线速直通），终端 DNS 填 `10.0.0.11`。飞牛 NAS 关机维护或重启，**全家国内上网、微信、刷抖音、智能家居 100% 不受影响，绝不断网**！

##### 步骤 1：在飞牛终端创建一次 Macvlan 网络
在飞牛终端通过 `ip a` 查看主物理网卡名称（通常为 `eth0` 或 `enp3s0`），执行一次创建（以局域网 `10.0.0.0/24`、主路由 `10.0.0.1` 为例）：
```bash
docker network create -d macvlan \
  --subnet=10.0.0.0/24 \
  --gateway=10.0.0.1 \
  -o parent=eth0 \
  cubex_macvlan
```

##### 步骤 2：启动独立 IP 容器（提供 Compose 与 Run 两种方式）

**方式 A：Docker Compose 编排（最推荐，在飞牛【Docker】->【项目】新建即可）**：
```yaml
version: '3.8'

services:
  cubex:
    image: fyasshole/cubex:latest  # 亦可使用 ghcr.io/zyeyefyass/cubex:latest
    container_name: cubex
    restart: unless-stopped
    cap_add:
      - NET_ADMIN
    devices:
      - /dev/net/tun:/dev/net/tun
    networks:
      cubex_macvlan:
        ipv4_address: 10.0.0.11          # 分配给 CubeX 的专属局域网独立 IP
    volumes:
      - /vol1/1000/cubex/data:/app/data # 飞牛持久化配置目录 (通用Linux请改为 /etc/cubex/data)
      - /vol1/1000/cubex/etc:/app/etc   # 核心配置文件目录 (通用Linux请改为 /etc/cubex/etc)
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
      - CUBEX_PORT=2026
      - CUBEX_LISTEN=:2026

networks:
  cubex_macvlan:
    external: true
```

**方式 B：Docker Run 单行指令启动**：
```bash
docker run -d \
  --name cubex \
  --restart unless-stopped \
  --network cubex_macvlan \
  --ip 10.0.0.11 \
  --cap-add=NET_ADMIN \
  --device /dev/net/tun:/dev/net/tun \
  -v /vol1/1000/cubex/data:/app/data \
  -v /vol1/1000/cubex/etc:/app/etc \
  -v /etc/localtime:/etc/localtime:ro \
  -e TZ=Asia/Shanghai \
  -e CUBEX_PORT=2026 \
  -e CUBEX_LISTEN=:2026 \
  fyasshole/cubex:latest
```

##### 步骤 3：网络配置与两套接入模式全景对照表

无论您是小白还是网络极客，请按照以下两种模式选择最适合您家庭网络的接入方式（以局域网 `10.0.0.0/24`、主路由 `10.0.0.1`、飞牛 NAS `10.0.0.2`、CubeX 独立 IP `10.0.0.11` 为例）：

---

#### 🌟 方案一【黄金解耦模式：主路由网关 + 旁路 DNS】（官方强烈推荐）
> **体验特征**：国内全流量物理线速直出（主路由 CPU 1%），CubeX 仅负责出海与 DNS。飞牛更新重启或维护，全家微信、刷抖音、智能家居 **100% 正常上网，绝不断网！**

| 网络角色 / 实体 | IP 地址 | 子网掩码 | 默认网关 (Gateway) | 首选 DNS | 备用 DNS (可选) | 配置要点说明 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **主路由器**<br>(小米/TP-Link/ROS等) | `10.0.0.1` | `255.255.255.0` | 运营商光猫 IP | 运营商/公共 DNS<br>(`223.5.5.5`) | `119.29.29.29` | **关键必做**：在主路由后台添加一条静态路由：`198.18.0.0/15 -> 下一跳指向 10.0.0.11`。<br>可在主路由 DHCP 选项中将下发 DNS 设为 `10.0.0.11`，全家全自动生效！ |
| **飞牛 fnOS 宿主机**<br>(或 PVE / Linux) | `10.0.0.2` | `255.255.255.0` | **`10.0.0.1`**<br>(主路由) | **`10.0.0.1`** 或 `223.5.5.5` | `119.29.29.29` | **严禁填 CubeX IP！** 飞牛自身直通主路由上网，彻底杜绝开机启动依赖死锁与内网应用断网。 |
| **CubeX 容器**<br>(Macvlan 独立 IP) | `10.0.0.11` | `255.255.255.0` | **`10.0.0.1`**<br>(主路由) | **`10.0.0.1`** 或 `223.5.5.5` | `119.29.29.29` | 容器自身通过主路由直连公网更新订阅与节点。 |
| **局域网各终端设备**<br>(手机/电脑/平板/电视) | `10.0.0.x`<br>(DHCP自动) | `255.255.255.0` | **`10.0.0.1`**<br>(主路由) | **`10.0.0.11`**<br>(CubeX 独立 IP) | `10.0.0.1`<br>(主路由) | **黄金核心**：国内流量由主路由硬件加速直通光猫；出海流量被 Fake-IP 解析，送达主路由后自动转给 CubeX 代理出海。 |

##### 💡 常见家用主路由添加「静态路由」速查指引：
在主路由后台找到【静态路由 / 内部路由】功能，新增一条并保存：
- **目的地址 (Destination)**：`198.18.0.0`
- **子网掩码 (Subnet Mask)**：`255.254.0.0`（或 CIDR `/15`；若路由器仅支持 `/16`，可添加两条：`198.18.0.0/16` 与 `198.19.0.0/16`）
- **网关 / 下一跳 (Gateway / Next Hop)**：填 CubeX 独立 IP（例如 `10.0.0.11`；若为 Host 模式则填飞牛 IP `10.0.0.2`）
- **出接口 (Interface)**：局域网（LAN / br-lan）

常见后台路径：
- **小米 / Redmi 路由器**：高级设置 ➔ 路由设置 ➔ 静态路由 ➔ 添加；
- **TP-Link 路由器**：高级设置 ➔ 路由设置 ➔ 静态路由表 ➔ 新增；
- **华硕 ASUS 路由器**：内部网络 (LAN) ➔ 路由设置 ➔ 启用静态路由 ➔ 填入并应用；
- **OpenWrt 路由器**：网络 ➔ 静态路由 ➔ 添加一条针对接口 `LAN` 的路由规则；
- **MikroTik RouterOS**：**无需手动配置**，进入 CubeX 控制台点击【一键配置】全自动搞定！

---

#### ⚡ 方案二【开箱免配模式：传统旁路由网关模式】
> **适用场景**：主路由为运营商赠送的光猫路由一体机或老旧路由器，后台不支持添加静态路由；或者不想改动主路由任何设置。

| 网络角色 / 实体 | IP 地址 | 子网掩码 | 默认网关 (Gateway) | 首选 DNS | 备用 DNS (可选) | 配置要点说明 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| **主路由器** | `10.0.0.1` | `255.255.255.0` | 运营商光猫 IP | 原厂默认设置 | - | **保持原厂默认，无需做任何改动**。 |
| **飞牛 fnOS 宿主机** | `10.0.0.2` | `255.255.255.0` | **`10.0.0.1`**<br>(主路由) | **`10.0.0.1`** | - | 宿主机自身网关保持指向主路由。 |
| **CubeX 容器** | `10.0.0.11` | `255.255.255.0` | **`10.0.0.1`**<br>(主路由) | **`10.0.0.1`** | - | 容器网关指向主路由。 |
| **需要加速的终端**<br>(如开发电脑/游戏机) | `10.0.0.x`<br>(静态指定) | `255.255.255.0` | **`10.0.0.11`**<br>(CubeX 独立 IP) | **`10.0.0.11`**<br>(CubeX 独立 IP) | `10.0.0.1`<br>(主路由) | 终端所有网络数据包直接先送到 CubeX 容器，国内流量由 CubeX 转发回主路由。 |

---

#### 💡 附：若采用途径 ② Host 共享网络模式（CubeX 共享飞牛 IP `10.0.0.2`），各参数怎么填？
- **飞牛 NAS 自身**：网关保持 `10.0.0.1`，DNS 保持 `10.0.0.1` 或公共 DNS（不变）；
- **方案一（主路由网关+旁路DNS）**：主路由静态路由写 `198.18.0.0/15 -> 10.0.0.2`（下一跳改为飞牛 IP）；终端设备网关填 `10.0.0.1`，DNS 填 `10.0.0.2`；
- **方案二（传统旁路由网关）**：终端设备网关填 `10.0.0.2`，DNS 填 `10.0.0.2`。

---

#### ⚡【免折腾极速版】途径 ②：Host 共享网络模式（单行拉起）

适合不想配置 Macvlan、且飞牛 53 端口没有被其他容器占用的用户：

```bash
# 飞牛 fnOS 专属单行指令 (共享飞牛 IP: 10.0.0.2)
docker run -d \
  --name cubex \
  --restart unless-stopped \
  --net=host \
  --cap-add=NET_ADMIN \
  --device /dev/net/tun:/dev/net/tun \
  -v /vol1/1000/cubex/data:/app/data \
  -v /vol1/1000/cubex/etc:/app/etc \
  -v /etc/localtime:/etc/localtime:ro \
  -e TZ=Asia/Shanghai \
  -e CUBEX_PORT=2026 \
  -e CUBEX_LISTEN=:2026 \
  fyasshole/cubex:latest
```
浏览器访问：`http://<飞牛IP>:2026`。

---

#### 📦 途径 ③：飞牛 Docker GUI【离线镜像导入】(国内免翻墙 · 0 超时风险)
1. 在 [Releases 最新发布页](https://github.com/zyeyefyass/CubeX/releases/latest) 下载最新版 `CubeX-docker-amd64.tar.gz` 离线镜像；
2. 飞牛桌面 ->【Docker】应用 ->【镜像仓库】-> 点击右上角【本地导入】上传 `.tar.gz`；
3. 【本地镜像】点击【创建容器】：
   - **网络**：可直接选 **`cubex_macvlan`** 指定独立 IP，或选 **`Host`**；
   - **环境权限**：勾选 **【高权限执行】(Privileged 或 NET_ADMIN)**；
   - **路径映射**：本地 `/vol1/1000/cubex/data` 映射容器内 `/app/data`；
   - **启动**：容器 2 秒内极速就绪！

---

<a id="deploy-pve"></a>
### 方式 D：Proxmox VE (PVE) CT 容器 (LXC) 极速部署

相比笨重的 KVM 完整虚拟机（动辄占用 2G~4G 内存与大量 CPU 损耗），在 PVE 中采用 **LXC 轻量级 CT 容器** 是极客运行 CubeX 的终极优雅姿态——**基础常驻仅 ~25MB~35MB，百级节点全量生产仅约百兆，零虚拟化损耗**！

#### 1. 创建 PVE LXC 容器
* 模板：Debian 12 或 Alpine Linux 均可；
* 资源分配：内存 256MB ~ 512MB，CPU 1 核，磁盘 4GB 足够；
* 网络：网桥绑定 `vmbr0`，IP 设置为静态 IP（例如 `10.0.0.11/24`），网关填主路由（`10.0.0.1`）。

#### 2. 在 PVE 宿主机开启 LXC TUN 隧道权限 (关键必做)
在 PVE 宿主机终端执行（假设新建的 CT 容器 VMID 为 `105`）：
```bash
# 允许该容器访问 /dev/net/tun 虚拟网卡设备
cat << 'EOF' >> /etc/pve/lxc/105.conf
lxc.cgroup2.devices.allow: c 10:200 rwm
lxc.mount.entry: /dev/net/tun dev/net/tun none bind,create=file
EOF
```

#### 3. 启动 CT 容器并一键部署
进入该 CT 容器控制台，直接执行：
```bash
# 进入容器工作目录
mkdir -p /opt/cubex && cd /opt/cubex

# 下载解压最新 Linux 资产包
curl -sSL -O https://github.com/zyeyefyass/CubeX/releases/latest/download/CubeX-linux-amd64.zip
unzip CubeX-linux-amd64.zip

# 运行一键安装部署服务
sudo bash install.sh
```
部署完毕后，`cubexd` 将作为 systemd 服务开机自启，浏览器直接访问 `http://10.0.0.11:2026` 即可！

---

<a id="deploy-linux-cli"></a>
### 方式 E：通用 Linux 终端一键全自动安装

适用于 Ubuntu / Debian / CentOS / Alpine 等任何标准 Linux 发行版：

```bash
# 官方一键智能嗅探脚本
curl -fsSL https://raw.githubusercontent.com/zyeyefyass/CubeX/main/install.sh | bash
```

脚本将全自动识别宿主环境、安装依赖、注册 `cubex` 运维 CLI 与 `systemd` 守护服务。常用运维指令：
```bash
cubex status    # 查看网关运行状态、内存与双引擎版本
cubex log       # 实时追踪全息诊断日志流
cubex reload    # 0 断流秒级热重载分流策略与规则
cubex restart   # 重新启动守护进程
```

---

<a id="community-feedback"></a>
## 💬 极客畅所欲言：意见征集与需求大厅

> [!IMPORTANT]
> **“独行快，众行远。”**  
> CubeX 从一个单二进制雏形走到今天已发布最新版的双引擎融合共生体，离不开每一位极客玩家的测试与真实反馈。

此版本已经**完全达到了设计初衷与高可用指标**，距离最终正式稳定版（Stable v1.0）仅有一步之遥！我们真诚地期待听到您的声音：

* 💡 **您在实际使用中遇到了哪些奇葩网络拓扑？**（例如：多 WAN 叠加、PPPoE 定时重拨、特定游戏主机 NAT 类型限制？）
* 💡 **在 UI 交互与数据看板上，您还希望看到哪些维度的图表或快捷控制？**
* 💡 **对于 IPv6 双栈、严格防 DNS 泄露、多宿主极速部署等场景，您还有哪些进阶想法？**

👉 **[点击这里直达 GitHub Issues 提交反馈与功能需求](https://github.com/zyeyefyass/CubeX/issues/new?title=%5BFeature+Request%2FFeedback%5D+Test+v2.5+%E4%BD%BF%E7%94%A8%E5%8F%8D%E9%A6%88)**

无论是一句肯定、一个 Bug 提交，还是一个天马行空的新功能脑洞，请在社区**畅所欲言**，与我们一起打造软路由领域最强悍的宿主共生网络系统！

---

<a id="compliance-statement"></a>
## 📜 净室合规与免责声明

1. **纯学术与工程探索**：本项目为网络协议栈、双引擎动态调度与容器伴生路由技术的工程探索与学术交流产物；
2. **合规用途界定**：系统面向正当跨境电商出口、远程协同办公、前沿 AI 学术研究与大模型接入；
3. **法律遵约责任**：严禁将本系统用于任何违反法律法规、侵害国家安全或扰乱社会秩序的违规活动。使用者对其运行环境和操作承担全部法律责任；
4. **零凭证承诺**：项目所有公开发布物均经过 `audit_release_artifacts.py` 自动化反向解包逆向审计，保证 0 机场订阅、0 私有节点、0 敏感凭证。

---

<div align="center">
  <b>CubeX · 让每台设备宛如物理肉身直连全球互联网</b><br>
  <sub>Copyright © 2026 CubeX Team. All rights reserved.</sub>
</div>
