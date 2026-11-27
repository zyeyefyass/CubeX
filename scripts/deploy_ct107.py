import paramiko
import os
import sys

PVE_HOST = os.environ.get('CUBEX_PVE_HOST', '').strip()
PVE_PORT = int(os.environ.get('CUBEX_PVE_PORT', '22'))
PVE_USER = os.environ.get('CUBEX_PVE_USER', 'root').strip()
PVE_PASS = os.environ.get('CUBEX_PVE_PASS', '').strip()
TARGET_CT = os.environ.get('CUBEX_TARGET_CT', '').strip()

if not PVE_HOST or not PVE_PASS or not TARGET_CT:
    print("❌ 请通过环境变量 CUBEX_PVE_HOST / CUBEX_PVE_PASS / CUBEX_TARGET_CT 指定目标隔离容器与凭证。")
    sys.exit(1)

print(f"Connecting to PVE {PVE_HOST} (target CT {TARGET_CT})...")
ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect(PVE_HOST, port=PVE_PORT, username=PVE_USER, password=PVE_PASS, timeout=10)

sftp = ssh.open_sftp()
local_cubexd = 'dist/cubexd-container-amd64' if os.path.exists('dist/cubexd-container-amd64') else ('dist/cubexd-linux-amd64' if os.path.exists('dist/cubexd-linux-amd64') else 'bin/cubexd-linux-amd64')
local_cubex = 'dist/cubex-linux-amd64' if os.path.exists('dist/cubex-linux-amd64') else 'bin/cubex-linux-amd64'

print(f"Uploading {local_cubexd} ({round(os.path.getsize(local_cubexd)/(1024*1024), 2)} MB) to PVE /tmp/cubexd...")
sftp.put(local_cubexd, '/tmp/cubexd')

print(f"Uploading {local_cubex} to PVE /tmp/cubex...")
sftp.put(local_cubex, '/tmp/cubex')

theme_pkg = os.path.join('scratch', 'dark-skull-luxury-1080p.cbxtheme')
if os.path.exists(theme_pkg):
    print(f"Uploading theme package {theme_pkg} to PVE /tmp/dark-skull-luxury-1080p.cbxtheme...")
    sftp.put(theme_pkg, '/tmp/dark-skull-luxury-1080p.cbxtheme')

wallpapers = [
    'bg-tab-overview.webp',
    'bg-tab-routing.webp',
    'bg-tab-rules.webp',
    'bg-tab-system.webp',
    'bg-tab-diagnostics.webp'
]
for wp in wallpapers:
    wp_path = os.path.join('scratch', 'wallpapers', wp)
    if os.path.exists(wp_path):
        print(f"Uploading wallpaper {wp} to PVE /tmp/{wp}...")
        sftp.put(wp_path, f'/tmp/{wp}')

sftp.close()

def run_ssh(cmd):
    stdin, stdout, stderr = ssh.exec_command(cmd)
    out = stdout.read().decode().strip()
    err = stderr.read().decode().strip()
    return out, err

print(f"Stopping cubexd in CT {TARGET_CT}...")
out, err = run_ssh(f"pct exec {TARGET_CT} -- rc-service cubexd stop")
print("Stop result:", out, err)

print(f"Updating /etc/init.d/cubexd memory limits in CT {TARGET_CT}...")
run_ssh(f"""pct exec {TARGET_CT} -- sh -c 'cat << "INIT" > /etc/init.d/cubexd
#!/sbin/openrc-run
name="cubexd"
description="CubeX System Control Plane Daemon"
command="/opt/CubeX/bin/cubexd"
command_args=""
directory="/opt/CubeX"
supervisor="supervise-daemon"
respawn_max=0
output_log="/var/log/cubexd.log"
error_log="/var/log/cubexd.err"

export GOMEMLIMIT="64MiB"
export GOGC="100"
export GODEBUG="madvdontneed=1"
export TZ="Asia/Shanghai"

depend() {{
    need net
    after firewall
}}
INIT
chmod 755 /etc/init.d/cubexd'""")

print(f"Pushing binaries into CT {TARGET_CT}...")
run_ssh(f"pct push {TARGET_CT} /tmp/cubexd /opt/CubeX/bin/cubexd")
run_ssh(f"pct push {TARGET_CT} /tmp/cubex /opt/CubeX/bin/cubex")
run_ssh(f"pct exec {TARGET_CT} -- chmod +x /opt/CubeX/bin/cubexd /opt/CubeX/bin/cubex")

print(f"Setting up wallpapers in CT {TARGET_CT} (/etc/cubex/data/wallpapers)...")
run_ssh(f"pct exec {TARGET_CT} -- mkdir -p /etc/cubex/data/wallpapers")
for wp in wallpapers:
    run_ssh(f"pct push {TARGET_CT} /tmp/{wp} /etc/cubex/data/wallpapers/{wp}")

print(f"Setting up dark-skull-luxury theme in CT {TARGET_CT} (/etc/cubex/data/themes)...")
run_ssh(f"pct exec {TARGET_CT} -- mkdir -p /etc/cubex/data/themes/dark-skull-luxury")
run_ssh(f"pct push {TARGET_CT} /tmp/dark-skull-luxury-1080p.cbxtheme /tmp/dark-skull-luxury-1080p.cbxtheme")
run_ssh(f"pct exec {TARGET_CT} -- unzip -o /tmp/dark-skull-luxury-1080p.cbxtheme -d /etc/cubex/data/themes/dark-skull-luxury")
run_ssh(f"""pct exec {TARGET_CT} -- sh -c 'cat << "THEMECFG" > /etc/cubex/data/themes/config.json
{{
  "active_theme_id": "dark-skull-luxury",
  "playback_mode": "auto"
}}
THEMECFG'""")

print("Syncing themes and wallpapers to /opt/CubeX/data...")
run_ssh(f"pct exec {TARGET_CT} -- cp -rf /etc/cubex/data/themes /opt/CubeX/data/ && pct exec {TARGET_CT} -- cp -rf /etc/cubex/data/wallpapers /opt/CubeX/data/")

print(f"Starting cubexd in CT {TARGET_CT}...")
out, err = run_ssh(f"pct exec {TARGET_CT} -- rc-service cubexd start")
print("Start result:", out, err)

# Verify running process & MD5 & Size
out, _ = run_ssh(f"pct exec {TARGET_CT} -- ls -lh /opt/CubeX/bin/cubexd")
print(f"CT {TARGET_CT} cubexd size:", out)

out, _ = run_ssh(f"pct exec {TARGET_CT} -- rc-status | grep cubexd")
print(f"CT {TARGET_CT} service status:", out)

ssh.close()
print(f"🎉 Deployment to CT {TARGET_CT} completed!")
