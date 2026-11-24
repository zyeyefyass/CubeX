import subprocess
import os

env_base = os.environ.copy()
env_base['CGO_ENABLED'] = '0'

# ═══════════════════════════════════════════════════════════════════
# Phase 1: 纯净标准编译 (独立二进制：-trimpath -ldflags="-s -w -buildid="，杜绝 AST 膨胀与运行时反噬)
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
# Phase 2: 容器内置守护进程纯净编译 (用于容器/镜像内嵌，体积极简)
# ═══════════════════════════════════════════════════════════════════
container_targets = [
    ('amd64', './cmd/cubexd', 'dist/cubexd-container-amd64'),
    ('arm64', './cmd/cubexd', 'dist/cubexd-container-arm64'),
]

for arch, pkg, out in container_targets:
    env = env_base.copy()
    env['GOOS'] = 'linux'
    env['GOARCH'] = arch
    cmd = ['go', 'build', '-trimpath', '-ldflags=-s -w -buildid=', '-o', out, pkg]
    print(f"Compiling {out} ({arch}) standard (container-slim)...")
    res = subprocess.run(cmd, env=env, capture_output=True, text=True)
    if res.returncode == 0:
        size = os.path.getsize(out)
        print(f"  ✅ Succeeded: {size} bytes ({round(size/(1024*1024), 2)} MB)")
    else:
        print(f"  ❌ Failed: {res.stderr}")
        exit(1)

print("\n🎉 Phase 2: Container-slim cubexd binaries compiled!")
print("🎉 All compilation complete!")

