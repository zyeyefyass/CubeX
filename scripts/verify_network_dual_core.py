#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Dual-Core Network Verification Script
Executes 2 full cycles of engine switching (Mihomo <-> sing-box)
and verifies domestic direct, foreign proxy, and DNS Fake-IP resolution.
"""

import sys
import time
import json
import subprocess
import urllib.request

def find_active_host():
    for candidate in ["10.0.0.7", "10.0.0.11", "127.0.0.1"]:
        try:
            req = urllib.request.Request(f"http://{candidate}:2026/api/v1/system/status")
            with urllib.request.urlopen(req, timeout=1.5) as r:
                return candidate
        except Exception:
            continue
    return "10.0.0.7"

HOST_CUBEX = find_active_host()
PORT_API = 2026
PORT_PROXY = 7890

def get_overview():
    try:
        req = urllib.request.Request(f"http://{HOST_CUBEX}:{PORT_API}/api/v1/system/overview")
        with urllib.request.urlopen(req, timeout=5) as resp:
            return json.loads(resp.read().decode())
    except Exception as e:
        print(f"[-] Failed to get system overview: {e}")
        return {}

def switch_engine(target):
    try:
        data = json.dumps({"engine": target}).encode()
        req = urllib.request.Request(
            f"http://{HOST_CUBEX}:{PORT_API}/api/v1/engine/switch",
            data=data,
            headers={"Content-Type": "application/json"},
            method="POST"
        )
        with urllib.request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read().decode())
    except Exception as e:
        print(f"[-] Failed to switch engine to {target}: {e}")
        return {"status": "error", "error": str(e)}

def test_dns(domain="www.google.com"):
    try:
        out = subprocess.check_output(
            ["powershell", "-Command", f"Resolve-DnsName -Name {domain} -Server {HOST_CUBEX}"],
            text=True,
            timeout=5
        )
        has_aaaa = False
        has_a = False
        a_ip = ""
        for line in out.splitlines():
            if "AAAA" in line and "fc00::" in line:
                has_aaaa = True
            if "198.18." in line:
                has_a = True
                a_ip = line.strip().split()[-1]
        if has_aaaa:
            return False, f"FAILED: Leaked AAAA fc00:: Fake-IP!"
        if has_a:
            return True, f"OK (Fake-IP: {a_ip})"
        return False, f"FAILED: No 198.18.x.x answer returned"
    except Exception as e:
        return False, f"ERROR: {e}"

def test_proxy():
    try:
        out = subprocess.check_output(
            ["curl.exe", "-x", f"http://{HOST_CUBEX}:{PORT_PROXY}", "-I", "--connect-timeout", "4", "https://www.google.com/generate_204"],
            text=True,
            timeout=6
        )
        if "204" in out or "200" in out:
            return True, "OK (HTTP 204/200)"
        return False, f"HTTP Error: {out[:80]}"
    except Exception as e:
        return False, f"Failed: {e}"

def test_direct():
    try:
        out = subprocess.check_output(
            ["curl.exe", "-I", "--connect-timeout", "3", "https://www.baidu.com"],
            text=True,
            timeout=5
        )
        if "200" in out or "302" in out:
            return True, "OK (Direct 200/302)"
        return False, f"HTTP Error: {out[:80]}"
    except Exception as e:
        return False, f"Failed: {e}"

def main():
    print("=" * 65)
    print("🔄 CubeX Dual-Core & Network 2-Cycle Round-Trip Verification")
    print("=" * 65)
    
    init_ov = get_overview()
    curr_engine = init_ov.get("active_engine", "singbox")
    print(f"[*] Initial Active Engine: {curr_engine}")
    
    all_passed = True
    
    for cycle in [1, 2]:
        print(f"\n--- [Cycle {cycle}/2] ---")
        
        # Step A: Switch to other engine
        target_a = "mihomo" if curr_engine == "singbox" else "singbox"
        print(f"[*] Step A: Switching to {target_a}...")
        res_a = switch_engine(target_a)
        if res_a.get("status") != "switched":
            print(f"[-] Switch to {target_a} failed: {res_a}")
            all_passed = False
        time.sleep(2.5)
        
        # Verify Direct
        d_ok, d_msg = test_direct()
        print(f"    ├─ Direct Connection : {'✅' if d_ok else '❌'} {d_msg}")
        # Verify Proxy
        p_ok, p_msg = test_proxy()
        print(f"    ├─ Proxy Connection  : {'✅' if p_ok else '❌'} {p_msg}")
        # Verify DNS
        dns_ok, dns_msg = test_dns()
        print(f"    └─ DNS Fake-IP Clean : {'✅' if dns_ok else '❌'} {dns_msg}")
        
        if not (d_ok and p_ok and dns_ok):
            all_passed = False

        # Step B: Switch back to original engine
        target_b = curr_engine
        print(f"[*] Step B: Switching back to {target_b}...")
        res_b = switch_engine(target_b)
        if res_b.get("status") != "switched":
            print(f"[-] Switch to {target_b} failed: {res_b}")
            all_passed = False
        time.sleep(2.5)
        
        # Verify Direct
        d_ok, d_msg = test_direct()
        print(f"    ├─ Direct Connection : {'✅' if d_ok else '❌'} {d_msg}")
        # Verify Proxy
        p_ok, p_msg = test_proxy()
        print(f"    ├─ Proxy Connection  : {'✅' if p_ok else '❌'} {p_msg}")
        # Verify DNS
        dns_ok, dns_msg = test_dns()
        print(f"    └─ DNS Fake-IP Clean : {'✅' if dns_ok else '❌'} {dns_msg}")
        
        if not (d_ok and p_ok and dns_ok):
            all_passed = False

    print("\n" + "=" * 65)
    if all_passed:
        print("🎉 VERIFICATION PASSED: Dual-core 2-cycle switching 100% stable!")
        print("=" * 65)
        sys.exit(0)
    else:
        print("❌ VERIFICATION FAILED: Anomalies detected in dual-core or network!")
        print("=" * 65)
        sys.exit(1)

if __name__ == "__main__":
    main()
