PWD = $(shell pwd)

GO = go
OL_DIR = $(abspath ./src)
OL_GO_FILES = $(shell find src/ -name '*.go')
LAMBDA_FILES = $(shell find lambda)

.PHONY: all
.PHONY: test-all
.PHONY: clean

all: ol sock/sock-init imgs/lambda

sock/sock-init: sock/sock-init.c
	${MAKE} -C sock

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

ol: $(OL_GO_FILES)
	env
	cd $(OL_DIR) && $(GO) build -mod vendor -o ol
	mv $(OL_DIR)/ol ./ol

test-all:
	python3 -u test.py

clean:
	rm -f ol
	rm -f imgs/lambda
	${MAKE} -C lambda clean
	${MAKE} -C sock clean
