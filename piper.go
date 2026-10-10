// Package piper ist ein eigenständiges, cross-platform Piper-TTS für Go.
//
// Beim Build wird über Build-Tags **nur die passende native Piper-Runtime**
// für das Ziel-OS/Arch eingebettet (Linux arm64, Windows amd64/arm64,
// Darwin arm64/amd64, …). Zur Laufzeit wird sie in ein Cache-Verzeichnis
// entpackt und ausgeführt. Kein cgo, keine externen Dateien im Repo-Betrieb.
//
//	go get github.com/LukasLow/go-piper
//	wav, err := piper.Synthesize("Guten Tag, ich bin ein Assistent.", piper.Options{})
package piper

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Options steuert die Synthese.
type Options struct {
	// Model ist der Pfad zur .onnx-Stimme. Leer = automatische Suche
	// (Modell neben den Assets bzw. unter ~/.cache/go-piper/models).
	Model string
	// Espeak ist der Pfad zu espeak-ng-data. Leer = aus den Assets.
	Espeak string
	// Speaker ist die Speaker-ID (Default 0).
	Speaker int
}

// Synthesize erzeugt aus text eine WAV-Datei (Bytes).
func Synthesize(text string, opt Options) ([]byte, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("go-piper: leerer Text")
	}

	runtimeDir, err := ensureAssets()
	if err != nil {
		return nil, err
	}
	piperDir := filepath.Join(runtimeDir, "piper")

	bin := filepath.Join(piperDir, exeName("piper"))
	if _, err := os.Stat(bin); err != nil {
		return nil, fmt.Errorf("go-piper: Piper-Binary fehlt (%s)", bin)
	}

	espeak := opt.Espeak
	if espeak == "" {
		espeak = filepath.Join(piperDir, "espeak-ng-data")
	}

	model := opt.Model
	if model == "" {
		model = findModel(runtimeDir)
	}
	if model == "" {
		return nil, fmt.Errorf("go-piper: keine .onnx-Stimme gefunden — Options.Model setzen")
	}

	tmp, err := os.CreateTemp("", "go-piper-*.wav")
	if err != nil {
		return nil, err
	}
	out := tmp.Name()
	tmp.Close()
	defer os.Remove(out)

	args := []string{"-q", "-m", model, "--espeak_data", espeak, "-s", fmt.Sprint(opt.Speaker), "-f", out}
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(), libPathEnv(piperDir))
	cmd.Stdin = strings.NewReader(text)

	if b, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("go-piper: piper fehlgeschlagen: %v\n%s", err, b)
	}
	return os.ReadFile(out)
}

// SynthesizeToFile wie Synthesize, schreibt aber direkt in path.
func SynthesizeToFile(path, text string, opt Options) error {
	wav, err := Synthesize(text, opt)
	if err != nil {
		return err
	}
	return os.WriteFile(path, wav, 0o644)
}

// ---- Asset-Entpackung ------------------------------------------------------

func ensureAssets() (string, error) {
	afs, tag, ok := nativeAssets()
	if !ok {
		return "", fmt.Errorf("go-piper: keine eingebetteten Assets für %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cache, "go-piper", tag)

	// Schnellprüfung: schon entpackt und unverändert?
	if haveVersion(dir, afs, tag) {
		return dir, nil
	}

	prefix := "embed/" + tag
	err = fs.WalkDir(afs, prefix, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(strings.TrimPrefix(p, prefix), "/")
		if rel == "" {
			return os.MkdirAll(dir, 0o755)
		}
		dst := filepath.Join(dir, rel)
		if d.IsDir() {
			return os.MkdirAll(dst, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		data, err := fs.ReadFile(afs, p)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0o644)
		if isExecutable(rel) {
			mode = 0o755
		}
		return os.WriteFile(dst, data, mode)
	})
	if err != nil {
		return "", fmt.Errorf("go-piper: Assets entpacken: %w", err)
	}

	// Symlinks bettet go:embed nicht ein -> aus Manifest nachbauen.
	if data, err := fs.ReadFile(afs, prefix+"/symlinks.txt"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) != 2 {
				continue
			}
			lp := filepath.Join(dir, parts[0])
			_ = os.Remove(lp)
			if err := os.Symlink(parts[1], lp); err != nil {
				return "", fmt.Errorf("go-piper: Symlink %s -> %s: %w", parts[0], parts[1], err)
			}
		}
	}

	writeVersion(dir, afs, tag)
	return dir, nil
}

func exeName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

func isExecutable(rel string) bool {
	base := filepath.Base(rel)
	return base == "piper" || strings.HasSuffix(base, ".exe")
}

func libPathEnv(dir string) string {
	switch runtime.GOOS {
	case "darwin":
		return "DYLD_LIBRARY_PATH=" + dir
	case "windows":
		return "PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH")
	default:
		return "LD_LIBRARY_PATH=" + dir
	}
}

func findModel(runtimeDir string) string {
	cands := []string{
		filepath.Join(runtimeDir, "model.onnx"),
		filepath.Join(runtimeDir, "model", "model.onnx"),
	}
	if cache, err := os.UserCacheDir(); err == nil {
		matches, _ := filepath.Glob(filepath.Join(cache, "go-piper", "models", "*.onnx"))
		cands = append(cands, matches...)
	}
	for _, c := range cands {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}
