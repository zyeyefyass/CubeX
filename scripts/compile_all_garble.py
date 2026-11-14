import subprocess
import os

env_base = os.environ.copy()
env_base['CGO_ENABLED'] = '0'

garble = r'C:\Users\zyy\go\bin\garble.exe'

# ═══════════════════════════════════════════════════════════════════
# Phase 1: Garble 混淆编译 (用于 GitHub 单独发布的独立二进制，防逆向)
# ═══════════════════════════════════════════════════════════════════
garble_targets = [
    ('amd64', './cmd/cubexd', 'dist/cubexd-linux-amd64'),
    ('amd64', './cmd/cubex',  'dist/cubex-linux-amd64'),
    ('arm64', './cmd/cubexd', 'dist/cubexd-linux-arm64'),
    ('arm64', './cmd/cubex',  'dist/cubex-linux-arm64'),
]

for arch, pkg, out in garble_targets:
    env = env_base.copy()
    env['GOOS'] = 'linux'
    env['GOARCH'] = arch
    cmd = [garble, '-literals', '-tiny', 'build', '-trimpath', '-ldflags=-s -w -buildid=', '-o', out, pkg]
    print(f"Compiling {out} ({arch}) with Garble...")
    res = subprocess.run(cmd, env=env, capture_output=True, text=True)
    if res.returncode == 0:
        size = os.path.getsize(out)
        print(f"  ✅ Succeeded: {size} bytes ({round(size/(1024*1024), 2)} MB)")
    else:
        print(f"  ❌ Failed: {res.stderr}")
        exit(1)

print("\n🎉 Phase 1: All 4 Garble binaries compiled!")

# ═══════════════════════════════════════════════════════════════════
# Phase 2: 标准编译 (用于容器/镜像内嵌，体积极简，无需混淆)
#   容器内只需 cubexd 守护进程，不需要 cubex CLI
# ═══════════════════════════════════════════════════════════════════
container_targets = [
    ('amd64', './cmd/cubexd', 'dist/cubexd-container-amd64'),
    ('arm64', './cmd/cubexd', 'dist/cubexd-container-arm64'),
]

for arch, pkg, out in container_targets:
    env = env_base.copy()
    env['GOOS'] = 'linux'
    env['GOARCH'] = arch
    cmd = ['go', 'build', '-trimpath', '-ldflags=-s -w', '-o', out, pkg]
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
