#!/usr/bin/env bash
# Kopiert eine lokale Piper-Runtime in die Embed-Struktur eines Ziel-Tags
# und erzeugt das Symlink-Manifest (go:embed bettet Symlinks nicht ein).
#
# Aufruf:  scripts/prepare-assets.sh <piper-dir> <os>_<arch>
# Beispiel: scripts/prepare-assets.sh ~/piper/piper linux_arm64
set -euo pipefail

src="$1"
tag="$2"
here="$(cd "$(dirname "$0")/.." && pwd)"
dst="$here/embed/$tag"

mkdir -p "$dst"
rm -rf "$dst/piper"
cp -R "$src" "$dst/piper"

# Symlinks erfassen: <link> <ziel>
( cd "$dst/piper" && find . -type l | while read -r l; do
    printf 'piper/%s %s\n' "${l#./}" "$(readlink "$l")"
  done ) > "$dst/symlinks.txt"

echo "bereit: $dst"
echo "-> für den Build noch assets_$tag.go nach dem Muster von assets_linux_arm64.go anlegen"
echo "   (//go:build <os> && <arch>  +  //go:embed embed/$tag)."
