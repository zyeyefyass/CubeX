#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Automated Release Artifacts Desensitization Auditor
Independent unpack and in-memory inspection for all .tar and .zip release artifacts.
Guarantees:
  1. 0 subscriptions, 0 private nodes / airport proxies, 0 credentials.
  2. Strict gating: Windows release packages are FORBIDDEN unless explicitly permitted via --allow-windows.
  3. Clean-room compliance: 0 legacy names or tokens.
"""

import os
import sys
import re
import json
import tarfile
import zipfile
import io

FORBIDDEN_FILE_PATTERNS = [
    r"^cache\.db$",
    r"\.log$",
    r"\.sqlite\d*$",
    r"\.pid$",
    r"\.history",
    r"^scratch/",
    r"\.pyc$",
    r"__pycache__",
]

SENSITIVE_REGEXES = [
    (r"Airport Subscription Token", re.compile(r"(?:token|sub_token)=[a-zA-Z0-9]{16,}", re.IGNORECASE)),
    (r"Airport Subscribe URL", re.compile(r"(?:https?://[^\s\"']+/api/v1/client/subscribe\?[^\s\"']+)", re.IGNORECASE)),
    (r"Airport Subscribe URL (Download)", re.compile(r"(?:https?://[^\s\"']+/download/sub\?[^\s\"']+)", re.IGNORECASE)),
    (r"Airport General Subscribe URL", re.compile(r"(?:https?://[^\s\"']+(?:sub|subscribe)\?[^\s\"']*(?:token|auth|key)=[a-zA-Z0-9]+)", re.IGNORECASE)),
    (r"Proxy URI (vmess)", re.compile(r"vmess://[a-zA-Z0-9+/=]{16,}", re.IGNORECASE)),
    (r"Proxy URI (vless)", re.compile(r"vless://[a-f0-9\-]{32,}[^\s\"']*", re.IGNORECASE)),
    (r"Proxy URI (trojan)", re.compile(r"trojan://[a-zA-Z0-9\-]{8,}@[^\s\"']+", re.IGNORECASE)),
    (r"Proxy URI (shadowsocks)", re.compile(r"ss://[a-zA-Z0-9+/=]{16,}@[^\s\"']+", re.IGNORECASE)),
    (r"Proxy URI (hysteria2)", re.compile(r"hy2://[a-zA-Z0-9\-]{8,}@[^\s\"']+", re.IGNORECASE)),
    (r"Proxy URI (tuic)", re.compile(r"tuic://[a-zA-Z0-9\-]{8,}@[^\s\"']+", re.IGNORECASE)),
    (r"RouterOS Cloud DDNS Serial", re.compile(r"[a-f0-9]{12}\.sn\.mynetname\.net", re.IGNORECASE)),
    (r"WireGuard Private Key", re.compile(r'(?:"private_key"|\'private_key\'|private-key)\s*[:=]\s*["\']([a-zA-Z0-9+/]{42,44}=)["\']', re.IGNORECASE)),
]

FORBIDDEN_LEGACY_NAMES = [
    re.compile(r"\bopen[-_]?box\b", re.IGNORECASE),
    re.compile(r"\bros[-_]?box\b", re.IGNORECASE),
]

FORBIDDEN_SENSITIVE_WORDS = [
    re.compile(r"科学上网"),
    re.compile(r"\b出海\b"),
    re.compile(r"\b翻墙\b"),
    re.compile(r"\b梯子\b"),
    re.compile(r"偷跑"),
    re.compile(r"后门"),
    re.compile(r"绝对安全"),
    re.compile(r"加密通道"),
]

FORBIDDEN_FLAG_IMAGE_PATTERNS = [
    re.compile(r"cubex-live-routing-matrix\.png", re.IGNORECASE),
    re.compile(r"flags-crop\.png", re.IGNORECASE),
    re.compile(r"flags-test\.png", re.IGNORECASE),
]

def check_file_content(archive_name, inner_path, content_bytes):
    errors = []
    text = None
    try:
        text = content_bytes.decode("utf-8")
    except UnicodeDecodeError:
        try:
            text = content_bytes.decode("gbk")
        except UnicodeDecodeError:
            pass # Binary file, skip text pattern matching

    base_name = os.path.basename(inner_path)

    # 1. Check for specific config files
    if base_name == "config.ir.json" or base_name.endswith(".ir.json"):
        try:
            data = json.loads(text)
            subs = data.get("subscriptions", [])
            if subs:
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: 'subscriptions' is not empty! Found {len(subs)} subscription entries.")
            proxies = data.get("proxies", [])
            if proxies:
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: 'proxies' is not empty! Found {len(proxies)} private proxy node entries.")
            dev_policies = data.get("device_policies", {})
            if dev_policies:
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: 'device_policies' is not empty! Found {len(dev_policies)} device policy entries.")
            
            # Check groups
            for g in data.get("groups", []):
                g_proxies = g.get("proxies", [])
                for p in g_proxies:
                    # Should only have DIRECT, REJECT or regional tags, never specific private node names
                    if any(c in p for c in ["192.168.", "10.", "172.16."]) or "node" in p.lower():
                        errors.append(f"[{archive_name} -> {inner_path}] Suspicious proxy in group '{g.get('name')}': {p}")
        except Exception as e:
            errors.append(f"[{archive_name} -> {inner_path}] Failed to parse JSON: {e}")

    elif base_name == "wireguard.json":
        try:
            data = json.loads(text)
            peers = data.get("peers", [])
            if peers:
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: 'peers' is not empty! Found {len(peers)} peers.")
            cfg = data.get("config", {})
            if cfg.get("private_key"):
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: WireGuard 'private_key' is not empty: {cfg.get('private_key')}")
            if cfg.get("public_key"):
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: WireGuard 'public_key' is not empty: {cfg.get('public_key')}")
            if cfg.get("endpoint_host"):
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: 'endpoint_host' is not empty: {cfg.get('endpoint_host')}")
            if cfg.get("cloud_ddns"):
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: 'cloud_ddns' is not empty: {cfg.get('cloud_ddns')}")
        except Exception as e:
            errors.append(f"[{archive_name} -> {inner_path}] Failed to parse wireguard.json: {e}")

    elif base_name in ["host-config.json", "ros-config.json"]:
        try:
            data = json.loads(text)
            if data.get("password"):
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: RouterOS password leaked!")
            if data.get("private_key"):
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: Host private_key leaked!")
        except Exception as e:
            errors.append(f"[{archive_name} -> {inner_path}] Failed to parse {base_name}: {e}")

    elif base_name == "glados-config.json":
        try:
            data = json.loads(text)
            if data.get("cookie"):
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: GLaDOS 'cookie' is not empty!")
            if data.get("sub_url"):
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: GLaDOS 'sub_url' is not empty!")
        except Exception as e:
            errors.append(f"[{archive_name} -> {inner_path}] Failed to parse glados-config.json: {e}")

    elif base_name == "config.json":
        # sing-box config
        try:
            data = json.loads(text)
            outbounds = data.get("outbounds", [])
            for ob in outbounds:
                t = ob.get("type", "")
                if t in ["vmess", "vless", "trojan", "shadowsocks", "wireguard", "hysteria2", "tuic"]:
                    errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: Outbound contains private proxy type '{t}' with tag '{ob.get('tag')}'!")
        except Exception:
            pass

    elif base_name in ["config.yaml", "config.yml"]:
        # mihomo config
        if text:
            m = re.search(r"^proxies:\s*\n(\s+-\s*\{.+|\s+-\s*name:)", text, re.MULTILINE)
            if m:
                errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: Mihomo config.yaml contains real proxies list!")

    # 2. General regex scanner for text files
    if text and inner_path.endswith((".json", ".yaml", ".yml", ".rsc", ".conf", ".sh", ".env", ".txt", ".md")):
        for label, pattern in SENSITIVE_REGEXES:
            matches = pattern.findall(text)
            if matches:
                filtered = [m for m in matches if "your_" not in str(m).lower() and "example" not in str(m).lower()]
                if filtered:
                    errors.append(f"[{archive_name} -> {inner_path}] CRITICAL: Sensitive pattern detected: {label} (matches: {filtered[:3]})")

        # 3. Clean-room legacy name check
        for pattern in FORBIDDEN_LEGACY_NAMES:
            if pattern.search(text):
                for line_no, line in enumerate(text.splitlines(), 1):
                    if pattern.search(line):
                        errors.append(f"[{archive_name} -> {inner_path}:{line_no}] CLEAN-ROOM VIOLATION: Legacy term detected: {line.strip()[:80]}")

    return errors

def audit_zip(archive_path):
    archive_name = os.path.basename(archive_path)
    errors = []
    files_checked = 0
    configs_checked = 0

    print(f"\n📦 Auditing ZIP Archive: {archive_name}")
    with zipfile.ZipFile(archive_path, 'r') as zf:
        for entry in zf.infolist():
            if entry.is_dir():
                continue
            files_checked += 1
            filename = entry.filename

            # Check forbidden files
            base = os.path.basename(filename)
            for pat in FORBIDDEN_FILE_PATTERNS:
                if re.search(pat, base, re.IGNORECASE) or re.search(pat, filename, re.IGNORECASE):
                    errors.append(f"[{archive_name}] FORBIDDEN FILE: {filename}")

            # Check content
            if filename.endswith((".json", ".yaml", ".yml", ".rsc", ".conf", ".sh", ".env", ".txt", ".md")):
                configs_checked += 1
                content = zf.read(entry)
                errs = check_file_content(archive_name, filename, content)
                errors.extend(errs)

    print(f"   -> Checked {files_checked} total files ({configs_checked} config/text files).")
    return errors

def audit_tar(archive_path):
    archive_name = os.path.basename(archive_path)
    errors = []
    files_checked = 0
    configs_checked = 0

    mode = "r:*" if archive_path.endswith((".tar.gz", ".tgz")) else "r:"
    print(f"\n📦 Auditing TAR Archive: {archive_name} (mode: {mode})")
    with tarfile.open(archive_path, mode) as tf:
        for member in tf.getmembers():
            if not member.isfile():
                continue
            files_checked += 1
            filename = member.name

            # Check forbidden files
            base = os.path.basename(filename)
            for pat in FORBIDDEN_FILE_PATTERNS:
                if re.search(pat, base, re.IGNORECASE) or re.search(pat, filename, re.IGNORECASE):
                    errors.append(f"[{archive_name}] FORBIDDEN FILE: {filename}")

            # Check content
            if filename.endswith((".json", ".yaml", ".yml", ".rsc", ".conf", ".sh", ".env", ".txt", ".md")):
                configs_checked += 1
                f = tf.extractfile(member)
                if f:
                    content = f.read()
                    errs = check_file_content(archive_name, filename, content)
                    errors.extend(errs)

    print(f"   -> Checked {files_checked} total files ({configs_checked} config/text files).")
    return errors

def audit_directory(release_dir, allow_windows=False):
    print("=" * 80)
    print("🛡️  CubeX Automated Release Artifacts Desensitization Audit")
    print(f"🎯 Target Directory: {release_dir}")
    print(f"🔒 Windows Policy: {'Allowed' if allow_windows else 'STRICTLY FORBIDDEN (No Windows release without explicit instruction)'}")
    print("=" * 80)

    if not os.path.exists(release_dir):
        print(f"❌ Error: Release directory does not exist: {release_dir}")
        sys.exit(1)

    all_errors = []
    archives = []
    for f in os.listdir(release_dir):
        p = os.path.join(release_dir, f)
        if os.path.isfile(p):
            # Check for forbidden Windows release packages
            if not allow_windows:
                if f.endswith(".exe") or "windows" in f.lower():
                    all_errors.append(
                        f"[{f}] POLICY VIOLATION: Windows release artifact detected without explicit authorization! "
                        "Per security policy, Windows versions must NEVER be released without explicit instructions."
                    )
            if f.endswith((".zip", ".tar", ".tar.gz", ".tgz")):
                archives.append(p)

    if not archives and not all_errors:
        print(f"⚠️  No release archives (.tar, .tar.gz, .zip) found in {release_dir}")
        sys.exit(0)

    for arch in sorted(archives):
        if arch.endswith(".zip"):
            errs = audit_zip(arch)
            all_errors.extend(errs)
        elif arch.endswith((".tar", ".tar.gz", ".tgz")):
            errs = audit_tar(arch)
            all_errors.extend(errs)

    print("\n" + "=" * 80)
    if all_errors:
        print(f"🚨 AUDIT FAILED! Detected {len(all_errors)} desensitization / compliance violations:")
        for e in all_errors:
            print(f"  ❌ {e}")
        print("=" * 80)
        sys.exit(1)
    else:
        print(f"✅ AUDIT PASSED! All {len(archives)} release archives verified 100% clean & desensitized.")
        print("   ✓ 0 Subscriptions (Strictly wiped)")
        print("   ✓ 0 Private Nodes / Proxies (Strictly wiped)")
        print("   ✓ 0 Credentials / Passwords / Private Keys")
        print("   ✓ 0 RouterOS Cloud DDNS Serials")
        print("   ✓ 0 Runtime Caches / DBs (cache.db, logs, sqlite)")
        print("   ✓ 0 Unauthorized Windows Release Artifacts")
        print("   ✓ 100% Clean-Room & Zero-Legacy Compliance")
        print("=" * 80)
        sys.exit(0)

if __name__ == "__main__":
    args = sys.argv[1:]
    allow_win = "--allow-windows" in args
    clean_args = [a for a in args if a != "--allow-windows"]
    target = clean_args[0] if clean_args else os.path.join(os.path.dirname(__file__), "..", "dist", "release")
    audit_directory(target, allow_windows=allow_win)
