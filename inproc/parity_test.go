//go:build linux && cgo

package inproc

import (
	"bytes"
	"encoding/binary"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Golden-Phonem-IDs, erzeugt mit `piper --debug` (Referenz) für diesen Text.
// Da die Vorverarbeitung (espeak -> NFD -> phoneme_id_map) deterministisch ist,
// muss go-piper exakt dieselben IDs liefern.
const goldenText = "Guten Tag. Ich bin ein Assistent. Eins, zwei, drei."

var goldenIDs = [][]int64{
	{1, 0, 66, 0, 120, 0, 33, 0, 122, 0, 32, 0, 59, 0, 26, 0, 3, 0, 32, 0, 120, 0, 51, 0, 122, 0, 23, 0, 10, 0, 2},
	{1, 0, 74, 0, 16, 0, 140, 0, 3, 0, 15, 0, 74, 0, 26, 0, 3, 0, 14, 0, 74, 0, 26, 0, 3, 0, 121, 0, 14, 0, 31, 0, 74, 0, 31, 0, 32, 0, 120, 0, 61, 0, 26, 0, 32, 0, 10, 0, 2},
	{1, 0, 120, 0, 14, 0, 74, 0, 26, 0, 31, 0, 8, 0, 3, 0, 32, 0, 31, 0, 34, 0, 120, 0, 14, 0, 74, 0, 8, 0, 3, 0, 17, 0, 92, 0, 120, 0, 14, 0, 74, 0, 10, 0, 2},
}

// TestPhonemeIDsGolden prüft, dass go-pipers Vorverarbeitung exakt Pipers
// IDs erzeugt (deterministisch). Braucht nur die Voice-Konfiguration
// (PIPER_CONFIG), kein Piper-Binary.
func TestPhonemeIDsGolden(t *testing.T) {
	cfg := configEnv(t)
	sentences, err := SentencesFor(goldenText, "de")
	if err != nil {
		t.Fatalf("Sentences: %v", err)
	}
	if len(sentences) != len(goldenIDs) {
		t.Fatalf("Satzanzahl: got %d, want %d (%q)", len(sentences), len(goldenIDs), sentences)
	}
	c, err := LoadConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range sentences {
		got := c.PhonemesToIDs(s)
		if !eqIDs(got, goldenIDs[i]) {
			t.Errorf("satz %d IDs weichen ab:\n got  %v\n want %v\n phone %q", i+1, got, goldenIDs[i], s)
		}
	}
}

// TestPiperAudioParity vergleicht die Ausgabe der Referenz-`piper`-Binary mit
// go-piper inproc bei deaktiviertem Rauschen (noise=0). Da beide onnxruntime
// unterschiedlich parallelisieren, ist die Ausgabe nicht bit-identisch, aber
// hoch korreliert; wir prüfen gleiche Länge und Korrelation > 0.95.
//
// Voraussetzungen (sonst Skip): PIPER_BIN, PIPER_MODEL, PIPER_CONFIG.
func TestPiperAudioParity(t *testing.T) {
	bin := os.Getenv("PIPER_BIN")
	model := os.Getenv("PIPER_MODEL")
	if bin == "" || model == "" {
		t.Skip("PIPER_BIN/PIPER_MODEL nicht gesetzt")
	}
	cfg := configEnv(t)
	espeak := os.Getenv("PIPER_ESPEAK")

	out := filepath.Join(t.TempDir(), "piper.wav")
	args := []string{"-q", "-m", model, "-f", out, "--noise_scale", "0", "--noise_w", "0"}
	if espeak != "" {
		args = append(args, "--espeak_data", espeak)
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin = bytes.NewReader([]byte(goldenText))
	cmd.Env = append(os.Environ(), "LD_LIBRARY_PATH="+filepath.Dir(bin))
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("piper: %v\n%s", err, b)
	}
	ref := wavPCM(t, mustReadBytes(t, out))

	ourWav, err := SynthesizeWith(goldenText, "de", model, cfg, Options{NoiseScale: 0, LengthScale: 1, NoiseW: 0})
	if err != nil {
		t.Fatalf("go-piper: %v", err)
	}
	our := wavPCM(t, ourWav)

	if len(ref) != len(our) {
		t.Fatalf("Länge unterschiedlich: piper %d, go-piper %d", len(ref), len(our))
	}
	corr := correlation(ref, our)
	t.Logf("Samples %d, Korrelation %.4f", len(ref), corr)
	if corr < 0.95 {
		t.Errorf("Korrelation zu klein: %.4f", corr)
	}
}

// ---- Helfer ---------------------------------------------------------------

func configEnv(t *testing.T) string {
	t.Helper()
	if c := os.Getenv("PIPER_CONFIG"); c != "" {
		return c
	}
	if _, err := os.Stat("testdata/config.json"); err == nil {
		return "testdata/config.json"
	}
	t.Skip("PIPER_CONFIG nicht gesetzt (und kein testdata/config.json)")
	return ""
}

func eqIDs(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func mustReadBytes(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func wavPCM(t *testing.T, wav []byte) []int16 {
	t.Helper()
	i := bytes.Index(wav, []byte("data"))
	if i < 0 || i+8 > len(wav) {
		t.Fatalf("kein data-Chunk (%d Bytes)", len(wav))
	}
	n := int(binary.LittleEndian.Uint32(wav[i+4 : i+8]))
	body := wav[i+8:]
	if n > len(body) {
		n = len(body)
	}
	out := make([]int16, n/2)
	for k := range out {
		out[k] = int16(binary.LittleEndian.Uint16(body[k*2:]))
	}
	return out
}

func correlation(a, b []int16) float64 {
	n := len(a)
	if n == 0 {
		return 0
	}
	var ma, mb float64
	for i := 0; i < n; i++ {
		ma += float64(a[i])
		mb += float64(b[i])
	}
	ma /= float64(n)
	mb /= float64(n)
	var num, da, db float64
	for i := 0; i < n; i++ {
		x, y := float64(a[i])-ma, float64(b[i])-mb
		num += x * y
		da += x * x
		db += y * y
	}
	if da == 0 || db == 0 {
		return 0
	}
	return num / (sqrt(da) * sqrt(db))
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 40; i++ {
		z = (z + x/z) / 2
	}
	return z
}
