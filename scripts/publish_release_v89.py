#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Test v8.9 GitHub 发布自动化脚本
- 创建/更新 Test-v8.9 Release（全中文精简大白话说明 + 资产清单表）
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
TAG = "Test-v8.9"
PROXY = "http://10.0.0.7:7890"

RELEASE_TITLE = "CubeX Test v8.9：疯狂动物城三级景深流光玻璃 · 120FPS 零卡顿渲染调度 · 双擎切换毫秒耗时穿透 · 宿主旁路维护逃生模式"

RELEASE_BODY = """# CubeX Test v8.9

本次更新重点实现了疯狂动物城主题三级景深流光玻璃美学重构、120FPS 零卡顿帧调度器、双引擎无感热切毫秒耗时精准穿透以及软路由宿主旁路维护逃生模式：

### 核心亮点
1. **疯狂动物城三级景深流光玻璃与赏景模式**：顶层保留高透真毛玻璃，内容子项改用半透明衬底与镜面高光，彻底根除滚动掉帧；右上角新增一键赏景模式，并注入 400ms 防抖保护窗与 45px 位移阈值，丝滑赏景不误退。
2. **120FPS 零卡顿渲染调度**：百级节点矩阵与规则列表启用浏览器原生虚拟化，视口外零重绘开销；植入帧调度器合并高频更新，后台标签页自动休眠，切回前台单次合并渲染。
3. **双核切换毫秒耗时精确穿透**：控制台无感热切后，顶部精确展示停核耗时、启动耗时、总耗时与规则条数；同时在切核瞬间一键静默截断在途请求，切核成功毫秒级提前唤醒看板，彻底告别报错弹窗与漫长等待。
4. **宿主旁路维护逃生模式**：一键开启旁路维护，软路由瞬间平滑切回原生直连与原生解析，局域网绝不断网；维护结束一键恢复接管。

---

### 📦 资产下载与校验清单 (Release Assets)

| 发布物文件名 | 适用宿主与架构 | 文件大小 | 说明与用途 |
| :--- | :--- | :--- | :--- |
| `CubeX-Test-v8.9-ros-amd64.tar` | 软路由系统 (x86 软路由) | 93.04 MB | 软路由系统原生未压缩容器镜像（支持管理终端拖入直装） |
| `CubeX-Test-v8.9-ros-arm64.tar` | 软路由系统 (RB5009/CCR/ARM) | 93.36 MB | 软路由系统 ARM64 原生容器镜像（适配 64 位 ARM 硬路由） |
| `CubeX-Test-v8.9-docker-amd64.tar.gz` | Docker / 爱快 / 飞牛 / NAS / PVE | 76.32 MB | 容器离线镜像直装导入压缩包（内置双核与全量 105 套规则） |
| `CubeX-Test-v8.9-linux-amd64.zip` | Linux amd64 (PVE/LXC/Ubuntu/Debian) | 87.48 MB | 完整独立开箱包（内含主程序、运维工具、双核引擎及 105 套规则） |
| `CubeX-Test-v8.9-linux-arm64.zip` | Linux arm64 (树莓派/香橙派/ARM软路由) | 80.16 MB | ARM64 完整归档发布包 |
| `cubexd-linux-amd64` | Linux amd64 | 42.29 MB | 纯单二进制控制面（代码加固版，用于在线平滑热更新） |
| `cubexd-linux-arm64` | Linux arm64 | 38.00 MB | ARM64 单二进制核心控制面（代码加固版） |
| `cubex-linux-amd64` | Linux amd64 | 16.99 MB | 纯命令行运维工具（终端直接运行） |
| `cubex-linux-arm64` | Linux arm64 | 15.25 MB | ARM64 纯命令行运维工具（终端直接运行） |
| `setup-routeros-v7.rsc` | 软路由终端编排脚本 | 17.50 KB | 软路由管理终端复制即用一键自动编排脚本 |
| `setup-openwrt.sh` | 开源软路由系统一键安装脚本 | 23.17 KB | 开源软路由系统原生适配安装脚本 |
| `install_pve_lxc.sh` | 容器一键脚本 | 10.48 KB | 极简容器一键安装脚本 |
| `install.sh` | 通用一键安装脚本 | 12.30 KB | 通用一键部署主脚本 |
"""


def gh_api(token, method, path, data=None):
    url = f"https://api.github.com/repos/{REPO}{path}"
    headers = [
        "-H", f"Authorization: token {token}",
        "-H", "Accept: application/vnd.github.v3+json",
        "-H", "User-Agent: CubeX-Release-Agent",
    ]
    cmd = ["curl.exe", "-s", "-x", PROXY, "-X", method] + headers

    temp_file = None
    if data:
        temp_file = tempfile.NamedTemporaryFile("w", delete=False, encoding="utf-8")
        json.dump(data, temp_file, ensure_ascii=False)
        temp_file.close()
        cmd.extend(["-H", "Content-Type: application/json; charset=utf-8", "-d", f"@{temp_file.name}"])

    cmd.append(url)
    try:
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
        output = res.stdout
    finally:
        if temp_file and os.path.exists(temp_file.name):
            os.remove(temp_file.name)

    status_cmd = ["curl.exe", "-s", "-o", "NUL", "-w", "%{http_code}", "-x", PROXY, "-X", method] + headers
    if data:
        status_cmd.extend(["-H", "Content-Type: application/json", "-d", json.dumps(data)])
    status_cmd.append(url)
    status_res = subprocess.run(status_cmd, capture_output=True, text=True, timeout=30)
    try:
        code = int(status_res.stdout.strip())
    except Exception:
        code = 200

    return code, output


def get_assets_map(token, release_id):
    code, body = gh_api(token, "GET", f"/releases/{release_id}/assets")
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
    elif name.endswith(".sh") or name.endswith(".rsc"):
        content_type = "text/plain"

    print(f"🚀 [正在上传] {name} ({round(size / 1024 / 1024, 2)} MB)...", flush=True)

    cmd = [
        "curl.exe",
        "--http1.1",
        "-x", PROXY,
        "-X", "POST",
        "-H", f"Authorization: token {token}",
        "-H", f"Content-Type: {content_type}",
        "-H", "Expect:",
        "--speed-time", "180",
        "--speed-limit", "512",
        "--data-binary", f"@{file_path}",
        upload_url,
    ]

    for attempt in range(1, 4):
        res = subprocess.run(cmd, capture_output=True, text=True, timeout=600)
        out = res.stdout.strip()
        try:
            asset_info = json.loads(out)
            if asset_info.get("state") == "uploaded" and asset_info.get("size") == size:
                print(f"   ✅ [上传成功] {name} (ID: {asset_info.get('id')})", flush=True)
                return True
        except Exception:
            pass

        assets = get_assets_map(token, release_id)
        if name in assets and assets[name].get("size") == size and assets[name].get("state") == "uploaded":
            print(f"   ✅ [校验成功] {name} 云端字节校验通过！", flush=True)
            return True

        if name in assets and assets[name].get("size") != size:
            print(f"   ⚠️ 字节不完整，正在清理残片并准备重试 (尝试 {attempt}/3)...", flush=True)
            gh_api(token, "DELETE", f"/releases/assets/{assets[name]['id']}")
            time.sleep(3)

        print(f"   ⚠️ 上传异常，第 {attempt} 次重试中...", flush=True)
        time.sleep(3)

    return False


def main():
    token = get_github_token()
    if not token:
        print("❌ 未能获取 GITHUB_TOKEN，请配置环境变量。")
        sys.exit(1)

    print("======================================================================")
    print(f"🚀 开始执行 CubeX {TAG} GitHub Release 生产发布流水线")
    print("======================================================================")

    # 1. 检查 Release 是否存在
    code, body = gh_api(token, "GET", f"/releases/tags/{TAG}")
    rel_id = None
    upload_url_tpl = None

    if code == 200:
        rel = json.loads(body)
        rel_id = rel["id"]
        upload_url_tpl = rel["upload_url"]
        print(f"ℹ️ Release {TAG} 已存在 (ID: {rel_id})，正在同步更新发布说明与标题...", flush=True)
        code, body = gh_api(token, "PATCH", f"/releases/{rel_id}", {
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
            "draft": False,
            "prerelease": False,
        })
        if code != 200:
            print(f"❌ 更新 Release 失败: {body}")
            sys.exit(1)
    elif code == 404:
        print(f"ℹ️ Release {TAG} 不存在，准备创建...", flush=True)
        code, body = gh_api(token, "POST", "/releases", {
            "tag_name": TAG,
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
            "draft": False,
            "prerelease": False,
        })
        if code != 201:
            print(f"❌ 创建 Release 失败: {body}")
            sys.exit(1)
        rel = json.loads(body)
        rel_id = rel["id"]
        upload_url_tpl = rel["upload_url"]
        print(f"✅ 成功创建 Release #{rel_id} ({TAG})", flush=True)
    else:
        print(f"❌ 查询 Release 状态异常: HTTP {code}")
        sys.exit(1)

    # 2. 上传 release 下的所有文件
    files = sorted([f for f in RELEASE_DIR.iterdir() if f.is_file()])
    print(f"\n📦 找到 {len(files)} 个待上传资产，开始逐一处理...", flush=True)

    for f in files:
        assets = get_assets_map(token, rel_id)
        if f.name in assets:
            cur_a = assets[f.name]
            if cur_a.get("size") == f.stat().st_size and cur_a.get("state") == "uploaded":
                print(f"   ✅ 资产 {f.name} 已存在且字节完全一致 ({cur_a.get('size')} 字节)，跳过上传。", flush=True)
                continue
            else:
                print(f"   - 资产 {f.name} 字节不一致 (本地 {f.stat().st_size} vs 云端 {cur_a.get('size')})，正在清理旧版本...", flush=True)
                gh_api(token, "DELETE", f"/releases/assets/{cur_a['id']}")
                time.sleep(2)

        ok = upload_asset(token, rel_id, upload_url_tpl, f)
        if not ok:
            print(f"❌ 资产 {f.name} 上传失败，终止发版。", flush=True)
            sys.exit(1)

    print(f"\n🎉 全部 {len(files)} 个资产上传并校验完成！", flush=True)

    # 3. 运行 10 周期存留红线
    print("\n🛡️ 正在执行全平台 10 版本存留清理门禁...", flush=True)
    enforce_script = SCRIPT_DIR / "enforce_10_version_retention.py"
    subprocess.run([sys.executable, str(enforce_script)], check=True)
    print("✅ 全平台 10 版本存留守护完成！", flush=True)


if __name__ == "__main__":
    main()
