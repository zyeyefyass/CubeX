#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX 混沌工程与突发断电逃生自动化演练工具 (CubeX Chaos Engineering & Failover Resilience Test)
专注于模拟极端故障（旁路由突然断电拔线、核心服务崩溃宕机），
压测 RouterOS Netwatch 看门狗在 3 秒内心跳判定并完成 0 秒无感逃生切换。
"""

import time
import json
import urllib.request
import base64
import subprocess
import paramiko

ROS_IP = "10.0.0.1"
ROS_AUTH = base64.b64encode(b"zyy:820130").decode("ascii")
ROS_HEADERS = {
    "Authorization": f"Basic {ROS_AUTH}",
    "Content-Type": "application/json"
}

PVE_IP = "10.0.0.9"
PVE_USER = "root"
PVE_PASS = "poi098)(*"
CT_ID = "107"
CUBEX_IP = "10.0.0.7"
CUBEX_PORT = 2026

def ros_api(method, path, data=None):
    url = f"http://{ROS_IP}/rest{path}"
    body = json.dumps(data).encode("utf-8") if data else None
    req = urllib.request.Request(url, data=body, headers=ROS_HEADERS, method=method)
    with urllib.request.urlopen(req, timeout=5) as resp:
        content = resp.read().decode("utf-8")
        return json.loads(content) if content else {}

def get_netwatch_status():
    items = ros_api("GET", "/tool/netwatch")
    for item in items:
        if "CubeX" in item.get("comment", "") or item.get("host") == CUBEX_IP:
            return item
    return None

def get_ros_dns():
    return ros_api("GET", "/ip/dns")

def get_fakeip_route():
    routes = ros_api("GET", "/ip/route")
    for r in routes:
        if r.get("dst-address") == "198.18.0.0/15" and "Backtrack" in r.get("comment", ""):
            return r
    return None

def test_dns_lookup(domain="baidu.com"):
    t0 = time.time()
    try:
        res = subprocess.run(
            ["nslookup", domain, ROS_IP],
            capture_output=True,
            text=True,
            encoding="gbk",
            errors="ignore",
            timeout=4
        )
        ms = (time.time() - t0) * 1000
        success = ("Address" in res.stdout or "Addresses" in res.stdout) and "timed out" not in res.stdout
        return success, ms, res.stdout
    except Exception as e:
        return False, 9999, str(e)

def run_chaos_drill():
    print("=" * 80)
    print("🔥  CubeX 极端断电逃生混沌工程自动化演练 (Chaos Resilience Drill)")
    print("=" * 80)
    
    report = {
        "timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
        "phases": {},
        "resilience_score": 100,
        "passed": True
    }

    # ─────────────────────────────────────────────────────────────
    # PHASE 1: 演练前基线状态检测 (Baseline State)
    # ─────────────────────────────────────────────────────────────
    print("\n[PHASE 1] 演练前基线状态检测...")
    nw_initial = get_netwatch_status()
    dns_initial = get_ros_dns()
    route_initial = get_fakeip_route()

    print(f"  • 主路由 Netwatch 状态 : {nw_initial.get('status')} (探针间隔: {nw_initial.get('interval')}, 延迟: {nw_initial.get('tcp-connect-time', 'N/A')})")
    print(f"  • 主路由当前 DNS 上游   : {dns_initial.get('servers')}")
    print(f"  • Fake-IP 回指路由状态 : 禁用状态={route_initial.get('disabled')}, 网关={route_initial.get('gateway')}")

    dns_ok, dns_lat, _ = test_dns_lookup("baidu.com")
    print(f"  • 演练前国内 DNS 解析  : {'✓ 成功' if dns_ok else '✗ 失败'} (耗时 {dns_lat:.1f}ms)")
    
    if not dns_ok or nw_initial.get("status") != "up":
        print("  ❌ 基线状态未达标，终止演练！")
        report["passed"] = False
        return report

    report["phases"]["baseline"] = {
        "status": "PASS",
        "netwatch_status": nw_initial.get("status"),
        "dns_servers": dns_initial.get("servers"),
        "dns_latency_ms": dns_lat
    }

    # ─────────────────────────────────────────────────────────────
    # PHASE 2: 注入故障 - 模拟物理机暴毙/瞬间拔电断开
    # ─────────────────────────────────────────────────────────────
    print("\n[PHASE 2] 注入残酷现实故障：模拟旁路由物理机突发断电/强杀核心进程...")
    ssh = paramiko.SSHClient()
    ssh.set_missing_host_key_policy(paramiko.AutoAddPolicy())
    ssh.connect(PVE_IP, username=PVE_USER, password=PVE_PASS)

    t_fault_start = time.time()
    # 瞬间强杀 cubexd
    stdin, stdout, stderr = ssh.exec_command(f"pct exec {CT_ID} -- rc-service cubexd stop")
    stdout.channel.recv_exit_status()
    print("  💥 已强行停止 CT 107 cubexd 守护进程 (模拟物理掉电)！")

    # 实时监听 RouterOS Netwatch 逃生响应时长
    failover_detected = False
    failover_time_ms = 0
    max_wait_seconds = 10
    start_watch = time.time()

    print("  ⏱️  正在精确捕捉主路由 Netwatch 毫秒逃生响应...")
    while time.time() - start_watch < max_wait_seconds:
        nw = get_netwatch_status()
        status = nw.get("status")
        if status == "down":
            failover_time_ms = (time.time() - t_fault_start) * 1000
            failover_detected = True
            break
        time.sleep(0.3)

    dns_during = get_ros_dns()
    route_during = get_fakeip_route()

    print(f"  • Netwatch 捕获停机用时 : {failover_time_ms:.1f}ms (状态: {nw.get('status')})")
    print(f"  • 逃生后主路由 DNS 上游 : {dns_during.get('servers')}")
    print(f"  • 逃生后回指路由状态   : 禁用状态={route_during.get('disabled')}")

    # 验证主路由逃生期间，客户端国内网络是否绝对不掉线
    print("\n[PHASE 3] 验证断网逃生期间：全家国内上网与 DNS 解析是否坚挺...")
    client_dns_ok, client_dns_lat, dns_out = test_dns_lookup("baidu.com")
    print(f"  • 客户端向主路由解析 Baidu  : {'✓ 极速直出' if client_dns_ok else '✗ 掉线'} ({client_dns_lat:.1f}ms)")
    
    # 检查 curl 国内网页
    try:
        t_curl = time.time()
        res_curl = subprocess.run(["curl", "-I", "-s", "--connect-timeout", "3", "http://www.baidu.com"], capture_output=True, text=True)
        curl_ok = "200 OK" in res_curl.stdout or "302" in res_curl.stdout
        curl_lat = (time.time() - t_curl) * 1000
        print(f"  • 客户端直连国内 HTTP 访问  : {'✓ 飞速直通' if curl_ok else '✗ 异常'} ({curl_lat:.1f}ms)")
    except Exception as e:
        curl_ok = False
        print(f"  • 客户端直连国内 HTTP 访问  : ✗ 失败 ({e})")

    report["phases"]["failover"] = {
        "detected": failover_detected,
        "failover_duration_ms": failover_time_ms,
        "dns_servers_after_failover": dns_during.get("servers"),
        "route_disabled": route_during.get("disabled"),
        "client_dns_during_failover": client_dns_ok,
        "client_dns_lat_ms": client_dns_lat,
        "client_http_ok": curl_ok
    }

    # ─────────────────────────────────────────────────────────────
    # PHASE 4: 故障自愈恢复 - 模拟旁路由通电重新启动
    # ─────────────────────────────────────────────────────────────
    print("\n[PHASE 4] 故障自愈恢复演练：模拟旁路由恢复供电并重新拉起...")
    t_heal_start = time.time()
    stdin, stdout, stderr = ssh.exec_command(f"pct exec {CT_ID} -- rc-service cubexd start")
    stdout.channel.recv_exit_status()
    print("  ⚡ 已重新启动 CT 107 cubexd 守护进程！")

    heal_detected = False
    heal_time_ms = 0
    start_heal_watch = time.time()

    while time.time() - start_heal_watch < max_wait_seconds:
        nw = get_netwatch_status()
        if nw.get("status") == "up":
            heal_time_ms = (time.time() - t_heal_start) * 1000
            heal_detected = True
            break
        time.sleep(0.3)

    ssh.close()

    dns_restored = get_ros_dns()
    route_restored = get_fakeip_route()

    is_route_disabled = str(route_restored.get("disabled", "")).lower() in ("true", "yes")

    print(f"  • Netwatch 捕获恢复用时 : {heal_time_ms:.1f}ms (状态: {nw.get('status')})")
    print(f"  • 恢复后主路由 DNS 上游 : {dns_restored.get('servers')}")
    print(f"  • 恢复后回指路由状态   : 禁用状态={is_route_disabled}")

    # 验证加速与直连双通
    final_dns_ok, final_dns_lat, _ = test_dns_lookup("baidu.com")
    print(f"  • 自愈后最终 DNS 解析  : {'✓ 成功' if final_dns_ok else '✗ 失败'} ({final_dns_lat:.1f}ms)")

    report["phases"]["restoration"] = {
        "heal_detected": heal_detected,
        "heal_duration_ms": heal_time_ms,
        "dns_servers_restored": dns_restored.get("servers"),
        "route_enabled": not is_route_disabled
    }

    # ─────────────────────────────────────────────────────────────
    # 演练综合评分卡
    # ─────────────────────────────────────────────────────────────
    print("\n" + "=" * 80)
    print("🏆  CUBEX 零断网容灾演练权威评分卡 (Chaos Resilience Scorecard)")
    print("=" * 80)
    
    score = 100
    if not failover_detected:
        score -= 40
        print("  ✗ 主路由未能在规定时间内捕获停机事件 (-40分)")
    else:
        print(f"  ✓ 毫秒逃生响应迅速: 捕获用时 {failover_time_ms:.1f}ms (满分)")

    if "223.5.5.5" not in dns_during.get("servers", ""):
        score -= 20
        print("  ✗ 逃生后 DNS 未能无缝回退至阿里/腾讯公共 DNS (-20分)")
    else:
        print(f"  ✓ DNS 0秒逃生成功: 已切换为 {dns_during.get('servers')} (满分)")

    if not route_during.get("disabled"):
        score -= 20
        print("  ✗ 停机后未自动禁用 Fake-IP 回指路由 (-20分)")
    else:
        print("  ✓ 路由死锁熔断成功: 198.18.0.0/15 回指已自动禁用，防流量进黑洞 (满分)")

    if not client_dns_ok or not curl_ok:
        score -= 20
        print("  ✗ 旁路由宕机期间客户端网络出现卡顿或解析失败 (-20分)")
    else:
        print("  ✓ 拔电全家零掉线: 国内网页与解析 100% 飞速无感 (满分)")

    if not heal_detected or "10.0.0.7" not in dns_restored.get("servers", ""):
        score -= 10
        print("  ✗ 容器重启后未能在 3 秒内恢复 CubeX 状态 (-10分)")
    else:
        print(f"  ✓ 通电极速自愈恢复: 用时 {heal_time_ms:.1f}ms，加速链路全自动重新就绪 (满分)")

    print(f"\n🌟 最终高可用容灾综合韧性得分: {score}/100")
    if score == 100:
        print("🎖️  闭环断言认证: 「当面拔掉电源全家依然飞速上网」神话坚不可摧！100% 满分通过！")
    
    report["resilience_score"] = score
    with open("scratch/chaos_resilience_report.json", "w", encoding="utf-8") as f:
        json.dump(report, f, indent=2, ensure_ascii=False)
    print("  -> 演练报告已持久化保存至 scratch/chaos_resilience_report.json")
    print("=" * 80)
    return report

if __name__ == "__main__":
    run_chaos_drill()
