<div align="center">

  <img src="docs/assets/logo-cubex.svg" width="160" height="160" alt="CubeX 3D Cyber Aurora Crystal Logo" />

  # 🌌 CubeX · 极光魔方

  ### 次世代赛博极光 · 双擎高可用旁路网关系统
  **Dual-Engine Cyber Gateway with Zero-Downtime Handoff**

  [![License](https://img.shields.io/badge/License-Proprietary_EULA-00F5D4?style=for-the-badge&logo=shield)](COPYRIGHT.md)
  [![Engine](https://img.shields.io/badge/Dual--Engine-Mihomo_%C3%97_sing--box_v1.14+-7B2CBF?style=for-the-badge&logo=go)](https://github.com/zyeyefyass/CubeX)
  [![RouterOS](https://img.shields.io/badge/Hardware-RouterOS_v7_Native_Sync-0077B6?style=for-the-badge&logo=mikrotik)](#)
  [![Platform](https://img.shields.io/badge/Platform-fnOS_%7C_PVE_LXC_%7C_Docker-3A86FF?style=for-the-badge)](#)
  [![Architecture](https://img.shields.io/badge/Arch-amd64_%7C_arm64-FF007F?style=for-the-badge)](#)

  <p align="center">
    <b>Mihomo 与 sing-box 两大派系的终极答案</b><br>
    一键秒切双擎 · 宿主机原生联动全自动编排 · 硬件级 WireGuard 穿透对齐 · 局域网设备分流矩阵 · 217heidai 权威规则 · 四维在线热更
  </p>

</div>

---

## ⚡ 为什么是 CubeX（极光魔方）？

软路由圈内关于 **Mihomo（精细化策略组流派）** 与 **sing-box（极速轻量现代协议流派）** 的宗教级争论已久。  
**CubeX 的诞生，旨在彻底终结这场内耗——成年人的世界，选择“全都要”！**

CubeX 是由原作者在继 **MiCube** 与 **SingCube** 两大独立项目之后，将底层双核同构融合、历经全面重构推出的跨世代终极旗舰项目。通过独创的 **CubeX IR（统一配置中间表示层）**，一份分流策略即可自动双向转译并驱动双内核，实现毫秒级零断网热切换（Make-Before-Break）与双核同台竞技。

---

## 📸 即将推出的 CubeX 四大核心功能规划与实机实景

### 🧭 规划一：【总览与分流】全域双擎主控驾驶舱

![01-dashboard-dual-engine](docs/assets/screenshots/01-dashboard-dual-engine.png)

* **⚡ 顶栏双核毫秒级热切胶囊 (Dual-Engine Handoff)**：  
  顶栏常驻 `[ ⚡ SingCube (极光青) ]` ⇄ `[ 🛡️ MiCube (矩阵金) ]` 热切开关。依托底层 **Make-Before-Break** 架构，切换内核时**无需重建防火墙、不断开存量长连接**，选中的出口节点状态自动无缝手递手（Handoff），大文件下载不断流、网络游戏不掉线。
* **⚔️ 双核同节点 AB 竞技场 (Benchmark Arena)**：  
  业内首创！同一出口节点同时唤醒 Mihomo 与 sing-box 两套底层协议栈同台狂飙，**TCP 握手延时、TLS 协商开销、内存占用与吞吐效率** 实时同屏对决，让数据告诉谁才是速度真神。
* **🧬 全链路沙盒分流推演 (Packet Tracer)**：  
  拒绝抓瞎排错！在搜索框输入任意测试域名（如 `netflix.com` 或 `bilibili.com`），即刻毫秒级可视化推演：`请求域名 -> DNS 嗅探解析 -> 命中具体规则 -> 归属策略组 -> 最终出口节点`，一目了然。
* **📊 全局健康度矩阵与一秒急救直连 (Panic Button)**：  
  右上方醒目的红色 **「1秒急救直连」** 按钮，遭遇任何配置失误或机场大面积故障时，一键清空转发恢复纯物理直连，断网自愈保障家庭网络底线。

---

### 📡 规划二：【穿透协同】RouterOS 原生 & Linux 宿主双活 WireGuard 与设备分流矩阵

![02-wireguard-matrix](docs/assets/screenshots/02-wireguard-matrix.png)

* **🔷 5 维硬件级对齐巡检报告 (RouterOS v7 Hardware Alignment)**：  
  原生打通 MikroTik REST API，自动探查并实时核验 5 项硬件对齐状态：
  1. `/interface/wireguard` 服务端接口监听状态（默认端口 13231）；
  2. `/ip/address` 穿透虚拟网段（10.88.0.1/24）网关绑定；
  3. `/ip/firewall/filter` 穿透端口外部公网访问自动放行；
  4. RouterOS 官方 **IP Cloud 免费动态域名** 探查；
  5. 移动端设备 Peer 密钥与 /32 隧道 IP 硬件对齐同步。
* **📱 手机 WireGuard 官方客户端一键扫码穿透**：  
  外网手机直接使用官方 WireGuard App 扫码秒连家庭内网，无论身在何地，手机网络自动直通内网 NAS、路由器管理面板与极光代理网关。
* **💻 飞牛 fnOS / Linux 宿主 WireGuard 极速穿透**：  
  一键部署宿主机 `wg0.conf`，利用 Linux 内核级直接跑满千兆带宽，实现 0 内存损耗。
* **📺 局域网设备分流控制矩阵 (Per-Device Matrix)**：  
  家庭网络终端精细化管控神器！无需进各个设备修改网关或代理，面板直观提供三段式胶囊滑块：
  * `🟢 智能规则分流`：适用于手机、PC、平板等日常终端；
  * `🔵 底层直连绕过 (Bypass)`：**专为家庭影院蓝光机、PT 下载机、打印机设计**，彻底绕过代理协议栈，保护 SMB1 局域网超大码率直通与 PT 纯净 IP；
  * `🟣 绑定特定出口 (Bound Node)`：为 PS5 游戏主机或 Apple TV 强行绑定特定低延迟或解锁专线。

---

### 🛡️ 规划三：【规则安全】217heidai 权威规则与严格 0 明文 DNS 泄露

![03-rules-security](docs/assets/screenshots/03-rules-security.png)

* **🛡️ 217heidai 权威去广告与隐私防护**：  
  内置 **233,481 条** 高精度离线 MRS 格式规则包，覆盖全网主流视频去广告、应用启动页净化与隐私追踪阻断。
* **🔒 严格 0 明文 DNS 泄露保障**：  
  杜绝 ISP 劫持与明文泄漏，非加密非受信 DNS 查询强制拦截阻断（SERVFAIL）。采用 Fake-IP 智能剥离技术，国内真实公网 IP 直连直出，兼顾极速访问与隐私安全。
* **📈 实时拦截探测日志看板**：  
  实时统计今日阻断次数与累计阻断总量，毫秒级捕获每个终端发起的恶意追踪请求并记录域名与来源。

---

### ⚙️ 规划四：【宿主联动】统一智能宿主环境原生联动与四维升级中枢

![04-system-host-base](docs/assets/screenshots/04-system-host-base.png)

* **🖥️ 统一智能宿主自适应探针**：  
  全自动嗅探宿主操作系统（**MikroTik RouterOS v7.x、飞牛 fnOS 专属环境、Proxmox VE LXC、标准 Linux Docker**），自动展示系统内核版本、CPU 核心数、实时内存负载与内核包转发状态（`ip_forward=1`）。
* **📜 防火墙与路由脚本一键下发与复制**：  
  * **一键复制 RouterOS WinBox 策略路由脚本**（包含 `table 8228`、Mangle 防死锁规则与 Netwatch 0 秒自动逃生）；
  * **一键复制 Linux / 飞牛宿主防火墙放行脚本**（iptables / nftables 自动配置）。
* **🚀 四维版本在线热更矩阵**：  
  彻底告别命令行折腾！面板支持针对四个层级进行独立在线体检与一键无损热更：
  1. **CubeX 主控守护进程** (`cubexd`)；
  2. **双代理内核**（Mihomo 与 sing-box v1.14+ 原生内核同步在线拉取）；
  3. **Geo 全量数据库与 217heidai MRS 规则包**；
  4. **宿主机底层系统包**（针对 Debian / fnOS 自动更新 wireguard-tools 与内核补丁）。
* **🤖 AI 智能网关运维助手 (Copilot)**：  
  内置本地/云端 LLM 驱动的智能运维助手，输入排障疑问（如“诊断 NAT 类型”、“排查 DNS 泄露”），一键全维度网络体检并生成修复指令。
* **💾 防手残时光机全量备份**：  
  一键打包导出包含双核配置、WireGuard 密钥及分流规则的全量快照，误操作随时导入秒级复原。

---

## ⚔️ 终极架构特性对比

| 特性维度 | 传统单一内核网关 | **CubeX · 极光魔方** |
| :--- | :--- | :--- |
| **底层代理内核** | 锁定单一内核（无法切换或需重配） | **Mihomo × sing-box v1.14+ 双擎无感毫秒级热切** |
| **同节点性能对比** | 无法横向比较，仅凭玄学感觉 | **独创双核同节点 AB 竞技场（TCP/TLS/内存实时对决）** |
| **分流排障手段** | 查看滚屏几千行的英文日志 | **全链路沙盒推演（输入域名即可视化推演命中流向）** |
| **RouterOS 联动** | 需用户自行手动在 WinBox 里写脚本 | **REST API 深度联动 + 5 项硬件对齐巡检 + WinBox 脚本自动生成** |
| **局域网设备控制** | 一刀切全局代理，蓝光机/PT经常翻车 | **Per-Device Matrix（智能分流 / 蓝光机底层直连 / 绑定特定出口）** |
| **DNS 与广告防护** | 简易 AdGuard / 经常出现明文 DNS 泄露 | **217heidai 233,481 条 MRS 规则 + 严格 0 明文 DNS 阻断** |
| **系统升级维护** | 手动敲命令解压升级，容易断网变砖 | **四维版本在线热更矩阵（主控、内核、规则、宿主）一键无损更新** |
| **逃生容错机制** | 崩溃时全家断网，必须手动拔网线 | **1秒急救直连按钮 (Panic) + 防手残时光机全量备份** |

---

## 🚀 极速安装与部署 (Quick Start)

### 1. 通用 Linux / 飞牛 fnOS 一键安装

```bash
curl -fsSL https://raw.githubusercontent.com/zyeyefyass/CubeX/main/scripts/install.sh | bash
```

安装完成后，打开浏览器访问：`http://<设备IP>:2026` 即可进入 Cyber Aurora 赛博极光全息驾驶舱。

### 2. Docker Compose 编排部署

```yaml
services:
  cubex:
    image: ghcr.io/zyeyefyass/cubex:latest
    container_name: cubex
    restart: unless-stopped
    network_mode: host
    cap_add:
      - NET_ADMIN
      - NET_BIND_SERVICE
      - SYS_MODULE
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - ./data:/opt/CubeX/data
      - ./etc:/opt/CubeX/etc
      - /etc/timezone:/etc/timezone:ro
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
      - CUBEX_PORT=2026
```

### 3. MikroTik RouterOS v7 容器部署

请参照 [docs/routeros-setup.md](docs/routeros-setup.md) 获取针对 RB5009、CCR2004、CCR2116 等设备的专用 VETH 容器快速部署模版。

---

## 🔒 知识产权与专有许可声明 (Proprietary Notice)

`CubeX（极光魔方）` 是经由原作者 **zyeyefyass** 针对 Mihomo 与 sing-box 双原生内核深度重构打造的闭源专有高性能旁路网关系统。

* 完整架构源码仅在官方私有仓库 `CubeX-Core` 中受控维护；
* 本公共仓库仅分发经符号剥离（`-ldflags="-s -w" -trimpath`）加固的官方发行版本、容器配置与使用文档；
* 本项目面向个人及家庭用户提供**完全免费、无隐私回传、零强制消费**的高品质使用体验；
* 未经原作者书面许可，严禁对二进制进行逆向工程、反编译、去除署名或用于任何形式的商业转售牟利。详细法律条款参见 [COPYRIGHT.md](COPYRIGHT.md) 与 [DECLARATION.md](DECLARATION.md)。

---

**Copyright © 2026 zyeyefyass. All Rights Reserved.**
