import socket
import struct
import time
import urllib.request
import os

def send(sock, text):
    data = text.encode('utf-8')
    sock.sendall(struct.pack('>i', len(data)) + data)

def messages(buf):
    out = []
    while len(buf) >= 4:
        n = struct.unpack('>i', buf[:4])[0]
        if n < 0:
            raise ValueError(f"negative length: {n}")
        if len(buf) < 4 + n:
            break
        out.append(buf[4:4 + n].decode('utf-8', 'replace'))
        buf = buf[4 + n:]
    return out, buf

sock = socket.create_connection(('10.0.0.186', 6553), timeout=5)
sock.settimeout(2)
send(sock, '159>>1>>0')
time.sleep(0.5)

# Request screenshot
print("Requesting screenshot...")
send(sock, '225>>')

buf = b''
shot_url = None
deadline = time.monotonic() + 10

while time.monotonic() < deadline:
    try:
        chunk = sock.recv(65536)
        if not chunk:
            break
        buf += chunk
    except socket.timeout:
        pass
    found, buf = messages(buf)
    for msg in found:
        if 'http' in msg and '.png' in msg:
            parts = msg.split('>>')
            for p in parts:
                if p.startswith('http'):
                    shot_url = p
                    break
    if shot_url:
        break
    time.sleep(0.2)

if shot_url:
    print("Found screenshot URL:", shot_url)
    print("Downloading WHILE KEEPING SOCKET OPEN...")
    os.makedirs('shots', exist_ok=True)
    try:
        req = urllib.request.Request(shot_url, headers={'User-Agent': 'Mozilla/5.0'})
        with urllib.request.urlopen(req, timeout=5) as resp:
            data = resp.read()
            with open('shots/tv_screenshot.png', 'wb') as f:
                f.write(data)
            print("Successfully saved screenshot to shots/tv_screenshot.png! Size:", len(data), "bytes")
    except Exception as e:
        print("Download error:", e)

sock.close()
