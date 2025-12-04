PWD=$(shell pwd)
WASM_TARGET=wasm32-unknown-unknown
GO=go
OL_DIR=$(abspath ./go)
OL_GO_FILES=$(shell find go/ -name '*.go')
LAMBDA_FILES = min-image/Dockerfile min-image/Makefile min-image/spin.c min-image/runtimes/python/server.py min-image/runtimes/python/setup.py min-image/runtimes/python/ol.c
BUILDTYPE?=debug
INSTALL_PREFIX?=/usr/local

# Detect host architecture for Linux
ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
    RUST_TARGET := x86_64-unknown-linux-gnu
else ifeq ($(ARCH),aarch64)
    RUST_TARGET := aarch64-unknown-linux-gnu
else
    $(error Unsupported architecture: $(ARCH))
endif

ifeq (${BUILDTYPE}, release)
	BUILD_FLAGS=--release
else
	BUILD_FLAGS=
endif

.PHONY: all
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

all: ol imgs/ol-wasm wasm-worker wasm-functions native-functions container-proxy

wasm-worker:
	cd wasm-worker && cargo build ${BUILD_FLAGS}
	cp wasm-worker/target/${BUILDTYPE}/wasm-worker ./ol-wasm

wasm-functions:
	cd bin-functions && ${MAKE} wasm-functions
	bash ./bin-functions/install-wasm.sh test-registry.wasm ${WASM_TARGET}
	ls test-registry.wasm/hashing.wasm test-registry.wasm/noop.wasm

native-functions: imgs/ol-wasm
	cd bin-functions && cargo build --release --target $(RUST_TARGET)
	bash ./bin-functions/install-native.sh registry $(RUST_TARGET)
	ls registry/hashing.tar.gz registry/noop.tar.gz # guarantee they were created

update-dependencies:
	cd wasm-image/runtimes/native && cargo update
	cd wasm-worker && cargo update
	cd bin-functions && cargo update
	cd container-proxy && cargo update

imgs/ol-min: ${LAMBDA_FILES}
	${MAKE} -C min-image
	docker build -t ol-min min-image
	touch imgs/ol-min

imgs/ol-wasm: imgs/ol-min wasm-image/runtimes/native/src/main.rs
	docker build -t ol-wasm wasm-image
	touch imgs/ol-wasm

install-python-bindings:
	cd python && pip install .

check-runtime:
	cd lambda/runtimes/rust && cargo check

container-proxy:
	cd container-proxy && cargo build ${BUILD_FLAGS}
	cp ./container-proxy/target/${BUILDTYPE}/open-lambda-container-proxy ./ol-container-proxy

ol: ${OL_GO_FILES}
	cd ${OL_DIR} && ${GO} build -o ../ol

build: ol wasm-worker container-proxy

install: build
	cp ol ${INSTALL_PREFIX}/bin/
	cp ol-wasm ${INSTALL_PREFIX}/bin/
	cp ol-container-proxy ${INSTALL_PREFIX}/bin/
	cp autocomplete/bash_autocomplete /etc/bash_completion.d/ol 

sudo-install: build
	sudo cp ol ${INSTALL_PREFIX}/bin/
	sudo cp ol-wasm ${INSTALL_PREFIX}/bin/
	sudo cp ol-container-proxy ${INSTALL_PREFIX}/bin/
	sudo cp autocomplete/bash_autocomplete /etc/bash_completion.d/ol 

test-all:
	sudo python3 -u ./scripts/test.py --worker_type=sock
	sudo python3 -u ./scripts/test.py --worker_type=docker --test_blocklist=max_mem_alloc
	sudo python3 -u ./scripts/sock_test.py
	sudo python3 -u ./scripts/bin_test.py --worker_type=wasm
	sudo python3 -u ./scripts/bin_test.py --worker_type=sock

fmt:
	#cd go && go fmt ...
	cd wasm-worker && cargo fmt
	cd bin-functions && cargo fmt
	cd container-proxy && cargo fmt

check-fmt:
	cd wasm-worker && cargo fmt --check
	cd bin-functions && cargo fmt --check
	cd container-proxy && cargo fmt --check

lint-go:
	revive -exclude go/vendor/... -config golint.toml go/...

lint-python:
	pylint scripts --ignore=build --disable=missing-docstring,multiple-imports,global-statement,invalid-name,W0511,W1510,R0801,W3101,broad-exception-raised

lint-functions:
	cd bin-functions && make lint
	cd container-proxy && cargo clippy

lint-wasm-worker:
	cd wasm-worker && cargo clippy

lint: lint-wasm-worker lint-functions lint-python lint-go

clean:
	rm -f ol imgs/ol-min imgs/ol-wasm
	${MAKE} -C lambda clean
	${MAKE} -C sock clean
