// Command vits testet die komplette In-Process-Kette:
// espeak-ng (Phoneme) -> phoneme_id_map -> ONNX Runtime (VITS) -> WAV.
//
//	CGO_ENABLED=1 go run ./cmd/vits <model.onnx> <model.onnx.json> <espeak-parent> "<text>" out.wav
package main

import (
	"fmt"
	"os"

	"github.com/LukasLow/go-piper/inproc"
)

func main() {
	if len(os.Args) < 6 {
		fmt.Fprintln(os.Stderr, `Aufruf: go run ./cmd/vits <model.onnx> <config.json> <espeak-parent> "TEXT" <out.wav>`)
		os.Exit(2)
	}
	modelPath, cfgPath, espeakParent, text, out := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5]

	cfg, err := inproc.LoadConfig(cfgPath)
	if err != nil {
		fatal(err)
	}
	phon, err := inproc.Phonemize(text, "de", espeakParent)
	if err != nil {
		fatal(err)
	}
	ids := cfg.PhonemesToIDs(phon)
	fmt.Printf("phoneme: %s\nids: %d\n", phon, len(ids))

	m, err := inproc.OpenModel(modelPath)
	if err != nil {
		fatal(err)
	}
	defer m.Close()

	samples, err := m.Infer(ids, [3]float32{0.667, 1.0, 0.8})
	if err != nil {
		fatal(err)
	}
	fmt.Printf("samples: %d (%.2fs)\n", len(samples), float64(len(samples))/float64(cfg.Audio.SampleRate))

	if err := os.WriteFile(out, inproc.FloatToWAV(samples, cfg.Audio.SampleRate), 0o644); err != nil {
		fatal(err)
	}
	fmt.Println("geschrieben:", out)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "Fehler:", err)
	os.Exit(1)
}
