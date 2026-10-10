//go:build darwin && arm64 && cgo

// Package inproc bindet die native Piper-Kette **in-process** per cgo an.
//
// Stufe 1: Phonemisierung über die espeak-ng-C-API (kein exec, kein Kindprozess).
// Die Lib wird über Build-Tags pro Plattform aus libs/<os>_<arch>/ gelinkt.
package inproc

/*
#cgo CFLAGS: -I${SRCDIR}/libs/darwin_arm64/include
#cgo LDFLAGS: -L${SRCDIR}/libs/darwin_arm64 -lespeak-ng -Wl,-rpath,${SRCDIR}/libs/darwin_arm64
#include <stdlib.h>
#include "speak_lib.h"
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

// Phonemize liefert IPA-Phoneme (space-separiert) für text in der Sprache voice
// (z. B. "de"). dataDir ist das Verzeichnis, das `espeak-ng-data/` enthält.
func Phonemize(text, voice, dataDir string) (string, error) {
	cdata := C.CString(dataDir)
	defer C.free(unsafe.Pointer(cdata))
	if C.espeak_Initialize(C.AUDIO_OUTPUT_SYNCHRONOUS, 0, cdata, 0) < 0 {
		return "", fmt.Errorf("inproc: espeak_Initialize(%q) fehlgeschlagen", dataDir)
	}

	cvoice := C.CString(voice)
	defer C.free(unsafe.Pointer(cvoice))
	if C.espeak_SetVoiceByName(cvoice) != 0 {
		return "", fmt.Errorf("inproc: espeak_SetVoiceByName(%q) fehlgeschlagen", voice)
	}

	ctext := C.CString(text)
	defer C.free(unsafe.Pointer(ctext))
	p := unsafe.Pointer(ctext)

	var b strings.Builder
	for {
		ph := C.espeak_TextToPhonemes((*unsafe.Pointer)(unsafe.Pointer(&p)), C.espeakCHARS_UTF8, C.espeakPHONEMES_IPA)
		if ph == nil {
			break
		}
		s := C.GoString(ph)
		if s == "" { // Ende: espeak liefert "" statt NULL
			break
		}
		b.WriteString(s)
		b.WriteByte(' ')
	}
	return strings.TrimSpace(b.String()), nil
}
