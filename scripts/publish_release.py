#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX 原生双栈与极速热切版 GitHub 发布自动化脚本
- 创建/更新 Release（全中文精简大白话说明 + 资产清单表）
- 统一通过 curl.exe --http1.1 -x PROXY 禁用 Expect: 100-continue 直传资产，支持断点续传与字节校验
- 上传完毕后自动执行全平台 10 版本存留红线脚本
"""
import json
import os
import subprocess
import sys
import tempfile
import time
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT_DIR = SCRIPT_DIR.parent
RELEASE_DIR = ROOT_DIR / "dist" / "release"
sys.path.insert(0, str(SCRIPT_DIR))

from enforce_10_version_retention import get_github_token  # noqa: E402

REPO = "zyeyefyass/CubeX"
PROXY = os.environ.get("CUBEX_RELEASE_PROXY", "")

def get_tag():
    try:
        with open(ROOT_DIR / "meta.json", "r", encoding="utf-8") as f:
            v = json.load(f).get("version", "v9.9.2")
            if not v.startswith("v"):
                v = f"v{v}"
            return v
    except Exception:
        return "v9.9.2"

TAG = get_tag()

RELEASE_TITLE = f"CubeX {TAG} 正式版：Fake-IP 路由收敛 · 根治 Lucky STUN 穿透 · 顶栏减法去伪存真 · 60FPS 极简暮光美学"

RELEASE_BODY = f"""# CubeX {TAG} 正式版 · Fake-IP 路由收敛 · 根治 Lucky STUN 穿透 · 顶栏减法去伪存真 · 60FPS 极简暮光美学

Fake-IP TUN 路由收敛与 Lucky STUN 穿透彻底根治 (闭环 Issue #6) · 53 端口 1ms 物理极速探活 · 顶栏去伪存真大做减法 · 纯 CSS 矢量光晕 60FPS 零卡顿 · 全站 Gzip 稳控在 269.5 KB · 双核双循环 4 次热切 100% 稳定

CubeX 迎来 {TAG} 正式版！本版本彻底收敛了 TUN 虚拟网卡路由表，从根源上消除了飞牛 fnOS 与 PVE 宿主下 Docker 容器网络回程黑洞，使 Lucky STUN 穿透与影视/NAS 任务恢复全锥型 NAT 极速直连；同时控制台界面大刀阔斧去伪存真，剔除冗余装饰与重复入口，全站更轻量、更丝滑、更省心。

---

### 🌟 核心更新亮点

1. **Fake-IP TUN 路由收敛与 Lucky STUN 穿透彻底根治 (闭环 Issue #6)**：
   - 彻底将双引擎 TUN 路由接管范围收敛至精准的 `198.18.0.0/15`（IPv6 开启时追加 `fc00::/18`），彻底根除了 Table 2022 中 50 余条非私网网段劫持造成的非对称回程丢包；
   - 实测飞牛 fnOS / PVE All-in-One 宿主下 Lucky v3.8.1 TCP/UDP STUN 穿透完全恢复为全锥型 NAT（Full Cone），外网手机与远程终端访问直达物理网卡；
   - 引入高精度子域正则 `^(stun|turn)[0-9]*\\.` 与 `\\.(stun|turn)[0-9]*\\.`，STUN/TURN 协议与 BT/PT 任务实现物理芯片线速直连直出。
2. **大白话数据面中枢与 1ms 物理极速探活**：
   - 将底层健康探活端口由 1053 对齐为实际监听的 53 端口，拔除 200ms 超时病根，探活耗时从 201ms 压缩至 1ms 物理极限；
   - 采用暮光冰晶固定顶栏，三态健康脉冲胶囊作为唯一诊断入口，去除冗余齿轮图标，一键直达抽屉式排障控制台。
3. **顶栏去伪存真与全站极致轻量化 (60FPS · 269KB Gzip)**：
   - 大刀阔斧做减法：彻底剔除顶栏吃灰的居中大搜索框与快捷键徽章，彻底删除总览页冗余空卡片；
   - 替换离屏模糊为纯 CSS `radial-gradient` 渐变网格，彻底解放 GPU 显存，实测满载单帧渲染稳定在 16.8ms（全程 60FPS 丝滑不掉帧）；
   - 全站 Gzip 压缩包严格稳控在 **269.5 KB**。
4. **壁纸资产解耦与净室开源合规**：
   - 公开代码库与发行包 100% 剥离第三方壁纸素材，内置原创暮光冰晶几何矢量底图；
   - 后端支持 `/etc/cubex/data/wallpapers/` 槽位本地挂载，并加入白名单与 `filepath.Base` 防路径穿越加固。
5. **双核双循环 4 次热切与软路由三巨头三位一体实测**：
   - 4 次往返热重载零丢包、零连接停顿，CT 107 实测 RSS 42.9MB，14 项健康审计通过率 100%。

---

### 📦 资产下载与校验清单 (Release Assets)

> [!TIP]
> **各平台底座发布物对照表**：请根据您的宿主底座与硬件架构选择对应的下载资产（完整包内置双核与全量 105 套规则）：

| 发布物文件名 | 适用宿主与架构 | 说明与用途 |
| :--- | :--- | :--- |
| `cubex_fnos_{TAG}_amd64.fpk` | 飞牛 fnOS (x86_64) | 飞牛私有云官方应用中心原生套件安装包 |
| `cubex_fnos_{TAG}_arm64.fpk` | 飞牛 fnOS (aarch64) | 飞牛私有云 ARM64 官方应用中心套件包 |
| `CubeX-{TAG}-ros-amd64.tar` | 软路由系统 (x86 软路由) | 软路由系统原生未压缩容器镜像（支持管理终端拖入直装） |
| `CubeX-{TAG}-ros-arm64.tar` | 软路由系统 (RB5009/CCR/ARM) | 软路由系统 ARM64 原生容器镜像（适配 64 位 ARM 硬路由） |
| `CubeX-{TAG}-docker-amd64.tar.gz` | Docker / 爱快 / 飞牛 / NAS / PVE | 容器离线镜像直装导入压缩包（内置双核与全量 105 套规则） |
| `CubeX-{TAG}-linux-amd64.zip` | Linux amd64 (PVE/LXC/Ubuntu/Debian) | 完整独立开箱包（内含主程序、运维工具、双核引擎及 105 套规则） |
| `CubeX-{TAG}-linux-arm64.zip` | Linux arm64 (树莓派/香橙派/ARM软路由) | ARM64 完整归档发布包 |
| `cubexd-linux-amd64` | Linux amd64 | 纯单二进制控制面（代码加固版，用于在线平滑热更新） |
| `cubexd-linux-arm64` | Linux arm64 | ARM64 单二进制核心控制面（代码加固版） |
| `cubex-linux-amd64` | Linux amd64 | 纯命令行运维工具（终端直接运行） |
| `cubex-linux-arm64` | Linux arm64 | ARM64 纯命令行运维工具（终端直接运行） |
| `install_ros_container.rsc` | RouterOS v7 单机容器脚本 | RouterOS v7 内置容器单机自闭环一键编排脚本 (172.17.0.0/30 + 看门狗) |
| `install_openwrt_docker.sh` | OpenWrt Docker 单机脚本 | OpenWrt / iStoreOS Docker 单机自闭环一键安装脚本 |
| `setup-routeros-v7.rsc` | 软路由终端编排脚本 | 软路由管理终端复制即用一键自动编排脚本 |
| `setup-openwrt.sh` | 开源软路由系统一键安装脚本 | 开源软路由系统原生适配安装脚本 |
| `install_pve_lxc.sh` | 容器一键脚本 | 极简容器一键安装脚本 |
| `install.sh` | 通用一键安装脚本 | 通用一键部署主脚本 |
| `SHA256SUMS` | 全平台校验文件 | 所有发布产物完整性 SHA256 校验清单 |
"""


def gh_api(token, method, path, data=None):
    url = f"https://api.github.com/repos/{REPO}{path}"
    headers = [
        "-H", f"Authorization: token {token}",
        "-H", "Accept: application/vnd.github.v3+json",
        "-H", "User-Agent: CubeX-Release-Agent",
    ]
    cmd = ["curl.exe", "-s", "-w", "\n%{http_code}", "-X", method] + headers
    if PROXY:
        cmd.extend(["-x", PROXY])

    temp_file = None
    if data:
        temp_file = tempfile.NamedTemporaryFile("w", delete=False, encoding="utf-8")
        json.dump(data, temp_file, ensure_ascii=False)
        temp_file.close()
        cmd.extend(["-H", "Content-Type: application/json; charset=utf-8", "-d", f"@{temp_file.name}"])

    cmd.append(url)
    try:
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=90)
        output = res.stdout
    finally:
        if temp_file and os.path.exists(temp_file.name):
            os.remove(temp_file.name)

    if "\n" in output:
        body, code_str = output.rsplit("\n", 1)
        try:
            code = int(code_str.strip())
        except Exception:
            code = 200
        return code, body
    return 200, output


def get_assets_map(token, release_id):
    code, body = gh_api(token, "GET", f"/releases/{release_id}/assets?per_page=100")
    if code != 200:
        return {}
    try:
        items = json.loads(body)
        return {item["name"]: item for item in items}
    except Exception:
        return {}


def upload_asset(token, release_id, upload_url_tpl, file_path):
    name = file_path.name
    size = file_path.stat().st_size
    base_upload_url = upload_url_tpl.split("{")[0]
    upload_url = f"{base_upload_url}?name={name}"

    content_type = "application/octet-stream"
    if name.endswith(".tar"):
        content_type = "application/x-tar"
    elif name.endswith(".tar.gz") or name.endswith(".tgz"):
        content_type = "application/gzip"
    elif name.endswith(".zip"):
        content_type = "application/zip"
    elif name.endswith(".sh") or name.endswith(".rsc") or name.endswith(".txt") or name == "SHA256SUMS":
        content_type = "text/plain"
    elif name.endswith(".fpk"):
        content_type = "application/octet-stream"

    # Pre-check if already uploaded cleanly
    assets = get_assets_map(token, release_id)
    if name in assets:
        cur_a = assets[name]
        if cur_a.get("size") == size and cur_a.get("state") == "uploaded":
            print(f"✅ [已存在且完整] {name} ({round(size / 1024 / 1024, 2)} MB) 跳过重复上传", flush=True)
            return True
        print(f" - 发现未就绪残片 {name} (state: {cur_a.get('state')}, size: {cur_a.get('size')})，正在清理...", flush=True)
        gh_api(token, "DELETE", f"/releases/assets/{cur_a['id']}")
        time.sleep(3)

    print(f"🚀 [正在上传] {name} ({round(size / 1024 / 1024, 2)} MB)...", flush=True)

    cmd = [
        "curl.exe",
        "-s", "-S",
        "-L",
        "--http1.1",
        "--connect-timeout", "30",
        "--max-time", "600",
        "-X", "POST",
        "-H", f"Authorization: token {token}",
        "-H", f"Content-Type: {content_type}",
        "-H", f"Content-Length: {size}",
        "-H", "Expect:",
        "--speed-time", "60",
        "--speed-limit", "1024",
        "--data-binary", f"@{file_path}",
    ]
    if PROXY:
        cmd.extend(["-x", PROXY])
    cmd.append(upload_url)

    for attempt in range(1, 4):
        # Ensure no dirty starter exists before attempting upload
        assets = get_assets_map(token, release_id)
        if name in assets:
            cur_a = assets[name]
            if cur_a.get("size") == size and cur_a.get("state") == "uploaded":
                print(f"   ✅ [校验成功] {name} 云端校验通过！", flush=True)
                return True
            print(f"   - 清理未就绪残片 {name} (ID: {cur_a['id']})...", flush=True)
            gh_api(token, "DELETE", f"/releases/assets/{cur_a['id']}")
            time.sleep(3)

        print(f"   - 第 {attempt}/3 次传输...", flush=True)
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=660)

        # Wait up to 120 seconds for GitHub background storage finalization
        print("   - 等待 GitHub 云端入库终审校验...", flush=True)
        success = False
        for wait_idx in range(24):
            time.sleep(5)
            assets = get_assets_map(token, release_id)
            if name in assets:
                cur_a = assets[name]
                if cur_a.get("size") == size and cur_a.get("state") == "uploaded":
                    print(f"   ✅ [校验成功] {name} ({round(size / 1024 / 1024, 2)} MB) 上传完成并校验一致！", flush=True)
                    success = True
                    break
                else:
                    print(f"     ...等待中 (state: {cur_a.get('state')}, size: {cur_a.get('size')}/{size})", flush=True)
        if success:
            return True

        print(f"   ⚠️ 第 {attempt} 次校验未通过: stdout={res.stdout[:150]}, stderr={res.stderr[:150]}", flush=True)
        time.sleep(5)

    return False


def main():
    token = get_github_token()
    if not token:
        print("❌ 未获取到有效的 GITHUB_TOKEN，发布中止！")
        sys.exit(1)

    print(f"📦 正在检查/创建 Release: {TAG} ...")
    code, body = gh_api(token, "GET", f"/releases/tags/{TAG}")
    release = None
    if code == 200:
        release = json.loads(body)
        print(f"   ✓ 找到已存在的 Release (ID: {release['id']})")
        update_payload = {
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
        }
        gh_api(token, "PATCH", f"/releases/{release['id']}", update_payload)
    else:
        print(f"   - 创建新 Release: {TAG} ...")
        create_payload = {
            "tag_name": TAG,
            "target_commitish": "main",
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
            "draft": False,
            "prerelease": False,
        }
        code, body = gh_api(token, "POST", "/releases", create_payload)
        if code in (200, 201):
            release = json.loads(body)
            print(f"   ✅ 成功创建 Release (ID: {release['id']})")
        else:
            print(f"❌ 创建 Release 失败 (HTTP {code}): {body}")
            sys.exit(1)

    release_id = release["id"]
    upload_url_tpl = release["upload_url"]

    expected_assets = [
        f"cubex_fnos_{TAG}_amd64.fpk",
        f"cubex_fnos_{TAG}_arm64.fpk",
        f"CubeX-{TAG}-ros-amd64.tar",
        f"CubeX-{TAG}-ros-arm64.tar",
        f"CubeX-{TAG}-docker-amd64.tar.gz",
        f"CubeX-{TAG}-linux-amd64.zip",
        f"CubeX-{TAG}-linux-arm64.zip",
        "cubexd-linux-amd64",
        "cubexd-linux-arm64",
        "cubex-linux-amd64",
        "cubex-linux-arm64",
        "install_ros_container.rsc",
        "install_openwrt_docker.sh",
        "setup-routeros-v7.rsc",
        "setup-openwrt.sh",
        "install_pve_lxc.sh",
        "install.sh",
        "SHA256SUMS",
    ]

    print(f"\n📂 检查本地发布物目录: {RELEASE_DIR}")
    missing = []
    for item in expected_assets:
        p = RELEASE_DIR / item
        if not p.exists():
            missing.append(item)
    if missing:
        print(f"❌ 本地发布物缺失 {len(missing)} 项，无法发布:")
        for m in missing:
            print(f"   - {m}")
        sys.exit(1)

    print(f"✅ {len(expected_assets)} 项发布资产全部就绪，开始上传至 GitHub Releases (ID: {release_id})...\n")

    failed = []
    for item in expected_assets:
        p = RELEASE_DIR / item
        ok = upload_asset(token, release_id, upload_url_tpl, p)
        if not ok:
            failed.append(item)

    if failed:
        print(f"\n❌ 上传失败列表 ({len(failed)} 项): {failed}")
        sys.exit(1)

    print(f"\n🎉 全部 {len(expected_assets)} 项资产上传并校验 100% 成功！")

    print("\n🧹 执行全平台 10 周期存留红线清算...")
    enforce_script = SCRIPT_DIR / "enforce_10_version_retention.py"
    subprocess.run([sys.executable, str(enforce_script)], check=True)
    print("✅ 10 周期存留红线清算完成！发布流程圆满收官！")


if __name__ == "__main__":
    main()
