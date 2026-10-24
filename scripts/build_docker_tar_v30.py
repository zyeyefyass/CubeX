# -*- coding: utf-8 -*-
import os
import io
import gzip
import json
import hashlib
import tarfile
import time

print("🔨 Generating CubeX-Test-v3.4-docker-amd64.tar.gz from v2.7 base...")

root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
base_path = os.path.join(root_dir, "dist", "release", "CubeX-Test-v2.9-docker-amd64.tar.gz")
if not os.path.exists(base_path):
    base_path = os.path.join(root_dir, "dist", "docker", "CubeX-Test-v2.6-docker-amd64.tar.gz")

if not os.path.exists(base_path):
    print(f"❌ Base file not found: {base_path}")
    exit(1)

with open(base_path, 'rb') as f:
    raw_bytes = f.read()

print(f"Loaded base ({len(raw_bytes)/(1024*1024):.2f} MB)...")

t_in = tarfile.open(fileobj=io.BytesIO(raw_bytes), mode='r:gz')
blobs = {}
manifest = None
index = None
oci_layout = None

for m in t_in.getmembers():
    f = t_in.extractfile(m)
    if f is not None:
        if m.name == 'manifest.json':
            manifest = json.loads(f.read().decode('utf-8'))
        elif m.name == 'index.json':
            index = f.read()
        elif m.name == 'oci-layout':
            oci_layout = f.read()
        else:
            blobs[m.name] = f.read()

cubexd_path = os.path.join(root_dir, "dist", "cubexd-linux-amd64")
with open(cubexd_path, 'rb') as f:
    new_cubexd = f.read()

print(f"Read new cubexd Test v3.4: {len(new_cubexd)} bytes")

# Build updated cubexd layer
buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w') as sub:
    ti = tarfile.TarInfo(name='app/bin')
    ti.type = tarfile.DIRTYPE
    ti.mode = 0o755
    ti.mtime = int(time.time())
    sub.addfile(ti)

    ti = tarfile.TarInfo(name='app/bin/cubexd')
    ti.type = tarfile.REGTYPE
    ti.mode = 0o755
    ti.size = len(new_cubexd)
    ti.mtime = int(time.time())
    sub.addfile(ti, io.BytesIO(new_cubexd))

uncompressed_layer = buf.getvalue()
diff_id = f"sha256:{hashlib.sha256(uncompressed_layer).hexdigest()}"
compressed_layer = gzip.compress(uncompressed_layer)
layer_digest = hashlib.sha256(compressed_layer).hexdigest()
new_layer_blob = f"blobs/sha256/{layer_digest}"

blobs[new_layer_blob] = compressed_layer

# Find which layer contains cubexd dynamically
layers = list(manifest[0]['Layers'])
layer_idx = -1
for i, l in enumerate(layers):
    data = blobs[l]
    try:
        sub = tarfile.open(fileobj=io.BytesIO(gzip.decompress(data)))
        if 'app/bin/cubexd' in sub.getnames():
            layer_idx = i
            break
    except Exception:
        pass

if layer_idx == -1:
    layer_idx = 4
print(f"Found cubexd target layer index: {layer_idx}")
layers[layer_idx] = new_layer_blob

# Update config json
cfg_name = manifest[0]['Config']
cfg = json.loads(blobs[cfg_name].decode('utf-8'))
cfg['rootfs']['diff_ids'][layer_idx] = diff_id
new_cfg_bytes = json.dumps(cfg, indent=2).encode('utf-8')
new_cfg_digest = hashlib.sha256(new_cfg_bytes).hexdigest()
new_cfg_blob = f"blobs/sha256/{new_cfg_digest}"
blobs[new_cfg_blob] = new_cfg_bytes

# Update manifest
manifest[0]['Config'] = new_cfg_blob
manifest[0]['Layers'] = layers
manifest[0]['RepoTags'] = ['cubex:Test-v3.4', 'cubex:latest']
manifest_bytes = json.dumps(manifest, indent=2).encode('utf-8')

# Write output tar.gz
out_path = os.path.join(root_dir, "dist", "release", "CubeX-Test-v3.4-docker-amd64.tar.gz")
os.makedirs(os.path.dirname(out_path), exist_ok=True)
with tarfile.open(out_path, mode='w:gz') as t_out:
    def add_data(name, b):
        ti = tarfile.TarInfo(name=name)
        ti.size = len(b)
        ti.mtime = int(time.time())
        ti.mode = 0o644
        t_out.addfile(ti, io.BytesIO(b))

    add_data('oci-layout', oci_layout or b'{"imageLayoutVersion":"1.0.0"}')
    if index:
        add_data('index.json', index)
    add_data('manifest.json', manifest_bytes)
    for b_name in [new_cfg_blob] + layers:
        add_data(b_name, blobs[b_name])

print(f"✅ Generated {out_path} ({os.path.getsize(out_path)/(1024*1024):.2f} MB)")
