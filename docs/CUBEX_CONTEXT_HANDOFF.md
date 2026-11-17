# CubeX 系统全景与项目交接白皮书 (Project Handoff)

> 本文档用于向 AI 助手（如 Chatbox / Claude / GPT）全面同步 CubeX 项目的架构设计、目录结构、开发规范与核心算法，以便在对话中无缝接盘开发。

---

## 一、项目身份与设计哲学

- **项目名称**：CubeX（宿主共生零感双引擎网关）
- **当前版本**：`v9.9.0` 纯净稳定版
- **核心定位**：专为智能家庭、高可用极客网络及软路由 NAS 设计的**宿主共生双引擎智能透明网关**。
- **设计哲学**：
  1. **零断网逃生（Zero Downtime）**：全家终端 DHCP 网关与首选 DNS 100% 保持在主路由原生 IP（10.0.0.1），拔掉旁路由电源全家国内网络 0 掉线、0 影响。
  2. **双引擎混动（Dual-Engine）**：融合 `Mihomo` 极速转发与 `sing-box v1.14+` 微秒级路由，统一基于 Universal IR 中间表示层解析，支持秒级热重载与崩溃无感自动切换。
  3. **极致轻量（Zero Overhead）**：纯 Go 单二进制控制面，千兆线速跑满时稳态内存锁定在极限低区间（~14MB-35MB）。
  4. **宿主共生（Host Symbiotic）**：原生适配 MikroTik RouterOS v7.x、爱快 iKuai、OpenWrt (firewall4)、飞牛 fnOS 与 Proxmox VE (LXC)。

---

## 二、代码仓库目录结构

```text
CubeX/
├── cmd/
│   ├── cubexd/                 # 核心控制面守护进程入口 (Server)
│   │   ├── main.go             # 程序初始化、命令行参数与启动引导
│   │   ├── routes_core.go      # 核心 HTTP API 路由 (引擎切换/系统状态/配置管理)
│   │   ├── routes_ha.go        # 高可用、RouterOS 联动、逃生控制路由
│   │   ├── routes_sub.go       # 规则集订阅与 DNS 重写管理
│   │   ├── routes_v97.go       # 三角测速、连接树与智能策略路由
│   │   └── dist/               # 前端静态构建物嵌入 (go:embed)
│   └── cubex/                  # 运维管理 CLI 命令行工具入口
│
├── internal/
│   ├── ai/                     # AI Copilot 诊断中枢与自愈建议引擎
│   ├── compiler/               # Universal IR 编译器 (生成 config.yaml / config.json)
│   │   ├── ir.go               # IR 抽象结构体定义
│   │   ├── mihomo.go           # Mihomo (Clash.Meta) 配置转译器
│   │   └── singbox.go          # sing-box v1.14+ 严格 JSON 拓扑编译器
│   ├── device/                 # 局域网终端探查、DHCP 租期监听与按设备策略绑定
│   ├── engine/                 # 双引擎 Supervisor 进程守护、热切换与自愈心跳
│   ├── host/                   # 通用宿主底座联动 (iKuai Web-RPC / OpenWrt UCI / fnOS)
│   ├── ros/                    # MikroTik RouterOS 深度联动核心
│   │   ├── routing.go          # ApplyRouting 自动化编排 (table cubex, 198.18 回指, 逃生)
│   │   ├── hardware.go         # 硬件探查、cubex-clients 清单同步、Mangle/NAT 规则下发
│   │   └── updater.go          # ROS REST API 底层 HTTP 客户端与固件升级通道
│   ├── security/               # 安全中枢 (合规审计、凭证隔离、0 泄露防线)
│   └── telemetry/              # 实时连接树、流量波形、诊断打包导出
│
├── web/                        # 前端交互界面 (Vue 3 + Vite + Tailwind CSS)
│   └── src/
│       ├── App.vue             # 暮光冰晶 Twilight Crystal 仪表盘主控视图
│       └── main.js
│
├── scripts/                    # 构建、安全脱敏审计、多架构发布脚本
└── docs/                       # 架构设计与用户教程文档
```

---

## 三、网络底层与路由核心算法

### 1. 同网段自由共存与 Fake-IP 精准回指
- **RFC 2544 标准网段**：CubeX 使用 `198.18.0.0/15` 作为出海 Fake-IP 池。
- **主路由静态回指**：RouterOS 注入一条静态路由：
  `dst-address=198.18.0.0/15 gateway=10.0.0.7 check-gateway=ping distance=1`
- **黑洞防外泄兜底**：同时注入 `dst-address=198.18.0.0/15 blackhole distance=200`，杜绝旁路由离线时 Fake-IP 泄漏至外网运营商光猫导致证书报错。

### 2. 加速终端独立分流与 DNS 防污染重定向（最新突破）
- **加速名单**：RouterOS `/ip/firewall/address-list` 中维护 `cubex-clients` 清单。
- **Mangle 策略路由**：
  `chain=prerouting action=mark-routing new-routing-mark=cubex src-address-list=cubex-clients dst-address-list=!LAN_SUBNET`
- **DNS 本地接管重定向 (NAT Redirect)**：
  针对 TCL/小米等电视固件暗中注入阿里公共 DNS（`223.5.5.5`）造成 IPv6 虚假地址（`2001::1`）污染与 SmartTube 播放卡死问题，CubeX 在 RouterOS 上自动下发非侵入式定向重定向规则：
  ```text
  /ip firewall nat add chain=dstnat action=redirect protocol=udp dst-port=53 src-address-list=cubex-clients dst-address-type=!local comment="CubeX-Client-DNS-Redirect"
  /ip firewall nat add chain=dstnat action=redirect protocol=tcp dst-port=53 src-address-list=cubex-clients dst-address-type=!local comment="CubeX-Client-DNS-Redirect"
  ```
  该规则将加速终端擅自发往外网公共 DNS 的请求，瞬间收归主路由并由 CubeX 净化为 Fake-IP（AAAA 记录彻底置空），从根本上消除 DNS 侧漏与假 IPv6 污染。

### 3. Netwatch 毫秒级断电逃生看门狗
RouterOS 探针每 3 秒检测旁路由存活。一旦 CubeX 容器宕机，看门狗在 50ms 内全自动将主路由 DNS 切回公共 DNS 并临时禁用分流规则，全网 0 秒无感回退为纯宽带直连。

---

## 四、开发与协作四大绝对红线

1. **合规中立专业网络工程用语**：严格遵守合规准则，严禁使用任何煽动性、敏感性词汇或国旗图标；统一规范为中立专业词汇（如“规则分流”、“网络加速”、“国际跨境协同”、“双栈防泄露加固”等）。
2. **净室架构与零旧残留**：禁止出现历史旧版残留命名，杜绝外部第三方通信软件越权控制通道。
3. **双核绝对同步**：任何网络优化、规则改动或功能新增，必须同时同步覆盖 `Mihomo` 与 `sing-box v1.14+`。
4. **无明确指令绝不发布 Windows 产物**：常规编译打包严格限定为 Linux 与 RouterOS 多架构底座。
