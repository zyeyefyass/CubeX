import paramiko

ssh = paramiko.SSHClient()
ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
ssh.connect('10.0.0.9', username='root', password='poi098)(*')

cmd = """pct exec 107 -- sh -c '
for p in $(pidof cubexd sing-box mihomo); do
  echo "=== PID $p ($(cat /proc/$p/comm)) total=$(ls -1 /proc/$p/fd | wc -l) ==="
  ls -l /proc/$p/fd | awk "{print \$NF}" | sed "s/:.*/:/" | sort | uniq -c | sort -rn | head -n 10
done
'"""

stdin, stdout, stderr = ssh.exec_command(cmd)
print(stdout.read().decode('utf-8', errors='ignore'))
err = stderr.read().decode('utf-8', errors='ignore')
if err:
    print('err:', err)
ssh.close()
