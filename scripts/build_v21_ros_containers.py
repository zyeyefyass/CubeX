# -*- coding: utf-8 -*-
import os
import io
import gzip
import json
import hashlib
import tarfile
import time

print("🔨 Starting true OCI/Docker container packaging for RouterOS Test v3.1...")

root_dir = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
dist_dir = os.path.join(root_dir, "dist")
release_dir = os.path.join(dist_dir, "release")
os.makedirs(release_dir, exist_ok=True)

# ═════════════════════════════════════════════════════════════════════════════
# 1. Package AMD64 RouterOS OCI Container
# ═════════════════════════════════════════════════════════════════════════════
base_tar_amd64 = os.path.join(dist_dir, "docker", "CubeX-Test-v0.99-ros-amd64-oci.tar")
cubexd_amd64 = os.path.join(dist_dir, "cubexd-linux-amd64")
cubex_cli_amd64 = os.path.join(dist_dir, "cubex-linux-amd64")
singbox_amd64 = os.path.join(root_dir, "bin", "amd64", "sing-box")
mihomo_amd64 = os.path.join(root_dir, "bin", "amd64", "mihomo")
out_tar_amd64 = os.path.join(release_dir, "CubeX-Test-v3.1-ros-amd64.tar")

print(f"\n📦 [AMD64] Opening base OCI tar: {base_tar_amd64}")
blobs = {}
with tarfile.open(base_tar_amd64, 'r') as t:
    manifest = json.loads(t.extractfile('manifest.json').read().decode())
    for m in t.getmembers():
        f = t.extractfile(m)
        if f is not None and m.name != 'manifest.json':
            blobs[m.name] = f.read()

cfg_name = manifest[0]['Config']
cfg = json.loads(blobs[cfg_name].decode())
layers = list(manifest[0]['Layers'])

# Locate the layer where app binaries/rules reside
cubexd_layer_idx = 4
for i, l in enumerate(layers):
    data = blobs[l]
    try:
        sub = tarfile.open(fileobj=io.BytesIO(data))
        if 'app/bin/cubexd' in sub.getnames() or 'app/bin/sing-box' in sub.getnames():
            cubexd_layer_idx = i
            break
    except Exception:
        pass

print(f"📦 [AMD64] Target layer index to update: {cubexd_layer_idx}")

with open(cubexd_amd64, 'rb') as f:
    new_cubexd = f.read()
with open(singbox_amd64, 'rb') as f:
    new_singbox = f.read()
new_cubex_cli = b""
if os.path.exists(cubex_cli_amd64):
    with open(cubex_cli_amd64, 'rb') as f:
        new_cubex_cli = f.read()
new_mihomo = b""
if os.path.exists(mihomo_amd64):
    with open(mihomo_amd64, 'rb') as f:
        new_mihomo = f.read()

rules_dir = os.path.join(root_dir, "etc", "rules")
meta_file = os.path.join(root_dir, "meta.json")
with open(meta_file, 'rb') as f:
    meta_bytes = f.read()

# Build updated layer tarball
buf = io.BytesIO()
with tarfile.open(fileobj=buf, mode='w') as sub:
    dirs = [
        'app', 'app/bin', 'app/etc', 'app/etc/rules', 'app/data',
        'app/defaults', 'app/defaults/etc', 'app/defaults/etc/rules', 'app/defaults/data'
    ]
    for d in dirs:
        ti = tarfile.TarInfo(name=d)
        ti.type = tarfile.DIRTYPE
        ti.mode = 0o755
        ti.mtime = int(time.time())
        sub.addfile(ti)

    # Binaries
    ti = tarfile.TarInfo(name='app/bin/cubexd')
    ti.type = tarfile.REGTYPE
    ti.mode = 0o755
    ti.size = len(new_cubexd)
    ti.mtime = int(time.time())
    sub.addfile(ti, io.BytesIO(new_cubexd))

    if new_cubex_cli:
        ti = tarfile.TarInfo(name='app/bin/cubex')
        ti.type = tarfile.REGTYPE
        ti.mode = 0o755
        ti.size = len(new_cubex_cli)
        ti.mtime = int(time.time())
        sub.addfile(ti, io.BytesIO(new_cubex_cli))

    ti = tarfile.TarInfo(name='app/bin/sing-box')
    ti.type = tarfile.REGTYPE
    ti.mode = 0o755
    ti.size = len(new_singbox)
    ti.mtime = int(time.time())
    sub.addfile(ti, io.BytesIO(new_singbox))

    if new_mihomo:
        ti = tarfile.TarInfo(name='app/bin/mihomo')
        ti.type = tarfile.REGTYPE
        ti.mode = 0o755
        ti.size = len(new_mihomo)
        ti.mtime = int(time.time())
        sub.addfile(ti, io.BytesIO(new_mihomo))

    # Meta
    for p in ['app/meta.json', 'app/defaults/meta.json']:
        ti = tarfile.TarInfo(name=p)
        ti.type = tarfile.REGTYPE
        ti.mode = 0o644
        ti.size = len(meta_bytes)
        ti.mtime = int(time.time())
        sub.addfile(ti, io.BytesIO(meta_bytes))

    # All 101 Rules fully bundled
    if os.path.exists(rules_dir):
        for r in os.listdir(rules_dir):
            rpath = os.path.join(rules_dir, r)
            if os.path.isfile(rpath):
                with open(rpath, 'rb') as rf:
                    rdata = rf.read()
                for prefix in ['app/etc/rules', 'app/defaults/etc/rules']:
                    ti = tarfile.TarInfo(name=f'{prefix}/{r}')
                    ti.type = tarfile.REGTYPE
                    ti.mode = 0o644
                    ti.size = len(rdata)
                    ti.mtime = int(time.time())
                    sub.addfile(ti, io.BytesIO(rdata))

    # Top-level geodata files in etc/
    etc_dir = os.path.join(root_dir, "etc")
    for geofile in ['GeoIP.dat', 'geosite.dat', 'geoip.metadb']:
        geopath = os.path.join(etc_dir, geofile)
        if os.path.exists(geopath):
            with open(geopath, 'rb') as gf:
                gdata = gf.read()
            for prefix in ['app/etc', 'app/defaults/etc']:
                ti = tarfile.TarInfo(name=f'{prefix}/{geofile}')
                ti.type = tarfile.REGTYPE
                ti.mode = 0o644
                ti.size = len(gdata)
                ti.mtime = int(time.time())
                sub.addfile(ti, io.BytesIO(gdata))

    # Entrypoint script
    entrypoint_path = os.path.join(root_dir, "scripts", "docker", "entrypoint.sh")
    if os.path.exists(entrypoint_path):
        with open(entrypoint_path, 'rb') as ef:
            edata = ef.read()
        ti = tarfile.TarInfo(name='app/entrypoint.sh')
        ti.type = tarfile.REGTYPE
        ti.mode = 0o755
        ti.size = len(edata)
        ti.mtime = int(time.time())
        sub.addfile(ti, io.BytesIO(edata))

uncompressed_layer = buf.getvalue()
diff_id = f"sha256:{hashlib.sha256(uncompressed_layer).hexdigest()}"
compressed_layer = gzip.compress(uncompressed_layer)
layer_digest = hashlib.sha256(compressed_layer).hexdigest()
new_layer_blob_name = f"blobs/sha256/{layer_digest}"

print(f"📦 [AMD64] New layer uncompressed: {len(uncompressed_layer)} bytes, compressed: {len(compressed_layer)} bytes")
blobs[new_layer_blob_name] = compressed_layer

# Prune obsolete/duplicate historical layers 5..10 (which contained old v0.99 binaries and bloated the archive by 57MB)
layers = layers[:cubexd_layer_idx+1]
layers[cubexd_layer_idx] = new_layer_blob_name

# Update config json
cfg['rootfs']['diff_ids'] = cfg['rootfs']['diff_ids'][:cubexd_layer_idx+1]
cfg['rootfs']['diff_ids'][cubexd_layer_idx] = diff_id
new_cfg_bytes = json.dumps(cfg, indent=2).encode('utf-8')
new_cfg_digest = hashlib.sha256(new_cfg_bytes).hexdigest()
new_cfg_blob_name = f"blobs/sha256/{new_cfg_digest}"
blobs[new_cfg_blob_name] = new_cfg_bytes

# Update manifest
manifest[0]['Config'] = new_cfg_blob_name
manifest[0]['Layers'] = layers
manifest_bytes = json.dumps(manifest, indent=2).encode('utf-8')

# Build OCI manifest & index
oci_manifest_layers = []
for l in layers:
    oci_manifest_layers.append({
        'mediaType': 'application/vnd.oci.image.layer.v1.tar+gzip',
        'size': len(blobs[l]),
        'digest': f"sha256:{l.split('/')[-1]}"
    })

oci_manifest = {
    'schemaVersion': 2,
    'mediaType': 'application/vnd.oci.image.manifest.v1+json',
    'config': {
        'mediaType': 'application/vnd.oci.image.config.v1+json',
        'size': len(new_cfg_bytes),
        'digest': f'sha256:{new_cfg_digest}'
    },
    'layers': oci_manifest_layers
}
oci_manifest_bytes = json.dumps(oci_manifest, indent=2).encode('utf-8')
oci_manifest_digest = hashlib.sha256(oci_manifest_bytes).hexdigest()
oci_manifest_blob_name = f'blobs/sha256/{oci_manifest_digest}'
blobs[oci_manifest_blob_name] = oci_manifest_bytes

oci_index = {
    'schemaVersion': 2,
    'mediaType': 'application/vnd.oci.image.index.v1+json',
    'manifests': [
        {
            'mediaType': 'application/vnd.oci.image.manifest.v1+json',
            'digest': f'sha256:{oci_manifest_digest}',
            'size': len(oci_manifest_bytes),
            'annotations': {
                'io.containerd.image.name': 'docker.io/library/cubex:latest',
                'org.opencontainers.image.ref.name': 'latest'
            },
            'platform': {
                'architecture': 'amd64',
                'os': 'linux'
            }
        }
    ]
}
index_bytes = json.dumps(oci_index, indent=2).encode('utf-8')

print(f"📦 [AMD64] Writing valid RouterOS OCI container to: {out_tar_amd64}")
with tarfile.open(out_tar_amd64, 'w') as out:
    def add_mem(name, b):
        ti = tarfile.TarInfo(name=name)
        ti.size = len(b)
        ti.mtime = int(time.time())
        ti.mode = 0o644
        out.addfile(ti, io.BytesIO(b))

    add_mem('oci-layout', b'{"imageLayoutVersion":"1.0.0"}')
    add_mem('index.json', index_bytes)
    add_mem('manifest.json', manifest_bytes)
    all_blobs = [oci_manifest_blob_name, new_cfg_blob_name] + layers
    for b in all_blobs:
        add_mem(b, blobs[b])

size_mb = os.path.getsize(out_tar_amd64) / (1024 * 1024)
print(f"✅ [AMD64] Done! Generated {out_tar_amd64} ({size_mb:.2f} MB)")

# ═════════════════════════════════════════════════════════════════════════════
# 2. Package ARM64 RouterOS OCI Container
# ═════════════════════════════════════════════════════════════════════════════
base_tar_arm64 = r'd:\Micube\dist\MiCube-arm64.tar'
cubexd_arm64 = os.path.join(dist_dir, "cubexd-linux-arm64")
cubex_cli_arm64 = os.path.join(dist_dir, "cubex-linux-arm64")
singbox_arm64 = os.path.join(root_dir, "bin", "arm64", "sing-box")
mihomo_arm64 = os.path.join(root_dir, "bin", "arm64", "mihomo")
out_tar_arm64 = os.path.join(release_dir, "CubeX-Test-v3.1-ros-arm64.tar")

if os.path.exists(base_tar_arm64):
    print(f"\n📦 [ARM64] Opening base OCI tar: {base_tar_arm64}")
    blobs_arm = {}
    with tarfile.open(base_tar_arm64, 'r') as t:
        manifest_arm = json.loads(t.extractfile('manifest.json').read().decode())
        for m in t.getmembers():
            f = t.extractfile(m)
            if f is not None and m.name != 'manifest.json':
                blobs_arm[m.name] = f.read()

    cfg_name_arm = manifest_arm[0]['Config']
    cfg_arm = json.loads(blobs_arm[cfg_name_arm].decode())
    layers_arm = list(manifest_arm[0]['Layers'])

    arm_layer_idx = 4
    for i, l in enumerate(layers_arm):
        data = blobs_arm[l]
        try:
            sub = tarfile.open(fileobj=io.BytesIO(data))
            if 'app/bin/cubexd' in sub.getnames() or 'app/bin/sing-box' in sub.getnames():
                arm_layer_idx = i
                break
        except Exception:
            pass

    with open(cubexd_arm64, 'rb') as f:
        new_cubexd_arm = f.read()
    with open(singbox_arm64, 'rb') as f:
        new_singbox_arm = f.read()
    new_cli_arm = b""
    if os.path.exists(cubex_cli_arm64):
        with open(cubex_cli_arm64, 'rb') as f:
            new_cli_arm = f.read()
    new_mihomo_arm = b""
    if os.path.exists(mihomo_arm64):
        with open(mihomo_arm64, 'rb') as f:
            new_mihomo_arm = f.read()

    buf_arm = io.BytesIO()
    with tarfile.open(fileobj=buf_arm, mode='w') as sub:
        for d in dirs:
            ti = tarfile.TarInfo(name=d)
            ti.type = tarfile.DIRTYPE
            ti.mode = 0o755
            ti.mtime = int(time.time())
            sub.addfile(ti)

        ti = tarfile.TarInfo(name='app/bin/cubexd')
        ti.type = tarfile.REGTYPE
        ti.mode = 0o755
        ti.size = len(new_cubexd_arm)
        ti.mtime = int(time.time())
        sub.addfile(ti, io.BytesIO(new_cubexd_arm))

        if new_cli_arm:
            ti = tarfile.TarInfo(name='app/bin/cubex')
            ti.type = tarfile.REGTYPE
            ti.mode = 0o755
            ti.size = len(new_cli_arm)
            ti.mtime = int(time.time())
            sub.addfile(ti, io.BytesIO(new_cli_arm))

        ti = tarfile.TarInfo(name='app/bin/sing-box')
        ti.type = tarfile.REGTYPE
        ti.mode = 0o755
        ti.size = len(new_singbox_arm)
        ti.mtime = int(time.time())
        sub.addfile(ti, io.BytesIO(new_singbox_arm))

        if new_mihomo_arm:
            ti = tarfile.TarInfo(name='app/bin/mihomo')
            ti.type = tarfile.REGTYPE
            ti.mode = 0o755
            ti.size = len(new_mihomo_arm)
            ti.mtime = int(time.time())
            sub.addfile(ti, io.BytesIO(new_mihomo_arm))

        for p in ['app/meta.json', 'app/defaults/meta.json']:
            ti = tarfile.TarInfo(name=p)
            ti.type = tarfile.REGTYPE
            ti.mode = 0o644
            ti.size = len(meta_bytes)
            ti.mtime = int(time.time())
            sub.addfile(ti, io.BytesIO(meta_bytes))

        # All 101 Rules fully bundled
        if os.path.exists(rules_dir):
            for r in os.listdir(rules_dir):
                rpath = os.path.join(rules_dir, r)
                if os.path.isfile(rpath):
                    with open(rpath, 'rb') as rf:
                        rdata = rf.read()
                    for prefix in ['app/etc/rules', 'app/defaults/etc/rules']:
                        ti = tarfile.TarInfo(name=f'{prefix}/{r}')
                        ti.type = tarfile.REGTYPE
                        ti.mode = 0o644
                        ti.size = len(rdata)
                        ti.mtime = int(time.time())
                        sub.addfile(ti, io.BytesIO(rdata))

        for geofile in ['GeoIP.dat', 'geosite.dat', 'geoip.metadb']:
            geopath = os.path.join(etc_dir, geofile)
            if os.path.exists(geopath):
                with open(geopath, 'rb') as gf:
                    gdata = gf.read()
                for prefix in ['app/etc', 'app/defaults/etc']:
                    ti = tarfile.TarInfo(name=f'{prefix}/{geofile}')
                    ti.type = tarfile.REGTYPE
                    ti.mode = 0o644
                    ti.size = len(gdata)
                    ti.mtime = int(time.time())
                    sub.addfile(ti, io.BytesIO(gdata))

    uncompressed_arm = buf_arm.getvalue()
    diff_id_arm = f"sha256:{hashlib.sha256(uncompressed_arm).hexdigest()}"
    compressed_arm = gzip.compress(uncompressed_arm)
    layer_digest_arm = hashlib.sha256(compressed_arm).hexdigest()
    new_arm_layer_blob = f"blobs/sha256/{layer_digest_arm}"

    blobs_arm[new_arm_layer_blob] = compressed_arm
    layers_arm[arm_layer_idx] = new_arm_layer_blob

    cfg_arm['rootfs']['diff_ids'][arm_layer_idx] = diff_id_arm
    new_cfg_bytes_arm = json.dumps(cfg_arm, indent=2).encode('utf-8')
    new_cfg_digest_arm = hashlib.sha256(new_cfg_bytes_arm).hexdigest()
    new_cfg_blob_arm = f"blobs/sha256/{new_cfg_digest_arm}"
    blobs_arm[new_cfg_blob_arm] = new_cfg_bytes_arm

    manifest_arm[0]['Config'] = new_cfg_blob_arm
    manifest_arm[0]['Layers'] = layers_arm
    manifest_bytes_arm = json.dumps(manifest_arm, indent=2).encode('utf-8')

    oci_manifest_layers_arm = []
    for l in layers_arm:
        oci_manifest_layers_arm.append({
            'mediaType': 'application/vnd.oci.image.layer.v1.tar+gzip',
            'size': len(blobs_arm[l]),
            'digest': f"sha256:{l.split('/')[-1]}"
        })

    oci_manifest_arm = {
        'schemaVersion': 2,
        'mediaType': 'application/vnd.oci.image.manifest.v1+json',
        'config': {
            'mediaType': 'application/vnd.oci.image.config.v1+json',
            'size': len(new_cfg_bytes_arm),
            'digest': f'sha256:{new_cfg_digest_arm}'
        },
        'layers': oci_manifest_layers_arm
    }
    oci_manifest_bytes_arm = json.dumps(oci_manifest_arm, indent=2).encode('utf-8')
    oci_manifest_digest_arm = hashlib.sha256(oci_manifest_bytes_arm).hexdigest()
    oci_manifest_blob_arm = f'blobs/sha256/{oci_manifest_digest_arm}'
    blobs_arm[oci_manifest_blob_arm] = oci_manifest_bytes_arm

    oci_index_arm = {
        'schemaVersion': 2,
        'mediaType': 'application/vnd.oci.image.index.v1+json',
        'manifests': [
            {
                'mediaType': 'application/vnd.oci.image.manifest.v1+json',
                'digest': f'sha256:{oci_manifest_digest_arm}',
                'size': len(oci_manifest_bytes_arm),
                'annotations': {
                    'io.containerd.image.name': 'docker.io/library/cubex:latest',
                    'org.opencontainers.image.ref.name': 'latest'
                },
                'platform': {
                    'architecture': 'arm64',
                    'os': 'linux'
                }
            }
        ]
    }
    index_bytes_arm = json.dumps(oci_index_arm, indent=2).encode('utf-8')

    print(f"📦 [ARM64] Writing valid RouterOS OCI container to: {out_tar_arm64}")
    with tarfile.open(out_tar_arm64, 'w') as out:
        def add_mem_arm(name, b):
            ti = tarfile.TarInfo(name=name)
            ti.size = len(b)
            ti.mtime = int(time.time())
            ti.mode = 0o644
            out.addfile(ti, io.BytesIO(b))

        add_mem_arm('oci-layout', b'{"imageLayoutVersion":"1.0.0"}')
        add_mem_arm('index.json', index_bytes_arm)
        add_mem_arm('manifest.json', manifest_bytes_arm)
        all_blobs_arm = [oci_manifest_blob_arm, new_cfg_blob_arm] + layers_arm
        for b in all_blobs_arm:
            add_mem_arm(b, blobs_arm[b])

    size_mb_arm = os.path.getsize(out_tar_arm64) / (1024 * 1024)
    print(f"✅ [ARM64] Done! Generated {out_tar_arm64} ({size_mb_arm:.2f} MB)")
else:
    print(f"⚠️  Base ARM64 tar not found at {base_tar_arm64}, skipping ARM64 OCI container generation.")

print("\n🎉 True RouterOS OCI Containers Packaging Complete!")
