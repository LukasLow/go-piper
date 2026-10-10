package inproc

import (
	"fmt"
	"strings"
)

// Options sind die VITS-Synthese-Parameter (Piper-Defaults).
type Options struct {
	NoiseScale  float32
	LengthScale float32
	NoiseW      float32
}

// DefaultOptions sind Pipers Standardwerte.
func DefaultOptions() Options { return Options{NoiseScale: 0.667, LengthScale: 1.0, NoiseW: 0.8} }

// Synthesize rendert text in der Stimme voice zu einer 16-bit-PCM-WAV — exakt
// wie Piper: pro Satz, und an jeder Klausel wird das Satzzeichen-Token
// angehängt (`,`/`;`/`:` + Space; `.`/`?`/`!` ohne), zwischen Sätzen 0,2 s Pause.
//
// Engine (espeak-ng + ONNX Runtime) und espeak-ng-data sind eingebettet; kein
// externes Binary, keine Installation.
func Synthesize(text, voice, modelPath, configPath string) ([]byte, error) {
	return SynthesizeWith(text, voice, modelPath, configPath, DefaultOptions())
}

// SynthesizeWith wie Synthesize, mit expliziten VITS-Parametern (z. B.
// NoiseScale=0, NoiseW=0 für deterministische Ausgabe/Parity-Tests).
func SynthesizeWith(text, voice, modelPath, configPath string, opt Options) ([]byte, error) {
	root, err := engineRoot()
	if err != nil {
		return nil, err
	}
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, err
	}
	sentences, err := Sentences(text, voice, root)
	if err != nil {
		return nil, err
	}

	// Pause zwischen Sätzen, wie Piper (sentenceSilenceSeconds = 0.2).
	silence := make([]float32, cfg.Audio.SampleRate/5)

	scales := [3]float32{opt.NoiseScale, opt.LengthScale, opt.NoiseW}
	var samples []float32
	for _, s := range sentences {
		ids := cfg.PhonemesToIDs(s)
		if len(ids) == 0 {
			continue
		}
		out, err := Infer(modelPath, ids, scales)
		if err != nil {
			return nil, err
		}
		// Piper hängt die Pause nach JEDEM Satz an (auch nach dem letzten).
		samples = append(samples, out...)
		samples = append(samples, silence...)
	}
	if len(samples) == 0 {
		return nil, fmt.Errorf("inproc: leerer Text")
	}
	return FloatToWAV(samples, cfg.Audio.SampleRate), nil
}

// Sentences liefert die Phonem-Strings pro Satz (Piper-genau). Nützlich für
// Debug-Vergleiche mit `piper --debug`.
func Sentences(text, voice, root string) ([]string, error) {
	var out []string
	var phon strings.Builder
	for _, cl := range splitClauses(text) {
		if cl.text != "" {
			p, err := Phonemize(cl.text, voice, root)
			if err != nil {
				return nil, err
			}
			phon.WriteString(p)
		}
		switch cl.punct {
		case ".":
			phon.WriteString(".")
		case "?":
			phon.WriteString("?")
		case "!":
			phon.WriteString("!")
		case ",":
			phon.WriteString(", ")
		case ";":
			phon.WriteString("; ")
		case ":":
			phon.WriteString(": ")
		}
		if cl.punct == "." || cl.punct == "?" || cl.punct == "!" {
			out = append(out, phon.String())
			phon.Reset()
		}
	}
	if strings.TrimSpace(phon.String()) != "" {
		out = append(out, phon.String())
	}
	return out, nil
}

// SentencesFor wie Sentences, löst das Engine-Root selbst auf.
func SentencesFor(text, voice string) ([]string, error) {
	root, err := engineRoot()
	if err != nil {
		return nil, err
	}
	return Sentences(text, voice, root)
}

type clause struct{ text, punct string }

// splitClauses zerlegt text an , ; : . ! ? (Zeichen bleibt als punct erhalten).
func splitClauses(text string) []clause {
	var out []clause
	var b strings.Builder
	flush := func(p string) {
		out = append(out, clause{strings.TrimSpace(b.String()), p})
		b.Reset()
	}
	for _, r := range text {
		if r == '\n' {
			r = ' '
		}
		switch r {
		case ',':
			flush(",")
		case ';':
			flush(";")
		case ':':
			flush(":")
		case '.':
			flush(".")
		case '!':
			flush("!")
		case '?':
			flush("?")
		default:
			b.WriteRune(r)
		}
	}
	if strings.TrimSpace(b.String()) != "" {
		out = append(out, clause{strings.TrimSpace(b.String()), ""})
	}
	return out
}
