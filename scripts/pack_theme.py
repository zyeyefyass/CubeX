#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Standard Theme Packaging Tool (pack_theme.py)
Transcodes 4K source videos into standard lightweight .cbxtheme distribution packages:
- 1080p 30fps H.264 (yuv420p, +faststart, muted)
- High-res poster.webp (1920px)
- Ultra-lightweight preview.webp (480px, < 30KB)
- Strict theme.json token specification
- Output .cbxtheme zip package targeted at 5MB ~ 10MB
"""

import os
import sys
import json
import time
import zipfile
import argparse
import subprocess
import tempfile
import shutil

ROOT_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))

def run_cmd(cmd):
    p = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if p.returncode != 0:
        print(f"Command failed: {' '.join(cmd)}\nError: {p.stderr}", file=sys.stderr)
        sys.exit(p.returncode)
    return p.stdout

def get_video_info(path):
    cmd = [
        "ffprobe", "-v", "error",
        "-show_entries", "format=duration,size,bit_rate",
        "-show_entries", "stream=width,height,r_frame_rate,codec_name",
        "-of", "json", path
    ]
    out = run_cmd(cmd)
    data = json.loads(out)
    fmt = data.get("format", {})
    streams = data.get("streams", [{}])
    stream = streams[0] if streams else {}
    return {
        "duration": float(fmt.get("duration", 0)),
        "size": int(fmt.get("size", 0)),
        "bit_rate": int(fmt.get("bit_rate", 0)),
        "width": int(stream.get("width", 0)),
        "height": int(stream.get("height", 0)),
        "codec": stream.get("codec_name", "")
    }

def pack_theme(video_path, theme_id, name, tier, accent, quality="1080p", crf=25, out_dir=None):
    start_time = time.time()
    video_path = os.path.abspath(video_path)
    if not os.path.exists(video_path):
        print(f"Error: Input video not found: {video_path}", file=sys.stderr)
        sys.exit(1)

    out_dir = os.path.abspath(out_dir or os.path.join(ROOT_DIR, "scratch"))
    os.makedirs(out_dir, exist_ok=True)

    print(f"🎬 Starting CubeX Theme Packing: {name} (ID: {theme_id})")
    print(f"   Source video: {video_path}")
    src_info = get_video_info(video_path)
    print(f"   Source info : {src_info['width']}x{src_info['height']} @ {src_info['duration']:.1f}s, {round(src_info['size']/1024/1024, 2)} MB")

    scale_dim = "1920:1080" if quality == "1080p" else "2560:1440"
    target_quality_label = "1080p" if quality == "1080p" else "2k"

    temp_stage = tempfile.mkdtemp(prefix="cbx_theme_")
    try:
        bg_mp4 = os.path.join(temp_stage, "bg.mp4")
        poster_webp = os.path.join(temp_stage, "poster.webp")
        preview_webp = os.path.join(temp_stage, "preview.webp")
        theme_json_path = os.path.join(temp_stage, "theme.json")

        # 1. Transcode video (H.264 yuv420p, muted, faststart, 30fps)
        print(f"🔨 [1/4] Transcoding video to {scale_dim} H.264 (CRF {crf}, faststart)...")
        ffmpeg_video_cmd = [
            "ffmpeg", "-y", "-i", video_path,
            "-vf", f"scale={scale_dim}:flags=lanczos,fps=30",
            "-c:v", "libx264", "-preset", "slow", "-crf", str(crf),
            "-profile:v", "high", "-level", "4.1",
            "-pix_fmt", "yuv420p", "-an", "-movflags", "+faststart",
            bg_mp4
        ]
        run_cmd(ffmpeg_video_cmd)

        # 2. Extract first frame poster.webp (1920 wide)
        print("📸 [2/4] Extracting high-res poster.webp...")
        ffmpeg_poster_cmd = [
            "ffmpeg", "-y", "-ss", "00:00:00.000", "-i", bg_mp4,
            "-vframes", "1",
            "-vf", "scale=1920:1080",
            "-c:v", "libwebp", "-quality", "85",
            poster_webp
        ]
        run_cmd(ffmpeg_poster_cmd)

        # 3. Extract lightweight preview.webp (480 wide, < 30KB)
        print("🖼️  [3/4] Generating ultra-compact preview.webp (< 30KB)...")
        ffmpeg_preview_cmd = [
            "ffmpeg", "-y", "-ss", "00:00:00.000", "-i", bg_mp4,
            "-vframes", "1",
            "-vf", "scale=480:-1",
            "-c:v", "libwebp", "-quality", "75",
            preview_webp
        ]
        run_cmd(ffmpeg_preview_cmd)

        # 4. Generate theme.json
        print("📝 [4/4] Generating theme.json token specification...")
        meta = {
            "id": theme_id,
            "name": name,
            "version": "1.0.0",
            "type": "video",
            "tier": tier,
            "description": "黑金曜石奢华动态壁纸，60FPS 顺滑循环 · 纯离线无感渲染",
            "accent": accent,
            "assets": {
                "video": "bg.mp4",
                "poster": "poster.webp",
                "preview": "preview.webp"
            },
            "tokens": {
                "--cbx-theme-accent": accent,
                "--cbx-theme-accent-glow": "rgba(229, 185, 92, 0.35)",
                "--cbx-theme-card-bg": "rgba(10, 12, 18, 0.75)",
                "--cbx-theme-card-border": "rgba(229, 185, 92, 0.25)"
            }
        }
        with open(theme_json_path, "w", encoding="utf-8") as f:
            json.dump(meta, f, ensure_ascii=False, indent=2)

        # Package into .cbxtheme zip
        pkg_name = f"{theme_id}-{target_quality_label}.cbxtheme"
        pkg_path = os.path.join(out_dir, pkg_name)
        with zipfile.ZipFile(pkg_path, "w", zipfile.ZIP_DEFLATED) as zf:
            zf.write(theme_json_path, "theme.json")
            zf.write(bg_mp4, "bg.mp4")
            zf.write(poster_webp, "poster.webp")
            zf.write(preview_webp, "preview.webp")

        duration_sec = time.time() - start_time
        pkg_size = os.path.getsize(pkg_path)
        bg_info = get_video_info(bg_mp4)
        poster_size = os.path.getsize(poster_webp)
        preview_size = os.path.getsize(preview_webp)

        print("\n" + "="*70)
        print("🎉 CubeX Theme Packaging Completed Successfully!")
        print("="*70)
        print(f"📦 Output Package  : {pkg_path}")
        print(f"📊 Package Size    : {pkg_size:,} bytes ({round(pkg_size/1024/1024, 2)} MB)")
        print(f"🎞️  Video Bitrate   : {bg_info['bit_rate'] // 1000} kbps")
        print(f"🖼️  poster.webp     : {poster_size:,} bytes ({round(poster_size/1024, 1)} KB)")
        print(f"🔍 preview.webp    : {preview_size:,} bytes ({round(preview_size/1024, 1)} KB)")
        print(f"⏱️  Packing Time    : {duration_sec:.2f}s")
        print("="*70)

        # Also copy preview.webp to web public assets directory for the showcase card
        showcase_dir = os.path.join(ROOT_DIR, "web", "public", "assets", "themes")
        os.makedirs(showcase_dir, exist_ok=True)
        showcase_dest = os.path.join(showcase_dir, f"preview-{theme_id}.webp")
        shutil.copy2(preview_webp, showcase_dest)
        print(f"✓ Copied showcase thumbnail to: {showcase_dest} ({round(preview_size/1024, 1)} KB)")

        return {
            "package_path": pkg_path,
            "package_size": pkg_size,
            "video_bitrate_kbps": bg_info['bit_rate'] // 1000,
            "poster_size": poster_size,
            "preview_size": preview_size,
            "duration_sec": duration_sec
        }
    finally:
        shutil.rmtree(temp_stage, ignore_errors=True)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="CubeX Theme Packaging Tool")
    parser.add_argument("video", help="Path to source 4K video")
    parser.add_argument("--id", default="dark-skull-luxury", help="Theme ID")
    parser.add_argument("--name", default="暗夜曜金 · 奢华骷髅", help="Theme display name")
    parser.add_argument("--tier", default="free-fan", help="Theme tier (free-fan, member)")
    parser.add_argument("--accent", default="#E5B95C", help="Accent hex color")
    parser.add_argument("--quality", default="1080p", choices=["1080p", "2k"], help="Target quality")
    parser.add_argument("--crf", type=int, default=25, help="H.264 CRF quality level (24-28)")
    parser.add_argument("--out-dir", default=None, help="Output directory")

    args = parser.parse_args()
    pack_theme(args.video, args.id, args.name, args.tier, args.accent, args.quality, args.crf, args.out_dir)
