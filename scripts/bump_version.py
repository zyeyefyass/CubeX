#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import sys
import re

TARGET_NUM = sys.argv[1] if len(sys.argv) > 1 else "6.5"
TARGET_VER = f"Test v{TARGET_NUM}"
TARGET_TAG = f"Test-v{TARGET_NUM}"
TARGET_WEB = f"{TARGET_NUM}.0"

def replace_in_file(path, pattern, repl):
    with open(path, 'r', encoding='utf-8') as f:
        content = f.read()
    new_content = re.sub(pattern, repl, content)
    if new_content != content:
        with open(path, 'w', encoding='utf-8') as f:
            f.write(new_content)
        print(f"✓ Updated {path}")
    else:
        print(f"- No change in {path}")

# 1. cmd/cubexd/main.go
replace_in_file("cmd/cubexd/main.go", r'version\s*=\s*"Test v6\.[0-9]+"', f'version = "{TARGET_VER}"')
replace_in_file("cmd/cubexd/main.go", r'srv\.mcpServer\s*=\s*mcp\.NewServer\(srv,\s*"Test v6\.[0-9]+"\)', f'srv.mcpServer = mcp.NewServer(srv, "{TARGET_VER}")')

# 2. internal/ros/updater.go
replace_in_file("internal/ros/updater.go", r'func \(m \*Manager\) getLocalPanelVersion\(\) string \{\s*return "Test v6\.[0-9]+"', f'func (m *Manager) getLocalPanelVersion() string {{\n\treturn "{TARGET_VER}"')
replace_in_file("internal/ros/updater.go", r'latestPanel\s*=\s*"Test v6\.[0-9]+"', f'latestPanel = "{TARGET_VER}"')

# 3. meta.json
replace_in_file("meta.json", r'"version":\s*"Test v6\.[0-9]+"', f'"version": "{TARGET_VER}"')

# 4. scripts/package_release.ps1
replace_in_file("scripts/package_release.ps1", r'\$Version\s*=\s*"Test-v6\.[0-9]+"', f'$Version = "{TARGET_TAG}"')

# 5. scripts/package_release.sh
replace_in_file("scripts/package_release.sh", r'VERSION="Test-v6\.[0-9]+"', f'VERSION="{TARGET_TAG}"')

# 6. web/package.json
replace_in_file("web/package.json", r'"version":\s*"6\.[0-9]+\.[0-9]+"', f'"version": "{TARGET_WEB}"')

# 7. web/src/App.vue fallbacks
with open("web/src/App.vue", "r", encoding="utf-8") as f:
    vue_text = f.read()
vue_text = re.sub(r"'Test v6\.[0-9]+'", f"'{TARGET_VER}'", vue_text)
with open("web/src/App.vue", "w", encoding="utf-8") as f:
    f.write(vue_text)
print("✓ Updated web/src/App.vue fallback versions")

print(f"\n🎉 Successfully bumped version to {TARGET_VER} ({TARGET_TAG}) across all files!")
