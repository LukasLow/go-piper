package inproc

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// engineRoot entpackt die eingebetteten Engine-Assets (Libs + espeak-ng-data)
// nach $XDG_CACHE_HOME/go-piper/engine/<tag> und liefert das Verzeichnis.
func engineRoot() (string, error) {
	fsys, tag, ok := bundledAssets()
	if !ok {
		return "", fmt.Errorf("inproc: keine eingebetteten Engine-Assets für diese Plattform")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "go-piper", "engine", tag)
	prefix := "libs/" + tag

	if haveVer(dir, fsys, prefix) {
		return dir, nil
	}
	if err := extract(dir, fsys, prefix); err != nil {
		return "", err
	}
	writeVer(dir, fsys, prefix)
	return dir, nil
}

func extract(dir string, fsys fs.FS, prefix string) error {
	return fs.WalkDir(fsys, prefix, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := p[len(prefix):]
		rel = trimSlash(rel)
		if rel == "" {
			return os.MkdirAll(dir, 0o755)
		}
		dst := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		return os.WriteFile(dst, data, 0o644)
	})
}

func trimSlash(s string) string {
	for len(s) > 0 && s[0] == '/' {
		s = s[1:]
	}
	return s
}

func haveVer(dir string, fsys fs.FS, prefix string) bool {
	b, err := os.ReadFile(filepath.Join(dir, ".version"))
	return err == nil && string(b) == assetHash(fsys, prefix)
}

func writeVer(dir string, fsys fs.FS, prefix string) {
	_ = os.WriteFile(filepath.Join(dir, ".version"), []byte(assetHash(fsys, prefix)), 0o644)
}

func assetHash(fsys fs.FS, prefix string) string {
	h := sha256.New()
	_ = fs.WalkDir(fsys, prefix, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			if info, e := d.Info(); e == nil {
				fmt.Fprintf(h, "%s:%d\n", p, info.Size())
			}
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}
