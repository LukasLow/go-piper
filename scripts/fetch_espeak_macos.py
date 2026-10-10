#!/usr/bin/env python3
"""Fetch libespeak-ng (macOS arm64) from the Homebrew bottle.

No Homebrew needed: reads the formula metadata, downloads the arm64 bottle
from ghcr.io (anonymous token) and extracts the dylibs.

Usage: fetch_espeak_macos.py <outdir>
"""
import io
import json
import os
import sys
import tarfile
import urllib.request

API = "https://formulae.brew.sh/api/formula/espeak-ng.json"
want = sys.argv[1] if len(sys.argv) > 1 else "/tmp/espeak-macos"

j = json.load(urllib.request.urlopen(API))
files = j["bottle"]["stable"]["files"]
order = ["arm64_tahoe", "arm64_sequoia", "arm64_sonoma", "arm64_ventura",
         "arm64_monterey", "arm64_big_sur"]
tags = [t for t in order if t in files]
if not tags:
    sys.exit("no arm64 bottle found; tags: " + ",".join(sorted(files)))
tag = tags[0]
blob = files[tag]["url"]
print("bottle:", tag)

tok = json.load(urllib.request.urlopen(
    "https://ghcr.io/token?service=ghcr.io&scope=repository:homebrew/core/espeak-ng:pull"))["token"]
req = urllib.request.Request(blob, headers={"Authorization": "Bearer " + tok})
data = urllib.request.urlopen(req).read()
print("bottle bytes:", len(data))

tf = tarfile.open(fileobj=io.BytesIO(data), mode="r:gz")
os.makedirs(want, exist_ok=True)
found = 0
for m in tf.getmembers():
    base = os.path.basename(m.name)
    if base in ("libespeak-ng.1.dylib", "libespeak-ng.dylib"):
        f = tf.extractfile(m)
        if f is None:          # symlink entry -> create symlink
            os.system(f'ln -sf "{m.linkname}" "{os.path.join(want, base)}"')
            print("symlink", base, "->", m.linkname)
            continue
        with open(os.path.join(want, base), "wb") as out:
            out.write(f.read())
        print("wrote", os.path.join(want, base))
        found += 1
print("done,", found, "file(s) in", want)
