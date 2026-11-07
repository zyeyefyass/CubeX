import socket
import concurrent.futures

ip = "10.0.0.186"

def check(p):
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
            s.settimeout(0.2)
            if s.connect_ex((ip, p)) == 0:
                return p
    except:
        pass
    return None

print("Scanning 30000-50000 on 10.0.0.186...")
with concurrent.futures.ThreadPoolExecutor(max_workers=300) as ex:
    for p in ex.map(check, range(30000, 50001)):
        if p:
            print("Found open port:", p, flush=True)
print("Done scan.")
