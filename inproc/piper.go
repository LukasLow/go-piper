package inproc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
)

// Config ist der Piper-`.onnx.json`-Begleitfile (Ausschnitt).
type Config struct {
	Audio struct {
		SampleRate int `json:"sample_rate"`
	} `json:"audio"`
	PhonemeIDMap map[string][]int64 `json:"phoneme_id_map"`
}

// LoadConfig liest <model>.onnx.json.
func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("inproc: config json: %w", err)
	}
	if c.Audio.SampleRate == 0 {
		c.Audio.SampleRate = 22050
	}
	if len(c.PhonemeIDMap) == 0 {
		return nil, fmt.Errorf("inproc: config ohne phoneme_id_map")
	}
	return &c, nil
}

// PhonemesToIDs bildet Pipers Sequenz: BOS, dann je Phonem (IDs + PAD), dann EOS.
func (c *Config) PhonemesToIDs(phonemes string) []int64 {
	ids := make([]int64, 0, len(phonemes)*2+2)
	ids = append(ids, c.PhonemeIDMap["^"]...) // BOS
	pad := c.PhonemeIDMap["_"]
	for _, r := range phonemes {
		v, ok := c.PhonemeIDMap[string(r)]
		if !ok {
			continue
		}
		ids = append(ids, v...)
		ids = append(ids, pad...)
	}
	ids = append(ids, c.PhonemeIDMap["$"]...) // EOS
	return ids
}

// FloatToWAV wandelt f32-Samples (mono, [-1,1]) in eine 16-bit-PCM-WAV-Datei.
// Wie Piper wird auf die Spitze normiert.
func FloatToWAV(samples []float32, sampleRate int) []byte {
	peak := 1e-9
	for _, s := range samples {
		if a := math.Abs(float64(s)); a > peak {
			peak = a
		}
	}
	n := len(samples)
	data := make([]byte, n*2)
	for i, s := range samples {
		v := int16((float64(s) / peak) * 32767.0)
		binary.LittleEndian.PutUint16(data[i*2:], uint16(v))
	}
	hdr := make([]byte, 44)
	copy(hdr[0:], "RIFF")
	binary.LittleEndian.PutUint32(hdr[4:], uint32(36+len(data)))
	copy(hdr[8:], "WAVE")
	copy(hdr[12:], "fmt ")
	binary.LittleEndian.PutUint32(hdr[16:], 16)
	binary.LittleEndian.PutUint16(hdr[20:], 1)
	binary.LittleEndian.PutUint16(hdr[22:], 1)
	binary.LittleEndian.PutUint32(hdr[24:], uint32(sampleRate))
	binary.LittleEndian.PutUint32(hdr[28:], uint32(sampleRate*2))
	binary.LittleEndian.PutUint16(hdr[32:], 2)
	binary.LittleEndian.PutUint16(hdr[34:], 16)
	copy(hdr[36:], "data")
	binary.LittleEndian.PutUint32(hdr[40:], uint32(len(data)))
	return append(hdr, data...)
}
