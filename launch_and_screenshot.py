import socket
import struct
import time
import sys

def send(sock, text):
    data = text.encode('utf-8')
    sock.sendall(struct.pack('>i', len(data)) + data)

def recv_msg(sock):
    header = sock.recv(4)
    if not header or len(header) < 4:
        return None
    length = struct.unpack('>i', header)[0]
    data = b''
    while len(data) < length:
        chunk = sock.recv(min(length - len(data), 4096))
        if not chunk:
            break
        data += chunk
    return data.decode('utf-8', 'replace')

sock = socket.create_connection(('10.0.0.186', 6553), timeout=5)
send(sock, '159>>1>>0')
res = recv_msg(sock)
print("Handshake response 1:", res)
res2 = recv_msg(sock)
print("Handshake response 2:", res2)

# Launch Aerial
print("Launching app.orchard.tv...")
send(sock, '223>>OPEN>>app.orchard.tv')
res3 = recv_msg(sock)
print("Launch response:", res3)

time.sleep(2)

# Request screenshot
print("Requesting screenshot...")
send(sock, '225>>')
res4 = recv_msg(sock)
print("Screenshot response:", res4)

sock.close()
