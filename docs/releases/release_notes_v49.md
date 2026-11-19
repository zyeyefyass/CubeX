# 🌌 CubeX Test v4.9 旗舰发布公告 · 双核开机自愈总线 · 3ms并发去重缓存 · 策略连接平滑回退 · 生产级稳定性加固

> 👑 **次世代双擎宿主共生零感旁路网关系统 (Host-Symbiotic Dual-Engine Gateway)**  
> 核心引擎：`Mihomo v1.19.33` × `sing-box v1.14.4` | 架构：Linux amd64 / arm64 / RouterOS v7 / 爱快 iKuai / 飞牛 fnOS / PVE LXC / OpenWrt

---

### 🌟 Test v4.9 核心亮点更新 (Key Innovations & Fixes)

#### 1. 🛡️ RouterOS 策略路由与 DNS 劫持开机自愈总线全面上线
- **停机逃生与开机恢复双向闭环**：彻底解决 CubeX 守护进程优雅停机逃生后，RouterOS 上的 `CubeX-DNS-Hijack` (NAT 53) 与 `CubeX-FakeIP-Backtrack` (`198.18.0.0/15`) 规则未能在开机时自动重新激活的隐患；
- **全生命周期自愈保护**：守护进程开机启动序列深度挂载 `RestoreRulesAfterBoot()` 与 `EnsureFakeIPRoute()`，全自动解除 RouterOS 规则的 `disabled` 状态并刷新 DNS 缓存，杜绝任何重启后策略路由丢失问题；
- **固件信息解码加固**：彻底根治 RouterOS `/rest/system/resource` 接口高版本返回数值字符串引发的反序列化假阳性未认证异常。

#### 2. ⚡ sing-box 编译内核直连通道死锁根除
- **消除递归劫持自锁**：修复 `makeSingboxDNSServer` 剥离 `detour: "direct"` 的缺陷，确保 `dns-direct` 严格保留直连出站标记；
- **DNS 双栈极速穿透**：杜绝 sing-box 本地出站 DNS 查询被自身 `port: 53` 路由规则递归捕获的问题，国内真实 IP 直连与 Fake-IP 映射秒级响应，彻底根除高并发下的 DNS 超时黑洞。

#### 3. 🚀 专家组三大性能与高可用特性全面落地
- **特性 1（多策略组并发测速去重与共享缓存）**：
  - 引入 20 秒延迟智能缓存（`delayCache`）搭配 In-Flight 飞行探针单飞合并（`delaySFGroup`）；
  - 相同节点重复探针耗时从 5.01s 骤降至 **3ms**（性能跃升 1600+ 倍），彻底终结多策略组测速风暴。
- **特性 2（Conntrack 僵尸连接定向清算）**：
  - 热重载及策略组选路切换时，定向清算 `198.18.0.0/15` Fake-IP 历史僵尸连接；
  - 强力保护本地 Web 控制台 2026 管理端口，实现秒级生效与零长连接粘连。
- **特性 3（TCP 握手期有边界平滑回退）**：
  - URLTest 策略组全面注入 `interrupt-exist-connections: false` 与 `tolerance: 50`；
  - 健康探测重新选路时，正在传输中的会议、下载与实时通话完全平滑保持，杜绝瞬断踢连。

#### 4. 📦 PVE LXC 伴生容器与 OpenRC 守护进程深度加固
- **一键安装脚本深度调优**：`install_pve_lxc.sh` 全自动配置容器独立 DNS 解析并自动规避 PVE 虚拟网卡防火墙拦截；
- **标准守护架构**：规范化 Alpine Linux OpenRC `supervise-daemon` 配置与 Shebang 校验，支持 OOM 防爆内存软上限（`GOMEMLIMIT 180MiB`）全自动接管守护。

#### 5. 🔄 双核双循环端到端回归验证 100% 满分通过
- 生产环境实测 `Mihomo ⇄ sing-box` 连续往返热重载 4 次，直连与境外业务协同秒级恢复，零连接停顿、零报错。

---

### 📦 资产下载与校验清单 (Release Assets)

| 发布物文件名 (Filename) | 适用宿主环境 (Platform/Target) | 文件类型 (Format) | 说明 (Description) |
| :--- | :--- | :--- | :--- |
| `CubeX-Test-v4.9-docker-amd64.tar.gz` | 爱快 iKuai / 飞牛 fnOS / OpenWrt / 通用 Linux | Docker 离线镜像 | 约 94MB，开箱即用，内置双核心与全量 101 套规则体系 |
| `CubeX-Test-v4.9-ros-amd64.tar` | MikroTik RouterOS v7 (x86 软路由) | RouterOS 容器 | 约 118MB 极简 OCI 单层容器包，宿主共生零侵权 |
| `CubeX-Test-v4.9-ros-arm64.tar` | MikroTik RouterOS v7 (RB5009 / CCR2004 等) | RouterOS 容器 | 约 117MB 极简 OCI ARM64 容器包 |
| `CubeX-Test-v4.9-linux-amd64.zip` | 通用 Linux amd64 (PVE CT / Debian / Ubuntu / Alpine) | 完整部署归档包 | 包含加固主程序、双内核、101 套规则库与 CLI |
| `CubeX-Test-v4.9-linux-arm64.zip` | 通用 Linux arm64 (树莓派 / 晶晨盒子 / ARM 服务器) | 完整部署归档包 | 包含 ARM64 加固主程序、双内核与完整规则库 |
| `cubexd-linux-amd64` | Linux amd64 宿主 / 容器 | 单二进制可执行文件 | 核心守护进程，支持 1 秒极速热升级替换 |
| `cubexd-linux-arm64` | Linux arm64 宿主 / 容器 | 单二进制可执行文件 | ARM64 核心守护进程 |
| `cubex-linux-amd64` | Linux amd64 宿主 / 容器 | 运维 CLI 工具 | 命令行运维控制台 |
| `cubex-linux-arm64` | Linux arm64 宿主 / 容器 | 运维 CLI 工具 | ARM64 命令行运维控制台 |
| `install_pve_lxc.sh` | Proxmox VE (PVE 7.x / 8.x) | 一键自动化脚本 | 宿主机 1 行命令秒级创建 LXC 伴生容器 |
| `install.sh` | 通用 Linux / 飞牛 fnOS / 爱快 | 一键自动化脚本 | 全自动动态嗅探网关与网络环境部署 |
| `setup-openwrt.sh` | OpenWrt 21.02 ~ 24.10+ | 一键自动化脚本 | 适配 fw3/fw4 双内核策略分流与 DNS 加固 |
| `setup-routeros-v7.rsc` | MikroTik RouterOS v7.x | 终端配置脚本 | 自动化策略路由、Netwatch 心跳与 VETH 网桥编排 |
