// Command vits testet die komplette eingebettete In-Process-Kette:
// espeak-ng -> phoneme_id_map -> ONNX Runtime (VITS) -> WAV.
// Kein externes Binary, keine Systeminstallation.
//
//	CGO_ENABLED=1 go run ./cmd/vits <model.onnx> <config.json> <out.wav> "TEXT" [voice]
package main

import (
	"fmt"
	"os"

	"github.com/LukasLow/go-piper/inproc"
)

func main() {
	if len(os.Args) < 5 {
		fmt.Fprintln(os.Stderr, `Aufruf: go run ./cmd/vits <model.onnx> <config.json> <out.wav> "TEXT" [voice]`)
		os.Exit(2)
	}
	model, cfg, out, text := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	voice := "de"
	if len(os.Args) > 5 {
		voice = os.Args[5]
	}

	// Debug: Phonem-Strings pro Satz (Vergleich mit `piper --debug`).
	if sents, err := inproc.SentencesFor(text, voice); err == nil {
		for i, s := range sents {
			fmt.Printf("[sentence %d] %q\n", i+1, s)
		}
	}

	wav, err := inproc.Synthesize(text, voice, model, cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(out, wav, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "Fehler:", err)
		os.Exit(1)
	}
	fmt.Printf("geschrieben: %s (%d Bytes)\n", out, len(wav))
}
