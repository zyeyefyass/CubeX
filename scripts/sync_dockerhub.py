#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Docker Hub Description & Overview Sync Script
- Syncs Short Description (< 100 chars) to Docker Hub API
- Syncs Full Description (Overview) within Docker Hub 25,000 bytes limit
- Injects absolute URL to Twilight Crystal Logo (assets/logo.png)
"""

import json
import os
import sys
import urllib.request

RAW_BASE = "https://raw.githubusercontent.com/zyeyefyass/CubeX/main/"
USERNAME = os.environ.get("DOCKERHUB_USERNAME") or "fyasshole"
REPO = "cubex"

def get_dockerhub_token():
    tok = os.environ.get("DOCKERHUB_TOKEN", "").strip()
    if tok:
        return tok
    # Fallback to local scratch file if present
    scratch_tok = os.path.join(os.path.dirname(os.path.dirname(os.path.abspath(__file__))), "scratch", ".dockerhub_token")
    if os.path.exists(scratch_tok):
        try:
            with open(scratch_tok, "r", encoding="utf-8") as f:
                return f.read().strip()
        except Exception:
            pass
    return ""

TOKEN = get_dockerhub_token()

SHORT_DESC = "CubeX v9.9 · 15MB 极轻量全屋网络中枢 · 影音/NAS专属分流 · 零感自愈双核网关"

OVERVIEW = f"""# 🌌 CubeX · 宿主共生双引擎智能透明网关

<div align="center">
  <img src="{RAW_BASE}assets/logo.png" width="130" height="130" alt="CubeX Twilight Crystal Logo" />
  <h3>网关不用改 · DNS不用改 · 告别单点故障与断网风险！极客随便折腾、全家永不断网的宿主共生透明代理系统</h3>
</div>

<div align="center">

[![GitHub Release](https://img.shields.io/badge/Release-v9.9.0_正式版-00F5D4?style=for-the-badge&logo=github)](https://github.com/zyeyefyass/CubeX/releases)
[![Docker Hub](https://img.shields.io/badge/Docker-fyasshole%2Fcubex-0db7ed?style=for-the-badge&logo=docker)](https://hub.docker.com/r/fyasshole/cubex)
[![Dual-Engine](https://img.shields.io/badge/Dual--Engine-Mihomo_%C3%97_sing--box_v1.14+-0077B6?style=for-the-badge&logo=go)](https://github.com/zyeyefyass/CubeX)
[![Host Base](https://img.shields.io/badge/Host-Docker_%7C_%E9%A3%9E%E7%89%9B_fnOS_%7C_PVE_%7C_RouterOS_%7C_%E7%88%B1%E5%BF%AB_iKuai-3A86FF?style=for-the-badge)](https://github.com/zyeyefyass/CubeX)
[![Arch](https://img.shields.io/badge/Arch-amd64_%7C_arm64-FF007F?style=for-the-badge)](https://github.com/zyeyefyass/CubeX)
[![License](https://img.shields.io/badge/License-Proprietary_EULA-7B2CBF?style=for-the-badge)](https://github.com/zyeyefyass/CubeX)

**纯 Go 单二进制自研架构 · 物理内存常驻仅 ~15MB~35MB · 国内千兆满血硬件直连 · 0 明文 DNS 泄露 · 217heidai 权威规则体系**

</div>

---

## 📢 v9.9.0 最新版本核心亮点

1. 🎬 **【影音与 NAS 专属无损直通模式】**：
   - 专为电视盒子、蓝光播放机、极空间/群晖/威联通/飞牛 NAS 定制；
   - TMDB/TVDB/Fanart 海报墙刮削精准走高速加速通道，BT/PT 高速大流量下载与局域网内网传输 100% 物理直连直通，兼顾高速刮削与满带宽直连，彻底杜绝非必要外部带宽损耗。
2. ⚡ **【顶栏一句话自定义网站分流】**：
   - 在规则管理中枢顶栏直接输入域名（如 `tmdb.org`），提供三大醒目按键一键指派：
     - `【⚡ 强制走加速】`
     - `【🏠 强制走直连】`
     - `【🛑 屏蔽该网站】`
   - 下方实时呈现分类标签卡，支持单点 `✕` 即时清除，秒级生效。
3. 📦 **【全屋配置时光机 (一键秒级快照与无损回滚)】**：
   - 一键将全屋所有设备的分流状态、自定义别名、MAC 映射及系统规则完整打包为结构化快照；
   - 支持带 SHA-256 核验的一键无损导入恢复，软硬件换机、重装系统或迁移底座零感无缝恢复。
4. 🧊 **【控制面极限瘦身与 14.9MB 稳态】**：
   - 彻底优化前端响应式缓存与后台长轮询调度，杜绝重复计算；
   - 守护控制面在 Linux 环境下物理常驻内存（RSS）稳定压制至 **11.4MB ~ 14.9MB**，核心引擎常驻内存仅 **28MB ~ 29MB**，双核混动平稳运行。
5. 🔄 **【双核混动极速热重载 (Mihomo × sing-box v1.14+)】**：
   - 保留顶栏经典混动切换胶囊 `[ 📦 sing-box 极速轻量 | 🐱 Mihomo 规则完备 ]`；
   - 双核往返热重载实测单次切换仅需 200~400ms，0 丢包、0 路由锁死。

---

## 💎 为什么“网关 + DNS 双双指向主路由”才是最优解？

折腾传统旁路由谁没有经历过网关服务波动引发的**局域网大面积断网痛点**？
以往不管是传统全包旁路由，还是改旁路 DNS 方案，一旦在弱电箱升级固件、换订阅节点或写错一行配置，全家设备立刻掉线弹窗！

**CubeX 打破魔咒：【网关 + DNS 双双指向主路由】的免折腾形态！**

1. 🛡️ **终端 100% 彻底躺平**：
   全家所有手机、电视、平板、电脑、智能家居，**通通保持出厂默认的 DHCP（网关=主路由，DNS=主路由）**！新设备进门连上 Wi-Fi 就能直接享受规则分流与广告拦截！
2. ⚡ **国内大流量满血直连**：
   国内高频大流量（微信、B站、爱奇艺、抖音、在线游戏、百度网盘、局域网 NAS）由主路由器物理芯片直接走默认路由直出，直接跑满 2.5G/万兆物理线速，主路由 CPU 占用仅 ~1%！
3. 🕊️ **随便怎么折腾绝不断网**：
   CubeX 在独立容器中作为智能出站网关，在后台升级固件、切换引擎、甚至拔掉容器，主路由依然保持直通，全家网络绝对永不断线，微信不重连，国内游戏不掉线，客厅电视播放丝滑如常！

---

## 🚀 Docker & 飞牛 fnOS 极简部署指南

### 途径 ①：Docker Compose 极速编排（推荐）

在宿主机上新建 `docker-compose.yml`：

```yaml
version: "3.8"

services:
  cubex:
    image: fyasshole/cubex:latest
    container_name: cubex
    restart: unless-stopped
    network_mode: host
    privileged: true
    cap_add:
      - NET_ADMIN
      - NET_RAW
    devices:
      - /dev/net/tun:/dev/net/tun
    volumes:
      - ./data:/app/data
      - ./etc:/app/etc
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
      - CUBEX_EDITION=docker
```

启动命令：
```bash
docker compose up -d
```

### 途径 ②：单行 Docker 命令直接运行

```bash
docker run -d \\
  --name cubex \\
  --restart unless-stopped \\
  --network host \\
  --privileged \\
  --cap-add=NET_ADMIN \\
  --cap-add=NET_RAW \\
  --device /dev/net/tun:/dev/net/tun \\
  -v $(pwd)/data:/app/data \\
  -v $(pwd)/etc:/app/etc \\
  -v /etc/localtime:/etc/localtime:ro \\
  -e TZ=Asia/Shanghai \\
  -e CUBEX_EDITION=docker \\
  fyasshole/cubex:latest
```

### 🔌 默认服务端口说明

| 端口 | 类型 | 说明与用途 |
| :--- | :--- | :--- |
| **2026** | HTTP | CubeX Web 交互控制台入口 (Twilight Crystal 暮光冰晶界面) |
| **7890** | HTTP/SOCKS5 | 混合代理端口 (Mixed Inbound) |
| **7893** | TCP/UDP | 透明网关 TProxy 端口 (TUN / 宿主策略路由回指) |
| **9090** | REST API | Clash / Mihomo 控制协议兼听端口 |
| **1053** | UDP/TCP | 内置低延迟防泄露 DNS 解析端口 |

---

## 🧭 主路由静态回指配置（仅需 30 秒）

为了让全家设备在保持网关与 DNS 指向主路由的前提下享受旁路智能分流，只需在您的主路由（RouterOS / 爱快 iKuai / OpenWrt / 华硕固件等）添加一条**静态回指路由**：

- **目标网段**：`198.18.0.0/15`
- **网关 / 下一跳 (Gateway)**：填写运行 CubeX Docker 容器的宿主机内网 IP (例如 `10.0.0.7`)

完成后，所有非国内业务流将在主路由命中该回指路由，毫秒级注入 CubeX 容器处理，直连流量 100% 保持在主路由芯片级直出！

---

## 🔗 更多资源与项目地址

- 完整源码与深度使用文档：[https://github.com/zyeyefyass/CubeX](https://github.com/zyeyefyass/CubeX)
- 最新正式版 Release 资产下载：[https://github.com/zyeyefyass/CubeX/releases](https://github.com/zyeyefyass/CubeX/releases)
- 架构授权：遵循专有闭源 EULA 授权协议，保留所有原创权利。
"""


def sync():
    if not TOKEN:
        print("❌ Error: No DOCKERHUB_TOKEN provided.")
        sys.exit(1)

    print(f"🔑 Authenticating with Docker Hub API as {USERNAME}...")
    login_url = "https://hub.docker.com/v2/users/login/"
    login_data = json.dumps({"username": USERNAME, "password": TOKEN}).encode("utf-8")
    login_req = urllib.request.Request(
        login_url,
        data=login_data,
        headers={"Content-Type": "application/json", "User-Agent": "CubeX-Sync"}
    )

    proxy = os.environ.get("HTTP_PROXY") or os.environ.get("http_proxy")
    if not proxy and os.name == "nt":
        proxy = "http://10.0.0.7:7890"

    if proxy:
        opener = urllib.request.build_opener(urllib.request.ProxyHandler({'http': proxy, 'https': proxy}))
    else:
        opener = urllib.request.build_opener()

    try:
        with opener.open(login_req, timeout=15) as resp:
            jwt_token = json.loads(resp.read().decode("utf-8"))["token"]
        print("✓ JWT Token obtained successfully!")
    except Exception as e:
        print(f"❌ Failed to login to Docker Hub: {e}")
        sys.exit(1)

    byte_len = len(OVERVIEW.encode("utf-8"))
    print(f"📦 Payload: Short Desc ({len(SHORT_DESC)} chars), Overview ({byte_len} bytes)")

    patch_url = f"https://hub.docker.com/v2/repositories/{USERNAME}/{REPO}/"
    patch_data = json.dumps({
        "description": SHORT_DESC,
        "full_description": OVERVIEW
    }).encode("utf-8")

    patch_req = urllib.request.Request(
        patch_url,
        data=patch_data,
        headers={
            "Content-Type": "application/json; charset=utf-8",
            "Authorization": f"Bearer {jwt_token}",
            "User-Agent": "CubeX-Sync"
        },
        method="PATCH"
    )

    print("🚀 Updating Docker Hub repository metadata...")
    try:
        with opener.open(patch_req, timeout=30) as resp:
            res = json.loads(resp.read().decode("utf-8"))
            print("🎉 Docker Hub metadata updated successfully!")
            print(f"  Description: {res.get('description')}")
            print(f"  Overview Length: {len(res.get('full_description', ''))} characters")
    except urllib.error.HTTPError as e:
        err_msg = e.read().decode("utf-8")
        print(f"❌ PATCH failed: HTTP {e.code}: {err_msg}")
        sys.exit(1)
    except Exception as e:
        print(f"❌ Unexpected error: {e}")
        sys.exit(1)


if __name__ == "__main__":
    sync()
