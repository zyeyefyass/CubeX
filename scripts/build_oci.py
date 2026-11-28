#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Unified Clean-Room OCI & Docker Image Builder
- Extracts ONLY Alpine L0 + L1 base layers from dist/docker base tarballs
  and strips any legacy non-clean-room paths (e.g. opt/MiCube).
- Builds ONE clean application layer using dist/cubexd-linux-{arch},
  precompiled cores (bin/{arch}/{sing-box,mihomo}), packaging/defaults/data/config.ir.json,
  and single-copy /app/defaults/etc/{rules,GeoIP.dat,geosite.dat} (no duplicate /app/etc copy,
  no unused geoip.metadb by default).
- Outputs:
  * CubeX-<ver>-ros-{amd64,arm64}.tar        (uncompressed outer tar, gzip-9 inner blobs for RouterOS)
  * CubeX-<ver>-docker-{amd64,arm64}.tar.gz  (uncompressed inner blobs, single-pass gzip-9 outer archive)
"""
import argparse
import gzip
import hashlib
import io
import json
import os
import re
import sys
import tarfile
import time

ROOT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
DIST_DIR = os.path.join(ROOT_DIR, "dist")
RELEASE_DIR = os.path.join(DIST_DIR, "release")

FORBIDDEN_BASE_PATH_RE = re.compile(r"(micube|open[-_]?box|ros[-_]?box)", re.IGNORECASE)


def read_meta_version() -> str:
    meta_file = os.path.join(ROOT_DIR, "meta.json")
    with open(meta_file, "r", encoding="utf-8") as mf:
        v_str = str(json.load(mf).get("version", "")).strip()
    if not v_str:
        raise RuntimeError("meta.json missing 'version'")
    return v_str.replace(" ", "-")


def find_base_oci_tar(arch: str) -> str:
    docker_dir = os.path.join(DIST_DIR, "docker")
    candidates = {
        "amd64": ["CubeX-Test-v0.99-ros-amd64-oci.tar", "CubeX-ros-amd64-oci-base.tar"],
        "arm64": ["CubeX-ros-arm64-oci-base.tar", "CubeX-Test-v0.99-ros-arm64-oci.tar"],
    }[arch]
    for name in candidates:
        p = os.path.join(docker_dir, name)
        if os.path.exists(p):
            return p
    raise FileNotFoundError(f"Base Alpine OCI tar not found for {arch} in {docker_dir}")


def extract_clean_alpine_layers(base_tar_path: str) -> tuple[list[bytes], dict]:
    """Extract only L0 and L1 from base tar, stripping any forbidden legacy paths or stale /app files."""
    with tarfile.open(base_tar_path, "r") as t:
        manifest = json.loads(t.extractfile("manifest.json").read().decode("utf-8"))
        cfg_name = manifest[0]["Config"]
        base_cfg = json.loads(t.extractfile(cfg_name).read().decode("utf-8"))
        layer_names = list(manifest[0]["Layers"])[:2]

        clean_layers: list[bytes] = []
        for idx, lname in enumerate(layer_names):
            raw = t.extractfile(lname).read()
            uncomp = gzip.decompress(raw) if raw[:2] == b"\x1f\x8b" else raw
            in_sub = tarfile.open(fileobj=io.BytesIO(uncomp), mode="r:")
            out_buf = io.BytesIO()
            stripped_count = 0
            with tarfile.open(fileobj=out_buf, mode="w:") as out_sub:
                for member in in_sub.getmembers():
                    norm = member.name.lstrip("./")
                    if FORBIDDEN_BASE_PATH_RE.search(norm) or norm == "app" or norm.startswith("app/"):
                        stripped_count += 1
                        continue
                    fobj = in_sub.extractfile(member) if member.isfile() else None
                    out_sub.addfile(member, fobj)
            if stripped_count:
                print(f"   🧹 Stripped {stripped_count} non-clean entries from base L{idx}")
            clean_layers.append(out_buf.getvalue())

    return clean_layers, base_cfg


def build_app_layer_uncompressed(arch: str, include_metadb: bool = False) -> bytes:
    """Build single clean application layer tar (no duplicate /app/etc, no CLI, clean default IR)."""
    cubexd_path = os.path.join(DIST_DIR, f"cubexd-linux-{arch}")
    mihomo_path = os.path.join(ROOT_DIR, "bin", arch, "mihomo")
    singbox_path = os.path.join(ROOT_DIR, "bin", arch, "sing-box")
    meta_path = os.path.join(ROOT_DIR, "meta.json")
    default_ir_path = os.path.join(ROOT_DIR, "packaging", "defaults", "data", "config.ir.json")
    entrypoint_path = os.path.join(ROOT_DIR, "scripts", "docker", "entrypoint.sh")
    rules_dir = os.path.join(ROOT_DIR, "etc", "rules")
    etc_dir = os.path.join(ROOT_DIR, "etc")

    for req in (cubexd_path, mihomo_path, singbox_path, meta_path, default_ir_path, entrypoint_path):
        if not os.path.exists(req):
            raise FileNotFoundError(f"Required asset missing for {arch}: {req}")

    with open(cubexd_path, "rb") as f:
        cubexd_bytes = f.read()
    with open(mihomo_path, "rb") as f:
        mihomo_bytes = f.read()
    with open(singbox_path, "rb") as f:
        singbox_bytes = f.read()

    if arch == "amd64":
        for bname, braw in [("cubexd", cubexd_bytes), ("mihomo", mihomo_bytes), ("sing-box", singbox_bytes)]:
            if b"v3 microarchitecture support" in braw or b"v4 microarchitecture support" in braw:
                raise RuntimeError(f"CRITICAL: Refusing x86-64-v3/v4 restricted binary for {bname}!")

    with open(meta_path, "rb") as f:
        meta_bytes = f.read()
    with open(default_ir_path, "rb") as f:
        ir_bytes = f.read()
    with open(entrypoint_path, "rb") as f:
        entrypoint_bytes = f.read().replace(b"\r\n", b"\n")

    now = int(time.time())
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as sub:
        dirs = [
            "app", "app/bin", "app/etc", "app/etc/rules", "app/data",
            "app/defaults", "app/defaults/etc", "app/defaults/etc/rules", "app/defaults/data",
        ]
        for d in dirs:
            ti = tarfile.TarInfo(name=d)
            ti.type = tarfile.DIRTYPE
            ti.mode = 0o755
            ti.mtime = now
            sub.addfile(ti)

        def add_file(rel: str, payload: bytes, mode: int):
            ti = tarfile.TarInfo(name=rel)
            ti.type = tarfile.REGTYPE
            ti.mode = mode
            ti.size = len(payload)
            ti.mtime = now
            sub.addfile(ti, io.BytesIO(payload))

        add_file("app/bin/cubexd", cubexd_bytes, 0o755)
        add_file("app/bin/sing-box", singbox_bytes, 0o755)
        add_file("app/bin/mihomo", mihomo_bytes, 0o755)
        add_file("app/entrypoint.sh", entrypoint_bytes, 0o755)
        add_file("app/meta.json", meta_bytes, 0o644)
        add_file("app/defaults/meta.json", meta_bytes, 0o644)
        add_file("app/defaults/data/config.ir.json", ir_bytes, 0o644)

        if os.path.isdir(rules_dir):
            for r in sorted(os.listdir(rules_dir)):
                rp = os.path.join(rules_dir, r)
                if os.path.isfile(rp):
                    with open(rp, "rb") as rf:
                        add_file(f"app/defaults/etc/rules/{r}", rf.read(), 0o644)

        geofiles = ["GeoIP.dat", "geosite.dat"] + (["geoip.metadb"] if include_metadb else [])
        for gf in geofiles:
            gp = os.path.join(etc_dir, gf)
            if os.path.exists(gp):
                with open(gp, "rb") as fp:
                    add_file(f"app/defaults/etc/{gf}", fp.read(), 0o644)

    return buf.getvalue()


def assemble_oci_archive(
    arch: str,
    version_tag: str,
    uncompressed_layers: list[bytes],
    base_cfg: dict,
    out_path: str,
    compress_inner_blobs: bool,
    outer_gzip: bool,
):
    """Write a complete OCI / Docker-load compatible archive."""
    blobs: dict[str, bytes] = {}
    layer_names: list[str] = []
    diff_ids: list[str] = []

    for idx, raw_layer in enumerate(uncompressed_layers):
        diff_ids.append(f"sha256:{hashlib.sha256(raw_layer).hexdigest()}")
        payload = gzip.compress(raw_layer, compresslevel=9) if compress_inner_blobs else raw_layer
        digest = hashlib.sha256(payload).hexdigest()
        bname = f"blobs/sha256/{digest}"
        blobs[bname] = payload
        layer_names.append(bname)

    cfg = {
        "architecture": arch,
        "os": "linux",
        "created": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
        "config": {
            "ExposedPorts": {
                "1053/tcp": {}, "1053/udp": {}, "2026/tcp": {},
                "7890/tcp": {}, "7891/tcp": {}, "7892/tcp": {},
                "7893/tcp": {}, "7894/tcp": {}, "7895/tcp": {}, "9090/tcp": {},
            },
            "Env": ["PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"],
            "Entrypoint": ["/app/entrypoint.sh"],
            "Volumes": {"/app/data": {}, "/app/etc": {}},
            "WorkingDir": "/app",
        },
        "rootfs": {"type": "layers", "diff_ids": diff_ids},
        "history": list(base_cfg.get("history", []))[:2] + [
            {"created_by": f"CubeX {version_tag} Clean-Room App Layer"}
        ],
    }

    cfg_bytes = json.dumps(cfg, indent=2).encode("utf-8")
    cfg_digest = hashlib.sha256(cfg_bytes).hexdigest()
    cfg_blob = f"blobs/sha256/{cfg_digest}"
    blobs[cfg_blob] = cfg_bytes

    repo_tags = [f"cubex:{version_tag}", "cubex:latest", "fyasshole/cubex:latest"]
    manifest = [{
        "Config": cfg_blob,
        "RepoTags": repo_tags,
        "Layers": layer_names,
    }]
    manifest_bytes = json.dumps(manifest, indent=2).encode("utf-8")

    layer_media = (
        "application/vnd.oci.image.layer.v1.tar+gzip"
        if compress_inner_blobs
        else "application/vnd.oci.image.layer.v1.tar"
    )
    oci_manifest = {
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.image.manifest.v1+json",
        "config": {
            "mediaType": "application/vnd.oci.image.config.v1+json",
            "size": len(cfg_bytes),
            "digest": f"sha256:{cfg_digest}",
        },
        "layers": [
            {
                "mediaType": layer_media,
                "size": len(blobs[ln]),
                "digest": f"sha256:{ln.split('/')[-1]}",
            }
            for ln in layer_names
        ],
    }
    oci_manifest_bytes = json.dumps(oci_manifest, indent=2).encode("utf-8")
    oci_manifest_digest = hashlib.sha256(oci_manifest_bytes).hexdigest()
    oci_manifest_blob = f"blobs/sha256/{oci_manifest_digest}"
    blobs[oci_manifest_blob] = oci_manifest_bytes

    oci_index = {
        "schemaVersion": 2,
        "mediaType": "application/vnd.oci.image.index.v1+json",
        "manifests": [{
            "mediaType": "application/vnd.oci.image.manifest.v1+json",
            "digest": f"sha256:{oci_manifest_digest}",
            "size": len(oci_manifest_bytes),
            "annotations": {
                "io.containerd.image.name": "docker.io/library/cubex:latest",
                "org.opencontainers.image.ref.name": "latest",
            },
            "platform": {"architecture": arch, "os": "linux"},
        }],
    }
    index_bytes = json.dumps(oci_index, indent=2).encode("utf-8")

    now = int(time.time())
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    open_kwargs = {"mode": "w:gz", "compresslevel": 9} if outer_gzip else {"mode": "w"}
    with tarfile.open(out_path, **open_kwargs) as out:
        def add_mem(name: str, payload: bytes):
            ti = tarfile.TarInfo(name=name)
            ti.size = len(payload)
            ti.mtime = now
            ti.mode = 0o644
            out.addfile(ti, io.BytesIO(payload))

        add_mem("oci-layout", b'{"imageLayoutVersion":"1.0.0"}')
        add_mem("index.json", index_bytes)
        add_mem("manifest.json", manifest_bytes)
        for bname in [oci_manifest_blob, cfg_blob] + layer_names:
            add_mem(bname, blobs[bname])

    size_mb = os.path.getsize(out_path) / (1024 * 1024)
    print(f"   ✅ Generated {os.path.basename(out_path)} ({size_mb:.2f} MB)")


def main() -> int:
    parser = argparse.ArgumentParser(description="CubeX Unified OCI/Docker Image Builder")
    parser.add_argument("--arch", choices=["amd64", "arm64", "all"], default="all")
    parser.add_argument("--format", choices=["ros", "docker", "all"], default="all")
    parser.add_argument("--include-metadb", action="store_true", help="Include unused geoip.metadb (8MB)")
    args = parser.parse_args()

    version_tag = read_meta_version()
    os.makedirs(RELEASE_DIR, exist_ok=True)
    arches = ["amd64", "arm64"] if args.arch == "all" else [args.arch]

    print(f"📦 Building Clean-Room OCI/Docker Images for {version_tag} (metadb={args.include_metadb})...")
    for arch in arches:
        base_path = find_base_oci_tar(arch)
        print(f"\n🔨 [{arch.upper()}] Base Alpine OCI: {os.path.basename(base_path)}")
        base_layers, base_cfg = extract_clean_alpine_layers(base_path)
        app_layer = build_app_layer_uncompressed(arch, include_metadb=args.include_metadb)
        layers = base_layers + [app_layer]
        print(f"   -> Clean layers ready (L0+L1 Alpine + App layer {len(app_layer)/(1024*1024):.2f} MB uncompressed)")

        if args.format in ("ros", "all"):
            ros_out = os.path.join(RELEASE_DIR, f"CubeX-{version_tag}-ros-{arch}.tar")
            assemble_oci_archive(
                arch, version_tag, layers, base_cfg, ros_out,
                compress_inner_blobs=True, outer_gzip=False,
            )

        if args.format in ("docker", "all"):
            docker_out = os.path.join(RELEASE_DIR, f"CubeX-{version_tag}-docker-{arch}.tar.gz")
            assemble_oci_archive(
                arch, version_tag, layers, base_cfg, docker_out,
                compress_inner_blobs=False, outer_gzip=True,
            )

    print("\n🎉 Unified OCI/Docker Image Packaging Complete!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
