import os
import sys
import time
import json
import paramiko

PVE_HOST = '10.0.0.9'
PVE_PORT = 22
PVE_USER = 'root'
PVE_PASS = 'poi098)(*'
CT_ID = '107'

LOCAL_DIR = os.path.dirname(os.path.abspath(__file__))
OUTPUT_DIR = os.path.join(LOCAL_DIR, '..', '..', 'scratch', 'p1_results')
os.makedirs(OUTPUT_DIR, exist_ok=True)

print("══════════════════════════════════════════════════════════════════════")
print("🔥 CubeX P1「极限压测与削峰门禁」自动化测试调度套件")
print("══════════════════════════════════════════════════════════════════════")

# 1. Connect to PVE via SSH
print(f"📡 [SSH] 连接 PVE 物理宿主机 {PVE_HOST}...")
ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect(PVE_HOST, port=PVE_PORT, username=PVE_USER, password=PVE_PASS, timeout=10)

def exec_ct(cmd, timeout=120):
    stdin, stdout, stderr = ssh.exec_command(f"pct exec {CT_ID} -- sh -c '{cmd}'", timeout=timeout)
    out = stdout.read().decode('utf-8', errors='ignore').strip()
    err = stderr.read().decode('utf-8', errors='ignore').strip()
    return out, err

# 2. Push scripts to CT 107
print("📦 [Deploy] 推送 P1 自动化测试脚本至 CT 107...")
sftp = ssh.open_sftp()
for script in ['collect_metrics.sh', 'switch_burnin.sh', 'fault_drill.sh']:
    local_path = os.path.join(LOCAL_DIR, script)
    remote_pve = f'/tmp/{script}'
    sftp.put(local_path, remote_pve)
    _, push_out, _ = ssh.exec_command(f"pct push {CT_ID} {remote_pve} /opt/CubeX/{script} -perms 755")
    push_out.channel.recv_exit_status()
sftp.close()

# Verify files in CT 107
out, _ = exec_ct("ls -l /opt/CubeX/*.sh")
print(f"✓ 脚本注入完成:\n{out}")

# 3. Start 1Hz Telemetry Collector in CT 107 background
print("\n📊 [Stage 1] 启动 1Hz /proc 低开销遥测记录仪...")
exec_ct("killall collect_metrics.sh 2>/dev/null || true")
exec_ct("nohup /bin/bash /opt/CubeX/collect_metrics.sh /tmp/cubex_p1_telemetry.csv /tmp/cubex_p1_phase >/tmp/collector.log 2>&1 &")
time.sleep(3)

# Verify collector is running
out, _ = exec_ct("head -n 2 /tmp/cubex_p1_telemetry.csv")
print(f"✓ 遥测记录仪首行采样数据:\n{out}")

# 4. Run Step 3: 带载 10 次双核往返热切拷机 (含 100ms 激进防误熔断探针)
print("\n══════════════════════════════════════════════════════════════════════")
print("🔥 [Step 3] 启动带载 10 次双核连续往返热切极限拷机...")
print("══════════════════════════════════════════════════════════════════════")
exec_ct("echo 'step3_burnin_start' > /tmp/cubex_p1_phase")
stdin, stdout, stderr = ssh.exec_command(f"pct exec {CT_ID} -- /bin/bash /opt/CubeX/switch_burnin.sh", timeout=300)

for line in stdout:
    print(line.strip())

burnin_err = stderr.read().decode('utf-8', errors='ignore').strip()
if burnin_err:
    print(f"⚠️ [Burnin Warning]: {burnin_err}")

# 5. Run Step 4: 三连故障注入演练与脱敏诊断卡取证
print("\n══════════════════════════════════════════════════════════════════════")
print("⚡ [Step 4] 启动三连故障注入演练与脱敏诊断卡取证...")
print("══════════════════════════════════════════════════════════════════════")
stdin, stdout, stderr = ssh.exec_command(f"pct exec {CT_ID} -- /bin/bash /opt/CubeX/fault_drill.sh", timeout=180)

for line in stdout:
    print(line.strip())

fault_err = stderr.read().decode('utf-8', errors='ignore').strip()
if fault_err:
    print(f"⚠️ [Fault Drill Notice]: {fault_err}")

# 6. Stop collector & Pull telemetry CSV & diagnostic JSONs
print("\n══════════════════════════════════════════════════════════════════════")
print("📥 [Pull] 停止采样器并拉取全量遥测 CSV 与脱敏诊断卡样本...")
print("══════════════════════════════════════════════════════════════════════")
exec_ct("killall collect_metrics.sh 2>/dev/null || true")
time.sleep(1)

# Pull files from CT 107 to PVE /tmp, then SFTP to host
files_to_pull = [
    ('/tmp/cubex_p1_telemetry.csv', 'cubex_p1_telemetry.csv'),
    ('/tmp/diag_kill9.json', 'diag_kill9.json'),
    ('/tmp/diag_dns_hang.json', 'diag_dns_hang.json'),
    ('/tmp/cubex_p1_false_trips.log', 'cubex_p1_false_trips.log'),
]

sftp = ssh.open_sftp()
for ct_file, local_name in files_to_pull:
    pve_tmp = f"/tmp/{local_name}"
    _, pull_out, _ = ssh.exec_command(f"pct pull {CT_ID} {ct_file} {pve_tmp}")
    pull_out.channel.recv_exit_status()
    time.sleep(0.5)
    dst = os.path.join(OUTPUT_DIR, local_name)
    try:
        sftp.get(pve_tmp, dst)
        print(f"✓ 成功拉取: {local_name} ({os.path.getsize(dst)} 字节)")
    except Exception as e:
        print(f"⚠️ 拉取 {local_name} 提示: {e}")

sftp.close()
ssh.close()

# 7. Analyze telemetry CSV
csv_path = os.path.join(OUTPUT_DIR, 'cubex_p1_telemetry.csv')
if os.path.exists(csv_path):
    print("\n══════════════════════════════════════════════════════════════════════")
    print("📈 [Analysis] P1 压测全链路遥测数据深度统计分析")
    print("══════════════════════════════════════════════════════════════════════")
    lines = open(csv_path, encoding='utf-8').read().strip().splitlines()
    header = lines[0].split(',')
    records = [l.split(',') for l in lines[1:] if len(l.split(',')) == len(header)]
    
    total_samples = len(records)
    rss_values = [float(r[6]) for r in records if r[6] != '']
    fds_values = [int(r[7]) for r in records if r[7] != '']
    zombies_values = [int(r[8]) for r in records if r[8] != '']
    health_codes = [r[10] for r in records if r[10] != '']

    min_rss = min(rss_values) if rss_values else 0
    max_rss = max(rss_values) if rss_values else 0
    avg_rss = sum(rss_values) / len(rss_values) if rss_values else 0
    
    initial_fds = fds_values[0] if fds_values else 0
    final_fds = fds_values[-1] if fds_values else 0
    max_fds = max(fds_values) if fds_values else 0
    max_zombies = max(zombies_values) if zombies_values else 0

    print(f"• 遥测采样周期总时长 : {total_samples} 秒 (1Hz 连续打点)")
    print(f"• 内存综合占用 (RSS)  : 基准最低 {min_rss:.2f} MB | 压测峰值 {max_rss:.2f} MB | 平均 {avg_rss:.2f} MB")
    print(f"• 文件描述符 (FD)    : 初始基线 {initial_fds} -> 最终回落 {final_fds} (峰值: {max_fds}, 净增量: {final_fds - initial_fds})")
    print(f"• 僵尸进程数 (Zombie): 最高峰值 {max_zombies} 个 (严格红线: 0)")
    
    code_counts = {}
    for c in health_codes:
        code_counts[c] = code_counts.get(c, 0) + 1
    print(f"• 探针状态码分布     : {code_counts}")

print("\n🎉 P1 自动化压测执行完毕！所有数据与样本已保存在 scratch/p1_results/ 目录！")
