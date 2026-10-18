import os
import sys
import json
import re

STANDARD_STRATEGY_TARGETS = {
    "DIRECT", "REJECT",
    "🇭🇰 香港自动", "🇭🇰 香港手动", "🇭🇰 香港故转", "🇭🇰 香港容灾",
    "🇹🇼 台湾自动", "🇹🇼 台湾手动", "🇹🇼 台湾故转", "🇹🇼 台湾容灾",
    "🇯🇵 日本自动", "🇯🇵 日本手动", "🇯🇵 日本故转", "🇯🇵 日本容灾",
    "🇸🇬🇯🇵🇰🇷 新日韩自动", "🇸🇬🇯🇵🇰🇷 新日韩手动", "🇸🇬🇯🇵🇰🇷 新日韩故转", "🇸🇬🇯🇵🇰🇷 新日韩容灾",
    "🇺🇸 美国自动", "🇺🇸 美国手动", "🇺🇸 美国故转", "🇺🇸 美国容灾",
    "🏠 自建自动", "🏠 自建手动", "🏠 自建故转", "🏠 自建容灾",
    "🌐 其它自动", "🌐 其它手动", "🌐 其它故转", "🌐 其它容灾",
}

def desensitize(target_dir):
    data_dir = os.path.join(target_dir, "data")
    etc_dir = os.path.join(target_dir, "etc")
    if not os.path.exists(data_dir):
        # RouterOS rootfs structure: /app/data, /app/etc
        data_dir = os.path.join(target_dir, "app", "data")
        etc_dir = os.path.join(target_dir, "app", "etc")

    print(f"🧹 Running strict desensitization on: {target_dir}")

    # 0. Thoroughly remove any backup directories (e.g. ros_backup, backups, etc.)
    import shutil
    for root, dirs, files in os.walk(target_dir, topdown=True):
        for d in list(dirs):
            if "backup" in d.lower() or d == "ros_backup":
                bp = os.path.join(root, d)
                try:
                    shutil.rmtree(bp)
                    print(f"  ✓ Removed backup directory: {bp}")
                    dirs.remove(d)
                except Exception as e:
                    print(f"  Error removing backup dir {bp}: {e}")

    # 1. Clean data/config.ir.json
    ir_path = os.path.join(data_dir, "config.ir.json")
    if os.path.exists(ir_path):
        try:
            with open(ir_path, "r", encoding="utf-8") as f:
                ir = json.load(f)
            
            # Wipe subscriptions & node proxies
            ir["subscriptions"] = []
            ir["proxies"] = []
            ir["device_policies"] = {}

            # Sanitize groups: strip ANY node proxy names
            for g in ir.get("groups", []):
                name = g.get("name", "")
                cat = g.get("category", "")
                if cat == "region" or any(k in name for k in ["香港", "台湾", "日本", "新日韩", "美国", "自建", "其它"]):
                    g["proxies"] = ["DIRECT"]
                    g["now"] = "DIRECT"
                else:
                    # Business groups (AI, YouTube, Google, etc.)
                    # Only allow strict matches in STANDARD_STRATEGY_TARGETS
                    g["proxies"] = [p for p in g.get("proxies", []) if p in STANDARD_STRATEGY_TARGETS]
                    if not g["proxies"]:
                        g["proxies"] = ["🇯🇵 日本自动", "🇭🇰 香港自动", "DIRECT"]
                    if g.get("now") not in STANDARD_STRATEGY_TARGETS:
                        g["now"] = g["proxies"][0]

            with open(ir_path, "w", encoding="utf-8") as f:
                json.dump(ir, f, indent=2, ensure_ascii=False)
            print(f"  ✓ Desensitized {ir_path} (0 subs, 0 proxies, pristine groups)")
        except Exception as e:
            print(f"  Error desensitizing IR: {e}")

    # 2. Clean etc/config.json (Sing-box)
    sb_path = os.path.join(etc_dir, "config.json")
    if os.path.exists(sb_path):
        try:
            with open(sb_path, "r", encoding="utf-8") as f:
                sb = json.load(f)
            clean_outbounds = []
            for ob in sb.get("outbounds", []):
                ob_type = ob.get("type", "")
                tag = ob.get("tag", "")
                if ob_type in ["direct", "block", "dns"]:
                    clean_outbounds.append(ob)
                elif ob_type in ["selector", "urltest"]:
                    filtered = [o for o in ob.get("outbounds", []) if o in ["direct", "block", "dns-out"] or o in STANDARD_STRATEGY_TARGETS]
                    if not filtered:
                        filtered = ["direct"]
                    ob["outbounds"] = filtered
                    clean_outbounds.append(ob)
            sb["outbounds"] = clean_outbounds
            with open(sb_path, "w", encoding="utf-8") as f:
                json.dump(sb, f, indent=2, ensure_ascii=False)
            print(f"  ✓ Desensitized {sb_path}")
        except Exception as e:
            print(f"  Error desensitizing sing-box: {e}")

    # 3. Clean etc/config.yaml (Mihomo)
    mh_path = os.path.join(etc_dir, "config.yaml")
    if os.path.exists(mh_path):
        try:
            with open(mh_path, "r", encoding="utf-8") as f:
                lines = f.readlines()
            new_lines = []
            skip_proxies = False
            for line in lines:
                if line.startswith("proxies:"):
                    new_lines.append("proxies: []\n")
                    skip_proxies = True
                    continue
                if skip_proxies:
                    if line.startswith("proxy-groups:") or (line and not line.startswith(" ") and not line.startswith("-") and ":" in line):
                        skip_proxies = False
                        new_lines.append(line)
                    continue
                new_lines.append(line)

            with open(mh_path, "w", encoding="utf-8") as f:
                f.writelines(new_lines)
            print(f"  ✓ Desensitized {mh_path}")
        except Exception as e:
            print(f"  Error desensitizing mihomo: {e}")

    # 4. Clean data/wireguard.json
    wg_path = os.path.join(data_dir, "wireguard.json")
    if os.path.exists(wg_path):
        try:
            with open(wg_path, "r", encoding="utf-8") as f:
                wg = json.load(f)
            cfg = wg.get("config", {})
            cfg["private_key"] = ""
            cfg["public_key"] = ""
            cfg["endpoint_host"] = ""
            cfg["cloud_ddns"] = ""
            wg["peers"] = []
            with open(wg_path, "w", encoding="utf-8") as f:
                json.dump(wg, f, indent=2, ensure_ascii=False)
            print(f"  ✓ Desensitized {wg_path} (0 keys, 0 peers, 0 ddns)")
        except Exception as e:
            print(f"  Error desensitizing wireguard: {e}")

    # 5. Clean data/host-config.json and data/ros-config.json
    for cfg_file in ["host-config.json", "ros-config.json"]:
        cp = os.path.join(data_dir, cfg_file)
        if os.path.exists(cp):
            try:
                with open(cp, "r", encoding="utf-8") as f:
                    c = json.load(f)
                c["password"] = ""
                c["host"] = "192.168.88.1"
                if "private_key" in c:
                    c["private_key"] = ""
                with open(cp, "w", encoding="utf-8") as f:
                    json.dump(c, f, indent=2, ensure_ascii=False)
                print(f"  ✓ Desensitized {cp}")
            except:
                pass

    # 5.1 Clean data/glados-config.json
    glados_path = os.path.join(data_dir, "glados-config.json")
    if os.path.exists(glados_path):
        try:
            with open(glados_path, "r", encoding="utf-8") as f:
                gc = json.load(f)
            gc["cookie"] = ""
            gc["sub_url"] = ""
            gc["email"] = ""
            gc["points"] = 0
            gc["left_days"] = "0"
            gc["synced_nodes"] = 0
            gc["last_checkin_status"] = ""
            gc["enabled"] = False
            with open(glados_path, "w", encoding="utf-8") as f:
                json.dump(gc, f, indent=2, ensure_ascii=False)
            print(f"  ✓ Desensitized {glados_path} (0 cookie, 0 sub_url)")
        except:
            pass

    # 6. Delete runtime artifacts and scratch files
    for root, dirs, files in os.walk(target_dir):
        for fl in files:
            if fl == "cache.db" or fl.endswith(".log") or fl.endswith(".sqlite") or fl.endswith(".pid"):
                p = os.path.join(root, fl)
                try:
                    os.remove(p)
                    print(f"  ✓ Removed runtime file: {fl}")
                except:
                    pass

if __name__ == "__main__":
    if len(sys.argv) > 1:
        desensitize(sys.argv[1])
    else:
        print("Usage: python desensitize.py <directory>")
