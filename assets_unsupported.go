//go:build !(linux && arm64)

// Fallback für Plattformen, für die (noch) keine Assets eingebettet sind.
// Füge eine Datei assets_<os>_<arch>.go nach dem Muster von
// assets_linux_arm64.go hinzu, um eine Plattform zu unterstützen.

package piper

import (
	"io/fs"
	"runtime"
)

func nativeAssets() (fs.FS, string, bool) {
	return nil, runtime.GOOS + "_" + runtime.GOARCH, false
}

func haveVersion(string, fs.FS, string) bool { return false }
func writeVersion(string, fs.FS, string)     {}

func assetHash(fs.FS, string) string { return "" }
