// Command server ist eine kleine interaktive Demo: Klick "Hallo senden" im
// Browser -> Text wird gestreamt (wie eine LLM-Antwort) und parallel zu Audio
// synthetisiert, inkl. Zeitmessung (erstes/letztes Wort: Text vs. Audio).
//
//	go run ./demo/server -model /pfad/de_DE-thorsten-high.onnx
//	# dann http://localhost:8080  im Browser öffnen
package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/LukasLow/go-piper"
)

const demoText = "Guten Tag. Ich bin ein freundlicher Assistent. " +
	"Beobachte, wie die Stimme zu sprechen beginnt, während der Text noch wächst. " +
	"Das nennt man Streaming. Sobald der erste Satz vollständig ist, erklingt die " +
	"erste Stimme. Danach laufen Text und Audio Seite an Seite weiter. So siehst du, " +
	"wie klein die Verzögerung bis zum ersten Ton wirklich ist. Vielen Dank fürs Zuhören."

func main() {
	model := flag.String("model", "", "Pfad zur .onnx-Stimme")
	addr := flag.String("addr", ":8080", "Listen-Adresse")
	llmDelay := flag.Duration("llm-delay", 55*time.Millisecond, "Verzögerung pro Wort")
	flag.Parse()

	opts := piper.Options{Model: *model}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "demo/web/index.html")
	})
	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		stream(w, opts, *llmDelay)
	})

	log.Printf("Demo läuft auf http://localhost%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func stream(w http.ResponseWriter, opts piper.Options, llmDelay time.Duration) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", 500)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")

	var mu sync.Mutex
	send := func(event string, data any) {
		b, _ := json.Marshal(data)
		mu.Lock()
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		flusher.Flush()
		mu.Unlock()
	}

	start := time.Now()
	words := strings.Fields(demoText)

	sentCh := make(chan string, 256)
	audioCh := make(chan []byte, 256)

	go func() { // Synthese (parallel)
		for s := range sentCh {
			if wav, err := piper.Synthesize(s, opts); err == nil {
				audioCh <- wav
			} else {
				log.Println("synth:", err)
			}
		}
		close(audioCh)
	}()

	var tFirstAudio, tLastAudio time.Duration
	go func() { // Audio -> SSE
		for wav := range audioCh {
			if tFirstAudio == 0 {
				tFirstAudio = time.Since(start)
			}
			send("audio", map[string]any{"wav": base64.StdEncoding.EncodeToString(wav)})
			tLastAudio = time.Since(start)
		}
		send("done", map[string]any{
			"firstAudioMs": tFirstAudio.Milliseconds(),
			"lastAudioMs":  tLastAudio.Milliseconds(),
			"firstTextMs":  0,
		})
	}()

	sentence := strings.Builder{}
	for i, word := range words {
		if i == 0 {
			send("firstText", map[string]any{"ms": time.Since(start).Milliseconds()})
		}
		sentence.WriteString(word)
		sentence.WriteByte(' ')
		send("word", map[string]any{"w": word, "ms": time.Since(start).Milliseconds()})
		time.Sleep(llmDelay)
		if strings.ContainsAny(word[len(word)-1:], ".!?") {
			sentCh <- strings.TrimSpace(sentence.String())
			sentence.Reset()
		}
	}
	if s := strings.TrimSpace(sentence.String()); s != "" {
		sentCh <- s
	}
	close(sentCh)
	send("lastText", map[string]any{"ms": time.Since(start).Milliseconds()})
}
