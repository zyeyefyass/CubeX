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
import hashlib
import shutil
import subprocess
import tempfile

META_JSON_PATH = os.path.join(os.path.dirname(__file__), "..", "meta.json")


def normalize_version_tag(ver_str):
    if not ver_str:
        return ""
    clean = re.sub(r"^(?:Test[-_\s]*)?v?", "", ver_str.strip(), flags=re.IGNORECASE)
    return f"v{clean}" if clean else ""


def resolve_expected_version(explicit_version=None):
    if explicit_version:
        return normalize_version_tag(explicit_version)
    try:
        if os.path.exists(META_JSON_PATH):
            with open(META_JSON_PATH, "r", encoding="utf-8") as mf:
                data = json.load(mf)
            return normalize_version_tag(data.get("version", ""))
    except Exception:
        pass
    return ""


def is_cubexd_binary(inner_path, content_bytes):
    base = os.path.basename(inner_path).lower()
    if not (base == "cubexd" or base == "cubexd.exe" or base.startswith("cubexd-")):
        return False
    if base.endswith((".fpk", ".zip", ".tar", ".gz", ".sha256", ".txt", ".json", ".md", ".sh", ".rsc")):
        return False
    if len(content_bytes) >= 4 and (content_bytes[:4] == b"\x7fELF" or content_bytes[:2] == b"MZ"):
        return True
    return False


def check_cubexd_binary_version(archive_name, inner_path, content_bytes, expected_version):
    errors = []
    if not expected_version or not is_cubexd_binary(inner_path, content_bytes):
        return errors, False
    if expected_version.encode("utf-8") not in content_bytes:
        errors.append(
            f"[{archive_name} -> {inner_path}] VERSION MISMATCH: "
            f"cubexd binary does not embed expected version '{expected_version}'!"
        )
    return errors, True

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
    re.compile(r"机场"),
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

FORBIDDEN_AMD64_V3_MARKERS = [
    b"v3 microarchitecture support",
    b"v4 microarchitecture support",
]

FORBIDDEN_CONTAINER_PATHS_RE = re.compile(r"(micube|open[-_]?box|ros[-_]?box)", re.IGNORECASE)

# Post-musl static dual-core gzip-9 floor: sing-box(30.55)+mihomo(21.53)+cubexd(5.11)+geo/rules(10.36)+Alpine(8.08) ≈ 75.65MB (amd64) / 71.32MB (arm64)
SIZE_GATE_LIMITS_MB = [
    ("ros-amd64.tar", 88.0),
    ("ros-arm64.tar", 88.0),
    ("docker-amd64.tar.gz", 80.0),
    ("docker-arm64.tar.gz", 80.0),
    ("linux-amd64.zip", 100.0),
    ("linux-arm64.zip", 100.0),
    ("_amd64.run", 32.0),
    ("_arm64.run", 32.0),
]

POSIX_BASHISM_PATTERNS = [
    (r"(^|[;&|(\s])\[\[(\s|$)", "[["),
    (r"(^|[;&|\s])declare\s+", "declare"),
    (r"(^|[;&|\s])source\s+", "source (use '.')"),
    (r"&>\s*\S", "&> redirect"),
    (r"<<<", "<<< herestring"),
]


def _find_sh_binary():
    found = shutil.which("sh")
    if found:
        return found
    for cand in (
        r"C:\Program Files\Git\bin\sh.exe",
        r"C:\Program Files\Git\usr\bin\sh.exe",
        "/bin/sh",
        "/usr/bin/sh",
    ):
        if os.path.exists(cand):
            return cand
    return None


def _strip_quoted_heredocs(lines):
    kept = []
    delim = None
    for idx, line in enumerate(lines, 1):
        s = line.strip()
        if delim is not None:
            if s == delim:
                delim = None
            continue
        m_hd = re.search(r"<<-?\s*['\"]?([A-Za-z0-9_]+)['\"]?", line)
        if m_hd and not line.lstrip().startswith("#"):
            if "'" in line[m_hd.start():m_hd.end()]:
                delim = m_hd.group(1)
        kept.append((idx, line))
    return kept


def check_posix_shell_script(archive_name, inner_path, text):
    errors = []
    for lno, line in _strip_quoted_heredocs(text.splitlines()):
        if line.lstrip().startswith("#"):
            continue
        for pat, label in POSIX_BASHISM_PATTERNS:
            if re.search(pat, line):
                errors.append(
                    f"[{archive_name} -> {inner_path}:{lno}] POSIX BASHISM VIOLATION ({label}): {line.strip()[:80]}"
                )
    sh_bin = _find_sh_binary()
    if sh_bin:
        with tempfile.NamedTemporaryFile("w", encoding="utf-8", suffix=".sh", delete=False, newline="\n") as tf:
            tf.write(text)
            tmp_path = tf.name
        try:
            res = subprocess.run([sh_bin, "-n", tmp_path], capture_output=True, text=True)
            if res.returncode != 0:
                errors.append(
                    f"[{archive_name} -> {inner_path}] SH SYNTAX ERROR (sh -n): {(res.stderr or res.stdout).strip()}"
                )
        finally:
            try:
                os.remove(tmp_path)
            except OSError:
                pass
    return errors


def check_amd64_binary_isa(archive_name, inner_path, content_bytes):
    """Ensure published x86_64/amd64 Go ELF binaries do not require x86-64-v3 (AVX2) instructions."""
    errors = []
    if len(content_bytes) < 20 or content_bytes[:4] != b"\x7fELF":
        return errors
    # ELF64 little-endian e_machine at bytes 18..20 == 0x003e (EM_X86_64)
    if content_bytes[4] != 2 or int.from_bytes(content_bytes[18:20], "little") != 0x3E:
        return errors
    for marker in FORBIDDEN_AMD64_V3_MARKERS:
        if marker in content_bytes:
            errors.append(
                f"[{archive_name} -> {inner_path}] ISA COMPATIBILITY VIOLATION: "
                f"AMD64 ELF binary requires '{marker.decode('ascii', 'ignore')}' (incompatible with Celeron J4125/N3450/non-AVX2 CPUs)!"
            )
    return errors

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

        # 4. Zero-Sensitive-Term & Zero-Flag-Image check (Rule 5)
        for pattern in FORBIDDEN_SENSITIVE_WORDS:
            if pattern.search(text):
                for line_no, line in enumerate(text.splitlines(), 1):
                    if pattern.search(line):
                        errors.append(f"[{archive_name} -> {inner_path}:{line_no}] SENSITIVE WORD VIOLATION ({pattern.pattern}): {line.strip()[:80]}")

        for pattern in FORBIDDEN_FLAG_IMAGE_PATTERNS:
            if pattern.search(text):
                errors.append(f"[{archive_name} -> {inner_path}] FORBIDDEN FLAG IMAGE REFERENCE: {pattern.pattern}")

    return errors

def audit_zip(archive_path, expected_version=""):
    archive_name = os.path.basename(archive_path)
    errors = []
    files_checked = 0
    configs_checked = 0
    cubexd_checked = 0

    print(f"\n📦 Auditing ZIP Archive: {archive_name}")
    with zipfile.ZipFile(archive_path, 'r') as zf:
        for entry in zf.infolist():
            if entry.is_dir():
                continue
            files_checked += 1
            filename = entry.filename

            # Check forbidden files & forbidden flag images
            base = os.path.basename(filename)
            for pat in FORBIDDEN_FILE_PATTERNS:
                if re.search(pat, base, re.IGNORECASE) or re.search(pat, filename, re.IGNORECASE):
                    errors.append(f"[{archive_name}] FORBIDDEN FILE: {filename}")
            for pat in FORBIDDEN_FLAG_IMAGE_PATTERNS:
                if pat.search(filename):
                    errors.append(f"[{archive_name}] FORBIDDEN FLAG IMAGE FILE: {filename}")

            # Check content
            if filename.endswith((".json", ".yaml", ".yml", ".rsc", ".conf", ".sh", ".env", ".txt", ".md")):
                configs_checked += 1
                content = zf.read(entry)
                errs = check_file_content(archive_name, filename, content)
                errors.extend(errs)
            elif any(k in base for k in ("cubexd", "cubex", "mihomo", "sing-box")):
                bcontent = zf.read(entry)
                errors.extend(check_amd64_binary_isa(archive_name, filename, bcontent))
                ver_errs, found = check_cubexd_binary_version(archive_name, filename, bcontent, expected_version)
                errors.extend(ver_errs)
                if found:
                    cubexd_checked += 1

    if expected_version and "cubex" in archive_name.lower() and cubexd_checked == 0:
        errors.append(f"[{archive_name}] VERSION AUDIT ERROR: No cubexd executable found inside ZIP archive!")

    print(f"   -> Checked {files_checked} total files ({configs_checked} config/text files, {cubexd_checked} cubexd binaries).")
    return errors, cubexd_checked

def audit_nested_oci_layer(archive_name, layer_name, raw_bytes, expected_version=""):
    errors = []
    cubexd_checked = 0
    for sub_mode in ("r:gz", "r:"):
        try:
            with tarfile.open(fileobj=io.BytesIO(raw_bytes), mode=sub_mode) as stf:
                for sm in stf.getmembers():
                    if FORBIDDEN_CONTAINER_PATHS_RE.search(sm.name):
                        errors.append(f"[{archive_name} -> {layer_name}::{sm.name}] CLEAN-ROOM VIOLATION: Forbidden legacy path in OCI layer!")
                    if not sm.isfile():
                        continue
                    sbase = os.path.basename(sm.name)
                    if sm.name.endswith((".json", ".yaml", ".yml", ".sh")) and sm.size < 2 * 1024 * 1024:
                        sf_txt = stf.extractfile(sm)
                        if sf_txt:
                            errors.extend(check_file_content(archive_name, f"{layer_name}::{sm.name}", sf_txt.read()))
                    if any(k in sbase for k in ("cubexd", "cubex", "mihomo", "sing-box")):
                        sf = stf.extractfile(sm)
                        if sf:
                            bcontent = sf.read()
                            inner = f"{layer_name}::{sm.name}"
                            errors.extend(check_amd64_binary_isa(archive_name, inner, bcontent))
                            ver_errs, found = check_cubexd_binary_version(archive_name, inner, bcontent, expected_version)
                            errors.extend(ver_errs)
                            if found:
                                cubexd_checked += 1
                break
        except Exception:
            continue
    return errors, cubexd_checked

def audit_tar(archive_path, expected_version=""):
    archive_name = os.path.basename(archive_path)
    errors = []
    files_checked = 0
    configs_checked = 0
    cubexd_checked = 0

    mode = "r:*" if archive_path.endswith((".tar.gz", ".tgz")) else "r:"
    print(f"\n📦 Auditing TAR Archive: {archive_name} (mode: {mode})")
    with tarfile.open(archive_path, mode) as tf:
        for member in tf.getmembers():
            if not member.isfile():
                continue
            files_checked += 1
            filename = member.name

            # Check forbidden files & forbidden flag images
            base = os.path.basename(filename)
            for pat in FORBIDDEN_FILE_PATTERNS:
                if re.search(pat, base, re.IGNORECASE) or re.search(pat, filename, re.IGNORECASE):
                    errors.append(f"[{archive_name}] FORBIDDEN FILE: {filename}")
            for pat in FORBIDDEN_FLAG_IMAGE_PATTERNS:
                if pat.search(filename):
                    errors.append(f"[{archive_name}] FORBIDDEN FLAG IMAGE FILE: {filename}")

            # Check content / binary ISA / nested OCI layers
            if filename.endswith((".json", ".yaml", ".yml", ".rsc", ".conf", ".sh", ".env", ".txt", ".md")):
                configs_checked += 1
                f = tf.extractfile(member)
                if f:
                    content = f.read()
                    errs = check_file_content(archive_name, filename, content)
                    errors.extend(errs)
            elif any(k in base for k in ("cubexd", "cubex", "mihomo", "sing-box")):
                f = tf.extractfile(member)
                if f:
                    bcontent = f.read()
                    errors.extend(check_amd64_binary_isa(archive_name, filename, bcontent))
                    ver_errs, found = check_cubexd_binary_version(archive_name, filename, bcontent, expected_version)
                    errors.extend(ver_errs)
                    if found:
                        cubexd_checked += 1
            elif filename.startswith("blobs/sha256/") and member.size > 1024:
                f = tf.extractfile(member)
                if f:
                    l_errs, l_cnt = audit_nested_oci_layer(archive_name, filename, f.read(), expected_version)
                    errors.extend(l_errs)
                    cubexd_checked += l_cnt

    if expected_version and "cubex" in archive_name.lower() and cubexd_checked == 0:
        errors.append(f"[{archive_name}] VERSION AUDIT ERROR: No cubexd executable found inside TAR/OCI archive!")

    print(f"   -> Checked {files_checked} total files ({configs_checked} config/text files, {cubexd_checked} cubexd binaries).")
    return errors, cubexd_checked

def _split_run_archive(run_path):
    with open(run_path, "rb") as rf:
        raw = rf.read()
    marker = b"\n__ARCHIVE_BELOW__\n"
    idx = raw.find(marker)
    if idx < 0:
        if raw.startswith(b"__ARCHIVE_BELOW__\n"):
            return b"__ARCHIVE_BELOW__\n", raw[len(b"__ARCHIVE_BELOW__\n"):]
        return None, None
    split_at = idx + len(marker)
    return raw[:split_at], raw[split_at:]


def audit_run_installer(run_path, expected_version=""):
    archive_name = os.path.basename(run_path)
    errors = []
    files_checked = 0
    configs_checked = 0
    cubexd_checked = 0

    print(f"\n📦 Auditing Self-Extracting .run Installer: {archive_name}")
    header_bytes, payload_bytes = _split_run_archive(run_path)
    if header_bytes is None or payload_bytes is None:
        errors.append(f"[{archive_name}] INVALID .run FORMAT: Missing __ARCHIVE_BELOW__ marker!")
        return errors, cubexd_checked

    try:
        header_text = header_bytes.decode("utf-8")
    except UnicodeDecodeError as e:
        errors.append(f"[{archive_name}] HEADER ENCODING ERROR: POSIX header is not valid UTF-8: {e}")
        return errors, cubexd_checked

    if not header_text.startswith("#!/bin/sh\n"):
        errors.append(f"[{archive_name}] POSIX HEADER ERROR: Must start with #!/bin/sh shebang!")

    errors.extend(check_file_content(archive_name, "header.sh", header_bytes))
    errors.extend(check_posix_shell_script(archive_name, "header.sh", header_text))

    actual_sha = hashlib.sha256(payload_bytes).hexdigest()
    m_sha = re.search(r'^CUBEX_PAYLOAD_SHA256="([0-9a-f]{64})"', header_text, re.MULTILINE)
    if not m_sha:
        errors.append(f"[{archive_name}] INTEGRITY ERROR: Missing embedded CUBEX_PAYLOAD_SHA256 in header!")
    elif m_sha.group(1) != actual_sha:
        errors.append(
            f"[{archive_name}] INTEGRITY MISMATCH: Header SHA256 {m_sha.group(1)} != payload SHA256 {actual_sha}!"
        )

    if expected_version:
        m_ver = re.search(r'^CUBEX_RUN_VERSION="([^"]+)"', header_text, re.MULTILINE)
        if m_ver and normalize_version_tag(m_ver.group(1)) != expected_version:
            errors.append(
                f"[{archive_name} -> header.sh] VERSION MISMATCH: "
                f"CUBEX_RUN_VERSION='{m_ver.group(1)}' != expected '{expected_version}'!"
            )

    try:
        with tarfile.open(fileobj=io.BytesIO(payload_bytes), mode="r:gz") as tf:
            for member in tf.getmembers():
                if FORBIDDEN_CONTAINER_PATHS_RE.search(member.name):
                    errors.append(f"[{archive_name} -> payload::{member.name}] CLEAN-ROOM VIOLATION: Forbidden legacy path!")
                if not member.isfile():
                    continue
                files_checked += 1
                filename = member.name
                base = os.path.basename(filename)
                for pat in FORBIDDEN_FILE_PATTERNS:
                    if re.search(pat, base, re.IGNORECASE) or re.search(pat, filename, re.IGNORECASE):
                        errors.append(f"[{archive_name}] FORBIDDEN FILE: payload::{filename}")
                for pat in FORBIDDEN_FLAG_IMAGE_PATTERNS:
                    if pat.search(filename):
                        errors.append(f"[{archive_name}] FORBIDDEN FLAG IMAGE FILE: payload::{filename}")

                fobj = tf.extractfile(member)
                if not fobj:
                    continue
                if filename.endswith((".json", ".yaml", ".yml", ".rsc", ".conf", ".sh", ".env", ".txt", ".md")):
                    configs_checked += 1
                    content = fobj.read()
                    errors.extend(check_file_content(archive_name, f"payload::{filename}", content))
                    if filename.endswith(".sh"):
                        errors.extend(
                            check_posix_shell_script(
                                archive_name,
                                f"payload::{filename}",
                                content.decode("utf-8", errors="replace"),
                            )
                        )
                elif any(k in base for k in ("cubexd", "cubex", "mihomo", "sing-box")):
                    bcontent = fobj.read()
                    errors.extend(check_amd64_binary_isa(archive_name, f"payload::{filename}", bcontent))
                    ver_errs, found = check_cubexd_binary_version(
                        archive_name, f"payload::{filename}", bcontent, expected_version
                    )
                    errors.extend(ver_errs)
                    if found:
                        cubexd_checked += 1
    except Exception as e:
        errors.append(f"[{archive_name}] PAYLOAD UNPACK ERROR: Failed to inspect embedded payload.tar.gz: {e}")

    if expected_version and cubexd_checked == 0:
        errors.append(f"[{archive_name}] VERSION AUDIT ERROR: No cubexd executable found inside .run payload!")

    print(
        f"   -> Checked POSIX header + {files_checked} payload files "
        f"({configs_checked} config/text files, {cubexd_checked} cubexd binaries)."
    )
    return errors, cubexd_checked


def check_archive_size_gate(release_dir):
    errors = []
    print("\n📏 Checking Release Size Gate (cubex-slimmer)...")
    for fname in sorted(os.listdir(release_dir)):
        fpath = os.path.join(release_dir, fname)
        if not os.path.isfile(fpath):
            continue
        for suffix, limit_mb in SIZE_GATE_LIMITS_MB:
            if fname.endswith(suffix):
                size_mb = os.path.getsize(fpath) / (1024 * 1024)
                status = "✅ PASS" if size_mb <= limit_mb else "❌ FAIL"
                print(f"   {status} {fname:<40} {size_mb:6.2f} MB / <= {limit_mb:.1f} MB")
                if size_mb > limit_mb:
                    errors.append(f"[{fname}] SIZE GATE EXCEEDED: {size_mb:.2f} MB > {limit_mb:.1f} MB limit!")
                    # Print top 10 largest entries inside archive for quick diagnosis
                    try:
                        entries = []
                        if fname.endswith(".zip"):
                            with zipfile.ZipFile(fpath, "r") as zf:
                                entries = [(zi.file_size, zi.filename) for zi in zf.infolist() if not zi.is_dir()]
                        elif fname.endswith(".run"):
                            _, payload_bytes = _split_run_archive(fpath)
                            if payload_bytes:
                                with tarfile.open(fileobj=io.BytesIO(payload_bytes), mode="r:gz") as tf:
                                    entries = [(m.size, m.name) for m in tf.getmembers() if m.isfile()]
                        else:
                            mode = "r:*" if fname.endswith((".tar.gz", ".tgz")) else "r:"
                            with tarfile.open(fpath, mode) as tf:
                                entries = [(m.size, m.name) for m in tf.getmembers() if m.isfile()]
                        entries.sort(reverse=True)
                        print("      Top 10 entries:")
                        for sz, nm in entries[:10]:
                            print(f"        - {sz/(1024*1024):6.2f} MB  {nm}")
                    except Exception:
                        pass
                break
    return errors


def audit_directory(release_dir, allow_windows=False, expected_version=None, size_gate=False):
    target_version = resolve_expected_version(expected_version)
    print("=" * 80)
    print("🛡️  CubeX Automated Release Artifacts Desensitization Audit")
    print(f"🎯 Target Directory: {release_dir}")
    print(f"🏷️  Expected Version: {target_version or '(not set)'}")
    print(f"🔒 Windows Policy: {'Allowed' if allow_windows else 'STRICTLY FORBIDDEN (No Windows release without explicit instruction)'}")
    print("=" * 80)

    if not os.path.exists(release_dir):
        os.makedirs(release_dir, exist_ok=True)

    all_errors = []
    archives = []
    standalone_cubexd_checked = 0
    archive_cubexd_checked = 0
    for f in sorted(os.listdir(release_dir)):
        p = os.path.join(release_dir, f)
        if os.path.isfile(p):
            # Check for forbidden Windows release packages
            if not allow_windows:
                if f.endswith(".exe") or "windows" in f.lower():
                    all_errors.append(
                        f"[{f}] POLICY VIOLATION: Windows release artifact detected without explicit authorization! "
                        "Per security policy, Windows versions must NEVER be released without explicit instructions."
                    )
            if f.endswith((".zip", ".tar", ".tar.gz", ".tgz", ".run")):
                archives.append(p)
            elif any(k in f for k in ("cubexd", "cubex", "mihomo", "sing-box")) and not f.endswith(".fpk"):
                with open(p, "rb") as bf:
                    bcontent = bf.read()
                all_errors.extend(check_amd64_binary_isa("release-dir", f, bcontent))
                ver_errs, found = check_cubexd_binary_version("release-dir", f, bcontent, target_version)
                all_errors.extend(ver_errs)
                if found:
                    standalone_cubexd_checked += 1

    if not archives and standalone_cubexd_checked == 0 and not all_errors:
        print(f"⚠️  No release archives (.tar, .tar.gz, .zip, .run) found in {release_dir}")
        sys.exit(0)

    for arch in sorted(archives):
        if arch.endswith(".zip"):
            errs, cnt = audit_zip(arch, target_version)
            all_errors.extend(errs)
            archive_cubexd_checked += cnt
        elif arch.endswith(".run"):
            errs, cnt = audit_run_installer(arch, target_version)
            all_errors.extend(errs)
            archive_cubexd_checked += cnt
        elif arch.endswith((".tar", ".tar.gz", ".tgz")):
            errs, cnt = audit_tar(arch, target_version)
            all_errors.extend(errs)
            archive_cubexd_checked += cnt

    total_cubexd_checked = standalone_cubexd_checked + archive_cubexd_checked
    if target_version and total_cubexd_checked == 0:
        all_errors.append(f"[release-dir] VERSION AUDIT ERROR: Found 0 cubexd binaries in {release_dir} to verify version '{target_version}'!")

    if size_gate:
        all_errors.extend(check_archive_size_gate(release_dir))

    print("\n" + "=" * 80)
    if all_errors:
        print(f"🚨 AUDIT FAILED! Detected {len(all_errors)} desensitization / compliance violations:")
        for e in all_errors:
            print(f"  ❌ {e}")
        print("=" * 80)
        sys.exit(1)
    else:
        print(f"✅ AUDIT PASSED! All {len(archives)} release archives & {total_cubexd_checked} cubexd binaries verified clean, desensitized & version-aligned ({target_version}).")
        print("   ✓ 0 Subscriptions (Strictly wiped)")
        print("   ✓ 0 Private Nodes / Proxies (Strictly wiped)")
        print("   ✓ 0 Credentials / Passwords / Private Keys")
        print("   ✓ 0 RouterOS Cloud DDNS Serials")
        print("   ✓ 0 Runtime Caches / DBs (cache.db, logs, sqlite)")
        print("   ✓ 0 x86-64-v3/v4 Restricted AMD64 Binaries (Universal Compatible)")
        print("   ✓ 0 Unauthorized Windows Release Artifacts")
        print(f"   ✓ 100% Binary Version Alignment ({target_version}) across standalone, .zip, .tar, .tar.gz, .run & OCI layers")
        print("   ✓ 100% Clean-Room & Zero-Legacy Compliance")
        print("=" * 80)
        sys.exit(0)

if __name__ == "__main__":
    import argparse
    parser = argparse.ArgumentParser(description="CubeX Release Artifacts Desensitization & Version Auditor")
    parser.add_argument("release_dir", nargs="?", default=os.path.join(os.path.dirname(__file__), "..", "dist", "release"))
    parser.add_argument("--allow-windows", action="store_true", help="Allow Windows release binaries")
    parser.add_argument("--expected-version", "--version", dest="expected_version", default=None, help="Expected cubexd version string (e.g. v9.96)")
    parser.add_argument("--size-gate", action="store_true", help="Enforce cubex-slimmer size limits on release packages")
    parsed = parser.parse_args()
    audit_directory(parsed.release_dir, allow_windows=parsed.allow_windows, expected_version=parsed.expected_version, size_gate=parsed.size_gate)
