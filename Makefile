PWD = $(shell pwd)
WASM_TARGET = wasm32-unknown-unknown

GO = go
OL_DIR = $(abspath ./src)
OL_GO_FILES = $(shell find src/ -name '*.go')
LAMBDA_FILES = $(shell find lambda)

.PHONY: all
.PHONY: install
.PHONY: test-all
.PHONY: clean
.PHONY: dependencies
.PHONY: wasm-programs
.PHONY: wasm-worker

all: dependencies ol imgs/lambda

wasm-worker:
	cd wasm-worker && cargo build --release
	cp wasm-worker/target/release/wasm-worker ./ol-wasm

wasm-programs:
	cd wasm-programs && cargo build --release --target $(wasm_target)
	cp wasm-programs/target/$(wasm_target)/release/*.wasm test-registry.wasm/

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	sudo docker build -t lambda lambda
	touch imgs/lambda

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
