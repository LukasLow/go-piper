//go:build linux && arm64

package piper

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed embed/linux_arm64
var embedded embed.FS

func nativeAssets() (fs.FS, string, bool) { return embedded, "linux_arm64", true }

// haveVersion/writeVersion: schnelle Entpack-Erkennung über einen Hash aus
// Dateinamen + Größen der eingebetteten Assets.
func haveVersion(dir string, afs fs.FS, tag string) bool {
	b, err := os.ReadFile(filepath.Join(dir, ".version"))
	if err != nil {
		return false
	}
	return string(b) == assetHash(afs, tag)
}

func writeVersion(dir string, afs fs.FS, tag string) {
	_ = os.WriteFile(filepath.Join(dir, ".version"), []byte(assetHash(afs, tag)), 0o644)
}

func assetHash(afs fs.FS, tag string) string {
	h := sha256.New()
	_ = fs.WalkDir(afs, "embed/"+tag, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		fmt.Fprintf(h, "%s:%d\n", p, info.Size())
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}
