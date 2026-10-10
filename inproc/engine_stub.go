//go:build !(cgo && arm64 && (linux || darwin))

package inproc

import "fmt"

func loadLibs(string) error {
	return fmt.Errorf("inproc: keine In-Process-Engine für diese Plattform")
}

func Phonemize(text, voice, dataParent string) (string, error) {
	return "", fmt.Errorf("inproc: In-Process-Engine für %s nicht verfügbar", platformTag())
}

func Infer(modelPath string, phonemeIDs []int64, scales [3]float32) ([]float32, error) {
	return nil, fmt.Errorf("inproc: In-Process-Engine für %s nicht verfügbar", platformTag())
}

func platformTag() string { return "diese Plattform" }
