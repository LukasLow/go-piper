#!/usr/bin/env python3
"""Make Homebrew dylibs self-contained: rewrite `@@HOMEBREW_PREFIX@@/opt/<f>/lib/<name>`
load-command strings to `@loader_path/<name>` (same byte length, NUL-padded).

Usage: relocate_macos.py <dir> [more dirs...]
"""
import os
import re
import subprocess
import sys

PAT = re.compile(rb"@@HOMEBREW_PREFIX@@/opt/[^/]+/lib/([A-Za-z0-9._+-]+\.dylib)")

for d in sys.argv[1:]:
    for fn in os.listdir(d):
        if not fn.endswith(".dylib"):
            continue
        p = os.path.join(d, fn)
        with open(p, "rb") as f:
            data = f.read()

        def repl(m: "re.Match[bytes]") -> bytes:
            name = m.group(1)
            t = b"@loader_path/" + name
            pad = len(m.group(0)) - len(t)
            assert pad >= 0, f"{fn}: target longer than original"
            return t + b"\0" * pad

        new, n = PAT.subn(repl, data)
        if n:
            with open(p, "wb") as f:
                f.write(new)
            # Der Byte-Patch verändert die Datei NACH dem Signieren und macht eine
            # vorhandene Code-Signatur ungültig. Auf Apple Silicon darf eine
            # dlopen'te Bibliothek ohne gültige Signatur den Prozess sofort
            # beenden (SIGKILL "Code Signature Invalid") -> ad-hoc neu signieren.
            subprocess.run(["codesign", "--force", "--sign", "-", p], check=True)
            print(f"{fn}: {n} reference(s) relocated + re-signed")
        else:
            print(f"{fn}: nothing to do")
