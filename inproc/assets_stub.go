//go:build !(cgo && arm64 && (linux || darwin))

package inproc

import "io/fs"

func bundledAssets() (fs.FS, string, bool) { return nil, "", false }

const (
	espeakLib = ""
	ortLib    = ""
)
