# 🌌 CubeX Test v4.8 旗舰发布公告 · OpenWrt 智能底座大屏 · 50% 纯净水晶穿透 · 极光天青温润重构 · 120 FPS 满帧引擎

> 👑 **次世代双擎宿主共生零感旁路网关系统 (Host-Symbiotic Dual-Engine Gateway)**  
> 核心引擎：`Mihomo v1.19.33` × `sing-box v1.14.4` | 架构：Linux amd64 / arm64 / RouterOS v7 / 爱快 iKuai / 飞牛 fnOS / PVE LXC / OpenWrt

---

### 🌟 Test v4.8 核心亮点更新 (Key Innovations & Fixes)

#### 1. 🌐 OpenWrt 智能底座大屏与全链路探活全面上线
- **UBUS/SSH 探活编排总线**：后端新增 `/api/v1/host/openwrt/inspect` 与 `/api/v1/host/openwrt/apply`，毫秒级读取 OpenWrt 固件版本、Linux 内核版本与网络架构。
- **全息大屏体检与状态诊断**：
  - 动态验证 `firewall4` (nftables) 引擎状态与转发链放行规则；
  - 实时校验 `198.18.0.0/15` 回指策略路由与路由表条目；
  - 校验 Dnsmasq DHCP Option 6 DNS 劫持防护配置；
  - 提供一键全自动体检加固与规则下发。

#### 2. 💎 每一板块 50% 纯净水晶穿透美学
- **彻底告别模糊色块与厚重遮罩**：全面剔除多层暗黑遮罩与 `backdrop-filter: blur(...)` 高斯模糊层。
- **100% 显色穿透**：所有一级主卡片统一重构为 `rgba(0, 18, 36, 0.50)` 顶级纯净暮光冰晶质感，底层疯狂动物城微缩原画 100% 高保真透视显色，光影层次深邃分明，极具科技美感与全息通透度。

#### 3. 🎨 全站刺眼明黄按钮消除 · 极光天青温润护眼重构
- **深度舒适视觉重塑**：全站 250+ 处刺眼明黄按钮（`#FF9E5E` / `#F59E0B` 与黑字 `text-[#1E1106]`）彻底重构为 Twilight 极光天青温润渐变（`from-sky-600 via-sky-500 to-cyan-500`）与冷白高清晰文字（`text-white`）。
- 长时间盯屏运维眼睛极度温润舒适，杜绝视觉疲劳与眩光感。

#### 4. ⚡ 120 FPS 满帧硬件加速引擎
- 全局主卡片引入 `contain: content; content-visibility: auto; transform: translateZ(0)` GPU 硬件加速管线。
- 百兆级节点矩阵渲染、千行策略日志滚屏及监控抽屉展开均保持 120 FPS 满帧丝滑，彻底消除拖拽与长列表卡顿。

#### 5. 🚀 Hysteria2 原生端口跳跃 (Port Hopping 20000-40000) 与双核秒级无感热重载
- 完整保留并加固标准 `mport` / `ports`（20000-40000）端口跳跃能力，彻底粉碎运营商 DPI 单流限速。
- 双引擎（`sing-box` ⇄ `mihomo`）双循环往返 4 轮热重载验证 100% 满分通过，零连接中断、零死锁。

---

### 📦 资产下载与校验清单 (Release Assets)

| 发布物文件名 (Filename) | 适用宿主环境 (Platform/Target) | 文件类型 (Format) | 说明 (Description) |
| :--- | :--- | :--- | :--- |
| `CubeX-Test-v4.8-docker-amd64.tar.gz` | 爱快 iKuai / 飞牛 fnOS / OpenWrt / 通用 Linux | Docker 离线镜像 | 约 94MB，开箱即用，内置双核心与规则体系 |
| `CubeX-Test-v4.8-ros-amd64.tar` | MikroTik RouterOS v7 (x86 软路由) | RouterOS 容器 | 约 117MB 极简 OCI 单层容器包，无感共生 |
| `CubeX-Test-v4.8-ros-arm64.tar` | MikroTik RouterOS v7 (RB5009 / CCR2004 等) | RouterOS 容器 | 约 116MB 极简 OCI ARM64 容器包 |
| `CubeX-Test-v4.8-linux-amd64.zip` | 通用 Linux amd64 (PVE CT / Debian / Ubuntu / Alpine) | 完整部署归档包 | 包含加固主程序、双内核、101 套规则库与 CLI |
| `CubeX-Test-v4.8-linux-arm64.zip` | 通用 Linux arm64 (树莓派 / 晶晨盒子 / ARM 服务器) | 完整部署归档包 | 包含 ARM64 加固主程序、双内核与完整规则库 |
| `cubexd-linux-amd64` | Linux amd64 宿主 / 容器 | 单二进制可执行文件 | 核心守护进程，支持 1 秒极速热升级替换 |
| `cubexd-linux-arm64` | Linux arm64 宿主 / 容器 | 单二进制可执行文件 | ARM64 核心守护进程 |
| `cubex-linux-amd64` | Linux amd64 宿主 / 容器 | 运维 CLI 工具 | 命令行运维控制台 |
| `cubex-linux-arm64` | Linux arm64 宿主 / 容器 | 运维 CLI 工具 | ARM64 命令行运维控制台 |
| `install_pve_lxc.sh` | Proxmox VE (PVE 7.x / 8.x) | 一键自动化脚本 | 宿主机 1 行命令秒级创建 LXC 伴生容器 |
| `install.sh` | 通用 Linux / 飞牛 fnOS / 爱快 | 一键自动化脚本 | 全自动动态嗅探网关与网络环境部署 |
| `setup-routeros-v7.rsc` | MikroTik RouterOS v7 | WinBox 终端脚本 | 一键配置 VETH、策略路由与 Netwatch 0 秒逃生 |
| `setup-openwrt.sh` | OpenWrt 21.02 / 22.03 / 23.05+ | 部署维护脚本 | Macvlan-Shim 离线容器网络一键初始化 |

> 🛡️ **严格安全声明与脱敏审计**：本发布包已通过 `scripts/audit_release_artifacts.py` 严格逆向解包脱敏审计，遵循零订阅、零私有代理节点、零私钥敏感凭证、零违规敏感词、零国家旗帜图红线，全网环境零硬编码自适应。
