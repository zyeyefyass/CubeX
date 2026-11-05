import os
import sys
import gzip
import base64
import re

LOADER_TEMPLATE = """#!/bin/sh
# ==============================================================================
# 🌌 CubeX Host-Symbiotic Zero-Perception Gateway
# Automated Deployment & Execution Engine (Test v4.2)
#
# Copyright (c) 2024-2026 CubeX Project. All Rights Reserved.
# Proprietary & Confidential Software. Unauthorized copying, reverse engineering,
# decompilation, or redistribution is strictly prohibited.
# ==============================================================================
set -e
_CUBEX_LOADER_VERSION="4.2"
_CUBEX_PAYLOAD="{payload}"

if command -v bash >/dev/null 2>&1; then
    _RUNNER="bash"
else
    _RUNNER="sh"
fi

_TMP_SCRIPT="/tmp/._cubex_run_$$.sh"
trap 'rm -f "$_TMP_SCRIPT" 2>/dev/null' EXIT INT TERM

if command -v base64 >/dev/null 2>&1 && command -v gunzip >/dev/null 2>&1; then
    echo "$_CUBEX_PAYLOAD" | base64 -d | gunzip > "$_TMP_SCRIPT"
elif command -v base64 >/dev/null 2>&1 && command -v gzip >/dev/null 2>&1; then
    echo "$_CUBEX_PAYLOAD" | base64 -d | gzip -d > "$_TMP_SCRIPT"
elif command -v openssl >/dev/null 2>&1 && command -v gzip >/dev/null 2>&1; then
    echo "$_CUBEX_PAYLOAD" | openssl base64 -d | gzip -d > "$_TMP_SCRIPT"
else
    echo "❌ [CubeX Protected Loader] Error: Missing base64 or gzip decoder on host." >&2
    exit 1
fi

chmod +x "$_TMP_SCRIPT"
"$_RUNNER" "$_TMP_SCRIPT" "$@"
_RET=$?
rm -f "$_TMP_SCRIPT" 2>/dev/null
exit $_RET
"""

def protect_shell_script(src_path, dst_path):
    with open(src_path, 'r', encoding='utf-8') as f:
        content = f.read()

    # Minify shell script safely: track Heredocs and strictly preserve all embedded Shebangs and templates
    lines = content.splitlines()
    clean_lines = []
    heredoc_stack = []
    heredoc_pattern = re.compile(r'<<-?\s*[\'"]?([A-Za-z0-9_]+)[\'"]?')

    for l in lines:
        s = l.strip()

        # If inside heredoc(s), preserve content bit-for-bit (NEVER strip comments/shebangs inside heredoc)
        if heredoc_stack:
            clean_lines.append(l)
            if s == heredoc_stack[-1]:
                heredoc_stack.pop()
            else:
                m = heredoc_pattern.search(l)
                if m:
                    heredoc_stack.append(m.group(1))
            continue

        if not s:
            continue

        # Check if line initiates a heredoc block
        m = heredoc_pattern.search(l)
        if m:
            heredoc_stack.append(m.group(1))
            clean_lines.append(l)
            continue

        # Strictly preserve ANY shebang (#!/sbin/openrc-run, #!/bin/sh, etc.)
        if s.startswith("#!"):
            clean_lines.append(l)
            continue

        # Strip standalone comments outside heredocs
        if s.startswith("#"):
            continue

        clean_lines.append(l)

    minified = "\n".join(clean_lines).encode('utf-8')

    # Security & Integrity Gating: verify no shebang was lost compared to source
    for check_sb in ["#!/sbin/openrc-run", "#!/bin/sh", "#!/usr/bin/env bash"]:
        if check_sb in content and check_sb.encode('utf-8') not in minified:
            raise RuntimeError(f"CRITICAL ERROR: Shebang '{check_sb}' in {src_path} was corrupted during minification!")

    compressed = gzip.compress(minified, compresslevel=9)
    b64_payload = base64.b64encode(compressed).decode('ascii')

    loader = LOADER_TEMPLATE.format(payload=b64_payload)
    with open(dst_path, 'w', encoding='utf-8', newline='\n') as f:
        f.write(loader)
    print(f"  ✓ Protected {src_path} -> {dst_path} ({len(content)} -> {len(loader)} bytes)")

def protect_ros_script(src_path, dst_path):
    with open(src_path, 'r', encoding='utf-8') as f:
        content = f.read()

    lines = content.splitlines()
    clean_lines = [
        "# 🌌 CubeX RouterOS Automated Deployment Engine (Test v4.1)",
        "# Proprietary & Confidential. All Rights Reserved."
    ]
    for l in lines:
        s = l.strip()
        if not s or s.startswith("#"):
            continue
        clean_lines.append(s)

    minified = "\n".join(clean_lines)
    with open(dst_path, 'w', encoding='utf-8', newline='\n') as f:
        f.write(minified)
    print(f"  ✓ Minified & Protected {src_path} -> {dst_path} ({len(content)} -> {len(minified)} bytes)")

def main():
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    release_dir = os.path.join(root, "dist", "release")
    os.makedirs(release_dir, exist_ok=True)

    print("🛡️ Protecting deployment scripts for Test v4.2...")
    # 1. OpenWrt (OpenWrt Busybox 默认未编译 base64 工具，必须保持原生纯 POSIX 脚本分发，确保 100% 免外部解码依赖)
    openwrt_src = os.path.join(root, "scripts", "openwrt", "setup-openwrt.sh")
    openwrt_dst = os.path.join(release_dir, "setup-openwrt.sh")
    import shutil
    shutil.copy2(openwrt_src, openwrt_dst)
    print(f"  ✓ Native Direct Script {openwrt_src} -> {openwrt_dst} ({os.path.getsize(openwrt_dst)} bytes)")

    # 2. Linux install.sh
    install_src = os.path.join(root, "scripts", "install.sh")
    install_dst = os.path.join(release_dir, "install.sh")
    protect_shell_script(install_src, install_dst)
    # Also write to root install.sh for curl -fsSL https://.../install.sh
    root_install = os.path.join(root, "install.sh")
    protect_shell_script(install_src, root_install)

    # 3. PVE LXC
    pve_src = os.path.join(root, "scripts", "install_pve_lxc.sh")
    pve_dst = os.path.join(release_dir, "install_pve_lxc.sh")
    protect_shell_script(pve_src, pve_dst)

    # 4. RouterOS .rsc
    ros_src = os.path.join(root, "scripts", "ros", "setup-routeros-v7.rsc")
    ros_dst = os.path.join(release_dir, "setup-routeros-v7.rsc")
    protect_ros_script(ros_src, ros_dst)

    print("🛡️ Script protection complete!")

if __name__ == '__main__':
    main()
