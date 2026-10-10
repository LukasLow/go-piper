//go:build !(linux && arm64 && cgo)

package inproc

import "fmt"

// Stub für Plattformen ohne cgo/Assets. Eine Plattform ergänzen = neue Datei
// inproc/phonemize_<os>_<arch>.go nach dem Muster von phonemize_linux_arm64.go
// plus passende libs/<os>_<arch>/.
func Phonemize(text, voice, dataDir string) (string, error) {
	return "", fmt.Errorf("inproc: keine In-Process-Implementierung für diese Plattform (cgo/Assets fehlen)")
}
