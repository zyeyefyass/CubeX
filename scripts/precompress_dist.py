#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
CubeX Static Asset Precompression Pipeline
Compresses all .js, .css, .html, and .svg assets in cmd/cubexd/dist using maximum gzip compression (level 9).
Replaces uncompressed assets with .gz files for zero-copy, zero-CPU runtime direct streaming.
"""

import os
import gzip

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), ".."))
DIST_DIR = os.path.join(ROOT, "cmd", "cubexd", "dist")

COMPRESS_EXTS = {".js", ".css", ".html", ".svg"}

def precompress():
    if not os.path.exists(DIST_DIR):
        print(f"Error: {DIST_DIR} does not exist")
        return

    orig_total = 0
    gz_total = 0
    count = 0

    for root, _, files in os.walk(DIST_DIR):
        for f in files:
            ext = os.path.splitext(f)[1].lower()
            if ext in COMPRESS_EXTS:
                src_path = os.path.join(root, f)
                gz_path = src_path + ".gz"
                
                with open(src_path, "rb") as fin:
                    raw_data = fin.read()
                
                orig_size = len(raw_data)
                orig_total += orig_size
                
                with open(gz_path, "wb") as fout:
                    with gzip.GzipFile(filename="", mode="wb", fileobj=fout, compresslevel=9, mtime=0) as gzout:
                        gzout.write(raw_data)
                
                gz_size = os.path.getsize(gz_path)
                gz_total += gz_size
                count += 1
                
                # Remove original uncompressed file
                os.remove(src_path)
                
                ratio = (1 - gz_size / orig_size) * 100
                rel = os.path.relpath(src_path, DIST_DIR)
                print(f"  ✓ {rel:35s}: {orig_size:7d} B -> {gz_size:7d} B (-{ratio:.1f}%)")

    saved = orig_total - gz_total
    ratio_total = (saved / orig_total * 100) if orig_total else 0
    print(f"\nPrecompression Summary:")
    print(f"  Files compressed : {count}")
    print(f"  Original size    : {orig_total:,} bytes ({orig_total/1024:.1f} KB)")
    print(f"  Precompressed    : {gz_total:,} bytes ({gz_total/1024:.1f} KB)")
    print(f"  Saved            : {saved:,} bytes (-{ratio_total:.1f}% reduction)")

if __name__ == "__main__":
    precompress()
