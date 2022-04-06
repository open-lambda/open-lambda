PWD=$(shell pwd)
WASM_TARGET=wasm32-unknown-unknown
CARGO=cargo +nightly
GO=go
OL_DIR=$(abspath ./src)
OL_GO_FILES=$(shell find src/ -name '*.go')
LAMBDA_FILES = lambda/Dockerfile lambda/Makefile lambda/spin.c lambda/runtimes/python/server.py lambda/runtimes/python/setup.py lambda/runtimes/python/ol.c
USE_LLVM?=1
BUILDTYPE?=debug

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
.PHONY: test-dir
.PHONY: check-runtime
.PHONY: container-proxy

all: ol imgs/lambda wasm-worker wasm-functions native-functions

wasm-worker:
	cd wasm-worker && ${CARGO} build ${BUILD_FLAGS} ${WASM_WORKER_FLAGS}
	cp wasm-worker/target/${BUILDTYPE}/wasm-worker ./ol-wasm

wasm-functions:
	cd bin-functions && make wasm-functions
	bash ./bin-functions/install-wasm.sh test-registry.wasm ${WASM_TARGET}

native-functions: imgs/lambda
	cd bin-functions && cross build --release
	bash ./bin-functions/install-native.sh test-registry

update-dependencies:
	cd lambda/runtimes/rust && ${CARGO} update
	cd wasm-worker && ${CARGO} update
	cd bin-functions && ${CARGO} update
	cd container-proxy && ${CARGO} update

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	sudo docker build -t lambda lambda
	touch imgs/lambda

install-python-bindings:
	cd scripts && python setup.py install

check-runtime:
	cd lambda/runtimes/rust && ${CARGO} check

container-proxy:
	cd container-proxy && ${CARGO} build ${BUILD_FLAGS}
	cp ./container-proxy/target/${BUILDTYPE}/open-lambda-container-proxy ./ol-container-proxy

test-dir:
	cp lambda/runtimes/rust/target/release/open-lambda-runtime ./test-registry/hello-rust.bin

ol: $(OL_GO_FILES)
	cd $(OL_DIR) && $(GO) build -o ../ol

install: ol
	cp ol /usr/local/bin

test-all:
	sudo python3 -u ./scripts/test.py

fmt:
	cd src && go fmt ...
	cd wasm-worker && cargo fmt
	cd bin-functions && cargo fmt

lint:
	cd wasm-worker && cargo clippy
	cd bin-functions && cargo clippy

clean:
	rm -f ol
	rm -f imgs/lambda
	${MAKE} -C lambda clean
	${MAKE} -C sock clean
