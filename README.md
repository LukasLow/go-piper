# go-piper

Eigenständiges, **cross-platform Piper-TTS für Go** — **kein cgo** im Kernpfad,
keine externen Dateien im Betrieb. Beim `go build` wird über **Build-Tags nur
die Runtime des Ziel-OS/Arch eingebettet**.

Zwei Pakete:

| Paket | Was | Status |
|-------|-----|--------|
| `piper` | Piper ausführen über die **eingebettete** Runtime (Kindprozess, cross-platform, kein cgo) | ✅ linux/arm64 |
| `inproc` | **In-Process** per **cgo**: espeak-ng + ONNX Runtime direkt gebunden (kein Piper-Binary, **kein `piper_phonemize`**) | ✅ linux/arm64 · ✅ darwin/arm64 |

## `piper` — Synthese (eingebettet, cross-platform)

```go
import "github.com/LukasLow/go-piper"

wav, err := piper.Synthesize("Guten Tag, ich bin ein Assistent.", piper.Options{
    Model: "/pfad/de_DE-thorsten-high.onnx",
})
```

Demo:
```bash
go run ./cmd -model /pfad/de_DE-thorsten-high.onnx "Guten Tag, ich bin ein Assistent."   # -> out.wav
```

## `inproc` — In-Process (cgo)

```go
import "github.com/LukasLow/go-piper/inproc"

phonemes, err := inproc.Phonemize("Guten Tag", "de", "/pfad/zu/espeak-parent")
// -> "ɡˈuːtən tˈɑːk"
```

Demo:
```bash
CGO_ENABLED=1 go run ./cmd/phonemize "Guten Tag" de /pfad/zu/espeak-parent
```

`inproc` bindet **espeak-ng** direkt an (kein exec). Die Lib wird pro Plattform
aus `inproc/libs/<os>_<arch>/` gelinkt (Build-Tags). Für linux/arm64 liegen
`libespeak-ng.so*` + `include/speak_lib.h` dort (gitignored, per Skript/CI holen).

## Wie das Per-OS-Embedding funktioniert (Paket `piper`)

```
assets_linux_arm64.go   //go:build linux && arm64     //go:embed embed/linux_arm64
assets_unsupported.go   //go:build !(...)              -> klare Fehlermeldung
```

`go build` kompiliert nur die passende Datei → nur die richtige Variante landet
drin. Beispielgrößen:

| Ziel | Binary |
|------|--------|
| `linux/arm64` (Assets drin) | ~55 MB |
| `windows/amd64` / `darwin/arm64` (Stub) | ~3 MB |

## Plattform hinzufügen

1. Runtime besorgen (rhasspy/piper Releases bzw. espeak-ng).
2. ```bash
   scripts/prepare-assets.sh /pfad/zu/piper linux_arm64
   ```
3. `assets_<os>_<arch>.go` (bzw. `inproc/phonemize_<os>_<arch>.go`) nach Muster anlegen.

## Technische Details

- **Keine Modul-Abhängigkeiten**: go-piper importiert ausschließlich die
  Standardbibliothek (`go.mod` hat kein `require`). Die Piper-Vorverarbeitung
  braucht Unicode-NFD; statt `golang.org/x/text` liegt dafür eine eingecheckte,
  generierte Tabelle bei (`inproc/nfd_tables.go`, erzeugt mit
  `python3 scripts/gennfd.py`, plus `inproc/nfd.go`). Sie stimmt im
  Piper-Zeichenbereich (Latein/IPA/Combining, U+0000–U+2FFF) exakt mit
  `x/text` überein.
- **Symlinks**: `go:embed` bettet keine Symlinks ein → `symlinks.txt`-Manifest,
  beim Entpacken nachgebaut.
- **Cache**: entpackt nach `$XDG_CACHE_HOME/go-piper/<tag>/` (idempotent).
- **cgo/inproc**: linkt gegen `inproc/libs/<tag>/` (rpath gesetzt); Datenpfad
  zeigt auf das Verzeichnis, das `espeak-ng-data/` enthält.
- **macOS-Signierung**: Auf Apple Silicon killt das System den ganzen Prozess
  (`SIGKILL "Code Signature Invalid"`), sobald eine per `dlopen` geladene Lib
  keine **gültige** Signatur hat. Deshalb signiert `scripts/relocate_macos.py`
  die relokierten dylibs nach dem Byte-Patch ad-hoc neu, und `inproc` prüft/
  repariert die entpackten Signaturen zusätzlich vor jedem `dlopen`
  (`ensureSigned`, nur darwin; sonst No-op).

## Status

- ✅ `piper`: linux/arm64 (embed + exec)
- ✅ `inproc`: **komplette In-Process-Kette** linux/arm64 — `cmd/vits`:
  espeak-ng (Phoneme) → `phoneme_id_map` → ONNX Runtime (VITS) → WAV.
  ```
  CGO_ENABLED=1 go run ./cmd/vits <model.onnx> <model.onnx.json> <espeak-parent> "Text" out.wav
  ```
- ✅ `inproc`: darwin/arm64 — komplette In-Process-Kette inkl. Relokation der
  Homebrew-espeak-dylib (`libpcaudio`, `@@HOMEBREW_PREFIX@@`) und Ad-hoc-Re-Sign
  nach dem Byte-Patch.
- ⬜ weitere Plattformen

## Lizenz

MIT (dieser Code). Gebündelte Komponenten: Piper MIT, ONNX Runtime MIT,
**espeak-ng GPL-3.0** (siehe `LICENSE`).
