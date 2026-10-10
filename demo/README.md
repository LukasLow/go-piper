# go-piper · demos

## Was hier ist

- **`streamdemo/`** — CLI: simuliert eine LLM-Antwort im Streaming und
  synthetisiert sie **parallel** mit Piper zu Audio. Misst erstes/letztes
  Wort (Text vs. Audio) und schreibt ein verflochtenes WAV.
  ```bash
  go run ./demo/streamdemo -model /pfad/de_DE-thorsten-high.onnx
  # -> demo/out/stream.wav + Zeitbilanz
  ```
- **`server/`** + **`web/`** — interaktive Browser-Demo: Klick „Hallo senden“,
  Text streamt rein, Audio wird parallel erzeugt und abgespielt, mit Live-Zeitmessung.
  ```bash
  go run ./demo/server -model /pfad/de_DE-thorsten-high.onnx
  # -> http://localhost:8080
  ```
  (Der Server nutzt SSE; die Seite in `web/index.html` spielt WAV-Häppchen per
  Web-Audio-Queue ab.)
- **`index.html`** — statische Ergebnisseite mit den zwei Test-Audios
  (`out/long1min.wav`, `out/stream.wav`) und der Messung.
- **`out/`** — erzeugte WAVs.

## Gemessene Streaming-Bilanz (Beispiel, linux/arm64)

| | Zeit |
|---|---|
| erstes Textwort | 0 ms |
| **erstes Audio** | **636 ms** |
| letztes Textwort | 8206 ms |
| letztes Audio | 13853 ms |
| Audio gesamt | 46,8 s |
| RTF | 0,30 |

**Deutung:** Die Verzögerung bis zum ersten Ton ist klein (< 1 s), weil Piper
**satzweise** synthetisiert und parallel zum Textstream läuft. Genau das ist für
einen Voice-Agent aus einer LLM-Streaming-Antwort entscheidend.

## Hinweis

Für die Web-Demo muss Piper auf dem Rechner laufen, auf dem der Server startet
(hier: die `embed/`-Assets bzw. das Modell). Auf macOS zuerst ein passendes
Piper-macOS-Build besorgen.
