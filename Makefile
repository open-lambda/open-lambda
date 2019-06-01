PWD = $(shell pwd)

GO = go
OL_DIR = $(abspath ./src)
OL_GO_FILES = $(shell find src/ -name '*.go')
LAMBDA_FILES = $(shell find lambda)

.PHONY: all
.PHONY: test-all
.PHONY: clean

all: bin/ol sock/sock-init imgs/lambda

sock/sock-init: sock/sock-init.c
	${MAKE} -C sock

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

bin/ol: $(OL_GO_FILES)
	env
	cd $(OL_DIR) && $(GO) build -mod vendor -o ol
	mkdir -p bin
	cp $(OL_DIR)/ol ./bin

test-all:
	python3 test.py

clean:
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda
	rm -rf testing/test_worker
	${MAKE} -C lambda clean
	${MAKE} -C sock clean
