PWD = $(shell pwd)
WASM_TARGET = wasm32-unknown-unknown

GO = go
OL_DIR = $(abspath ./src)
OL_GO_FILES = $(shell find src/ -name '*.go')
LAMBDA_FILES = $(shell find lambda)

.PHONY: install
.PHONY: test-all
.PHONY: clean
.PHONY: update-dependencies
.PHONY: wasm-programs
.PHONY: wasm-worker
.PHONY: test-dir

all: dependencies ol imgs/lambda wasm-worker wasm-programs

wasm-worker:
	cd wasm-worker && cargo build --release
	cp wasm-worker/target/release/wasm-worker ./ol-wasm

wasm-programs: imgs/lambda
	cd wasm-programs && cross build --release
	cd wasm-programs && cargo build --release --target $(WASM_TARGET)
	bash ./wasm-programs/install.sh ${WASM_TARGET}

update-dependencies:
	cd wasm-worker && cargo update
	cd wasm-programs && cargo update

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

test-dir:
	cp lambda/runtimes/rust/target/release/open-lambda-runtime ./test-registry/hello-rust.bin

ol: $(OL_GO_FILES)
	cd $(OL_DIR) && $(GO) build -o ../ol

install: ol
	cp ol /usr/local/bin

test-all:
	sudo python3 -u test.py

clean:
	rm -f ol
	rm -f imgs/lambda
	${MAKE} -C lambda clean
	${MAKE} -C sock clean
