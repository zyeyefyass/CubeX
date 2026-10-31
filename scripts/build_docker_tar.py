# -*- coding: utf-8 -*-
import os
import io
import gzip
import json
import hashlib
import tarfile
import time

root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
with open(os.path.join(root_dir, "meta.json"), "r", encoding="utf-8") as f:
    meta = json.load(f)
ver_tag = meta.get("version", "Test v4.7").replace(" ", "-")

print(f"🔨 Generating CubeX-{ver_tag}-docker-amd64.tar.gz...")

base_path = None
for cand in [
    os.path.join(root_dir, "dist", "release", "CubeX-Test-v4.6-docker-amd64.tar.gz"),
    os.path.join(root_dir, "dist", "release", "CubeX-Test-v4.5-docker-amd64.tar.gz"),
    os.path.join(root_dir, "dist", "docker", "CubeX-Test-v2.6-docker-amd64.tar.gz")
]:
    if os.path.exists(cand):
        base_path = cand
        break

if not base_path or not os.path.exists(base_path):
    print(f"❌ Base docker file not found!")
    exit(1)

with open(base_path, 'rb') as f:
    raw_bytes = f.read()

print(f"Loaded base from {base_path} ({len(raw_bytes)/(1024*1024):.2f} MB)...")

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

rules_dir = os.path.join(root_dir, "etc", "rules")
meta_file = os.path.join(root_dir, "meta.json")
with open(meta_file, 'rb') as f:
    meta_bytes = f.read()

buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w') as sub:
    for d in ['app', 'app/bin', 'app/etc', 'app/etc/rules', 'app/data', 'app/defaults', 'app/defaults/etc', 'app/defaults/etc/rules', 'app/defaults/data']:
        ti = tarfile.TarInfo(name=d)
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

    for p in ['app/meta.json', 'app/defaults/meta.json']:
        ti = tarfile.TarInfo(name=p)
        ti.type = tarfile.REGTYPE
        ti.mode = 0o644
        ti.size = len(meta_bytes)
        ti.mtime = int(time.time())
        sub.addfile(ti, io.BytesIO(meta_bytes))

    if os.path.exists(rules_dir):
        for r in os.listdir(rules_dir):
            rpath = os.path.join(rules_dir, r)
            if os.path.isfile(rpath):
                with open(rpath, 'rb') as rf:
                    rdata = rf.read()
                for prefix in ['app/etc/rules', 'app/defaults/etc/rules']:
                    ti = tarfile.TarInfo(name=f"{prefix}/{r}")
                    ti.type = tarfile.REGTYPE
                    ti.mode = 0o644
                    ti.size = len(rdata)
                    ti.mtime = int(time.time())
                    sub.addfile(ti, io.BytesIO(rdata))

diff_id = f"sha256:{hashlib.sha256(buf.getvalue()).hexdigest()}"
new_layer_gz = gzip.compress(buf.getvalue())
new_layer_digest = hashlib.sha256(new_layer_gz).hexdigest()
new_layer_blob = f"blobs/sha256/{new_layer_digest}"
blobs[new_layer_blob] = new_layer_gz

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

cfg_name = manifest[0]['Config']
cfg = json.loads(blobs[cfg_name].decode('utf-8'))
cfg['rootfs']['diff_ids'][layer_idx] = diff_id
new_cfg_bytes = json.dumps(cfg, indent=2).encode('utf-8')
new_cfg_digest = hashlib.sha256(new_cfg_bytes).hexdigest()
new_cfg_blob = f"blobs/sha256/{new_cfg_digest}"
blobs[new_cfg_blob] = new_cfg_bytes

manifest[0]['Config'] = new_cfg_blob
manifest[0]['Layers'] = layers
manifest[0]['RepoTags'] = [f'cubex:{ver_tag}', 'cubex:latest']
manifest_bytes = json.dumps(manifest, indent=2).encode('utf-8')

out_path = os.path.join(root_dir, "dist", "release", f"CubeX-{ver_tag}-docker-amd64.tar.gz")
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
