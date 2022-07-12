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
.PHONY: check-runtime
.PHONY: container-proxy
.PHONY: fmt check-fmt

all: ol imgs/lambda wasm-worker wasm-functions native-functions

wasm-worker:
	cd wasm-worker && ${CARGO} build ${BUILD_FLAGS} ${WASM_WORKER_FLAGS}
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
	cd lambda/runtimes/native && ${CARGO} update
	cd wasm-worker && ${CARGO} update
	cd bin-functions && ${CARGO} update
	cd container-proxy && ${CARGO} update

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

install-python-bindings:
	cd scripts && python setup.py install

check-runtime:
	cd lambda/runtimes/rust && ${CARGO} check

container-proxy:
	cd container-proxy && ${CARGO} build ${BUILD_FLAGS}
	cp ./container-proxy/target/${BUILDTYPE}/open-lambda-container-proxy ./ol-container-proxy

ol: $(OL_GO_FILES)
	cd $(OL_DIR) && $(GO) build -o ../ol

install: ol
	cp ol /usr/local/bin

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

lint:
	cd wasm-worker && cargo clippy
	cd bin-functions && cargo clippy
	pylint scripts --ignore build

clean:
	rm -f ol
	rm -f imgs/lambda
	${MAKE} -C lambda clean
	${MAKE} -C sock clean
