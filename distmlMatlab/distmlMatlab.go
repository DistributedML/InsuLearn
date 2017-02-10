package distmlMatlab

import (
	//"fmt"
	"sort"
	"unsafe"
)

//#cgo CFLAGS: -m64 -Dchar16_t=uint16_t -DmxUint64=uint64_t -DmxInt64=int64_t -I include -Wl,--export-all-symbolse

/*
#cgo CFLAGS: -m64 -Dchar16_t=uint16_t -I include -Wl,--export-all-symbolse
#cgo LDFLAGS: -L lib -lmex -lmx -leng -lmat
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <inttypes.h>
#include "engine.h"
extern void ReadData(Engine *ep, char *x, char *y);
extern double* FitModel(Engine *ep, double *paramlen);
extern void Predict(Engine *ep, double *model, int paramlen, double *weight, double *size);
extern void PushModel(Engine *ep, double *model, int paramlen, double *rvec, int veclen, double modelC, double location);
extern void InitializeGlobal(Engine *ep, double nodes, double modelD);
extern double* GetGlobalModel(Engine *ep, double *paramlen);
extern void PredictGlobal(Engine *ep, double *model, int paramlen, double *acc, double *cor);
*/
import "C"

type MatlabEngine *C.Engine

type MatModel struct {
	Params []float64
	Weight float64
	Size   float64
}

type MatGlobalModel struct {
	Params []float64
}

type Slice struct {
	sort.IntSlice
	idx []int
}

func NewModel(x, y string) MatModel {
	ep := StartEngine()
	model := MatModel{nil, 0.0, 0.0}
	ReadData(ep, x, y)
	model.Fit(ep)
	model.Weight, model.Size = model.Test(ep)
	StopEngine(ep)
	return model
}

func TestModel(x, y string, model *MatModel) {
	ep := StartEngine()
	ReadData(ep, x, y)
	model.Weight, model.Size = model.Test(ep)
	StopEngine(ep)
}

func GetError(x, y string, model MatModel) float64 {
	ep := StartEngine()
	ReadData(ep, x, y)
	weight, _ := model.Test(ep)
	StopEngine(ep)
	return weight
}

func GetErrorGlobal(x, y string, model MatGlobalModel) (float64, float64) {
	ep := StartEngine()
	ReadData(ep, x, y)
	acc, cor := model.Test(ep)
	StopEngine(ep)
	return acc, cor
}

func CompactGlobal(models map[int]MatModel, modelR map[int]map[int]float64, modelC map[int]float64, modelD float64) MatGlobalModel {
	var paramlen C.double
	N := len(modelR) - 1

	//TODO ugly, should change
	for k1, row := range modelR {
		if N < k1 {
			N = k1
		}
		for k2, _ := range row {
			if N < k2 {
				N = k2
			}
		}
	}

	N++

	ep := StartEngine()
	gmodel := MatGlobalModel{nil}
	C.InitializeGlobal(ep, C.double(N), C.double(modelD))
	for i := 0; i < N; i++ {
		if modelR[i] != nil {
			rvec := make([]float64, N)
			for j := 0; j < N; j++ {
				rvec[j] = modelR[i][j]
			}
			cparams := (*C.double)(&models[i].Params[0])
			crvec := (*C.double)(&rvec[0])
			C.PushModel(ep, cparams, C.int(len(models[i].Params)), crvec, C.int(N), C.double(modelC[i]), C.double(i))
		}
	}
	cparams := C.GetGlobalModel(ep, &paramlen)
	gmodel.Params = doubleToFloats(cparams, int(paramlen))

	StopEngine(ep)

	return gmodel
}

func StartEngine() *C.Engine {
	return C.engOpenSingleUse(nil, nil, nil)
}

func StopEngine(ep *C.Engine) {
	C.engClose(ep)
}

func ReadData(ep *C.Engine, x, y string) {
	C.ReadData(ep, C.CString(x), C.CString(y))
}

func (model *MatModel) Fit(ep *C.Engine) {
	var paramlen C.double
	cparams := C.FitModel(ep, &paramlen)
	model.Params = doubleToFloats(cparams, int(paramlen))
}

func (model *MatModel) Test(ep *C.Engine) (float64, float64) {
	var weight C.double
	var size C.double
	cparams := (*C.double)(&model.Params[0])
	C.Predict(ep, cparams, C.int(len(model.Params)), &weight, &size)
	return float64(weight), float64(size)
}

func (model *MatGlobalModel) Test(ep *C.Engine) (float64, float64) {
	var acc C.double
	var cor C.double
	cparams := (*C.double)(&model.Params[0])
	C.PredictGlobal(ep, cparams, C.int(len(model.Params)), &acc, &cor)
	return float64(acc), float64(cor)
}

func doubleToFloats(in *C.double, size int) []float64 {
	//defer C.free(unsafe.Pointer(in))
	out := (*[1 << 30]float64)(unsafe.Pointer(in))[:size:size]
	return out
}

func (s Slice) Swap(i, j int) {
	s.IntSlice.Swap(i, j)
	s.idx[i], s.idx[j] = s.idx[j], s.idx[i]
}

func NewSlice(n ...int) *Slice {
	s := &Slice{IntSlice: sort.IntSlice(n), idx: make([]int, len(n))}
	for i := range s.idx {
		s.idx[i] = i
	}
	return s
}
