//go:build darwin && arm64 && cgo

package inproc

import (
	"embed"
	"io/fs"
)

//go:embed libs/darwin_arm64
var bundled embed.FS

func bundledAssets() (fs.FS, string, bool) { return bundled, "darwin_arm64", true }

const (
	espeakLib = "libespeak-ng.dylib"
	ortLib    = "libonnxruntime.dylib"
)
