//go:build linux && arm64 && cgo

package inproc

import (
	"embed"
	"io/fs"
)

//go:embed libs/linux_arm64
var bundled embed.FS

func bundledAssets() (fs.FS, string, bool) { return bundled, "linux_arm64", true }

const (
	espeakLib = "libespeak-ng.so"
	ortLib    = "libonnxruntime.so"
)
