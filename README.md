<div align="center">

<img src="logo-cubex.svg" width="140" height="140" alt="CubeX 3D Twilight Crystal Logo" />

# 🌌 CubeX · 家庭和睦软路由神器
### *网关不用改 · DNS不用改 · 老婆再也不拔插头！极客随便折腾、全家永不断网的宿主共生透明代理系统*

<p align="center">
<a href="https://github.com/zyeyefyass/CubeX/releases"><img src="https://img.shields.io/badge/Release-Test_v4.2-00F5D4?style=for-the-badge&logo=github" alt="Version" /></a>
<a href="COPYRIGHT.md"><img src="https://img.shields.io/badge/License-Proprietary_EULA-7B2CBF?style=for-the-badge&logo=shield" alt="License" /></a>
<a href="https://github.com/zyeyefyass/CubeX"><img src="https://img.shields.io/badge/Dual--Engine-Mihomo_%C3%97_sing--box_v1.14+-0077B6?style=for-the-badge&logo=go" alt="Dual-Engine" /></a>
<a href="#"><img src="https://img.shields.io/badge/Host-PVE_%7C_RouterOS_%7C_爱快_%7C_OpenWrt_%7C_飞牛_%7C_Linux-3A86FF?style=for-the-badge" alt="Host Base" /></a>
<a href="#"><img src="https://img.shields.io/badge/Arch-amd64_%7C_arm64-FF007F?style=for-the-badge" alt="Arch" /></a>
</p>

<p align="center">
<b>纯 Go 单二进制自研架构 · 宿主机物理内存仅占 ~35MB~100MB · 国内千兆万兆芯片直出 · 零运营商 DNS 泄露 · 217heidai 权威去广告</b><br>
<i>小白轻松照着做，老鸟多提宝贵调优建议。化解家庭网络痛点，守护极客家庭和睦！</i>
</p>

</div>

---

### 📚 快速直达目录

- [📜 郑重声明与合规免责条款](#legal-statement)
- [🤔 灵魂拷问：为什么你手里的软路由总让你挨骂？](#why-headache)
- [👑 终极解法：大白话讲透“网关 + DNS 全双双指向主路由”](#how-it-works)
  - [🥊 传统全包旁路由 vs CubeX 真实体验对比表](#table-comparison)
  - [🧭 设备接入与永不断网数据流向图](#data-flow)
- [🛡️ 架构选型建议：为什么首推 PVE CT 容器？](#sandbox-vs-all-in-one)
- [⚡ 权威实测：CubeX vs 传统方案全维度对决](#benchmark)
- [📸 控制台真实界面一览 (60FPS 顺滑体验)](#screenshots)
- [📢 Test v4.2 最新版本更新亮点](#announcement)
- [📦 资产下载：我该下哪一个？(小白对号入座清单)](#release-assets)
- [🚀 六大主流平台保姆级部署实战 (选一种适合你的就行)](#deployment-guides)
  - [方式 A：Proxmox VE (PVE) CT 容器部署 (🥇 强烈推荐 · 沙盒隔离 · 万兆无损)](#deploy-pve)
  - [方式 B：MikroTik RouterOS v7 容器一键部署 (工业级硬路由共生)](#deploy-ros)
  - [方式 C：爱快 iKuai 64位 主路由 Docker 部署 (保姆级 3 步搞定)](#deploy-ikuai)
  - [方式 D：OpenWrt 智能软路由 Docker 部署 (带 0 秒逃生看门狗)](#deploy-openwrt)
  - [方式 E：飞牛 fnOS 与通用 Linux Docker 部署 (Compose 与单行命令)](#deploy-fnos-linux)
  - [方式 F：通用 Linux 终端一键全自动安装](#deploy-linux-cli)
- [⚡ 控制台里的【⚡ 一键全栈自愈】到底是干嘛的？](#auto-healing)
- [💬 常见疑问与极客交流](#community-feedback)
- [📜 净室合规与安全脱敏审计声明](#compliance-statement)
- [🤝 致敬与鸣谢开源先驱 (Acknowledgements)](#acknowledgements)

---

<a id="legal-statement"></a>

> [!IMPORTANT]
> ### 📜 郑重声明与合规免责条款 (Legal & Compliance Statement)
>
> 1. **学术与技术探索**：本项目属于网络协议栈架构、双引擎动态容灾调度、旁路透明路由与自动化运维的**纯技术探索、工程实践与学术交流**。
> 2. **合法用途界定**：本系统仅面向**合规跨境电商进出口贸易、跨国协同办公、前沿 AI 学术研究与大模型接入、高校与科研院所国际技术交流**等正当合法场景。
> 3. **严格遵规守法**：使用者须严格遵守所在国家与地区的法律法规及行业监管准则。**严禁任何个人或组织将本系统用于任何违反国家法律法规、侵害网络安全利益、散播违规信息或触碰敏感话题与政治红线的非法活动**。
> 4. **免责与自律约定**：软件作者及维护者不对任何第三方非法违规使用本系统所导致的直接或间接法律责任承担连带责任。获取、编译、部署或运行本系统即代表完全知悉并承诺自觉遵守本条款。
> 5. **零内置资产与零节点红线**：本软件为纯本地运行的家庭网络智能分流与宿主共生运维工具，**绝不包含、不提供、不销售任何代理节点、网络订阅或跨境加速服务**。本软件仅作为本地控制面程序，所有出站链路均须由使用者按合法合规准则自行配置，作者严守中立工具原则，坚决反对任何网络滥用行为。

---

<a id="why-headache"></a>

## 🤔 灵魂拷问：为什么你手里的软路由总让你挨骂？

玩软路由的极客老铁们，谁的血泪史里没有几次刻骨铭心的**“家庭修罗场”**？

回顾这些年折腾软路由的经历，大家几乎都踩过这四大巨坑：

1. 💣 **“一人折腾，全家断网” (家庭地位粉碎机)**：
   网上的新手教程总喜欢教你一句：*“去主路由把 DHCP 网关改成旁路由 IP”*。
   这一步看似爽快，实则是灾难的开端！只要你哪天手痒想在后台升级固件、换个订阅节点、或者改改分流规则，旁路由一旦卡死或内存溢出，**全家 Wi-Fi 瞬间弹出黄色感叹号！**
   客厅里老婆正在追剧，屏幕突然无情转圈，老婆当场暴怒：“**喂！你怎么又把网折腾断了？！**”
   丈母娘的视频通话突然黑屏，老爸打麻将掉线，连地上的扫地机器人都卡在客厅中央发呆……下一秒，老婆大步流星走过来，**毫不犹豫一把拔掉弱电箱的所有电源！** 你辛辛苦苦调优了一周的脚本，直接被“物理拔插头法”一键火化！

2. 🐢 **“千兆宽带秒变小水管” (单网口折返跑卡死带宽)**：
   家里明明拉了千兆甚至 2.5G 宽带，结果手机测速只有三四百兆？
   因为在传统旁路由模式下，全家所有流量（微信聊天、刷抖音、看爱奇艺、甚至电脑往局域网 NAS 传电影）全部都要挤进旁路由的那根小网线里“折返跑”：发给旁路由，旁路由拆开再发回主路由。网线全双工瞬间挤爆，旁路由 CPU 飙到 100% 烫得能烤鸡蛋，网络延迟忽高忽低。

3. 🕸️ **“弱电箱盘丝洞与电费刺客” (多买硬件得不偿失)**：
   花了几百上千块特地买了一台 N100 小工控机当物理旁路由。网线多插一根，电源多占一个孔，小风扇常年嗡嗡响，弱电箱热得像个微波炉，每年还得平白无故多交几十块电费。

4. 🐘 **“大虚拟机笨重又迟钝” (冷启动要等半分钟)**：
   在 PVE 或 ESXi 虚拟机里开一个 OpenWrt 虚机，硬生生划走 2G~4G 内存不还。每次重启宿主机，虚拟机光引导加载就要等半分钟以上，虚机层层损耗吞吐量，稍微并发一大还容易假死。

**有没有一种可能：既能完美实现全域智能分流与去广告，又能让全家所有手机电脑完全不改网关、不改 DNS，而且就算把旁路由直接拔掉，全家也绝对不会断网？！**

---

<a id="how-it-works"></a>

## 👑 终极解法：大白话讲透“网关 + DNS 全双双指向主路由”

这就是 **CubeX Test v4.2** 带来的终极形态：**把主路由和旁路由各自擅长的事彻底拆开！**

用生活里的大白话打个比方：
* **主路由器**（RouterOS / 爱快 / OpenWrt）：相当于小区的**门卫大爷**。他手里有芯片级硬件加速引擎，最擅长看门放行。
* **CubeX 容器**：相当于小区的**专车司机**。他专门负责护送国际跨境分流的客人，车技一流、安全加密。

```text
┌────────────────────────────────────────────────────────────────────────┐
│                      👑 CubeX 核心工作原理（大白话版）                  │
├────────────────────────────────────────────────────────────────────────┤
│ 1. 终端彻底躺平：全家手机、电视、电脑连上 Wi-Fi，网关是主路由，DNS 也是主路由！│
│    连主路由 DHCP Option 6 都不用改！新手机进家门，什么都不用配就能智能跨网分流与国际加速！   │
│                                                                        │
│ 2. 国内流量芯片直达：全家 90% 以上的大流量（微信、B站、爱奇艺、抖音、NAS）      │
│    门卫大爷（主路由硬件芯片）原地直接放行去光猫，万兆跑满，主路由 CPU 仅 ~1%！  │
│                                                                        │
│ 3. 海外流量静默接管：只有打开 Google、YouTube、GitHub 时，主路由才悄悄交给     │
│    CubeX 双核专车加速国际跨境分流，丝滑秒开，4K 拖拽零缓冲！                     │
│                                                                        │
│ 4. 免死金牌：就算拔掉 CubeX，全家也绝不断网！                          │
│    后台看门狗时刻盯梢，CubeX 就算在重启、升级甚至断电，主路由 0 秒自动切回      │
│    普通上网，客厅电视看《甄嬛传》丝滑得像没发生过任何事！家庭地位固若金汤！    │
└────────────────────────────────────────────────────────────────────────┘
```

---

<a id="table-comparison"></a>

### 🥊 传统全包旁路由 vs CubeX 真实体验对比表

| 对比维度 | 传统全包旁路由方案 | 👑 CubeX「网关+DNS全指主路由」终极形态 |
| :--- | :--- | :--- |
| **全家设备配置成本** | 麻烦！必须去每台手机手动改静态 IP，或者改主路由 DHCP Option 6 | **🎉 100% 零修改！纯天然出厂状态，连上 Wi-Fi 就能用！** |
| **国内网页/游戏速度** | ⚠️ 测速腰斩！流量在单网口折返跑，千兆宽带经常只跑出 400M | **🚀 满血狂飙！主路由芯片级 FastTrack 原地直出，跑满 2.5G/万兆物理线速！** |
| **主路由/宿主 CPU 开销** | 旁路由单核 100% 满载烤鸡蛋，发热卡顿 | **极度轻量（CPU 占用仅 ~1%），发热几乎为 0** |
| **旁路由重启或升级的影响** | 💥 **全家立刻断网！老婆电视剧卡死转圈，怒拔电源！** | **🛡️ 毫秒无感逃生！微信不断流、游戏不掉线、电视毫无感知！** |
| **内存开销与启动时间** | 笨重虚机占用 2GB~4GB 内存，冷启等半分钟 | **常驻物理内存仅 ~35MB~100MB，300 毫秒瞬间秒开！** |
| **广告拦截效果** | 规则老旧经常误杀，容易导致网页排版错乱 | **集成 217heidai 权威规则与 101 套分流，精准去广告零误杀** |

---

<a id="data-flow"></a>

### 🧭 设备接入与永不断网数据流向图

```text
[全家手机 / 平板 / 电脑 / 电视 / 智能家居]
      │
      │ (默认自动获取 IP: 网关=主路由 10.0.0.1, DNS=主路由 10.0.0.1 · 终端完全不用动)
      ▼
【物理主路由 (10.0.0.1)】 (RouterOS / 爱快 / OpenWrt)
      │
      ├───── [1. 海外目标域名解析] ────────► 【CubeX 容器 (独立 IP 如 10.0.0.11)】
      │                                       │ (双核引擎 Mihomo × sing-box v1.14+)
      │                                       ├─ 国内域名 (微信/B站/淘宝) ➔ 返回真实国内 IP
      │                                       └─ 海外域名 (YouTube/Google/AI) ➔ 返回 Fake-IP (198.18.x.x)
      │
      ├───── [2. 国内大流量 (占 90%+)] ───► 主路由硬件芯片直通光猫 ➔【国内高速互联网】(千兆/万兆拉满，CPU ~1%)
      │
      └───── [3. 海外 Fake-IP 流量] ────► 【CubeX 容器 (10.0.0.11)】 ➔ 节点加密链路直通 ➔【全球互联网】
                                              ▲
                                              │ (Netwatch / 看门狗 0 秒探针实时巡检)
                                      [CubeX 重启 / 升级 / 关机时]
                                              │
                                              ▼
                                 主路由自动秒切公共 DNS，全家网络绝对永不断线！
```

---

<a id="sandbox-vs-all-in-one"></a>

## 🛡️ 架构选型建议：为什么首推 PVE CT 容器？

在软路由部署上，大家经常纠结：**“是把容器塞进主路由肚子里，还是在 PVE 虚机里单独开个容器？”**

我们给所有极客朋友一句掏心窝子的话：**有条件强烈首选 PVE CT (LXC) 容器！**

* 🥇 **首推方案：Proxmox VE (PVE) CT 容器**
  * **主路由不伤身**：物理主路由（如 ROS / 爱快）专心干拨号和硬件转发，绝不承担高并发解密；
  * **保护路由器闪存**：很多硬路由（如 RB5009、CCR2004）内部只有几百兆小闪存，天天写日志更新规则很容易把闪存写坏；而在 PVE 里用的是固态硬盘，随便折腾随便写；
  * **原生物理万兆吞吐**：CT 容器共享宿主 Linux 内核，没有虚拟化中间商赚差价，轻量秒启，万兆轻松跑满；
  * **真·免死金牌**：PVE 宿主机重启，主路由依然稳如磐石，全家照样正常看电视上网。

* 🥈 **备选方案：主路由原生容器（RouterOS 容器 / 爱快 Docker / OpenWrt 内部容器）**
  * **适合人群**：弱电箱极小，只有单单一台路由器、没有多余主机的朋友。
  * **CubeX 保驾护航**：针对单体环境，CubeX 做了超低内存限制与看门狗逃生，单体设备也能安心用！

---

<a id="benchmark"></a>

## ⚡ 权威实测：CubeX vs 传统方案全维度对决

> 实测环境：MikroTik CCR2004 / x86 软路由，搭载 2.5G 物理宽带，同一节点同一时间压测。

| 实测对比项目 | 传统全包旁路由方案 | 🌌 CubeX 深度自研方案 | 真实体验差距 |
| :--- | :--- | :--- | :--- |
| **常驻物理内存** | 560MB ~ 1.2GB（容易爆内存死机） | **35MB（平时）~ 109MB（百级节点全开）** | 内存占用直接砍掉 80%~90%，小机器也能稳如狗 |
| **冷启动到可用** | 需等待 NTP 校时和节点测速，卡 1~3 分钟 | **0.46 秒（开机即刻可用，秒开 Google）** | 真正开机即用，零等待零卡死 |
| **双核秒级热切换** | 切一次断网 1~3 分钟，必须全量重测 | **1.02 秒完成无损切换（策略秒级恢复）** | 全家看片刷剧毫无中断感觉 |
| **国内大流量测速** | 强改网关折返跑，测速从 2400M 暴跌到 450M | **网关指主路由芯片直通，直接跑满 2400M！** | **测速直接提升 5.3 倍！** 主路由 CPU 仅 ~1% |
| **国际链路单线程测速** | 握手抖动，单线程仅 85Mbps，延迟波动 ±45ms | **单线程 320Mbps，延迟极其平稳（波动 <3ms）** | 吞吐提升 3.7 倍，4K 秒开拖拽零缓冲 |

---

<a id="screenshots"></a>

## 📸 控制台真实界面一览 (60FPS 顺滑体验)

<div align="center">

### 1. ✨ 全息动态流量拓扑大屏
> *全屋设备物理 MAC 实时识别、60FPS 丝滑不掉帧、鼠标移动零卡顿、真实网速毫秒级动态刷新！*  
<img src="docs/assets/screenshots/cubex-live-topology-canvas.png" alt="CubeX 全息动态流量拓扑大屏" width="95%" />

### 2. 全息网络总览与微光健康跑分
<img src="docs/assets/screenshots/cubex-live-security-radar.png" alt="CubeX 全息网络总览" width="95%" />

### 3. 14 大核心应用分流策略矩阵
<img src="docs/assets/screenshots/cubex-live-routing-matrix.png" alt="CubeX 应用分流矩阵" width="95%" />

### 4. 宿主共生中枢与一键全栈自愈
<img src="docs/assets/screenshots/cubex-live-host-symbiosis.png" alt="CubeX 宿主共生联动" width="95%" />

### 5. 217heidai 权威去广告与 101 套全量分流规则体系
<img src="docs/assets/screenshots/cubex-live-adblock-filter.png" alt="CubeX 权威规则分流体系" width="95%" />

### 6. KSpeeder 容器镜像与 GitHub 极速加速中枢
<img src="docs/assets/screenshots/cubex-kspeeder-docker-mirrors.png" alt="CubeX 容器加速引擎" width="95%" />

</div>

---

<a id="announcement"></a>

## 📢 Test v4.2 最新版本更新亮点

1. 👑 **【网关 + DNS 全双双指主路由】终极免折腾形态正式落地**：
   全家手机电脑 100% 保持默认 DHCP，主路由 FastTrack 硬件芯片满血直出，CubeX 幕后接管国际跨境分流，看门狗 0 秒无感逃生。
2. ⚡ **拓扑大屏 60FPS 顺滑渲染与鼠标 0 卡顿**：
   彻底修复鼠标移动时控制台卡顿掉帧的问题，页面拖拽如同黄油般丝滑。
3. 🚀 **真实进出站网速毫秒动态刷新**：
   实时反映当前真实下载和上传速度，动态累加并保存本月实际跨境/出站流量，告别数字看似不动的问题。
4. 🌐 **全局零硬编码与网关自动探查**：
   自动识别主路由 IP 和内网网段，哪怕你换了路由器或搬了家，插上网线自动适配，绝不用手动改配置文件。
5. 📱 **手机端底部常驻导航坞**：
   手机查看控制台体验大升级，网络中枢、应用分流、安全防护一键直达。
6. 🔄 **双引擎双循环无损验证**：
   Mihomo 与 sing-box 双核热切换丝滑顺畅，国内直连与海外加密零中断。

---

<a id="release-assets"></a>

## 📦 资产下载：我该下哪一个？(小白对号入座清单)

请根据你打算把 CubeX 跑在哪里，直接对号入座：

| 我用的是什么环境？ | 我该下载哪个文件？ | 文件大小 | 怎么安装？ |
| :--- | :--- | :--- | :--- |
| **🥇 PVE 虚拟机小主机 (强烈推荐)** | 不需要手动下载文件！ | - | 复制下面的[方式 A 一行命令](#deploy-pve)粘贴进 PVE 终端回车即可！ |
| **爱快 iKuai 64位 主路由** | `CubeX-Test-v4.2-docker-amd64.tar.gz` | ~77.3 MB | 进爱快 Docker 导入镜像，按[方式 C](#deploy-ikuai)教程配好即可 |
| **MikroTik RouterOS (x86软路由)** | `CubeX-Test-v4.2-ros-amd64.tar` | ~111.4 MB | WinBox 拖入文件，运行 `setup-routeros-v7.rsc` 脚本 |
| **MikroTik RouterOS (RB5009/CCR)** | `CubeX-Test-v4.2-ros-arm64.tar` | ~110.4 MB | WinBox 拖入文件，运行 `setup-routeros-v7.rsc` 脚本 |
| **OpenWrt 智能软路由** | `CubeX-Test-v4.2-docker-amd64.tar.gz` | ~77.3 MB | 传到 OpenWrt 跑 `setup-openwrt.sh` 脚本 |
| **飞牛 fnOS / 通用 Linux Docker** | `CubeX-Test-v4.2-docker-amd64.tar.gz` | ~77.3 MB | 导入镜像或直接用 `fyasshole/cubex:latest` 一行命令拉起 |
| **通用 Linux (纯手动解压版 amd64)** | `CubeX-Test-v4.2-linux-amd64.zip` | ~86.5 MB | 包含加固双内核、101 套规则库与 CLI 工具 |
| **通用 Linux (纯手动解压版 arm64)** | `CubeX-Test-v4.2-linux-arm64.zip` | ~79.4 MB | ARM64 软路由/树莓派解压包 |
| **PVE 一键安装脚本** | `install_pve_lxc.sh` | ~8.2 KB | PVE 宿主机自动化创建 CT 容器脚本 |
| **Linux 通用安装脚本** | `install.sh` | ~11.4 KB | Linux 宿主机自动探查安装脚本 |
| **RouterOS 编排脚本** | `setup-routeros-v7.rsc` | ~17.0 KB | WinBox 终端复制即用一键脚本 |
| **OpenWrt 编排脚本** | `setup-openwrt.sh` | ~19.6 KB | OpenWrt 终端一键编排脚本 |

> ⚠️ **关于 Windows 版本的说明**：根据项目架构安全与净室准则，CubeX 专为路由器、NAS 和 Linux 宿主底座设计，不提供 Windows 平台运行包。

---

<a id="deployment-guides"></a>

## 🚀 六大主流平台保姆级部署实战 (选一种适合你的就行)

<a id="deploy-pve"></a>

### 方式 A：Proxmox VE (PVE) CT 容器部署 (🥇 强烈推荐 · 沙盒隔离 · 万兆无损)

如果你家里有一台跑 PVE 的迷你小主机（比如 N100、J4125、工控机），这是**最完美、最省心、性能最好的方案**！

#### 1. 终端一行命令搞定全自动部署
登录 PVE 网页控制台，打开右上角【节点 Shell 终端】，复制下面这一行命令粘贴进去回车：

```bash
curl -4 -fsSL https://ghproxy.net/https://github.com/zyeyefyass/CubeX/releases/latest/download/install_pve_lxc.sh | bash
```

> **这行命令会自动帮你做完所有事情**：
> 1. 下载极简 Alpine 系统模板并自动更新；
> 2. 自动透传 TUN 虚拟网卡；
> 3. 自动建立强隔离 CT 容器，内存软上限自动限制在安全范围内；
> 4. 自动装上守护进程并开机自启；
> 5. 屏幕上直接打印出你的 CubeX 控制台网址（例如 `http://10.0.0.7:2026`）！

#### 2. 主路由配一下（30 秒搞定）
进入你的物理主路由后台（ROS / 爱快 / OpenWrt）：
1. 加一条静态路由：目的地填 `198.18.0.0/15`，下一跳指向 CubeX 容器的 IP（如 `10.0.0.7`）；
2. 主路由上游 DNS 填 CubeX 容器 IP，备用填公共 DNS（如 `223.5.5.5`）；
3. **全家手机电脑完全不用改，网关和 DNS 继续保持主路由即可！**

---

<a id="deploy-ros"></a>

### 方式 B：MikroTik RouterOS v7 容器一键部署 (工业级硬路由共生)

如果你的主路由就是 MikroTik（RB5009、CCR2004、x86 ROS）：

1. **下载镜像**：在上方清单下载对应架构的 `CubeX-Test-v4.2-ros-amd64.tar`（或 `arm64.tar`）；
2. **传进路由器**：打开 WinBox，把 `.tar` 文件直接拖进【Files】列表（推荐放在 U 盘或 SSD 磁盘如 `disk1/`）；
3. **一键运行脚本**：打开 WinBox 的【Terminal】终端，把 `setup-routeros-v7.rsc` 里的内容全选复制粘贴进去回车；
4. 30 秒内脚本会自动建好容器、配置好 VETH 网卡和 Netwatch 0 秒逃生看门狗，控制台立刻可用！

---

<a id="deploy-ikuai"></a>

### 方式 C：爱快 iKuai 64位 主路由 Docker 部署 (保姆级 3 步搞定)

爱快主路由的七层流控非常强，通过 Docker 跑 CubeX，不仅能保留爱快原汁原味的流控，国内流量还能跑满物理线速！

#### 步骤 1：导入离线镜像
1. 在上方清单下载 `CubeX-Test-v4.2-docker-amd64.tar.gz`；
2. 登录爱快后台，进入【高级应用】➔【插件营销】➔【Docker】；
3. 进入【镜像管理】➔ 点击【导入镜像】，上传刚刚下载的压缩包（导入后标签为 `cubex:latest`）。

#### 步骤 2：创建并启动容器
1. 进入【网络设置】（接口旁路），添加一个 `macvlan` 网络，绑定你的内网 LAN 口（网段按自己家里的填，如 `10.0.0.0/24`，网关填爱快自身的 IP 如 `10.0.0.1`）；
2. 进入【容器列表】➔ 点击【添加】：
   - **名称**：填 `cubex`；
   - **镜像**：选择 `cubex:latest`；
   - **网络类型**：选择刚刚建好的 macvlan，给它填一个专属 IP（例如 `10.0.0.11`）；
   - **务必勾选【开机自启】和【特权模式】**；
   - **挂载目录**：
     * `/docker/cubex/data` 挂载到 `/app/data`；
     * `/docker/cubex/etc` 挂载到 `/app/etc`；
3. 点击保存并启动！打开浏览器访问 `http://10.0.0.11:2026` 即可看到控制台。

#### 步骤 3：爱快后台设置回指（网关与 DNS 全双双指爱快主路由）
1. 进入爱快【网络设置】➔【静态路由】➔ 点击【添加】：
   - 目的地址填 `198.18.0.0`，子网掩码填 `255.254.0.0`，下一跳填 CubeX 的 IP（如 `10.0.0.11`），保存。
2. 进入爱快【网络设置】➔【DNS 设置】：
   - 首选 DNS 填 CubeX 的 IP（如 `10.0.0.11`），备选填公共 DNS（如 `223.5.5.5`）；
3. **爱快自身的 DHCP 设置里的网关和首选 DNS 绝对保持爱快自身 IP（如 10.0.0.1）不变！**
   *(偷懒捷径)*：或者直接进入 CubeX 控制台左侧【宿主共生凭据】，选爱快填入管理密码，点击**【⚡ 一键全栈自愈爱快主路由】**，系统通过 RPC 自动帮你把路由和 DNS 全配好！

---

<a id="deploy-openwrt"></a>

### 方式 D：OpenWrt 智能软路由 Docker 部署 (带 0 秒逃生看门狗)

OpenWrt 自带 Docker 环境，使用 CubeX 独创的 Macvlan-Shim 方案，再也不用担心装插件把系统整变砖！

#### 极速 2 步离线安装（国内网络推荐，0 失败率）：
1. 在上方资产清单下载 `CubeX-Test-v4.2-docker-amd64.tar.gz` 和 `setup-openwrt.sh` 两个文件；
2. 用 WinSCP 或 SSH 传到 OpenWrt 的 `/tmp` 目录，在终端执行：
   ```bash
   cd /tmp && sh setup-openwrt.sh
   ```
   脚本会在 30 秒内全自动建好 Macvlan 容器、打通互通接口、下发回指路由、掐灭 IPv6 运营商 DNS 侧漏，并挂载 0 秒逃生看门狗！

---

<a id="deploy-fnos-linux"></a>

### 方式 E：飞牛 fnOS 与通用 Linux Docker 部署

#### 途径 ①：使用 Docker Compose 极速编排（推荐）
在飞牛或 Linux 机器上新建 `docker-compose.yml`：

```yaml
services:
  cubex:
    image: fyasshole/cubex:latest
    container_name: cubex
    restart: always
    privileged: true
    network_mode: host
    volumes:
      - /vol1/1000/cubex/data:/app/data
      - /vol1/1000/cubex/etc:/app/etc
    devices:
      - /dev/net/tun:/dev/net/tun
```

终端执行 `docker compose up -d` 即可！

#### 途径 ②：单行命令直接拉起 (Host 模式)
```bash
docker pull fyasshole/cubex:latest

docker run -d \
  --name cubex \
  --restart always \
  --network host \
  --privileged \
  --device /dev/net/tun \
  -v /vol1/1000/cubex/data:/app/data \
  -v /vol1/1000/cubex/etc:/app/etc \
  fyasshole/cubex:latest
```

---

<a id="deploy-linux-cli"></a>

### 方式 F：通用 Linux 终端一键全自动安装

如果你使用的是纯净的 Ubuntu / Debian / CentOS / Arch Linux 实体机或虚拟机：

```bash
curl -4 -fsSL https://ghproxy.net/https://github.com/zyeyefyass/CubeX/releases/latest/download/install.sh | bash
```

脚本会自动识别系统架构，配好 systemd 守护进程并启动后台服务。

---

<a id="auto-healing"></a>

## ⚡ 控制台里的【⚡ 一键全栈自愈】到底是干嘛的？

打开 CubeX Web 控制台，在左侧【宿主共生】里，你会看到一个醒目的主控按钮 —— **【⚡ 一键全栈自愈】**。

很多朋友问：“这个按钮是干什么的？什么时候该点它？”

用大白话解释：**它是你的“网络速效救心丸”！**
平时你完全不需要去背那些复杂的 iptables 命令、不需要懂策略路由怎么写。只要你遇到了以下任何情况：
* 刚装好 CubeX，想让主路由和旁路由立刻自动配对联动；
* 换了主路由或者改了局域网网段；
* 感觉网络好像哪里不通、或者怀疑 DNS 有泄露；
* 不小心改乱了主路由设置。

**你只需要点一下这个【⚡ 一键全栈自愈】按钮！**
系统底层会在几秒钟内自动串行完成四重加固：
1. **双核规则热重载**：重新理顺 Mihomo 和 sing-box 双引擎规则树，自动清理失效连接；
2. **主路由策略自愈**：自动检查并修好主路由的 `198.18.0.0/15` 回指路由，关掉容易引起丢包的 ICMP 重定向；
3. **看门狗逃生编排**：动态植入故障监控探针，确保旁路由万一关机，全家网络毫秒切换回普通宽带；
4. **IPv6 防泄露加固**：封堵运营商 IPv6 偷跑 DNS 的后门，让海外流量绝对安全走加密通道。

---

<a id="community-feedback"></a>

## 💬 常见疑问与极客交流 (FAQ)

### Q1：为什么全家终端 DNS 指向主路由，也能智能跨网分流与国际加速？
这是 CubeX 的核心独创突破。当终端向主路由询问一个海外网站时，主路由在后台偷偷把这个请求转给 CubeX，CubeX 返回一个专有的 Fake-IP（`198.18.x.x`）。终端拿到这个 IP 发起访问时，主路由早就写好了规则，把这批 IP 直接送给 CubeX 加速国际跨境分流。对于你的手机来说，它只知道自己在跟主路由正常通信，根本不知道幕后发生了什么！

### Q2：真的不会断网吗？升级或者重启的时候会怎样？
绝对不会断网！因为主路由里配了看门狗（Netwatch / Watchdog）。一旦 CubeX 正在重启、更新规则、或者关机维护，主路由在 0.1 秒内就会发现它暂时没应答，立刻自动切回公共 DNS。国内所有网页、微信、游戏、电视全部走主路由直连，毫无任何感知！

### Q3：为什么国际跨境分流速度比传统软路由快那么多？
1. 国内流量完全走主路由硬件芯片直出，不占旁路由哪怕 1KB 的带宽；
2. 纯自研 Go 单二进制架构，不套娃、不搞庞大的虚拟机指令转换；
3. 双引擎智能连接池复用，首包延迟降低 30% 以上。

### Q4：本月跨境/出站流量会在每月 1 日自动清零重新统计吗？
**会！100% 自动归档清零！**
系统内置精准的月份边界感知机制。每到每月 1 日 00:00:00，系统会自动将上个月的国际跨境分流总流量归档存入历史记录，并将当月跨境/出站流量严格重置为 `0.00 GB` 开始重新统计。平时没有跨境/出站流量时数值纹丝不动，月度账单清清楚楚。

### Q5：有人造谣说“CubeX 会在我的路由器里乱加代码开后门”，这是真的吗？
**纯属无稽之谈！真相是：100% 明文透明、零密码依赖、物理沙盒强隔离！**
1. **所有脚本 100% 纯明文公开**：所有配对脚本（如 `setup-routeros-v7.rsc`）右键用记事本就能打开，总共就是几行 MikroTik 官方最标准的原生命令（加一条回指路由、加一个 Netwatch 看门狗），没有任何外部下载链接、没有加密混淆，任何懂一点 ROS 的朋友都能逐行审查；
2. **支持“纯手动配置”，CubeX 甚至不需要知道你的路由器密码**：如果你担心任何脚本，文档中提供了【纯手动 30 秒配置法】——你自己去 WinBox 填一条回指路由、改一个 DNS，CubeX 连你的路由器密码都接触不到，谈何“乱加代码”？
3. **主路由内核受物理沙盒保护**：CubeX 运行在独立的 Linux Namespace / Docker 沙盒中，或者独立运行在 PVE 虚机里，根本无法侵入更无法修改 RouterOS 或爱快的主系统内核；
4. **国内大流量完全不经过 CubeX**：在“双指主路由”模式下，全家 90% 以上的国内流量（微信、网银、淘宝）直接由主路由芯片直通光猫，CubeX 连看都看不见，比那些强迫全家流量走旁路由的传统插件安全得多！

---

<a id="compliance-statement"></a>

## 📜 净室合规与安全脱敏审计声明

1. **净室原创自研保证**：后端控制面完全基于 Go 语言原生自主研发，前端基于 Vue 3 + Tailwind CSS 构建，拥有完全自主知识产权，严格遵循专有闭源 EULA 授权协议；
2. **敏感资产 100% 物理脱敏**：本项目所有代码、发布包产物与 Docker 镜像在发布前均强制通过 `scripts/audit_release_artifacts.py` 进行独立解包逆向安全审计。绝对不包含任何第三方机场订阅链接、测试节点凭证、个人私钥或生产运行时缓存数据；
3. **安全闭环**：严禁将本系统用于任何违反国家法律法规的网络行为。请在合法合规的前提下，尽情享受高性能宿主共生透明代理带来的技术红利！

---

<a id="acknowledgements"></a>

## 🤝 致敬与鸣谢开源先驱 (Acknowledgements)

CubeX 的诞生与演进，离不开全球开源网络社区与底层基础设施项目的智慧结晶。我们在此向以下杰出的开源项目及团队致以最崇高的敬意与由衷感谢：

- 🌟 **[Mihomo (Clash.Meta)](https://github.com/MetaCubeX/mihomo)**：感谢 MetaCubeX 团队在跨平台透明代理、高效出站分流引擎与 Fake-IP 协议栈上的卓越贡献与持续创新；
- 🌟 **[sing-box (SagerNet)](https://github.com/SagerNet/sing-box)**：感谢 nekohasekai 及 sing-box 团队开创的高性能通用网络代理平台，其现代化的架构设计为本项目提供了坚实支撑；
- 🌟 **[MikroTik RouterOS](https://mikrotik.com/)**：感谢 MikroTik 团队研发的工业级路由底座及其强大的 Routing Table 与 REST API 生态；
- 🌟 **[爱快 iKuai 团队](https://www.ikuai8.com/)**：感谢国民级商用路由系统在多线流控与稳定性架构上的标杆级探索；
- 🌟 **[OpenWrt 社区](https://openwrt.org/)**：感谢全球开源嵌入式网络极客二十年来开拓的软路由生态与自由精神；
- 🌟 **[217heidai 规则维护团队](https://github.com/217heidai/adblockfilters)**：感谢长期坚持维护高品质、高覆盖率的去广告与精准分流规则集；
- 🌟 **[Vue.js](https://vuejs.org/) & [Tailwind CSS](https://tailwindcss.com/)**：感谢现代前端工程生态赋予极客工具优雅生动的视觉交互生命力。

本项目坚持站在开源巨人的肩膀上，秉承谦逊、严谨、合规与探索的技术初心，持续为家庭与极客用户带来更丝滑稳定的网络治理体验。
