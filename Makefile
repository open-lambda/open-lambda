PWD=$(shell pwd)
WASM_TARGET=wasm32-unknown-unknown
GO=go
OL_DIR=$(abspath ./src)
OL_GO_FILES=$(shell find src/ -name '*.go')
LAMBDA_FILES = lambda/Dockerfile lambda/Makefile lambda/spin.c lambda/runtimes/python/server.py lambda/runtimes/python/setup.py lambda/runtimes/python/ol.c
USE_LLVM?=1
BUILDTYPE?=debug
INSTALL_PREFIX?=/usr/local

ifeq (${USE_LLVM}, 1)
	WASM_WORKER_FLAGS=--features=llvm-backend
else
	WASM_WORKER_FLAGS=
endif

ifeq (${BUILDTYPE}, release)
	BUILD_FLAGS=--release
else
	BUILD_FLAGS=
endif

.PHONY: install
.PHONY: test-all
.PHONY: clean
.PHONY: update-dependencies
.PHONY: wasm-functions
.PHONY: wasm-worker
.PHONY: native-functions
.PHONY: check-runtime
.PHONY: container-proxy
.PHONY: fmt check-fmt

all: ol imgs/lambda wasm-worker wasm-functions native-functions

wasm-worker:
	cd wasm-worker && cargo build ${BUILD_FLAGS} ${WASM_WORKER_FLAGS}
	cp wasm-worker/target/${BUILDTYPE}/wasm-worker ./ol-wasm

wasm-functions:
	cd bin-functions && ${MAKE} wasm-functions
	bash ./bin-functions/install-wasm.sh test-registry.wasm ${WASM_TARGET}
	ls test-registry.wasm/hashing.wasm test-registry.wasm/noop.wasm

native-functions: imgs/lambda
	cd bin-functions && cross build --release
	bash ./bin-functions/install-native.sh test-registry
	ls test-registry/hashing.bin test-registry/noop.bin # guarantee they were created

update-dependencies:
	cd lambda/runtimes/native && cargo update
	cd wasm-worker && cargo update
	cd bin-functions && cargo update
	cd container-proxy && cargo update

imgs/lambda: ${LAMBDA_FILES}
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

install-python-bindings:
	cd scripts && python setup.py install

check-runtime:
	cd lambda/runtimes/rust && cargo check

container-proxy:
	cd container-proxy && cargo build ${BUILD_FLAGS}
	cp ./container-proxy/target/${BUILDTYPE}/open-lambda-container-proxy ./ol-container-proxy

ol: ${OL_GO_FILES}
	cd ${OL_DIR} && ${GO} build -o ../ol

install: ol wasm-worker
	cp ol ${INSTALL_PREFIX}/bin/
	cp ol-wasm ${INSTALL_PREFIX}/bin/

test-all:
	sudo python3 -u ./scripts/test.py --worker_type=sock
	sudo python3 -u ./scripts/test.py --worker_type=docker --test_filter=ping_test,numpy
	sudo python3 -u ./scripts/sock_test.py
	sudo python3 -u ./scripts/bin_test.py --worker_type=wasm
	sudo python3 -u ./scripts/bin_test.py --worker_type=sock

fmt:
	#cd src && go fmt ...
	cd wasm-worker && cargo fmt
	cd bin-functions && cargo fmt

check-fmt:
	cd wasm-worker && cargo fmt --check
	cd bin-functions && cargo fmt --check

lint-go:
	revive -exclude src/vendor/... -config golint.toml src/...

lint: #go-lint
	pylint scripts --ignore=build --disable=missing-docstring,multiple-imports,global-statement,invalid-name,W0511,W1510,R0801,W3101
	cd wasm-worker && cargo clippy
	cd bin-functions && cargo clippy

clean:
	rm -f ol
	rm -f imgs/lambda
	${MAKE} -C lambda clean
	${MAKE} -C sock clean
