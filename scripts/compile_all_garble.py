import os
import shutil
import subprocess

env_base = os.environ.copy()
env_base['CGO_ENABLED'] = '0'

# ═══════════════════════════════════════════════════════════════════
# Phase 1: 纯净标准编译 (-trimpath -ldflags="-s -w -buildid="，杜绝 AST 膨胀)
# ═══════════════════════════════════════════════════════════════════
clean_targets = [
    ('amd64', './cmd/cubexd', 'dist/cubexd-linux-amd64'),
    ('amd64', './cmd/cubex',  'dist/cubex-linux-amd64'),
    ('arm64', './cmd/cubexd', 'dist/cubexd-linux-arm64'),
    ('arm64', './cmd/cubex',  'dist/cubex-linux-arm64'),
]

for arch, pkg, out in clean_targets:
    env = env_base.copy()
    env['GOOS'] = 'linux'
    env['GOARCH'] = arch
    cmd = ['go', 'build', '-trimpath', '-ldflags=-s -w -buildid=', '-o', out, pkg]
    print(f"Compiling {out} ({arch}) with pure flags...")
    res = subprocess.run(cmd, env=env, capture_output=True, text=True)
    if res.returncode == 0:
        size = os.path.getsize(out)
        print(f"  ✅ Succeeded: {size} bytes ({round(size/(1024*1024), 2)} MB)")
    else:
        print(f"  ❌ Failed: {res.stderr}")
        exit(1)

print("\n🎉 Phase 1: All 4 clean binaries compiled!")

# ═══════════════════════════════════════════════════════════════════
# Phase 2: 容器内置守护进程复用同步 (消除相同参数的重复二次编译)
# ═══════════════════════════════════════════════════════════════════
for src, dst in [
    ('dist/cubexd-linux-amd64', 'dist/cubexd-container-amd64'),
    ('dist/cubexd-linux-arm64', 'dist/cubexd-container-arm64'),
]:
    shutil.copy2(src, dst)
    print(f"  ✅ Synced container binary {dst} <- {src}")

print("🎉 All compilation complete!")

