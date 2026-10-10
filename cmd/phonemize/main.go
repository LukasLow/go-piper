// Command phonemize testet die In-Process-Phonemisierung (cgo + espeak-ng).
//
//	CGO_ENABLED=1 go run ./cmd/phonemize "Guten Tag" de /pfad/espeak-ng-data
package main

import (
	"fmt"
	"os"

	"github.com/LukasLow/go-piper/inproc"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, `Aufruf: go run ./cmd/phonemize "TEXT" <voice> <espeak-ng-data-dir>`)
		os.Exit(2)
	}
	ph, err := inproc.Phonemize(os.Args[1], os.Args[2], os.Args[3])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println(ph)
}
