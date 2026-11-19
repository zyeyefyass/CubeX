import paramiko
import os

PVE_HOST = '10.0.0.9'
PVE_PORT = 22
PVE_USER = 'root'
PVE_PASS = 'poi098)(*'

print(f"Connecting to PVE {PVE_HOST}...")
ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect(PVE_HOST, port=PVE_PORT, username=PVE_USER, password=PVE_PASS, timeout=10)

sftp = ssh.open_sftp()
local_cubexd = 'dist/cubexd-linux-amd64' if os.path.exists('dist/cubexd-linux-amd64') else ('dist/cubexd-container-amd64' if os.path.exists('dist/cubexd-container-amd64') else 'bin/cubexd-linux-amd64')
local_cubex = 'dist/cubex-linux-amd64' if os.path.exists('dist/cubex-linux-amd64') else 'bin/cubex-linux-amd64'

print(f"Uploading {local_cubexd} ({round(os.path.getsize(local_cubexd)/(1024*1024), 2)} MB) to PVE /tmp/cubexd...")
sftp.put(local_cubexd, '/tmp/cubexd')

print(f"Uploading {local_cubex} to PVE /tmp/cubex...")
sftp.put(local_cubex, '/tmp/cubex')

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

print("Stopping cubexd in CT 107...")
out, err = run_ssh("pct exec 107 -- rc-service cubexd stop")
print("Stop result:", out, err)

print("Updating /etc/init.d/cubexd memory limits...")
run_ssh("""pct exec 107 -- sh -c 'cat << "INIT" > /etc/init.d/cubexd
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

depend() {
    need net
    after firewall
}
INIT
chmod 755 /etc/init.d/cubexd'""")

print("Pushing binaries into CT 107...")
run_ssh("pct push 107 /tmp/cubexd /opt/CubeX/bin/cubexd")
run_ssh("pct push 107 /tmp/cubex /opt/CubeX/bin/cubex")
run_ssh("pct exec 107 -- chmod +x /opt/CubeX/bin/cubexd /opt/CubeX/bin/cubex")

print("Setting up wallpapers in CT 107 (/etc/cubex/data/wallpapers)...")
run_ssh("pct exec 107 -- mkdir -p /etc/cubex/data/wallpapers")
for wp in wallpapers:
    run_ssh(f"pct push 107 /tmp/{wp} /etc/cubex/data/wallpapers/{wp}")

print("Starting cubexd in CT 107...")
out, err = run_ssh("pct exec 107 -- rc-service cubexd start")
print("Start result:", out, err)

# Verify running process & MD5 & Size
out, _ = run_ssh("pct exec 107 -- ls -lh /opt/CubeX/bin/cubexd")
print("CT 107 cubexd size:", out)

out, _ = run_ssh("pct exec 107 -- rc-status | grep cubexd")
print("CT 107 service status:", out)

ssh.close()
print("🎉 Deployment to CT 107 completed!")
