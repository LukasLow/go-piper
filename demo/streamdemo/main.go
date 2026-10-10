// Command streamdemo simuliert eine LLM-Antwort im Streaming und synthetisiert
// sie **nebenläufig** mit Piper zu Audio — wie ein Voice-Agent.
//
// Gemessen wird: erstes/letztes Textwort vs. erstes/letztes Audio.
//
//	go run ./demo/streamdemo -model /pfad/de_DE-thorsten-high.onnx
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/LukasLow/go-piper"
)

const defaultText = "Guten Tag. Ich bin ein freundlicher Assistent. Heute " +
	"möchte ich dir zeigen, wie flüssig Sprache aus einem Text entstehen kann, " +
	"während der Text noch geschrieben wird. Stell dir vor, ein Sprachmodell " +
	"schreibt eine Antwort Wort für Wort, und genau währenddessen beginnt die " +
	"Stimme schon zu sprechen. Das nennt man Streaming, und es fühlt sich viel " +
	"lebendiger an als Warten. In diesem Beispiel liest du den Text mit und hörst " +
	"gleichzeitig zu. Am Anfang ist alles noch still, doch sobald der erste Satz " +
	"vollständig ist, erklingt die erste Stimme. Danach läuft beides Seite an Seite " +
	"weiter, Satz für Satz, bis die Antwort zu Ende ist. So siehst du genau, wie " +
	"kurz die Verzögerung bis zum ersten Ton wirklich ist. Ich hoffe, das klingt " +
	"angenehm und natürlich, und vielleicht hilft es dir, deine eigene Anwendung " +
	"noch besser zu machen. Vielen Dank fürs Zuhören, und bis zum nächsten Mal."

func main() {
	model := flag.String("model", "", "Pfad zur .onnx-Stimme")
	llmDelay := flag.Duration("llm-delay", 55*time.Millisecond, "Verzögerung pro Wort (simuliertes LLM-Streaming)")
	outPath := flag.String("o", "demo/out/stream.wav", "Ausgabe-WAV (verflochten)")
	text := flag.String("text", defaultText, "Text")
	flag.Parse()

	words := strings.Fields(*text)
	opts := piper.Options{Model: *model}

	sentCh := make(chan string, 1024)   // fertige Sätze -> Synthese
	audioCh := make(chan []byte, 1024)  // synthetisierte WAV-Häppchen
	consumerDone := make(chan struct{})
	var synthWG sync.WaitGroup

	// Synthese-Goroutine (läuft parallel zum Textstream).
	synthWG.Add(1)
	go func() {
		defer synthWG.Done()
		for s := range sentCh {
			wav, err := piper.Synthesize(s, opts)
			if err != nil {
				fmt.Fprintln(os.Stderr, "synth:", err)
				continue
			}
			audioCh <- wav
		}
		close(audioCh)
	}()

	// Konsument: sammelt Audio + misst Audio-Zeitpunkte (einziger Schreiber).
	var (
		mu                       sync.Mutex
		chunks                   [][]byte
		tFirstAudio, tLastAudio  time.Duration
	)
	go func() {
		defer close(consumerDone)
		for wav := range audioCh {
			mu.Lock()
			if tFirstAudio == 0 {
				tFirstAudio = time.Since(startTime)
			}
			chunks = append(chunks, wav)
			tLastAudio = time.Since(startTime)
			mu.Unlock()
		}
	}()

	startTime = time.Now()
	var tFirstText, tLastText time.Duration
	var sent strings.Builder

	for i, w := range words {
		if i == 0 {
			tFirstText = time.Since(startTime)
		}
		sent.WriteString(w)
		sent.WriteByte(' ')
		time.Sleep(*llmDelay)
		tLastText = time.Since(startTime)

		if strings.ContainsAny(w[len(w)-1:], ".!?") {
			sentCh <- strings.TrimSpace(sent.String())
			sent.Reset()
		}
	}
	if s := strings.TrimSpace(sent.String()); s != "" {
		sentCh <- s
	}
	close(sentCh)
	synthWG.Wait()
	<-consumerDone

	mu.Lock()
	defer mu.Unlock()

	_ = os.MkdirAll("demo/out", 0o755)
	if err := concatWav(*outPath, chunks); err != nil {
		fmt.Fprintln(os.Stderr, "wav:", err)
		os.Exit(1)
	}
	audioTotal := durOf(chunks)
	renderTotal := tLastAudio

	fmt.Printf("\n=== Streaming-Bilanz ===\n")
	fmt.Printf("Text gestreamt bis:   %8s  (letztes Textwort)\n", ms(tLastText))
	fmt.Printf("Erstes Textwort bei:  %8s\n", ms(tFirstText))
	fmt.Printf("ERSTES AUDIO bei:     %8s   <- Verzögerung bis zum Ton\n", ms(tFirstAudio))
	fmt.Printf("LETZTES AUDIO bei:   %8s\n", ms(tLastAudio))
	fmt.Printf("Audio gesamt:         %8s\n", ms(audioTotal))
	fmt.Printf("Renderzeit (parallel):%8s\n", ms(renderTotal))
	if audioTotal > 0 {
		fmt.Printf("RTF gesamt:           %.2f\n", renderTotal.Seconds()/audioTotal.Seconds())
	}
	fmt.Printf("WAV: %s\n", *outPath)
}

var startTime time.Time

func ms(d time.Duration) string { return d.Round(time.Millisecond).String() }

func durOf(chunks [][]byte) time.Duration {
	var bytesPerMs float64 = 22050 * 2 / 1000 // 16-bit mono @22050
	var total int
	for _, c := range chunks {
		if len(c) > 44 {
			total += len(c) - 44
		}
	}
	return time.Duration(float64(total)/bytesPerMs) * time.Millisecond
}

// concatWav schreibt einen WAV aus mehreren Einzel-WAVs (Header nur einmal).
func concatWav(path string, chunks [][]byte) error {
	var data bytes.Buffer
	for _, c := range chunks {
		if len(c) > 44 {
			data.Write(c[44:])
		}
	}
	const sr = 22050
	hdr := make([]byte, 44)
	copy(hdr[0:], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(36+data.Len()))
	copy(hdr[8:], "WAVE")
	copy(hdr[12:], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)
	binary.LittleEndian.PutUint16(hdr[20:], 1)
	binary.LittleEndian.PutUint16(hdr[22:], 1)
	binary.LittleEndian.PutUint32(hdr[24:], sr)
	binary.LittleEndian.PutUint32(hdr[28:], sr*2)
	binary.LittleEndian.PutUint16(hdr[32:], 2)
	binary.LittleEndian.PutUint16(hdr[34:], 16)
	copy(hdr[36:], "data")
	binary.LittleEndian.PutUint32(hdr[40:], uint32(data.Len()))

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(hdr); err != nil {
		return err
	}
	_, err = f.Write(data.Bytes())
	return err
}
