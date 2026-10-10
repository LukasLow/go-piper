// Command piper ist eine winzige Demo-CLI für die go-piper-Lib.
//
//	go run ./cmd "Guten Tag, ich bin ein Assistent."   # -> out.wav
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/LukasLow/go-piper"
)

func main() {
	out := flag.String("o", "out.wav", "Ausgabedatei")
	model := flag.String("model", "", "Pfad zur .onnx-Stimme")
	flag.Parse()

	text := strings.TrimSpace(strings.Join(flag.Args(), " "))
	if text == "" {
		fmt.Fprintln(os.Stderr, `Aufruf:  go run ./cmd "Dein Text"`)
		os.Exit(2)
	}

	if err := piper.SynthesizeToFile(*out, text, piper.Options{Model: *model}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("geschrieben: %s\n", *out)
}
