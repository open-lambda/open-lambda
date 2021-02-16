package server

import (
	"log"
	wasmer "github.com/wasmerio/wasmer-go/wasmer"
)

func makeEmscriptenBindings(store *wasmer.Store) (*wasmer.ImportObject) {
	invokeIiFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeIiiFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeIiiiFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeIiiiiFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeViFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeViiFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	invokeViiiiFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	getTempRet0 := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	setTempRet0 := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	_abortFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			log.Fatal("Got abort")

			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	abortFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			log.Fatal("Got abort")

			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	assertFailFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			log.Fatal("Assert failed")

			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	buildEnvironmentFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	getTimeFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaAllocateExceptionFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaBeginCatchFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaRethrowPrimaryExceptionFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaCurrentPrimaryExceptionFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaPureVirtualFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaThrowFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			result := make([]wasmer.Value, 0)
			return result, nil
		},
	)

	cxaUncaughtExceptionsFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	cxaIncrementExceptionRefcountFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	cxaDecrementExceptionRefcountFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	jsToPythonFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	libcCurrentSigrtminFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	libcCurrentSigrtmaxFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	waitFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	waitFdFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	lockFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	clockFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	endpwentFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	execvFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	execveFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	forkFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	fpathconfFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	gaiStrerrorFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getEnvFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getAddrInfoFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getHostByAddrFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getItimerFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getNameInfoFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getProtoByNameFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getPwentFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getPwnamFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	gmTimeRFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireArrayFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireCallFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireCallMemberFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireGetMemberIntFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireIncrefFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes( wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireInt16ArrayFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireBytesFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireEqualFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadExitFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireObjectFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwirePushArrayFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwirePushObjectPairFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireSetupFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireNewFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireGetBoolFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireStringUcs1Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireSubarrayFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireUint16ArrayFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireGetDtypeFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireDeleteMemberObjFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireDirFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	hiwireDoubleFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.F64), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	getTimeOfDayFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)
	clockGetResFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	chrootFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	confstrFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	dlcloseFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	dlerrorFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	dlopenFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	dlsymFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	emscriptenAsmConstFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	emscriptenExitWithLiveRuntimeFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	emscriptenGetHeapSizeFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	emscriptenLongjmpFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	posixSpawnFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	posixSpawnFileActionsAddup2Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)
	
	posixSpawnFileActionsAddOpenFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	posixSpawnFileActionsDestroyFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	posixSpawnAttrSetFlagsFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	posixSpawnNpFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadAttrDestroyFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadCleanupPopFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadCleanupPushFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadAttrSetStackSizeFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadCondInitFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadCondTimedWaitFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadCondWaitFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadCreateFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadDetachFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadEqualFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadMutexAttrDestroyFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadMutexAttrInitFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pthreadSigmaskFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	putenvFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pyimportInitFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	pyproxyNewFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	runpythonFinalizeJsFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	setEnvFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	siginterruptFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	sigPendingFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	strftimeFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	strftimeLFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	setGroupsFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	sysconfFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	usleepFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	wait3Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	wait4Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	waitIdFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)
	
	utimesFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	setPwentFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	posixSpawnFileActionsAddCloseFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	mktimeFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)
	
	emscriptenMemcpyBigFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	emscriptenResizeHeapFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)


	alarmFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	clockGetTimeFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	inetAddrFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	llvmCopysignF32Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.F64, wasmer.F64), wasmer.NewValueTypes(wasmer.F64)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	llvmLog10F64Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.F64), wasmer.NewValueTypes(wasmer.F64)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	llvmStacksaveFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	llvmTrapFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	killFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	mapFileFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	setErrnoFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32), wasmer.NewValueTypes()),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall10Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall12Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall14Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall15Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall20Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall102Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall114Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall118Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall121Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall122Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall125Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall132Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall133Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall140Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall142Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall144Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall145Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall147Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall148Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall150Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall151Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall152Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall153Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall163Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall168Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall180Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall181Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall183Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall191Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall192Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall193Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall194Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall195Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall196Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall197Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall198Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall199Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	syscall200Func := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.I32, wasmer.I32), wasmer.NewValueTypes(wasmer.I32)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	f64remFunc := wasmer.NewFunction(
		store,
		wasmer.NewFunctionType(wasmer.NewValueTypes(wasmer.F64, wasmer.F64), wasmer.NewValueTypes(wasmer.F64)),
		func(args []wasmer.Value) ([]wasmer.Value, error) {
			panic("Not implemented yet")
		},
	)

	memoryBase := wasmer.NewGlobal(
		store,
		wasmer.NewGlobalType(wasmer.NewValueType(wasmer.I32), wasmer.IMMUTABLE),
		wasmer.NewI32(0),
	)

	tableBase := wasmer.NewGlobal(
		store,
		wasmer.NewGlobalType(wasmer.NewValueType(wasmer.I32), wasmer.IMMUTABLE),
		wasmer.NewI32(0),
	)
	
	tempDoublePtr := wasmer.NewGlobal(
		store,
		wasmer.NewGlobalType(wasmer.NewValueType(wasmer.I32), wasmer.IMMUTABLE),
		wasmer.NewI32(0),
	)

	dynamicTopPtr := wasmer.NewGlobal(
		store,
		wasmer.NewGlobalType(wasmer.NewValueType(wasmer.I32), wasmer.IMMUTABLE),
		wasmer.NewI32(0),
	)

	notANumber := wasmer.NewGlobal(
		store,
		wasmer.NewGlobalType(wasmer.NewValueType(wasmer.F64), wasmer.IMMUTABLE),
		wasmer.NewF64(0.0),
	)

	memlimits, _ := wasmer.NewLimits(160, 512)

	memory := wasmer.NewMemory(
		store, wasmer.NewMemoryType(memlimits),
	)

/*FIXME	table := wasmer.NewTable(
		store, wasmer.NewMemoryType(memlimits),
	)*/

	importObject := wasmer.NewImportObject()

	importObject.Register(
		"asm2wasm",
		map[string]wasmer.IntoExtern{
			"f64-rem": f64remFunc,
		},
	)

	importObject.Register(
		"global",
		map[string]wasmer.IntoExtern{
			"NaN": notANumber,
			"Infinity": notANumber,
		},
	)

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
			"___wait": waitFunc,
			"___wasi_fd_write": waitFdFunc,
			"__exit": lockFunc,
			"__memory_base": memoryBase,
			"__table_base": tableBase,
			"_abort": _abortFunc,
			"_alarm": alarmFunc,
			"_chroot": chrootFunc,
			"_clock": clockFunc,
			"_clock_getres": clockGetResFunc,
			"_clock_gettime": clockGetTimeFunc,
			"_clock_settime": clockGetResFunc,
			"_confstr": confstrFunc,
			"_dlclose": dlcloseFunc,
			"_dlerror": dlerrorFunc,
			"_dlopen": dlopenFunc,
			"_dlsym": dlsymFunc,
			"_emscripten_asm_const_i": emscriptenAsmConstFunc,
			"_emscripten_exit_with_live_runtime": emscriptenExitWithLiveRuntimeFunc,
			"_emscripten_get_heap_size": emscriptenGetHeapSizeFunc,
			"_emscripten_longjmp": emscriptenLongjmpFunc,
			"_emscripten_memcpy_big": emscriptenMemcpyBigFunc,
			"_emscripten_resize_heap": emscriptenResizeHeapFunc,
			"_endpwent": endpwentFunc,
			"_execv": execvFunc,
			"_execve": execveFunc,
			"_exit": lockFunc,
			"_fexecve": execveFunc,
			"_fork": forkFunc,
			"_fpathconf": fpathconfFunc,
			"_gai_strerror": gaiStrerrorFunc,
			"_getaddrinfo": getAddrInfoFunc,
			"_getenv": getEnvFunc,
			"_gethostbyaddr": getHostByAddrFunc,
			"_gethostbyname": gaiStrerrorFunc,
			"_getitimer": getItimerFunc,
			"_getloadavg": getItimerFunc,
			"_getnameinfo": getNameInfoFunc,
			"_getprotobyname": getProtoByNameFunc,
			"_getpwent": getPwentFunc,
			"_getpwnam": getPwnamFunc,
			"_getpwuid": getPwnamFunc,
			"_gettimeofday": getTimeOfDayFunc,
			"_gmtime": getPwnamFunc,
			"_gmtime_r": gmTimeRFunc,
			"_hiwire_array": hiwireArrayFunc,
			"_hiwire_bytes": hiwireBytesFunc,
			"_hiwire_call": hiwireCallFunc,
			"_hiwire_call_member": hiwireCallMemberFunc,
			"_hiwire_copy_to_ptr": hiwireCallFunc,
			"_hiwire_decref": lockFunc,
			"_hiwire_delete_member_obj": hiwireDeleteMemberObjFunc,
			"_hiwire_delete_member_string": hiwireDeleteMemberObjFunc,
			"_hiwire_dir": hiwireDirFunc,
			"_hiwire_double": hiwireDoubleFunc,
			"_hiwire_equal": hiwireEqualFunc,
			"_hiwire_float32array": hiwireEqualFunc,
			"_hiwire_float64array": hiwireEqualFunc,
			"_hiwire_get_bool": hiwireGetBoolFunc,
			"_hiwire_get_byteLength": hiwireGetBoolFunc,
			"_hiwire_get_byteOffset": hiwireGetBoolFunc,
			"_hiwire_get_dtype": hiwireGetDtypeFunc,
			"_hiwire_get_global": hiwireGetDtypeFunc,
			"_hiwire_get_iterator": hiwireGetDtypeFunc,
			"_hiwire_get_length": hiwireGetDtypeFunc,
			"_hiwire_get_member_int": hiwireGetMemberIntFunc,
			"_hiwire_get_member_obj": hiwireGetMemberIntFunc,
			"_hiwire_get_member_string": hiwireGetMemberIntFunc,
			"_hiwire_greater_than": hiwireGetMemberIntFunc,
			"_hiwire_greater_than_equal": hiwireGetMemberIntFunc,
			"_hiwire_incref": hiwireIncrefFunc,
			"_hiwire_int": hiwireIncrefFunc,
			"_hiwire_int16array": hiwireInt16ArrayFunc,
			"_hiwire_int32array": hiwireInt16ArrayFunc,
			"_hiwire_int8array": hiwireInt16ArrayFunc,
			"_hiwire_is_function": hiwireIncrefFunc,
			"_hiwire_is_on_wasm_heap": hiwireIncrefFunc,
			"_hiwire_is_typedarray": hiwireIncrefFunc,
			"_hiwire_less_than": hiwireEqualFunc,
			"_hiwire_less_than_equal": hiwireEqualFunc,
			"_hiwire_new": hiwireNewFunc,
			"_hiwire_next": hiwireGetBoolFunc,
			"_hiwire_nonzero": hiwireGetBoolFunc,
			"_hiwire_not_equal": hiwireEqualFunc,
			"_hiwire_object": hiwireObjectFunc,
			"_hiwire_push_array": hiwirePushArrayFunc,
			"_hiwire_push_object_pair": hiwirePushObjectPairFunc,
			"_hiwire_set_member_obj": hiwirePushObjectPairFunc,
			"_hiwire_set_member_string": hiwirePushObjectPairFunc,
			"_hiwire_setup": hiwireSetupFunc,
			"_hiwire_string_ascii": alarmFunc,
			"_hiwire_string_ucs1": hiwireStringUcs1Func,
			"_hiwire_string_ucs2": hiwireStringUcs1Func,
			"_hiwire_string_ucs4": hiwireStringUcs1Func,
			"_hiwire_subarray": hiwireSubarrayFunc,
			"_hiwire_throw_error": lockFunc,
			"_hiwire_to_string": alarmFunc,
			"_hiwire_typeof": alarmFunc,
			"_hiwire_uint16array": hiwireUint16ArrayFunc,
			"_hiwire_uint32array": hiwireUint16ArrayFunc,
			"_hiwire_uint8array": hiwireUint16ArrayFunc,
			"_inet_addr": inetAddrFunc,
			"_kill": killFunc,
			"_killpg": killFunc,
			"_llvm_copysign_f32": llvmCopysignF32Func,
			"_llvm_copysign_f64": llvmCopysignF32Func,
			"_llvm_log10_f64": llvmLog10F64Func,
			"_llvm_log2_f64": llvmLog10F64Func,
			"_llvm_stackrestore": lockFunc,
			"_llvm_stacksave": llvmStacksaveFunc,
			"_llvm_trap": llvmTrapFunc,
			"_localtime_r": killFunc,
			"_longjmp": emscriptenLongjmpFunc,
			"_mktime": mktimeFunc,
			"_nanosleep": killFunc,
			"_pathconf": killFunc,
			"_posix_spawn": posixSpawnFunc,
			"_posix_spawn_file_actions_addclose": posixSpawnFileActionsAddCloseFunc,
			"_posix_spawn_file_actions_adddup2": posixSpawnFileActionsAddup2Func,
			"_posix_spawn_file_actions_addopen": posixSpawnFileActionsAddOpenFunc,
			"_posix_spawn_file_actions_destroy": posixSpawnFileActionsDestroyFunc,
			"_posix_spawn_file_actions_init": posixSpawnFileActionsDestroyFunc,
			"_posix_spawnattr_destroy": posixSpawnFileActionsDestroyFunc,
			"_posix_spawnattr_init": posixSpawnFileActionsDestroyFunc,
			"_posix_spawnattr_setflags": posixSpawnAttrSetFlagsFunc,
			"_posix_spawnattr_setpgroup": posixSpawnAttrSetFlagsFunc,
			"_posix_spawnattr_setschedparam": posixSpawnAttrSetFlagsFunc,
			"_posix_spawnattr_setschedpolicy": posixSpawnAttrSetFlagsFunc,
			"_posix_spawnp": posixSpawnNpFunc,
			"_pthread_attr_destroy": pthreadAttrDestroyFunc,
			"_pthread_attr_init": pthreadAttrDestroyFunc,
			"_pthread_attr_setstacksize": pthreadAttrSetStackSizeFunc,
			"_pthread_cleanup_pop": pthreadCleanupPopFunc,
			"_pthread_cleanup_push": pthreadCleanupPushFunc,
			"_pthread_cond_destroy": pthreadAttrDestroyFunc,
			"_pthread_cond_init": pthreadCondInitFunc,
			"_pthread_cond_signal": pthreadAttrDestroyFunc,
			"_pthread_cond_timedwait": pthreadCondTimedWaitFunc,
			"_pthread_cond_wait": pthreadCondWaitFunc,
			"_pthread_condattr_init": pthreadAttrDestroyFunc,
			"_pthread_condattr_setclock": pthreadCondWaitFunc,
			"_pthread_create": pthreadCreateFunc,
			"_pthread_detach": pthreadDetachFunc,
			"_pthread_equal": pthreadEqualFunc,
			"_pthread_exit": pthreadExitFunc,
			"_pthread_join": pthreadEqualFunc,
			"_pthread_mutexattr_destroy": pthreadMutexAttrDestroyFunc,
			"_pthread_mutexattr_init": pthreadMutexAttrInitFunc,
			"_pthread_mutexattr_settype": pthreadEqualFunc,
			"_pthread_setcancelstate": pthreadEqualFunc,
			"_pthread_sigmask": pthreadSigmaskFunc,
			"_putenv": putenvFunc,
			"_pyimport_init": pyimportInitFunc,
			"_pyproxy_init": pyimportInitFunc,
			"_pyproxy_new": pyproxyNewFunc,
			"_pyproxy_use": pyproxyNewFunc,
			"_raise": pyproxyNewFunc,
			"_runpython_finalize_js": runpythonFinalizeJsFunc,
			"_runpython_init_js": runpythonFinalizeJsFunc,
			"_sched_yield": runpythonFinalizeJsFunc,
			"_setenv": setEnvFunc,
			"_setgroups": setGroupsFunc,
			"_setitimer": setEnvFunc,
			"_setpwent": setPwentFunc,
			"_sigemptyset": pyproxyNewFunc,
			"_sigfillset": pyproxyNewFunc,
			"_siginterrupt": siginterruptFunc,
			"_sigismember": siginterruptFunc,
			"_signal": siginterruptFunc,
			"_sigpending": sigPendingFunc,
			"_strftime": strftimeFunc,
			"_strftime_l": strftimeLFunc,
			"_sysconf": sysconfFunc,
			"_system": sysconfFunc,
			"_time": sysconfFunc,
			"_times": sysconfFunc,
			"_unsetenv": sysconfFunc,
			"_usleep": usleepFunc,
			"_utimes": utimesFunc,
			"_wait": usleepFunc,
			"_wait3": wait3Func,
			"_wait4": wait4Func,
			"_waitid": waitIdFunc,
			"_waitpid": wait3Func,
			"abortOnCannotGrowMemory": usleepFunc,
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
			"DYNAMICTOP_PTR": dynamicTopPtr,
			"gb": dynamicTopPtr,
			"fb": dynamicTopPtr,
			"STACKTOP": dynamicTopPtr,
			"STACK_MAX": dynamicTopPtr,
			"memory": memory,
			"table": setTempRet0,
			"tempDoublePtr": tempDoublePtr,
			"invoke_ii": invokeIiFunc,
			"invoke_iii": invokeIiiFunc,
			"invoke_iiii": invokeIiiiFunc,
			"invoke_iiiii": invokeIiiiiFunc,
			"invoke_vi": invokeViFunc,
			"invoke_vii": invokeViiFunc,
			"invoke_viiii": invokeViiiiFunc,
		},
	)

	return importObject
}
