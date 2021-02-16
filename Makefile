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
.PHONY: wasm

all: dependencies ol imgs/lambda

wasm:
	cd wasm-programs && cargo build --release --target $(WASM_TARGET)
	cp wasm-programs/target/$(WASM_TARGET)/release/*.wasm test-registry.wasm/

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
