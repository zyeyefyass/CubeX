#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX 原生双栈与极速热切版 GitHub 发布自动化脚本
- 创建/更新 Release（全中文精简大白话说明 + 资产清单表）
- 统一通过 curl.exe --http1.1 -x PROXY 禁用 Expect: 100-continue 直传资产，支持断点续传与字节校验
- 上传完毕后自动执行全平台 10 版本存留红线脚本
"""
import argparse
import json
import os
import re
import socket
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT_DIR = SCRIPT_DIR.parent
RELEASE_DIR = ROOT_DIR / "dist" / "release"
sys.path.insert(0, str(SCRIPT_DIR))

from enforce_10_version_retention import get_github_token, get_dockerhub_token  # noqa: E402

REPO = "zyeyefyass/CubeX"

def detect_proxy():
    env = os.environ.get("CUBEX_RELEASE_PROXY") or os.environ.get("HTTPS_PROXY") or os.environ.get("HTTP_PROXY")
    if env:
        if env.strip().upper() == "DIRECT":
            return ""
        return env
    for host in ["127.0.0.1"]:
        try:
            s = socket.socket()
            s.settimeout(0.3)
            s.connect((host, 7890))
            s.close()
            return f"http://{host}:7890"
        except Exception:
            pass
    return ""

PROXY = detect_proxy()

def get_tag():
    try:
        with open(ROOT_DIR / "meta.json", "r", encoding="utf-8") as f:
            v = json.load(f).get("version", "v9.93")
            if not v.startswith("v"):
                v = f"v{v}"
            return v
    except Exception:
        return "v9.93"

TAG = get_tag()

RELEASE_TITLE = f"CubeX {TAG} · DNS重写持久化 · 静默非ROS日志 · 测速与防绕过加固 · 网络中枢对比度优化 · 安全直通暂停"

RELEASE_BODY = f"""1. **修复 DNS 重写单条添加持久化**：新增单条域名解析或局域网 A 记录后立即原子落盘并同步双核心，刷新界面与重启服务不再丢失记录。
2. **静默非 ROS 底座日志刷屏**：未填写路由凭据或运行在独立模式、爱快、开源软路由底座时自动跳过相关巡检，不再产生误导性告警刷屏。
3. **优化应用分流节点测速超时**：放宽测速时间窗口并增加双地址回退探测，改善自建 Hysteria2 等协议节点测速误判超时问题，结果即时保存。
4. **修复纯 IP 节点下 Fake-IP 绕过隐患**：纯 IP 节点或空节点列表不再触发核心空域名通配直出，同时补齐直连解析地址端口拆分，保障分流准确。
5. **提升 PC 端「网络中枢」对比度**：优化拓扑条、设备卡片、筛选条与文字边框对比度，深浅色界面均清晰省眼。
6. **新增安全直通「一键暂停代理 / 恢复运行」入口**：暂停时保持控制台与 DNS 解析在线并返回真实公网地址全直连放行，不改主路由也不导致全家断网，并与三大底座「撤回主路由规则」明确区分。

---

### 📦 资产下载与校验清单 (Release Assets)

> [!TIP]
> **各平台底座发布物对照表**：请根据您的宿主底座与硬件架构选择对应的下载资产（完整包内置双核与全量 105 套规则）：

| 发布物文件名 | 适用宿主与架构 | 说明与用途 |
| :--- | :--- | :--- |
| `cubex_fnos_{TAG}_amd64.fpk` | 飞牛 fnOS (x86_64) | 飞牛私有云官方应用中心原生套件安装包 |
| `cubex_fnos_{TAG}_arm64.fpk` | 飞牛 fnOS (aarch64) | 飞牛私有云 ARM64 官方应用中心套件包 (适配 OECT / 树莓派 / RK3566) |
| `CubeX-{TAG}-ros-amd64.tar` | 软路由系统 (x86 软路由) | 软路由系统原生未压缩容器镜像（支持管理终端拖入直装） |
| `CubeX-{TAG}-ros-arm64.tar` | 软路由系统 (RB5009/CCR/ARM) | 软路由系统 ARM64 原生容器镜像（适配 64 位 ARM 硬路由） |
| `CubeX-{TAG}-docker-amd64.tar.gz` | Docker / 爱快 / 飞牛 / NAS / PVE | x86 容器离线镜像直装导入压缩包（内置双核与全量 105 套规则） |
| `CubeX-{TAG}-docker-arm64.tar.gz` | Docker / 树莓派 / ARM64 NAS | ARM64 容器离线镜像直装导入压缩包（内置双核与全量 105 套规则） |
| `CubeX-{TAG}-linux-amd64.zip` | 通用 Linux (x86_64 / Debian / PVE) | 完整离线发布归档包（含纯净版双核心、规则集与安装脚本） |
| `CubeX-{TAG}-linux-arm64.zip` | 通用 Linux (ARM64 / aarch64) | ARM64 完整离线归档包（含纯净版双核心、规则集与安装脚本） |
| `cubexd-linux-amd64` | Linux x86_64 (单二进制) | 后端控制面主守护进程单二进制（支持原地替换平滑热升级） |
| `cubexd-linux-arm64` | Linux ARM64 (单二进制) | ARM64 后端控制面主守护进程单二进制（支持原地替换平滑热升级） |
| `cubex-linux-amd64` | Linux x86_64 (CLI) | 命令行终端运维管理工具单二进制 |
| `cubex-linux-arm64` | Linux ARM64 (CLI) | ARM64 命令行终端运维管理工具单二进制 |
| `install_ros_container.rsc` | RouterOS v7 容器 | RouterOS 原生容器一键部署与看门狗配置脚本 |
| `install_openwrt_docker.sh` | OpenWrt Docker | OpenWrt / iStoreOS 容器一键安装脚本 |
| `setup-routeros-v7.rsc` | RouterOS 终端 | RouterOS 旁路策略路由与看门狗一键编排脚本 |
| `setup-openwrt.sh` | OpenWrt 终端 | OpenWrt / iStoreOS 原生安装与联动脚本 |
| `install_pve_lxc.sh` | Proxmox VE | PVE 容器一键创建与迁移升级脚本 |
| `install.sh` | 通用一键安装脚本 | 通用主机一键部署与平滑升级主脚本 |
| `SHA256SUMS` | 全平台校验文件 | 全部 19 项发布产物完整性 SHA256 校验清单 |
"""


def gh_api(token, method, path, data=None):
    url = f"https://api.github.com/repos/{REPO}{path}"
    headers = [
        "-H", f"Authorization: token {token}",
        "-H", "Accept: application/vnd.github.v3+json",
        "-H", "User-Agent: CubeX-Release-Agent",
    ]
    curl_bin = r"C:\Program Files\Git\mingw64\bin\curl.exe"
    if not os.path.exists(curl_bin):
        curl_bin = "curl.exe"
    cmd = [curl_bin, "-s", "-w", "\n%{http_code}", "-X", method] + headers
    if PROXY:
        cmd.extend(["-x", PROXY])

    temp_file = None
    if data:
        temp_file = tempfile.NamedTemporaryFile("w", delete=False, encoding="utf-8")
        json.dump(data, temp_file, ensure_ascii=True)
        temp_file.close()
        cmd.extend(["-H", "Content-Type: application/json; charset=utf-8", "-d", f"@{temp_file.name}"])

    cmd.append(url)
    try:
        res = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=90)
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


def _upload_via_openssl(token: str, upload_url: str, file_path: Path, content_type: str) -> tuple[int, str, str]:
    """使用 Python OpenSSL http.client 分块流式上传，规避 Windows Schannel 大文件断流。"""
    import http.client
    from urllib.parse import urlparse

    parsed = urlparse(upload_url)
    host = parsed.hostname
    port = parsed.port or 443
    path_q = parsed.path + (f"?{parsed.query}" if parsed.query else "")
    total_size = file_path.stat().st_size

    if PROXY:
        p_url = urlparse(PROXY)
        conn = http.client.HTTPSConnection(p_url.hostname, p_url.port or 8080, timeout=120)
        conn.set_tunnel(host, port)
    else:
        conn = http.client.HTTPSConnection(host, port, timeout=120)

    try:
        conn.putrequest("POST", path_q)
        conn.putheader("Authorization", f"token {token}")
        conn.putheader("Content-Type", content_type)
        conn.putheader("Content-Length", str(total_size))
        conn.putheader("User-Agent", "CubeX-Release-Agent")
        conn.putheader("Accept", "application/vnd.github.v3+json")
        conn.endheaders()

        sent = 0
        next_report = 15 * 1024 * 1024
        t0 = time.time()
        with open(file_path, "rb") as fh:
            while True:
                chunk = fh.read(65536)
                if not chunk:
                    break
                conn.send(chunk)
                sent += len(chunk)
                if sent >= next_report or sent == total_size:
                    elapsed = max(0.1, time.time() - t0)
                    mb_sent = sent / 1024 / 1024
                    mb_tot = total_size / 1024 / 1024
                    kbps = sent / 1024 / elapsed
                    print(f"     -> OpenSSL 流式已发送 {mb_sent:.1f}/{mb_tot:.1f} MB ({kbps:.0f} KB/s)", flush=True)
                    next_report += 15 * 1024 * 1024

        resp = conn.getresponse()
        body = resp.read().decode("utf-8", errors="replace")
        return resp.status, body, ""
    except Exception as exc:
        return 0, "", str(exc)
    finally:
        try:
            conn.close()
        except Exception:
            pass


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

    curl_bin = r"C:\Program Files\Git\mingw64\bin\curl.exe"
    if not os.path.exists(curl_bin):
        curl_bin = "curl.exe"

    cmd = [
        curl_bin,
        "--http1.1",
        "-s", "-S",
        "-w", "\n%{http_code}",
        "-L",
        "--connect-timeout", "25",
        "--speed-limit", "10240",
        "--speed-time", "60",
        "--max-time", "900",
        "-X", "POST",
        "-H", f"Authorization: token {token}",
        "-H", f"Content-Type: {content_type}",
        "-H", "Expect:",
        "--data-binary", f"@{file_path}",
    ]
    if PROXY:
        cmd.extend(["-x", PROXY])
    cmd.append(upload_url)

    for attempt in range(1, 5):
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

        use_openssl = (size >= 20 * 1024 * 1024) or (attempt >= 3)
        if use_openssl:
            print(f"   - 第 {attempt}/4 次传输 (OpenSSL 流式)...", flush=True)
            status_code, _, err_msg = _upload_via_openssl(token, upload_url, file_path, content_type)
            ret_code = 0 if status_code in (200, 201) else 1
            http_code = str(status_code) if status_code else "000"
        else:
            print(f"   - 第 {attempt}/4 次传输 (curl)...", flush=True)
            res = subprocess.run(cmd, capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=960)
            lines = res.stdout.strip().splitlines()
            http_code = lines[-1].strip() if lines else ""
            ret_code = res.returncode
            err_msg = res.stderr[:150]

        # Wait for GitHub background storage finalization
        print("   - 等待 GitHub 云端入库终审校验...", flush=True)
        success = False
        max_waits = 16 if (ret_code == 0 and http_code in ("200", "201")) else 8
        for wait_idx in range(max_waits):
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

        print(f"   ⚠️ 第 {attempt} 次校验未通过: code={ret_code}, http={http_code}, err={err_msg}", flush=True)
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
    else:
        lc, lb = gh_api(token, "GET", "/releases?per_page=20")
        if lc == 200:
            for item in json.loads(lb):
                if item.get("draft") and (
                    item.get("tag_name") == TAG
                    or item.get("name", "").startswith(f"CubeX {TAG}")
                ):
                    release = item
                    print(f"   ✓ 找到已存在的 Draft Release (ID: {release['id']})")
                    break
    if release:
        update_payload = {
            "tag_name": TAG,
            "target_commitish": "main",
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
        }
        gh_api(token, "PATCH", f"/releases/{release['id']}", update_payload)
    else:
        print(f"   - 创建新 Release (Draft 状态): {TAG} ...")
        create_payload = {
            "tag_name": TAG,
            "target_commitish": "main",
            "name": RELEASE_TITLE,
            "body": RELEASE_BODY,
            "draft": True,
            "prerelease": False,
        }
        code, body = gh_api(token, "POST", "/releases", create_payload)
        if code in (200, 201):
            release = json.loads(body)
            print(f"   ✅ 成功创建 Draft Release (ID: {release['id']})")
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
        f"CubeX-{TAG}-docker-arm64.tar.gz",
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

    print(f"✅ {len(expected_assets)} 项发布资产全部就绪，开始上传至 GitHub Draft Release (ID: {release_id})...\n")

    ordered_assets = sorted(expected_assets, key=lambda n: (RELEASE_DIR / n).stat().st_size)
    small_assets = [n for n in ordered_assets if (RELEASE_DIR / n).stat().st_size < 20 * 1024 * 1024]
    large_assets = [n for n in ordered_assets if (RELEASE_DIR / n).stat().st_size >= 20 * 1024 * 1024]

    failed = []
    for item in small_assets:
        if not upload_asset(token, release_id, upload_url_tpl, RELEASE_DIR / item):
            failed.append(item)
    if large_assets:
        for item in large_assets:
            if not upload_asset(token, release_id, upload_url_tpl, RELEASE_DIR / item):
                failed.append(item)

    if failed:
        print(f"\n❌ 上传失败列表 ({len(failed)} 项): {failed}")
        sys.exit(1)

    print(f"\n🔍 [终审核验] 逐一核验 GitHub 云端资产状态 (state: uploaded & 字节数一致)...")
    final_assets = get_assets_map(token, release_id)
    verify_failed = False
    for item in expected_assets:
        p = RELEASE_DIR / item
        exp_sz = p.stat().st_size
        if item not in final_assets:
            print(f"   ❌ 缺失云端资产: {item}")
            verify_failed = True
            continue
        cur = final_assets[item]
        state = cur.get("state")
        actual_sz = cur.get("size")
        if state != "uploaded" or actual_sz != exp_sz:
            print(f"   ❌ 资产异常: {item} (state={state}, size={actual_sz}/{exp_sz})")
            verify_failed = True
        else:
            print(f"   ✓ {item}: state=uploaded, size={actual_sz:,} 字节 (PASS)")

    if verify_failed:
        print("\n🚨 资产终审核验未通过，Release 保持 Draft 状态，中止正式发布！")
        sys.exit(1)

    print(f"\n🎉 全部 {len(expected_assets)} 项资产在 Draft 状态下核验 100% 满分通过！")
    print("🚀 正在将 Release 由 Draft 切换为正式发布 (draft: false)...")
    patch_code, patch_res = gh_api(token, "PATCH", f"/releases/{release_id}", {"draft": False})
    if patch_code == 200:
        pub_rel = json.loads(patch_res)
        print(f"✅ 正式发布成功！Release URL: {pub_rel.get('html_url')}")
    else:
        print(f"❌ 切换正式发布失败 (HTTP {patch_code}): {patch_res}")
        sys.exit(1)

    print("\n🐳 执行 Docker Hub 仓库说明页自动同步...")
    sync_dockerhub_description(TAG)

    print("\n🧹 执行全平台 10 周期存留红线清算...")
    enforce_script = SCRIPT_DIR / "enforce_10_version_retention.py"
    subprocess.run([sys.executable, str(enforce_script)], check=True)
    print("✅ 10 周期存留红线清算完成！发布流程圆满收官！")


DOCKERHUB_MAX_BYTES = 25000
_FORBIDDEN_WORDS = (
    "科学上网", "出海", "翻墙", "梯子", "机场", "偷跑", "后门",
    "绝对安全", "加密通道", "WireGuard", "wireguard",
    "自我迭代", "自我演化", "自主进化",
)


def _extract_changelog_highlights(tag: str) -> str:
    """从 CHANGELOG.md 提取指定版本的核心条目（自动回退至当前 RELEASE_BODY）。"""
    cl_path = ROOT_DIR / "CHANGELOG.md"
    if cl_path.exists():
        try:
            text = cl_path.read_text(encoding="utf-8")
            ver_clean = tag.lstrip("v")
            pat = re.compile(
                rf"^##\s+\[(?:v?{re.escape(ver_clean)})\]\s*[^\n]*\n(.*?)(?=\n---|\n##\s+\[|\Z)",
                re.MULTILINE | re.DOTALL,
            )
            m = pat.search(text)
            if m:
                lines = []
                for ln in m.group(1).splitlines():
                    s = ln.strip()
                    if not s or s.startswith("### "):
                        continue
                    lines.append(ln)
                if lines:
                    return "\n".join(lines).strip()
        except Exception:
            pass
    return RELEASE_BODY.split("\n---\n", 1)[0].strip()


def sanitize_and_cap_dockerhub_text(text: str, max_bytes: int = DOCKERHUB_MAX_BYTES) -> str:
    """强制脱敏扫描（零敏感词/零国旗图）并按 UTF-8 字节上限安全截断。"""
    for w in _FORBIDDEN_WORDS:
        if w in text:
            raise ValueError(f"Docker Hub 说明命中禁词红线: {w}")
    # 过滤国旗区域指示符 Emoji (U+1F1E6..U+1F1FF)
    cleaned = re.sub(r"[\U0001F1E6-\U0001F1FF]{2}", "", text)
    raw = cleaned.encode("utf-8")
    if len(raw) <= max_bytes:
        return cleaned
    ellipsis = "\n\n> 内容已按平台字节上限自动截断，完整说明请查阅 GitHub Releases。"
    limit = max_bytes - len(ellipsis.encode("utf-8"))
    cut = raw[:limit].decode("utf-8", errors="ignore")
    if "\n" in cut:
        cut = cut.rsplit("\n", 1)[0]
    return cut.rstrip() + ellipsis


def build_dockerhub_description(tag: str, highlights: str = "") -> tuple[str, str]:
    """构建对齐 README.md、无重复主标题、全合规常青版 Docker Hub 短描述与完整 Overview。"""
    ver = tag if tag.startswith("v") else f"v{tag}"
    hl = (highlights or _extract_changelog_highlights(ver)).strip()
    short_raw = f"CubeX {ver} · 双核共生网关 · 网关DNS永指主路由 · 断流逃生 · 多架构兼容"
    # Docker Hub repository short description 严格限制 <= 100 UTF-8 字节
    short_bytes = short_raw.encode("utf-8")
    short_desc = short_bytes[:96].decode("utf-8", errors="ignore").rstrip() if len(short_bytes) > 96 else short_raw

    full_desc = f"""# CubeX · 全自动宿主共生双核网关（{ver}）

> **核心架构卖点：终端网关和 DNS 永远指向主路由，全屋永不断线。**
> 无需改动手机、电脑或电视的任何默认网关设置；结合 Fake-IP（`198.18.0.0/15`）精准分流、双核心（`sing-box v1.14+` ⇄ `mihomo`）毫秒热切熔断，以及主路由看门狗（RouterOS / 爱快 iKuai / OpenWrt）断流逃生与自愈回切机制，彻底告别传统旁路由升级、重启或宕机导致的全屋断网隐患。

- **GitHub 官方仓库与发布页**：[https://github.com/zyeyefyass/CubeX](https://github.com/zyeyefyass/CubeX) ｜ [Releases 完整安装包与校验下载](https://github.com/zyeyefyass/CubeX/releases/latest)
- **Web 控制台默认端口**：`http://<宿主机IP>:2026`

---

## 本版核心更新（{ver}）

{hl}

---

## 为什么选择 CubeX？（架构对比与性能基线）

| 对比维度 | 传统全包旁路由（改网关/全流量接管） | CubeX 宿主共生零感网关 |
| :--- | :--- | :--- |
| **终端接入设置** | 需逐台改动手机、电脑默认网关或 DHCP | **100% 零修改**：所有设备网关与 DNS 永远指向主路由 |
| **故障与重启影响** | 旁路关机、升级或核心异常即全家断网 | **看门狗秒级逃生**：异常自动切回主路由直连，恢复自动回切 |
| **国内流量路径** | 全部绕行旁路由内核，白白损耗转发带宽 | **国内直出主路由芯片**，仅 `198.18.0.0/15` 虚拟映射走分流通道 |
| **物理内存开销** | 常驻 512MB ~ 2GB，小内存主机易 OOM | 纯 Go 单二进制控制面，静置 ~15MB、双核稳态 ~30MB~40MB |
| **核心防泄露调度** | 单核故障无热备，常见 DNS 回环或泄露 | `sing-box` ⇄ `mihomo` 毫秒级热切，严格拦截明文解析泄露 |

---

## 硬件指令集、老内核与挂载规范

1. **多架构通用兼容（`linux/amd64` & `linux/arm64`）**：
   - `amd64` 容器镜像内置官方 **`-compatible-` 通用指令集**核心，稳定运行于 Intel Celeron **J4125** / N100 等无 AVX2 指令集的老旧 x86_64 CPU；
   - `arm64` 镜像原生适配树莓派、ARM64 软路由与各类 ARM NAS 设备。
2. **老旧内核自动适配**：
   - 针对群晖 DSM 7.x（Linux `4.4` 内核）等老内核环境自动切换兼容协议栈；启动前自动预检 `/dev/net/tun` 与宿主 DNS 自环状态。
3. **容器核心路径规范（必读）**：
   - 配置文件与规则集目录：容器内 **`/app/etc`**
   - 持久化状态与历史账本目录：容器内 **`/app/data`**
   - 必须使用 `--network host`、`--cap-add=NET_ADMIN` 并映射 `--device /dev/net/tun:/dev/net/tun`。

---

## 各宿主平台部署指南（直接复制即可运行）

### 一、 飞牛 fnOS 部署（原生 `.fpk` 套件 / 单行 Compose）

#### 方式 1：应用中心手动安装 `.fpk`（推荐）
1. 前往 [GitHub Releases](https://github.com/zyeyefyass/CubeX/releases/latest) 下载对应架构包：`cubex_fnos_{ver}_amd64.fpk`（x86_64）或 `cubex_fnos_{ver}_arm64.fpk`（ARM64）。
2. 登录飞牛桌面 → **应用中心 → 手动安装** 上传 `.fpk`，或通过 SSH 执行终端命令行安装：

```bash
sudo appcenter-cli install-fpk cubex_fnos_{ver}_amd64.fpk
```

#### 方式 2：飞牛单行一键 Compose 部署
直接拉取官方对齐最新路径（`/app/etc`、`/app/data`）的编排模版并启动：

```bash
curl -fsSL https://raw.githubusercontent.com/zyeyefyass/CubeX/main/scripts/docker/docker-compose-fnos.yml -o docker-compose.yml && docker compose up -d
```

安装完成后打开飞牛桌面图标或访问 `http://<飞牛NAS-IP>:2026` 即可进入控制台；支持在【局域网设备】中将 NAS 设为影音直连模式，实现大文件满速直出与刮削加速协同。

---

### 二、 群晖 NAS（DSM 7）与通用 Docker / Compose 部署

#### 第 1 步（群晖 DSM 必看）：激活 `/dev/net/tun` 虚拟网卡驱动
群晖默认未自动创建 TUN 设备，建议在「控制面板 → 任务计划」中添加开机触发 root 脚本（或 SSH 首次执行）：

```bash
ls -l /dev/net/tun || {{
  insmod /lib/modules/tun.ko
  mkdir -p /dev/net
  mknod /dev/net/tun c 10 200
  chmod 600 /dev/net/tun
}}
```

#### 第 2 步：`docker run` 命令行一键启动（群晖 / 通用 Linux）

```bash
docker run -d \\
  --name cubex \\
  --restart unless-stopped \\
  --network host \\
  --privileged \\
  --cap-add=NET_ADMIN \\
  --cap-add=NET_RAW \\
  --device /dev/net/tun:/dev/net/tun \\
  -v /volume1/docker/cubex/etc:/app/etc \\
  -v /volume1/docker/cubex/data:/app/data \\
  -v /etc/localtime:/etc/localtime:ro \\
  -e TZ=Asia/Shanghai \\
  -e GOMEMLIMIT=128MiB \\
  fyasshole/cubex:latest
```

#### 第 3 步：`docker-compose.yml` 完整编排模版

```yaml
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
      - ./etc:/app/etc
      - ./data:/app/data
      - /etc/localtime:/etc/localtime:ro
    environment:
      - TZ=Asia/Shanghai
      - CUBEX_ENV=production
      - GOMEMLIMIT=128MiB
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### 无外网直连环境：Docker 离线镜像导入
前往 [GitHub Releases](https://github.com/zyeyefyass/CubeX/releases/latest) 下载 `CubeX-{ver}-docker-amd64.tar.gz`（或 `arm64.tar.gz`），上传至宿主机后执行：

```bash
docker load -i CubeX-{ver}-docker-amd64.tar.gz
```

导入后镜像标签为 `fyasshole/cubex:latest` 与 `cubex:latest`（兼容不同离线包命名），按上方 `docker run` 或 Compose 命令启动即可。

---

### 三、 OpenWrt / iStoreOS 容器与联动部署

#### 方式 1：在线一键全自动脚本（Macvlan-Shim + 看门狗）

```bash
sh -c "$(curl -fsSL https://gh-proxy.com/https://raw.githubusercontent.com/zyeyefyass/CubeX/main/scripts/install_openwrt_docker.sh)"
```

#### 方式 2：纯本地离线导入（无外网推荐）
从 Releases 下载 `CubeX-{ver}-docker-amd64.tar.gz`（或 `arm64.tar.gz`）与 `setup-openwrt.sh` 上传至 OpenWrt 的 `/tmp` 目录后执行：

```bash
cd /tmp && sh setup-openwrt.sh
```

脚本将自动导入离线镜像、建立 Macvlan 独立旁路容器、打通宿主通信桥接、下发 `198.18.0.0/15` 回指路由并注入 `cubex-watchdog` 自动逃生与恢复看门狗。部署完毕访问控制台默认端口 `http://<CubeX容器IP>:2026`。

---

### 四、 MikroTik RouterOS v7 / 爱快 iKuai / Proxmox VE（LXC）

- **MikroTik RouterOS v7 容器**：下载 `CubeX-{ver}-ros-amd64.tar`（或 `arm64.tar`）拖入 WinBox `Files` 目录，在 Terminal 导入一键脚本（详见 [Releases](https://github.com/zyeyefyass/CubeX/releases/latest) 中的 `install_ros_container.rsc` 与 `setup-routeros-v7.rsc`）：
  ```routeros
  /tool fetch url="https://gh-proxy.com/https://raw.githubusercontent.com/zyeyefyass/CubeX/main/install_ros_container.rsc" mode=https dst-path=install.rsc; /import install.rsc
  ```
- **爱快 iKuai Docker**：在爱快后台【镜像管理】导入 `CubeX-{ver}-docker-amd64.tar.gz`，新建 Macvlan 容器（勾选开机自启与特权模式，分别挂载到 `/app/etc` 与 `/app/data`），随后进入控制台填爱快管理凭据一键自愈即可。
- **Proxmox VE（PVE）隔离 CT 容器（强烈推荐）**：在 PVE 节点 Shell 执行：
  ```bash
  bash -c "$(curl -fsSL https://gh-proxy.com/https://raw.githubusercontent.com/zyeyefyass/CubeX/main/install_pve_lxc.sh)"
  ```

---

## 启动后 3 步开箱流程

1. **访问控制台**：浏览器打开 `http://<CubeX所在IP>:2026`。
2. **宿主共生与立即自愈**：在【宿主共生】填写主路由（RouterOS / 爱快 / OpenWrt）凭据并点击 **【立即自愈】**；如使用小米、华硕、TP-Link 等家用路由器，只需在主路由后台新增一条静态路由 `198.18.0.0/15 → <CubeX IP>`，并将局域网 DHCP 首选 DNS 指向 CubeX IP（备用 DNS 建议留空防抢答）。
3. **导入配置并核验运行**：导入订阅或自建节点并选择策略组出站节点；如宿主机启用了 DHCP 续租，请确认其自身 `/etc/resolv.conf` 指向主路由或公共 DNS。

---

- **完整发布资产、SHA256 校验与版本历史**：[GitHub Releases 官方发布页](https://github.com/zyeyefyass/CubeX/releases)
"""
    return short_desc, sanitize_and_cap_dockerhub_text(full_desc, DOCKERHUB_MAX_BYTES)


def sync_dockerhub_description(tag: str, highlights: str = "") -> bool:
    """调用 Docker Hub API 更新仓库 short description 与 full_description 并回查确认。"""
    d_user = os.environ.get("DOCKERHUB_USERNAME") or "fyasshole"
    d_tok = get_dockerhub_token()
    if not d_tok:
        print("   ⚠️ 未找到 DOCKERHUB_TOKEN，跳过 Docker Hub 说明页同步。")
        return False

    short_desc, full_desc = build_dockerhub_description(tag, highlights)
    byte_len = len(full_desc.encode("utf-8"))
    print(f"   -> 说明文本脱敏校验通过，UTF-8 字节数: {byte_len} / {DOCKERHUB_MAX_BYTES}")

    opener = urllib.request.build_opener(
        urllib.request.ProxyHandler({"http": PROXY, "https": PROXY})
    ) if PROXY else urllib.request.build_opener()

    login_req = urllib.request.Request(
        "https://hub.docker.com/v2/users/login/",
        data=json.dumps({"username": d_user, "password": d_tok}).encode("utf-8"),
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with opener.open(login_req, timeout=20) as r:
        jwt = json.loads(r.read().decode("utf-8"))["token"]

    repo_url = f"https://hub.docker.com/v2/repositories/{d_user}/cubex/"
    patch_payload = json.dumps(
        {"description": short_desc, "full_description": full_desc},
        ensure_ascii=True,
    ).encode("utf-8")
    patch_req = urllib.request.Request(
        repo_url,
        data=patch_payload,
        headers={
            "Authorization": f"Bearer {jwt}",
            "Content-Type": "application/json",
        },
        method="PATCH",
    )
    with opener.open(patch_req, timeout=25) as r:
        patch_status = r.status
        patch_resp = json.loads(r.read().decode("utf-8"))

    # 回查 GET 确认已生效
    with opener.open(urllib.request.Request(repo_url, method="GET"), timeout=20) as r:
        get_status = r.status
        verify_data = json.loads(r.read().decode("utf-8"))

    verified_desc = verify_data.get("full_description") or ""
    verified_bytes = len(verified_desc.encode("utf-8"))
    ver_tag = tag if tag.startswith("v") else f"v{tag}"
    if patch_status != 200 or get_status != 200 or ver_tag not in verified_desc:
        raise RuntimeError(f"Docker Hub 回查校验异常: patch={patch_status}, get={get_status}, resp={patch_resp}")

    print(f"   ✅ Docker Hub Overview 更新并回查确认成功！PATCH HTTP {patch_status}, GET HTTP {get_status}, 最终字节数: {verified_bytes}")
    return True


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="CubeX GitHub Release & Docker Hub Sync Tool")
    parser.add_argument("--sync-dockerhub-only", action="store_true", help="仅同步更新 Docker Hub 仓库说明页")
    parser.add_argument("--tag", default="", help="指定同步版本号（默认读取 meta.json）")
    args = parser.parse_args()
    if args.sync_dockerhub_only:
        target_tag = args.tag.strip() or TAG
        if not target_tag.startswith("v"):
            target_tag = f"v{target_tag}"
        ok = sync_dockerhub_description(target_tag)
        sys.exit(0 if ok else 1)
    main()
