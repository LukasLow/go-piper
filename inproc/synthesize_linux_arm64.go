//go:build linux && arm64 && cgo

package inproc

/*
#cgo CFLAGS: -I${SRCDIR}/libs/linux_arm64/include
#cgo LDFLAGS: -L${SRCDIR}/libs/linux_arm64 -lonnxruntime -Wl,-rpath,${SRCDIR}/libs/linux_arm64
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include "onnxruntime_c_api.h"

// ---- dünner C-Shim um die ONNX-Runtime-C-API ------------------------------
typedef struct {
  const OrtApi* api;
  OrtEnv* env;
  OrtSessionOptions* opts;
  OrtSession* session;
  OrtMemoryInfo* mem;
} lux_ort;

static lux_ort* lux_ort_open(const char* model) {
  lux_ort* h = (lux_ort*)calloc(1, sizeof(lux_ort));
  if (!h) return NULL;
  h->api = OrtGetApiBase()->GetApi(ORT_API_VERSION);
  if (!h->api) { free(h); return NULL; }
  if (h->api->CreateEnv(ORT_LOGGING_LEVEL_ERROR, "go-piper", &h->env)) { free(h); return NULL; }
  if (h->api->CreateSessionOptions(&h->opts)) { free(h); return NULL; }
  h->api->SetIntraOpNumThreads(h->opts, 1);
  h->api->SetSessionGraphOptimizationLevel(h->opts, ORT_ENABLE_ALL);
  if (h->api->CreateSession(h->env, model, h->opts, &h->session)) { free(h); return NULL; }
  if (h->api->CreateCpuMemoryInfo(OrtArenaAllocator, OrtMemTypeDefault, &h->mem)) { free(h); return NULL; }
  return h;
}

static void lux_ort_close(lux_ort* h) {
  if (!h) return;
  const OrtApi* api = h->api;
  if (h->mem) api->ReleaseMemoryInfo(h->mem);
  if (h->session) api->ReleaseSession(h->session);
  if (h->opts) api->ReleaseSessionOptions(h->opts);
  if (h->env) api->ReleaseEnv(h->env);
  free(h);
}

// lux_ort_run: input=[1,T] int64, input_lengths=[1] int64, scales=[3] f32 -> output f32
static int lux_ort_run(lux_ort* h, int64_t* input, int64_t t, float* scales,
                       float** out, int64_t* out_len) {
  const OrtApi* api = h->api;
  int64_t in_shape[2] = {1, t};
  int64_t len_shape[1] = {1};
  int64_t scales_shape[1] = {3};
  int64_t ilen = t;

  OrtValue *in_v = NULL, *len_v = NULL, *scales_v = NULL;
  if (api->CreateTensorWithDataAsOrtValue(h->mem, input, (size_t)t * sizeof(int64_t),
        in_shape, 2, ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, &in_v)) return 1;
  if (api->CreateTensorWithDataAsOrtValue(h->mem, &ilen, sizeof(int64_t),
        len_shape, 1, ONNX_TENSOR_ELEMENT_DATA_TYPE_INT64, &len_v)) return 2;
  if (api->CreateTensorWithDataAsOrtValue(h->mem, scales, 3 * sizeof(float),
        scales_shape, 1, ONNX_TENSOR_ELEMENT_DATA_TYPE_FLOAT, &scales_v)) return 3;

  const char* in_names[3] = {"input", "input_lengths", "scales"};
  const OrtValue* ins[3] = {in_v, len_v, scales_v};
  const char* out_names[1] = {"output"};
  OrtValue* outs[1] = {NULL};
  if (api->Run(h->session, NULL, in_names, ins, 3, out_names, 1, outs)) return 4;

  OrtTensorTypeAndShapeInfo* info = NULL;
  if (api->GetTensorTypeAndShape(outs[0], &info)) return 5;
  size_t n = 0;
  if (api->GetTensorShapeElementCount(info, &n)) return 6;
  float* data = NULL;
  if (api->GetTensorMutableData(outs[0], (void**)&data)) return 7;
  float* buf = (float*)malloc(n * sizeof(float));
  memcpy(buf, data, n * sizeof(float));
  *out = buf;
  *out_len = (int64_t)n;

  api->ReleaseTensorTypeAndShapeInfo(info);
  api->ReleaseValue(outs[0]);
  api->ReleaseValue(in_v);
  api->ReleaseValue(len_v);
  api->ReleaseValue(scales_v);
  return 0;
}

static void lux_free(void* p) { free(p); }
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Model lädt ein Piper-VITS-Modell und kapselt das ONNX-Runtime-Session-Handle.
type Model struct {
	h *C.lux_ort
}

// OpenModel lädt modelPath (.onnx) in eine ORT-Session.
func OpenModel(modelPath string) (*Model, error) {
	c := C.CString(modelPath)
	defer C.free(unsafe.Pointer(c))
	h := C.lux_ort_open(c)
	if h == nil {
		return nil, fmt.Errorf("inproc: ORT-Modell %q konnte nicht geladen werden", modelPath)
	}
	return &Model{h: h}, nil
}

// Close gibt die Session frei.
func (m *Model) Close() {
	if m.h != nil {
		C.lux_ort_close(m.h)
		m.h = nil
	}
}

// Infer rechnet VITS: phonemeIDs -> f32-Audio (mono).
// scales = [noise_scale, length_scale, noise_w].
func (m *Model) Infer(phonemeIDs []int64, scales [3]float32) ([]float32, error) {
	if len(phonemeIDs) == 0 {
		return nil, fmt.Errorf("inproc: leere phonemeIDs")
	}
	var outPtr *C.float
	var outLen C.int64_t
	rc := C.lux_ort_run(m.h, (*C.int64_t)(unsafe.Pointer(&phonemeIDs[0])), C.int64_t(len(phonemeIDs)),
		(*C.float)(unsafe.Pointer(&scales[0])), &outPtr, &outLen)
	if rc != 0 {
		return nil, fmt.Errorf("inproc: ORT-Inferenz fehlgeschlagen (rc=%d)", int(rc))
	}
	if outPtr == nil || outLen == 0 {
		return nil, fmt.Errorf("inproc: ORT lieferte keine Ausgabe")
	}
	defer C.lux_free(unsafe.Pointer(outPtr))
	out := unsafe.Slice((*float32)(unsafe.Pointer(outPtr)), int(outLen))
	// kopieren, bevor der Puffer freigegeben wird
	cp := make([]float32, len(out))
	copy(cp, out)
	return cp, nil
}
