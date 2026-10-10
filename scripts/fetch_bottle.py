#!/usr/bin/env python3
"""Fetch dylibs from a Homebrew bottle into <outdir> (no Homebrew needed).

Usage: fetch_bottle.py <formula> <outdir> [name-substring ...]
Example: fetch_bottle.py pcaudiolib /tmp/x libpcaudio
"""
import io
import json
import os
import sys
import tarfile
import urllib.request

formula = sys.argv[1]
outdir = sys.argv[2]
wanted = sys.argv[3:] or ["dylib"]

j = json.load(urllib.request.urlopen(f"https://formulae.brew.sh/api/formula/{formula}.json"))
files = j["bottle"]["stable"]["files"]
order = ["arm64_tahoe", "arm64_sequoia", "arm64_sonoma", "arm64_ventura",
         "arm64_monterey", "arm64_big_sur"]
tags = [t for t in order if t in files]
if not tags:
    sys.exit("no arm64 bottle: " + ",".join(sorted(files)))
blob = files[tags[0]]["url"]

tok = json.load(urllib.request.urlopen(
    f"https://ghcr.io/token?service=ghcr.io&scope=repository:homebrew/core/{formula}:pull"))["token"]
data = urllib.request.urlopen(urllib.request.Request(blob, headers={"Authorization": "Bearer " + tok})).read()
tf = tarfile.open(fileobj=io.BytesIO(data), mode="r:gz")
os.makedirs(outdir, exist_ok=True)
for m in tf.getmembers():
    base = os.path.basename(m.name)
    if not base.endswith(".dylib"):
        continue
    if not any(w in base for w in wanted):
        continue
    f = tf.extractfile(m)
    if f is None:
        os.system(f'ln -sf "{m.linkname}" "{os.path.join(outdir, base)}"')
    else:
        with open(os.path.join(outdir, base), "wb") as out:
            out.write(f.read())
    print("wrote", os.path.join(outdir, base))
