PWD=$(shell pwd)
WASM_TARGET=wasm32-unknown-unknown
CARGO=cargo +nightly
GO=go
OL_DIR=$(abspath ./src)
OL_GO_FILES=$(shell find src/ -name '*.go')
LAMBDA_FILES=$(shell find lambda)
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
.PHONY: wasm-programs
.PHONY: wasm-worker
.PHONY: native-programs
.PHONY: test-dir
.PHONY: check-runtime
.PHONY: db-proxy

all: ol imgs/lambda wasm-worker wasm-programs native-programs

wasm-worker:
	cd wasm-worker && ${CARGO} build ${BUILD_FLAGS} ${WASM_WORKER_FLAGS}
	cp wasm-worker/target/${BUILDTYPE}/wasm-worker ./ol-wasm

wasm-programs:
	cd programs && ${CARGO} build --release --target $(WASM_TARGET)
	bash ./programs/install-wasm.sh test-registry.wasm ${WASM_TARGET}

native-programs: imgs/lambda
	cd programs && cross build --release
	bash ./programs/install-native.sh test-registry

update-dependencies:
	cd lambda/runtimes/rust && ${CARGO} update
	cd wasm-worker && ${CARGO} update
	cd programs && ${CARGO} update
	cd db-proxy && ${CARGO} update

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

check-runtime:
	cd lambda/runtimes/rust && ${CARGO} check

db-proxy:
	cd db-proxy && ${CARGO} build ${BUILD_FLAGS}
	cp ./db-proxy/target/${BUILDTYPE}/db-proxy ./ol-database-proxy

test-dir:
	cp lambda/runtimes/rust/target/release/open-lambda-runtime ./test-registry/hello-rust.bin

ol: $(OL_GO_FILES)
	cd $(OL_DIR) && $(GO) build -o ../ol

install: ol
	cp ol /usr/local/bin

test-all:
	sudo python3 -u ./scripts/test.py

clean:
	rm -f ol
	rm -f imgs/lambda
	${MAKE} -C lambda clean
	${MAKE} -C sock clean
