//go:build cgo && arm64 && (linux || darwin)

// In-Process-Glue: die nativen Libs werden zur Laufzeit per dlopen aus dem
// entpackten Cache geladen (kein Linken, keine Platzierungsabhängigkeit).
package inproc

/*
#cgo linux  CFLAGS: -I${SRCDIR}/libs/linux_arm64/include
#cgo darwin CFLAGS: -I${SRCDIR}/libs/darwin_arm64/include
#cgo linux  LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include "onnxruntime_c_api.h"
#include "speak_lib.h"

// ---- espeak-ng (dynamisch) ----
typedef int (*fn_espeak_Init)(espeak_AUDIO_OUTPUT, int, const char*, int);
typedef int (*fn_espeak_Voice)(const char*);
typedef const char* (*fn_espeak_TTP)(const void**, int, int);
static fn_espeak_Init  p_espeak_Init;
static fn_espeak_Voice p_espeak_Voice;
static fn_espeak_TTP   p_espeak_TTP;

static int lux_espeak_load(const char* lib) {
  void* h = dlopen(lib, RTLD_NOW | RTLD_LOCAL);
  if (!h) return 1;
  p_espeak_Init  = (fn_espeak_Init) dlsym(h, "espeak_Initialize");
  p_espeak_Voice = (fn_espeak_Voice)dlsym(h, "espeak_SetVoiceByName");
  p_espeak_TTP   = (fn_espeak_TTP)  dlsym(h, "espeak_TextToPhonemes");
  if (!p_espeak_Init || !p_espeak_Voice || !p_espeak_TTP) return 2;
  return 0;
}
static int lux_espeak_init(const char* data_parent) {
  return p_espeak_Init(AUDIO_OUTPUT_SYNCHRONOUS, 0, data_parent, 0);
}
static int lux_espeak_voice(const char* voice) { return p_espeak_Voice(voice); }

// espeak_TextToPhonemes in Text schreiben (IPA, UTF-8). Rueckgabe: Laenge.
static int lux_espeak_phonemize(const char* text, char* out, int outlen) {
  if (!p_espeak_TTP) return -1;
  const char* p = text;
  int n = 0;
  for (;;) {
    const char* ph = p_espeak_TTP((const void**)&p, 1, 0x02);
    if (!ph || !*ph) break;
    int l = (int)strlen(ph);
    if (n + l + 1 >= outlen) break;
    memcpy(out + n, ph, l); n += l; out[n++] = ' ';
  }
  if (n > 0 && out[n-1] == ' ') n--;
  out[n] = 0;
  return n;
}

// ---- ONNX Runtime (dynamisch) ----
typedef const OrtApiBase* (*fn_OrtGetApiBase)(void);
static const OrtApi* g_ort;

static int lux_ort_load(const char* lib) {
  void* h = dlopen(lib, RTLD_NOW | RTLD_LOCAL);
  if (!h) return 1;
  fn_OrtGetApiBase f = (fn_OrtGetApiBase)dlsym(h, "OrtGetApiBase");
  if (!f) return 2;
  const OrtApiBase* b = f();
  if (!b) return 3;
  g_ort = b->GetApi(ORT_API_VERSION);
  return g_ort ? 0 : 4;
}

typedef struct { OrtEnv* env; OrtSessionOptions* opts; OrtSession* session; OrtMemoryInfo* mem; } lux_ort;

static lux_ort* lux_ort_open(const char* model) {
  if (!g_ort) return NULL;
  lux_ort* h = (lux_ort*)calloc(1, sizeof(lux_ort));
  if (!h) return NULL;
  if (g_ort->CreateEnv(ORT_LOGGING_LEVEL_ERROR, "go-piper", &h->env)) { free(h); return NULL; }
  if (g_ort->CreateSessionOptions(&h->opts)) { free(h); return NULL; }
  // Exakt wie Piper: Graph-Optimierung aus, Threads auf Default.
  (void)g_ort->SetSessionGraphOptimizationLevel(h->opts, ORT_DISABLE_ALL);
  if (g_ort->CreateSession(h->env, model, h->opts, &h->session)) { free(h); return NULL; }
  if (g_ort->CreateCpuMemoryInfo(OrtArenaAllocator, OrtMemTypeDefault, &h->mem)) { free(h); return NULL; }
  return h;
}
static void lux_ort_close(lux_ort* h) {
  if (!h) return;
  if (h->mem) g_ort->ReleaseMemoryInfo(h->mem);
  if (h->session) g_ort->ReleaseSession(h->session);
  if (h->opts) g_ort->ReleaseSessionOptions(h->opts);
  if (h->env) g_ort->ReleaseEnv(h->env);
  free(h);
}
static int lux_ort_run(lux_ort* h, int64_t* input, int64_t t, float* scales, float** out, int64_t* out_len) {
  int64_t in_shape[2] = {1, t}, len_shape[1] = {1}, sc_shape[1] = {3}, ilen = t;
  OrtValue *in_v=NULL,*len_v=NULL,*sc_v=NULL;
  if (g_ort->CreateTensorWithDataAsOrtValue(h->mem, input, (size_t)t*sizeof(int64_t), in_shape, 2, ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, &in_v)) return 1;
  if (g_ort->CreateTensorWithDataAsOrtValue(h->mem, &ilen, sizeof(int64_t), len_shape, 1, ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, &len_v)) return 2;
  if (g_ort->CreateTensorWithDataAsOrtValue(h->mem, scales, 3*sizeof(float), sc_shape, 1, ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, &sc_v)) return 3;
  const char* in_names[3] = {"input","input_lengths","scales"};
  const OrtValue* ins[3] = {in_v, len_v, sc_v};
  const char* out_names[1] = {"output"};
  OrtValue* outs[1] = {NULL};
  if (g_ort->Run(h->session, NULL, in_names, ins, 3, out_names, 1, outs)) return 4;
  OrtTensorTypeAndShapeInfo* info=NULL;
  if (g_ort->GetTensorTypeAndShape(outs[0], &info)) return 5;
  size_t n=0;
  if (g_ort->GetTensorShapeElementCount(info, &n)) return 6;
  float* data=NULL;
  if (g_ort->GetTensorMutableData(outs[0], (void**)&data)) return 7;
  float* buf=(float*)malloc(n*sizeof(float));
  memcpy(buf, data, n*sizeof(float));
  *out=buf; *out_len=(int64_t)n;
  g_ort->ReleaseTensorTypeAndShapeInfo(info);
  g_ort->ReleaseValue(outs[0]);
  g_ort->ReleaseValue(in_v); g_ort->ReleaseValue(len_v); g_ort->ReleaseValue(sc_v);
  return 0;
}
static void lux_free(void* p) { free(p); }
*/
import "C"

import (
	"fmt"
	"strings"
	"sync"
	"unsafe"
)

var (
	loadOnce sync.Once
	loadErr  error
)

// loadLibs lädt espeak + onnxruntime aus dir (einmalig).
func loadLibs(dir string) error {
	loadOnce.Do(func() {
		// macOS: kaputte/fehlende Signaturen auf den entpackten dylibs reparieren,
		// bevor dlopen sie lädt — sonst killt das System den Prozess sofort.
		// Auf anderen Plattformen ein No-op.
		_ = ensureSigned(dir)
		es := C.CString(dir + "/lib/" + espeakLib)
		defer C.free(unsafe.Pointer(es))
		if rc := C.lux_espeak_load(es); rc != 0 {
			loadErr = fmt.Errorf("inproc: dlopen espeak fehlgeschlagen (rc=%d)", int(rc))
			return
		}
		or := C.CString(dir + "/lib/" + ortLib)
		defer C.free(unsafe.Pointer(or))
		if rc := C.lux_ort_load(or); rc != 0 {
			loadErr = fmt.Errorf("inproc: dlopen onnxruntime fehlgeschlagen (rc=%d)", int(rc))
			return
		}
	})
	return loadErr
}

// Phonemize liefert IPA-Phoneme. dataParent ist das Verzeichnis, das
// `espeak-ng-data/` enthält.
func Phonemize(text, voice, dataParent string) (string, error) {
	root, err := engineRoot()
	if err != nil {
		return "", err
	}
	if err := loadLibs(root); err != nil {
		return "", err
	}
	cd := C.CString(dataParent)
	defer C.free(unsafe.Pointer(cd))
	if C.lux_espeak_init(cd) < 0 {
		return "", fmt.Errorf("inproc: espeak_Initialize(%q) fehlgeschlagen", dataParent)
	}
	cv := C.CString(voice)
	defer C.free(unsafe.Pointer(cv))
	if C.lux_espeak_voice(cv) != 0 {
		return "", fmt.Errorf("inproc: espeak_SetVoiceByName(%q) fehlgeschlagen", voice)
	}
	ct := C.CString(text)
	defer C.free(unsafe.Pointer(ct))
	buf := C.malloc(1 << 16)
	defer C.free(buf)
	n := C.lux_espeak_phonemize(ct, (*C.char)(buf), C.int(1<<16))
	if n < 0 {
		return "", fmt.Errorf("inproc: Phonemisierung fehlgeschlagen")
	}
	return strings.TrimSpace(C.GoStringN((*C.char)(buf), n)), nil
}

// Infer rechnet VITS: phonemeIDs -> f32-Audio. modelPath ist der .onnx-Pfad.
func Infer(modelPath string, phonemeIDs []int64, scales [3]float32) ([]float32, error) {
	root, err := engineRoot()
	if err != nil {
		return nil, err
	}
	if err := loadLibs(root); err != nil {
		return nil, err
	}
	if len(phonemeIDs) == 0 {
		return nil, fmt.Errorf("inproc: leere phonemeIDs")
	}
	cm := C.CString(modelPath)
	defer C.free(unsafe.Pointer(cm))
	h := C.lux_ort_open(cm)
	if h == nil {
		return nil, fmt.Errorf("inproc: ORT-Modell %q konnte nicht geladen werden", modelPath)
	}
	defer C.lux_ort_close(h)

	var outPtr *C.float
	var outLen C.int64_t
	rc := C.lux_ort_run(h, (*C.int64_t)(unsafe.Pointer(&phonemeIDs[0])), C.int64_t(len(phonemeIDs)),
		(*C.float)(unsafe.Pointer(&scales[0])), &outPtr, &outLen)
	if rc != 0 || outPtr == nil || outLen == 0 {
		return nil, fmt.Errorf("inproc: ORT-Inferenz fehlgeschlagen (rc=%d)", int(rc))
	}
	defer C.lux_free(unsafe.Pointer(outPtr))
	out := unsafe.Slice((*float32)(unsafe.Pointer(outPtr)), int(outLen))
	cp := make([]float32, len(out))
	copy(cp, out)
	return cp, nil
}
