# -*- coding: utf-8 -*-
import os
import io
import gzip
import json
import hashlib
import tarfile
import time

root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
meta_file = os.path.join(root_dir, "meta.json")
version_tag = "Test-v5.9"
if os.path.exists(meta_file):
    try:
        with open(meta_file, 'r', encoding='utf-8') as mf:
            mdata = json.load(mf)
            v_str = mdata.get("version", "Test v5.9")
            version_tag = v_str.replace(" ", "-")
    except Exception as e:
        print(f"Warning: could not read version: {e}")

print(f"Generating CubeX-{version_tag}-docker-amd64.tar.gz (Container-Slim Enabled)...")

base_path = os.path.join(root_dir, "dist", "release", "CubeX-Test-v4.5-docker-amd64.tar.gz")
if not os.path.exists(base_path):
    base_path = os.path.join(root_dir, "dist", "docker", "CubeX-Test-v2.6-docker-amd64.tar.gz")

if not os.path.exists(base_path):
    print(f"Base file not found: {base_path}")
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

# 第一刀瘦身：使用标准编译的 cubexd-container-amd64 (~12MB)，并同步最新的通用兼容版 mihomo 与 sing-box
cubexd_path = os.path.join(root_dir, "dist", "cubexd-container-amd64")
if not os.path.exists(cubexd_path):
    cubexd_path = os.path.join(root_dir, "dist", "cubexd-linux-amd64")

mihomo_path = os.path.join(root_dir, "bin", "amd64", "mihomo")
singbox_path = os.path.join(root_dir, "bin", "amd64", "sing-box")
entrypoint_path = os.path.join(root_dir, "scripts", "docker", "entrypoint.sh")

for req_p in (cubexd_path, mihomo_path, singbox_path):
    if not os.path.exists(req_p):
        print(f"CRITICAL: Required amd64 binary missing: {req_p}")
        exit(1)

with open(cubexd_path, 'rb') as f:
    new_cubexd = f.read()
with open(mihomo_path, 'rb') as f:
    new_mihomo = f.read()
with open(singbox_path, 'rb') as f:
    new_singbox = f.read()
with open(meta_file, 'rb') as f:
    meta_bytes = f.read()
entrypoint_bytes = b""
if os.path.exists(entrypoint_path):
    with open(entrypoint_path, 'rb') as f:
        entrypoint_bytes = f.read()

# Safety gate: never package a v3-only amd64 core binary
for bname, braw in [("cubexd", new_cubexd), ("mihomo", new_mihomo), ("sing-box", new_singbox)]:
    if b"v3 microarchitecture support" in braw or b"v4 microarchitecture support" in braw:
        print(f"CRITICAL: Refusing to package x86-64-v3/v4 restricted binary for {bname}!")
        exit(1)

print(f"Read cubexd ({cubexd_path}): {len(new_cubexd)/(1024*1024):.2f} MB")
print(f"Read mihomo ({mihomo_path}): {len(new_mihomo)/(1024*1024):.2f} MB")
print(f"Read sing-box ({singbox_path}): {len(new_singbox)/(1024*1024):.2f} MB")

# Locate cfg and layers first, and strip stale app/bin/* & meta/entrypoint from all earlier layers
cfg_name = manifest[0]['Config']
cfg = json.loads(blobs[cfg_name].decode('utf-8'))
layers = list(manifest[0]['Layers'])

stale_paths = {
    'app/bin/cubexd', 'app/bin/mihomo', 'app/bin/sing-box',
    'app/meta.json', 'app/entrypoint.sh',
}
layer_idx = -1
for i, l in enumerate(layers):
    data = blobs[l]
    try:
        raw_tar = gzip.decompress(data)
        sub = tarfile.open(fileobj=io.BytesIO(raw_tar))
        names = set(sub.getnames())
        if 'app/bin/cubexd' in names and layer_idx == -1:
            layer_idx = i
        if names & stale_paths:
            clean_buf = io.BytesIO()
            with tarfile.open(fileobj=clean_buf, mode='w') as clean_out:
                for member in sub.getmembers():
                    if member.name.lstrip('./') in stale_paths:
                        continue
                    fobj = sub.extractfile(member) if member.isfile() else None
                    clean_out.addfile(member, fobj)
            clean_raw = clean_buf.getvalue()
            clean_gz = gzip.compress(clean_raw)
            new_blob_digest = hashlib.sha256(clean_gz).hexdigest()
            new_blob_name = f"blobs/sha256/{new_blob_digest}"
            blobs[new_blob_name] = clean_gz
            layers[i] = new_blob_name
            cfg['rootfs']['diff_ids'][i] = f"sha256:{hashlib.sha256(clean_raw).hexdigest()}"
    except Exception:
        pass

# Build updated app binary layer containing fresh cubexd + compatible mihomo + sing-box
buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w') as sub:
    ti = tarfile.TarInfo(name='app/bin')
    ti.type = tarfile.DIRTYPE
    ti.mode = 0o755
    ti.mtime = int(time.time())
    sub.addfile(ti)

    for rel_name, payload, mode in [
        ('app/bin/cubexd', new_cubexd, 0o755),
        ('app/bin/mihomo', new_mihomo, 0o755),
        ('app/bin/sing-box', new_singbox, 0o755),
        ('app/meta.json', meta_bytes, 0o644),
    ] + ([('app/entrypoint.sh', entrypoint_bytes, 0o755)] if entrypoint_bytes else []):
        ti = tarfile.TarInfo(name=rel_name)
        ti.type = tarfile.REGTYPE
        ti.mode = mode
        ti.size = len(payload)
        ti.mtime = int(time.time())
        sub.addfile(ti, io.BytesIO(payload))

uncompressed_layer = buf.getvalue()
diff_id = f"sha256:{hashlib.sha256(uncompressed_layer).hexdigest()}"
compressed_layer = gzip.compress(uncompressed_layer)
layer_digest = hashlib.sha256(compressed_layer).hexdigest()
new_layer_blob = f"blobs/sha256/{layer_digest}"

blobs[new_layer_blob] = compressed_layer

if layer_idx == -1:
    layer_idx = len(layers) - 1
print(f"Found cubexd target layer index: {layer_idx}")
layers[layer_idx] = new_layer_blob

# Update config json
cfg['rootfs']['diff_ids'][layer_idx] = diff_id
new_cfg_bytes = json.dumps(cfg, indent=2).encode('utf-8')
new_cfg_digest = hashlib.sha256(new_cfg_bytes).hexdigest()
new_cfg_blob = f"blobs/sha256/{new_cfg_digest}"
blobs[new_cfg_blob] = new_cfg_bytes

# Update manifest
manifest[0]['Config'] = new_cfg_blob
manifest[0]['Layers'] = layers
manifest[0]['RepoTags'] = [f'cubex:{version_tag}', 'cubex:latest']
manifest_bytes = json.dumps(manifest, indent=2).encode('utf-8')

# Write output tar.gz
out_path = os.path.join(root_dir, "dist", "release", f"CubeX-{version_tag}-docker-amd64.tar.gz")
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

print(f"Done! Generated {out_path} ({os.path.getsize(out_path)/(1024*1024):.2f} MB)")

# 🌟 生成 ARM64 Docker 离线镜像 (针对飞牛 fnOS ARM64 / OECT / 树莓派 / 开源软路由)
ros_arm64_path = os.path.join(root_dir, "dist", "release", f"CubeX-{version_tag}-ros-arm64.tar")
out_arm64_path = os.path.join(root_dir, "dist", "release", f"CubeX-{version_tag}-docker-arm64.tar.gz")
if os.path.exists(ros_arm64_path):
    print(f"Generating ARM64 Docker offline archive: {out_arm64_path}...")
    import shutil
    with open(ros_arm64_path, 'rb') as f_in, gzip.open(out_arm64_path, 'wb', compresslevel=6) as f_out:
        shutil.copyfileobj(f_in, f_out)
    print(f"Done! Generated {out_arm64_path} ({os.path.getsize(out_arm64_path)/(1024*1024):.2f} MB)")
else:
    print(f"Notice: {ros_arm64_path} not found yet (will be generated by build_ros_containers.py)")
