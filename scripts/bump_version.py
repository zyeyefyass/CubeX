#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Atomic Version Management & Multi-File Consistency Synchronization
Ensures 100% atomic version alignment across all 9 core backend, frontend,
installer, and release script files.
"""
import sys
import os
import re
import subprocess

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
os.chdir(ROOT)

TARGET_NUM = sys.argv[1] if len(sys.argv) > 1 else "7.9"
TARGET_VER = f"Test v{TARGET_NUM}"
TARGET_TAG = f"Test-v{TARGET_NUM}"
TARGET_WEB = f"{TARGET_NUM}.0"

def replace_in_file(path, pattern, repl):
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    new_content = re.sub(pattern, repl, content)
    if new_content != content:
        with open(path, 'w', encoding='utf-8', newline='') as f:
            f.write(new_content)
        print(f"✓ Updated {path}")
    else:
        print(f"- No change in {path}")

print(f"🚀 Atomically bumping CubeX version to {TARGET_VER} ({TARGET_TAG})...")

# 1. cmd/cubexd/main.go
replace_in_file("cmd/cubexd/main.go", r'version\s*=\s*"Test v[0-9]+\.[0-9]+"', f'version = "{TARGET_VER}"')
replace_in_file("cmd/cubexd/main.go", r'srv\.mcpServer\s*=\s*mcp\.NewServer\(srv,\s*"Test v[0-9]+\.[0-9]+"\)', f'srv.mcpServer = mcp.NewServer(srv, "{TARGET_VER}")')

# 2. internal/ros/updater.go
replace_in_file("internal/ros/updater.go", r'func \(m \*Manager\) getLocalPanelVersion\(\) string \{\s*return "Test v[0-9]+\.[0-9]+"', f'func (m *Manager) getLocalPanelVersion() string {{\n\treturn "{TARGET_VER}"')
replace_in_file("internal/ros/updater.go", r'latestPanel\s*=\s*"Test v[0-9]+\.[0-9]+"', f'latestPanel = "{TARGET_VER}"')

# 3. meta.json
replace_in_file("meta.json", r'"version":\s*"Test v[0-9]+\.[0-9]+"', f'"version": "{TARGET_VER}"')

# 4. scripts/package_release.ps1
replace_in_file("scripts/package_release.ps1", r'\$Version\s*=\s*"Test-v[0-9]+\.[0-9]+"', f'$Version = "{TARGET_TAG}"')

# 5. scripts/package_release.sh
replace_in_file("scripts/package_release.sh", r'VERSION="Test-v[0-9]+\.[0-9]+"', f'VERSION="{TARGET_TAG}"')

# 6. web/package.json
replace_in_file("web/package.json", r'"version":\s*"[0-9]+\.[0-9]+\.[0-9]+"', f'"version": "{TARGET_WEB}"')

# 7. web/src/App.vue fallbacks
with open("web/src/App.vue", "r", encoding="utf-8") as f:
    vue_text = f.read()
vue_text = re.sub(r"'Test v[0-9]+\.[0-9]+'", f"'{TARGET_VER}'", vue_text)
with open("web/src/App.vue", "w", encoding="utf-8", newline='') as f:
    f.write(vue_text)
print("✓ Updated web/src/App.vue fallback versions")

# 8. install scripts fallback tags
replace_in_file("scripts/install.sh", r'REL_TAG="Test-v[0-9]+\.[0-9]+"', f'REL_TAG="{TARGET_TAG}"')
replace_in_file("scripts/install_pve_lxc.sh", r'REL_TAG="Test-v[0-9]+\.[0-9]+"', f'REL_TAG="{TARGET_TAG}"')

# 9. Regenerate install.sh using protect_script.py
protect_script = os.path.join(ROOT, "scripts", "protect_script.py")
if os.path.exists(protect_script):
    subprocess.run([sys.executable, protect_script], check=True)
    print("✓ Regenerated protected install.sh loader")

# 10. Verify consistency across all 9 core files
print("\n🔍 Verifying atomic version consistency across all 9 core files...")
files_to_verify = {
    'cmd/cubexd/main.go': r'version\s*=\s*"Test v([0-9]+\.[0-9]+)"',
    'internal/ros/updater.go': r'return\s*"Test v([0-9]+\.[0-9]+)"',
    'meta.json': r'"version":\s*"Test v([0-9]+\.[0-9]+)"',
    'web/package.json': r'"version":\s*"([0-9]+\.[0-9]+)\.0"',
    'scripts/package_release.ps1': r'\$Version\s*=\s*"Test-v([0-9]+\.[0-9]+)"',
    'scripts/package_release.sh': r'VERSION="Test-v([0-9]+\.[0-9]+)"',
    'scripts/install.sh': r'REL_TAG="Test-v([0-9]+\.[0-9]+)"',
    'install.sh': r'_CUBEX_LOADER_VERSION="([0-9]+\.[0-9]+)"',
    'scripts/install_pve_lxc.sh': r'REL_TAG="Test-v([0-9]+\.[0-9]+)"',
}

detected = {}
for p, pat in files_to_verify.items():
    with open(os.path.join(ROOT, p), 'r', encoding='utf-8', errors='ignore') as f:
        content = f.read()
    m = re.search(pat, content)
    detected[p] = m.group(1) if m else 'NOT_FOUND'

unique = set(detected.values())
if len(unique) == 1 and list(unique)[0] == TARGET_NUM:
    print(f"🎉 100% ATOMIC ALIGNMENT VERIFIED! All 9 files strictly set to {TARGET_VER} ({TARGET_TAG})!\n")
else:
    print(f"❌ VERSION DIVERGENCE DETECTED: {detected}")
    sys.exit(1)
