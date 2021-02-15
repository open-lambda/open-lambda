package server

import (
	"log"
	"net/http"
	"strings"
	"fmt"
	"io/ioutil"

	wasmer "github.com/wasmerio/wasmer-go/wasmer"
)

type WasmHandler func(http.ResponseWriter, []string, []byte) error

type WasmServer struct {
	engine *wasmer.Engine
	store *wasmer.Store
}

func (server *WasmServer) RunLambda(w http.ResponseWriter, rsrc []string, args []byte) error {
	wasmBytes, err := ioutil.ReadFile("test-registry.wasm/pyodide.asm.wasm")

	if err != nil {
		log.Fatal(err)
	}

	// TODO cache the module
	module, err := wasmer.NewModule(server.store, wasmBytes)

	if err != nil {
		log.Fatal(err)
	}

	invokeIiFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeIiiFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeIiiiFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeIiiiiFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeViFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeViiFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeViiiiFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	getTempRet0 := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	setTempRet0 := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	abortFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			log.Fatal("Got abort")

			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	assertFailFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			log.Fatal("Assert failed")

			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	buildEnvironmentFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	getTimeFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaAllocateExceptionFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaBeginCatchFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaRethrowPrimaryExceptionFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaCurrentPrimaryExceptionFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaPureVirtualFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaThrowFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaUncaughtExceptionsFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	cxaIncrementExceptionRefcountFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	cxaDecrementExceptionRefcountFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	jsToPythonFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	libcCurrentSigrtminFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	libcCurrentSigrtmaxFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	lockFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	inetAddrFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	killFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	mapFileFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	setErrnoFunc := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall10Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall12Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall14Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall15Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall20Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall102Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall114Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall118Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall121Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall122Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall125Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall132Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall133Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall140Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall142Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall144Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall145Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall147Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall148Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall150Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall151Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall152Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall153Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall163Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall168Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall180Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall181Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall183Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall191Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall192Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall193Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall194Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall195Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall196Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall197Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall198Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall199Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall200Func := wasmer.NewFunction(
		server.store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	importObject := wasmer.NewImportObject()

	importObject.Register(
		"env",
		map[string]wasmer.IntoExtern{
			"abort": abortFunc,
			"___syscall3": syscall10Func,
			"___syscall4": syscall10Func,
			"___syscall5": syscall10Func,
			"___syscall6": syscall10Func,
			"___syscall9": syscall10Func,
			"___syscall10": syscall10Func,
			"___syscall12": syscall12Func,
			"___syscall14": syscall14Func,
			"___syscall15": syscall15Func,
			"___syscall20": syscall20Func,
			"___syscall29": syscall20Func,
			"___syscall33": syscall20Func,
			"___syscall34": syscall20Func,
			"___syscall36": syscall20Func,
			"___syscall38": syscall20Func,
			"___syscall39": syscall20Func,
			"___syscall40": syscall20Func,
			"___syscall41": syscall20Func,
			"___syscall42": syscall20Func,
			"___syscall51": syscall20Func,
			"___syscall54": syscall20Func,
			"___syscall57": syscall20Func,
			"___syscall60": syscall20Func,
			"___syscall63": syscall20Func,
			"___syscall64": syscall20Func,
			"___syscall66": syscall20Func,
			"___syscall75": syscall20Func,
			"___syscall77": syscall20Func,
			"___syscall83": syscall20Func,
			"___syscall85": syscall20Func,
			"___syscall91": syscall20Func,
			"___syscall94": syscall20Func,
			"___syscall96": syscall20Func,
			"___syscall97": syscall20Func,
			"___syscall102": syscall102Func,
			"___syscall114": syscall114Func,
			"___syscall118": syscall118Func,
			"___syscall121": syscall121Func,
			"___syscall122": syscall122Func,
			"___syscall125": syscall125Func,
			"___syscall132": syscall132Func,
			"___syscall133": syscall133Func,
			"___syscall140": syscall140Func,
			"___syscall142": syscall142Func,
			"___syscall144": syscall144Func,
			"___syscall145": syscall145Func,
			"___syscall147": syscall147Func,
			"___syscall148": syscall148Func,
			"___syscall150": syscall150Func,
			"___syscall151": syscall151Func,
			"___syscall152": syscall152Func,
			"___syscall153": syscall153Func,
			"___syscall163": syscall163Func,
			"___syscall168": syscall168Func,
			"___syscall180": syscall180Func,
			"___syscall181": syscall181Func,
			"___syscall183": syscall183Func,
			"___syscall191": syscall191Func,
			"___syscall192": syscall192Func,
			"___syscall193": syscall193Func,
			"___syscall194": syscall194Func,
			"___syscall195": syscall195Func,
			"___syscall196": syscall196Func,
			"___syscall197": syscall197Func,
			"___syscall198": syscall198Func,
			"___syscall199": syscall199Func,
			"___syscall200": syscall200Func,
			"___syscall201": syscall200Func,
			"___syscall202": syscall200Func,
			"___syscall205": syscall200Func,
			"___syscall207": syscall200Func,
			"___syscall209": syscall200Func,
			"___syscall211": syscall200Func,
			"___syscall212": syscall200Func,
			"___syscall218": syscall200Func,
			"___syscall219": syscall200Func,
			"___syscall220": syscall200Func,
			"___syscall221": syscall200Func,
			"___syscall268": syscall200Func,
			"___syscall269": syscall200Func,
			"___syscall272": syscall200Func,
			"___syscall295": syscall200Func,
			"___syscall296": syscall200Func,
			"___syscall297": syscall200Func,
			"___syscall298": syscall200Func,
			"___syscall300": syscall200Func,
			"___syscall301": syscall200Func,
			"___syscall302": syscall200Func,
			"___syscall303": syscall200Func,
			"___syscall304": syscall200Func,
			"___syscall305": syscall200Func,
			"___syscall306": syscall200Func,
			"___syscall308": syscall200Func,
			"___syscall320": syscall200Func,
			"___syscall324": syscall200Func,
			"___syscall330": syscall200Func,
			"___syscall331": syscall200Func,
			"___syscall333": syscall200Func,
			"___syscall334": syscall200Func,
			"___syscall337": syscall200Func,
			"___syscall340": syscall200Func,
			"___syscall345": syscall200Func,
			"___setErrNo": setErrnoFunc,
			"___map_file": mapFileFunc,
			"___lock": lockFunc,
			"___unlock": lockFunc,
			"___wait": lockFunc,
			"___wasi_fd_write": lockFunc,
			"__exit": lockFunc,
			"_abort": lockFunc,
			"_alarm": lockFunc,
			"_chroot": lockFunc,
			"_clock": lockFunc,
			"_clock_getres": lockFunc,
			"_clock_gettime": lockFunc,
			"_clock_settime": lockFunc,
			"_confstr": lockFunc,
			"_dlclose": lockFunc,
			"_dlerror": lockFunc,
			"_dlopen": lockFunc,
			"_dlsym": lockFunc,
			"_emscripten_asm_const_i": lockFunc,
			"_emscripten_exit_with_live_runtime": lockFunc,
			"_emscripten_get_heap_size": lockFunc,
			"_emscripten_longjmp": lockFunc,
			"_emscripten_memcpy_big": lockFunc,
			"_emscripten_resize_heap": lockFunc,
			"_endpwent": lockFunc,
			"_execv": lockFunc,
			"_execve": lockFunc,
			"_exit": lockFunc,
			"_fexecve": lockFunc,
			"_fork": lockFunc,
			"_fpathconf": lockFunc,
			"_gai_strerror": lockFunc,
			"_getaddrinfo": lockFunc,
			"_getenv": lockFunc,
			"_gethostbyaddr": lockFunc,
			"_gethostbyname": lockFunc,
			"_getitimer": lockFunc,
			"_getloadavg": lockFunc,
			"_getnameinfo": lockFunc,
			"_getprotobyname": lockFunc,
			"_getpwent": lockFunc,
			"_getpwnam": lockFunc,
			"_getpwuid": lockFunc,
			"_gettimeofday": lockFunc,
			"_gmtime": lockFunc,
			"_gmtime_r": lockFunc,
			"_hiwire_array": lockFunc,
			"_hiwire_bytes": lockFunc,
			"_hiwire_call": lockFunc,
			"_hiwire_call_member": lockFunc,
			"_hiwire_copy_to_ptr": lockFunc,
			"_hiwire_decref": lockFunc,
			"_hiwire_delete_member_obj": lockFunc,
			"_hiwire_delete_member_string": lockFunc,
			"_hiwire_dir": lockFunc,
			"_hiwire_double": lockFunc,
			"_hiwire_equal": lockFunc,
			"_hiwire_float32array": lockFunc,
			"_hiwire_float64array": lockFunc,
			"_hiwire_get_bool": lockFunc,
			"_hiwire_get_byteLength": lockFunc,
			"_hiwire_get_byteOffset": lockFunc,
			"_hiwire_get_dtype": lockFunc,
			"_hiwire_get_global": lockFunc,
			"_hiwire_get_iterator": lockFunc,
			"_hiwire_get_length": lockFunc,
			"_hiwire_get_member_int": lockFunc,
			"_hiwire_get_member_obj": lockFunc,
			"_hiwire_get_member_string": lockFunc,
			"_hiwire_greater_than": lockFunc,
			"_hiwire_greater_than_equal": lockFunc,
			"_hiwire_incref": lockFunc,
			"_hiwire_int": lockFunc,
			"_hiwire_int16array": lockFunc,
			"_hiwire_int32array": lockFunc,
			"_hiwire_int8array": lockFunc,
			"_hiwire_is_function": lockFunc,
			"_hiwire_is_on_wasm_heap": lockFunc,
			"_hiwire_is_typedarray": lockFunc,
			"_hiwire_less_than": lockFunc,
			"_hiwire_less_than_equal": lockFunc,
			"_hiwire_new": lockFunc,
			"_hiwire_next": lockFunc,
			"_hiwire_nonzero": lockFunc,
			"_hiwire_not_equal": lockFunc,
			"_hiwire_object": lockFunc,
			"_hiwire_push_array": lockFunc,
			"_hiwire_push_object_pair": lockFunc,
			"_hiwire_set_member_obj": lockFunc,
			"_hiwire_set_member_string": lockFunc,
			"_hiwire_setup": lockFunc,
			"_hiwire_string_ascii": lockFunc,
			"_hiwire_string_ucs1": lockFunc,
			"_hiwire_string_ucs2": lockFunc,
			"_hiwire_string_ucs4": lockFunc,
			"_hiwire_subarray": lockFunc,
			"_hiwire_throw_error": lockFunc,
			"_hiwire_to_string": lockFunc,
			"_hiwire_typeof": lockFunc,
			"_hiwire_uint16array": lockFunc,
			"_hiwire_uint32array": lockFunc,
			"_hiwire_uint8array": lockFunc,
			"_inet_addr": inetAddrFunc,
			"_kill": killFunc,
			"_killpg": killFunc,
			"_llvm_copysign_f32": killFunc,
			"_llvm_copysign_f64": killFunc,
			"_llvm_log10_f64": killFunc,
			"_llvm_log2_f64": killFunc,
			"_llvm_stackrestore": killFunc,
			"_llvm_stacksave": killFunc,
			"_llvm_trap": killFunc,
			"_localtime_r": killFunc,
			"_longjmp": killFunc,
			"_mktime": killFunc,
			"_nanosleep": killFunc,
			"_pathconf": killFunc,
			"_posix_spawn": killFunc,
			"_posix_spawn_file_actions_addclose": killFunc,
			"_posix_spawn_file_actions_adddup2": killFunc,
			"_posix_spawn_file_actions_addopen": killFunc,
			"_posix_spawn_file_actions_destroy": killFunc,
			"_posix_spawn_file_actions_init": killFunc,
			"_posix_spawnattr_destroy": killFunc,
			"_posix_spawnattr_init": killFunc,
			"_posix_spawnattr_setflags": killFunc,
			"_posix_spawnattr_setpgroup": killFunc,
			"_posix_spawnattr_setschedparam": killFunc,
			"_posix_spawnattr_setschedpolicy": killFunc,

			"___libc_current_sigrtmin": libcCurrentSigrtminFunc,
			"___libc_current_sigrtmax": libcCurrentSigrtmaxFunc,
			"___js2python": jsToPythonFunc,
			"___clock_gettime": getTimeFunc,
			"___buildEnvironment": buildEnvironmentFunc,
			"___assert_fail": assertFailFunc,
			"___cxa_allocate_exception": cxaAllocateExceptionFunc,
			"___cxa_throw": cxaThrowFunc,
			"___cxa_uncaught_exceptions": cxaUncaughtExceptionsFunc,
			"___cxa_begin_catch": cxaBeginCatchFunc,
			"___cxa_pure_virtual": cxaPureVirtualFunc,
			"___cxa_rethrow_primary_exception": cxaRethrowPrimaryExceptionFunc,
			"___cxa_current_primary_exception": cxaCurrentPrimaryExceptionFunc,
			"___cxa_decrement_exception_refcount": cxaDecrementExceptionRefcountFunc,
			"___cxa_increment_exception_refcount": cxaIncrementExceptionRefcountFunc,
			"getTempRet0": getTempRet0,
			"setTempRet0": setTempRet0,
			"invoke_ii": invokeIiFunc,
			"invoke_iii": invokeIiiFunc,
			"invoke_iiii": invokeIiiiFunc,
			"invoke_iiiii": invokeIiiiiFunc,
			"invoke_vi": invokeViFunc,
			"invoke_vii": invokeViiFunc,
			"invoke_viiii": invokeViiiiFunc,
		},
	)

	instance, err := wasmer.NewInstance(module, importObject)

	if err == nil {
		log.Printf("Loaded and compiled wasm code")
	} else {
		log.Fatal(err)
	}

	content, err := ioutil.ReadFile(fmt.Sprintf("test-registry.wasm/%s", rsrc))
	if err != nil {
		log.Fatal(err)
	}
	
	// Convert []byte to string and print to screen
	code := string(content)
	
	log.Printf("Running code %s", code)

	loadFunc, _ := instance.Exports.GetFunction("loadPackagesFromIports")
	runFunc, _ := instance.Exports.GetFunction("runPython")

	loadFunc(code)
	runFunc(code)

	return nil
}

func (server *WasmServer) HandleInternal(w http.ResponseWriter, r *http.Request) error {
	log.Printf("%s %s", r.Method, r.URL.Path)

	defer r.Body.Close()

	if r.Method != "POST" {
		return fmt.Errorf("Only POST allowed (found %s)", r.Method)
	}

	rbody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	log.Printf("Body %s", rbody)

	rsrc := strings.Split(r.URL.Path, "/")
	if len(rsrc) < 2 {
		return fmt.Errorf("no path arguments provided in URL")
	}

	routes := map[string] WasmHandler{
		"run": server.RunLambda,
	}

	if h, ok := routes[rsrc[1]]; ok {
		return h(w, rsrc[2:], rbody)
	} else {
		return fmt.Errorf("unknown op %s", rsrc[1])
	}
}

func (server *WasmServer) Handle(w http.ResponseWriter, r *http.Request) {
	if err := server.HandleInternal(w, r); err != nil {
		log.Printf("Request Handler Failed: %v", err)
		w.WriteHeader(500)
		w.Write([]byte(fmt.Sprintf("%v\n", err)))
	}
}

func (server *WasmServer) cleanup() {
}

func NewWasmServer() (*WasmServer, error) {
	log.Printf("Starting WASM Server")

	//wasmBytes, _ := ioutil.ReadFile("pyodide.asm.wasm")

	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)

    log.Printf("Created WASM engine")

	server := &WasmServer{ engine, store }

	http.HandleFunc("/", server.Handle)

	return server, nil
}
