PWD = $(shell pwd)

LAMBDA_BIN=lambda/bin
REG_BIN:=registry/bin

WORKER_GO_FILES = $(shell find worker/ -name '*.go')
LAMBDA_FILES = $(shell find lambda)
POOL_FILES = $(shell find cache-entry)
PIP_FILES = $(shell find pip-installer)

GO = $(abspath ./hack/go.sh)
GO_PATH = hack/go
WORKER_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker
ADMIN_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker/admin

LAMBDA_DIR = $(abspath ./lambda)
PIPBENCH_DIR = $(abspath ./pipbench)

.PHONY: all
all : .git/hooks/pre-commit sock/sock-init imgs/lambda imgs/pip-installer bin/admin

.git/hooks/pre-commit: util/pre-commit
	cp util/pre-commit .git/hooks/pre-commit

sock/sock-init: sock/sock-init.c
	${MAKE} -C sock

imgs/lambda : $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

imgs/pip-installer : $(PIP_FILES)
	docker build -t pip-installer pip-installer
	touch imgs/pip-installer

bin/admin : $(WORKER_GO_FILES)
	cd $(ADMIN_DIR) && $(GO) install
	mkdir -p bin
	cp $(GO_PATH)/bin/admin ./bin

.PHONY: test test-config

test-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/configs/worker-config.json)

# run go unit tests in initialized environment
test : test-config imgs/lambda
	#cd $(WORKER_DIR) && $(GO) test ./handler -v
	cd $(WORKER_DIR) && $(GO) test ./server -v

.PHONY: cachetest cachetest-config
cachetest-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/configs/worker-config-cache.json)

# run go unit tests in initialized environment
cachetest : cachetest-config imgs/lambda imgs/cache-entry
	#cd $(WORKER_DIR) && $(GO) test ./handler -v
	cd $(WORKER_DIR) && $(GO) test ./server -v

.PHONY: clean
clean :
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda
	rm -rf testing/test_worker testing/test_cache
	${MAKE} -C lambda clean
