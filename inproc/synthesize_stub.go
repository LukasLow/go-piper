//go:build !(cgo && arm64 && (linux || darwin))

package inproc

import "fmt"

// Stub für Plattformen ohne cgo/(onnxruntime+espeak). Modell bauen =
// synthesize_<os>_<arch>.go nach Muster von synthesize_linux_arm64.go anlegen
// plus passende libs/<os>_<arch>/.
type Model struct{}

func OpenModel(string) (*Model, error) {
	return nil, fmt.Errorf("inproc: keine In-Process-Inferenz für diese Plattform (cgo/Assets fehlen)")
}

func (m *Model) Close() {}

func (m *Model) Infer([]int64, [3]float32) ([]float32, error) {
	return nil, fmt.Errorf("inproc: keine In-Process-Inferenz für diese Plattform")
}
