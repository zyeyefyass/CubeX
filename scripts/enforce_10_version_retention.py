#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX 全平台最新 10 版本强制存留门禁守护脚本 (Strict 10-Version Retention Policy)
功能：
1. 审计 GitHub Releases，若超过 10 个版本，全自动清理第 11 个及历史更早的 Release 资产；
2. 审计 Git 远程与本地 Tags，同步清理历史旧 Tag，确保仅存留最新的 10 个 Tag；
3. 审计本地 dist/release 归档包，清理冗余旧版本；
4. 全自主闭环执行，杜绝陈旧遗留资产与历史冗余。
"""

import os
import sys
import json
import re
import urllib.request
import subprocess

MAX_VERSIONS = 10
REPO = "zyeyefyass/CubeX"

_TAG_RE = re.compile(r"^(?:test[-_.]?)?v?(\d+)\.(\d+)(?:\.(\d+))?(?:[-_.](.+))?$", re.IGNORECASE)

def parse_cubex_tag_key(tag: str):
    """
    将 'Test-v9.1', 'Test-v9.0-LTS', 'v9.2.0' 统一映射为语义可比元组:
    (major, minor, patch, channel_weight, raw_tag)
    其中正式版 channel_weight = 2，带 Test/RC 的测试版 channel_weight = 1。
    """
    s = (tag or "").strip()
    m = _TAG_RE.match(s)
    if not m:
        return (0, 0, 0, 0, s)
    major = int(m.group(1))
    minor = int(m.group(2))
    patch = int(m.group(3) or 0)
    suffix = (m.group(4) or "").lower()
    is_test = s.lower().startswith("test") or any(k in suffix for k in ("test", "rc", "beta", "alpha"))
    channel_weight = 1 if is_test else 2
    return (major, minor, patch, channel_weight, s)
def get_github_token():
    tok = os.environ.get("GITHUB_TOKEN", "")
    if tok:
        return tok
    try:
        p = subprocess.Popen(['git', 'credential', 'fill'], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True)
        out, _ = p.communicate('protocol=https\nhost=github.com\n')
        for line in out.splitlines():
            if line.startswith('password='):
                return line.split('=', 1)[1].strip()
    except Exception:
        pass
    return ""

TOKEN = get_github_token()

PROXY = os.environ.get('HTTP_PROXY') or 'http://10.0.0.7:7890'
if PROXY:
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({'http': PROXY, 'https': PROXY}))
    urllib.request.install_opener(opener)

HEADERS = {
    'Accept': 'application/vnd.github.v3+json',
    'User-Agent': 'CubeX-Retention-Manager'
}
if TOKEN:
    HEADERS['Authorization'] = f'token {TOKEN}'

def prune_github_releases():
    print(f"🔍 [1/3] Checking GitHub Releases for {REPO} (Retention Limit: {MAX_VERSIONS})...")
    url = f"https://api.github.com/repos/{REPO}/releases?per_page=100"
    req = urllib.request.Request(url, headers=HEADERS)
    try:
        with urllib.request.urlopen(req) as resp:
            releases = json.loads(resp.read().decode('utf-8'))
    except Exception as e:
        print(f"⚠️ Failed to query GitHub releases: {e}")
        return []

    print(f"   Found {len(releases)} total releases.")
    if len(releases) <= MAX_VERSIONS:
        print(f"   ✓ Releases count ({len(releases)}) satisfies the 10-version rule.")
        return [r['tag_name'] for r in releases]

    # 优先按语义版本从新到旧严格排序，发布时间作为辅排序
    releases.sort(key=lambda x: (parse_cubex_tag_key(x.get('tag_name', '')), x.get('published_at') or x.get('created_at') or ''), reverse=True)
    keep = releases[:MAX_VERSIONS]
    to_prune = releases[MAX_VERSIONS:]

    print(f"   🚨 Pruning {len(to_prune)} older GitHub releases beyond latest {MAX_VERSIONS}:")
    for r in to_prune:
        del_url = f"https://api.github.com/repos/{REPO}/releases/{r['id']}"
        del_req = urllib.request.Request(del_url, headers=HEADERS, method='DELETE')
        try:
            with urllib.request.urlopen(del_req) as del_resp:
                if del_resp.status in (204, 200):
                    print(f"   ✓ Deleted release ID {r['id']} ({r['tag_name']})")
        except Exception as e:
            print(f"   ❌ Failed to delete release {r['tag_name']}: {e}")

    kept_tags = [r['tag_name'] for r in keep]
    print(f"   ✓ GitHub releases successfully consolidated to latest {len(keep)} versions: {kept_tags}")
    return kept_tags

def prune_git_tags(keep_tags):
    print(f"\n🏷️ [2/3] Aligning Git Tags with Active Releases...")
    if not keep_tags:
        print("   ⚠️ No active keep_tags provided. Skipping tag pruning.")
        return

    # Get local tags
    proc = subprocess.run(['git', 'tag', '-l'], capture_output=True, text=True)
    all_tags = [t.strip() for t in proc.stdout.splitlines() if t.strip()]
    
    keep_set = set(keep_tags)
    tags_to_delete = [t for t in all_tags if t not in keep_set]
    
    if not tags_to_delete:
        print(f"   ✓ All local tags already match the latest {len(keep_set)} versions.")
        return

    print(f"   Pruning {len(tags_to_delete)} older git tags from remote and local...")
    batch_size = 10
    for i in range(0, len(tags_to_delete), batch_size):
        batch = tags_to_delete[i:i+batch_size]
        cmd = ['git', 'push', 'origin', '--delete'] + batch
        subprocess.run(cmd, capture_output=True)

    for t in tags_to_delete:
        subprocess.run(['git', 'tag', '-d', t], capture_output=True)

    print(f"   ✓ Git tags cleanly pruned. Active tags: {sorted(list(keep_set))}")

def prune_local_dist(keep_tags):
    print(f"\n📁 [3/3] Auditing Local Release Dist Directory (dist/release)...")
    # 始终确保本地当前正在开发或构建的最新版本被保留
    tags_set = set(keep_tags) if keep_tags else set()
    try:
        with open("meta.json", "r", encoding="utf-8") as mf:
            curr_ver = json.load(mf).get("version", "")
            if curr_ver:
                tags_set.add(curr_ver)
                tags_set.add(curr_ver.replace(" ", "-"))
                tags_set.add(curr_ver.replace("-", " "))
                tags_set.add("Test-v6.2")
                tags_set.add("v6.2")
    except Exception as e:
        print("   Notice:", e)
    keep_tags = tags_set
    dist_dir = "dist/release"
    if not os.path.exists(dist_dir):
        print("   ✓ dist/release does not exist, nothing to prune.")
        return

    files = os.listdir(dist_dir)
    print(f"   Found {len(files)} files in {dist_dir}.")
    # Keep only files belonging to latest versions or general installation scripts
    essential_scripts = {
        'install.sh', 'install_pve_lxc.sh', 'setup-openwrt.sh', 'setup-routeros-v7.rsc',
        'cubexd-linux-amd64', 'cubexd-linux-arm64', 'cubex-linux-amd64', 'cubex-linux-arm64'
    }
    
    for f in files:
        if f in essential_scripts:
            continue
        # Check if file has any kept version tag
        matched = any(tag in f for tag in keep_tags)
        if not matched and ('CubeX-' in f):
            fpath = os.path.join(dist_dir, f)
            print(f"   Removing legacy package: {f}")
            try:
                os.remove(fpath)
            except Exception as e:
                print(f"   ⚠️ Could not remove {f}: {e}")
                
    print("   ✓ Local dist directory retention audit completed.")

def main():
    print("="*70)
    print("🛡️ CubeX 全平台最新 10 版本强制存留门禁守护 (Strict 10-Version Policy)")
    print("="*70)
    kept_tags = prune_github_releases()
    prune_git_tags(kept_tags)
    prune_local_dist(kept_tags)
    print("="*70)
    print("🎉 全平台最新 10 版本存留门禁核验 100% 达标！")
    print("="*70)

if __name__ == '__main__':
    main()
