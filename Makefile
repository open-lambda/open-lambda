PWD = $(shell pwd)

CLIENT_BIN:=worker/prof/client/client
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
all : .git/hooks/pre-commit imgs/lambda imgs/cache-entry imgs/pip-installer bin/admin sock/sock-init

.git/hooks/pre-commit: util/pre-commit
	cp util/pre-commit .git/hooks/pre-commit

imgs/lambda : $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

imgs/cache-entry : $(POOL_FILES)
	${MAKE} -C cache-entry
	docker build -t cache-entry cache-entry
	touch imgs/cache-entry

imgs/pip-installer : $(PIP_FILES)
	docker build -t pip-installer pip-installer
	touch imgs/pip-installer

bin/admin : $(WORKER_GO_FILES)
	cd $(ADMIN_DIR) && $(GO) install
	mkdir -p bin
	cp $(GO_PATH)/bin/admin ./bin

.PHONY: test test-config

test-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/worker-config.json)

# run go unit tests in initialized environment
test : test-config imgs/lambda
	#cd $(WORKER_DIR) && $(GO) test ./handler -v
	cd $(WORKER_DIR) && $(GO) test ./server -v

sock/sock-init : sock/sock-init.c
	${MAKE} -C sock

# TODO: eventually merge this with default tests
.PHONY: socktest socktest-config
socktest-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/worker-config-sock.json)

socktest : socktest-config imgs/lambda sock/sock-init
	mkdir -p /tmp/olpkgs
	cd $(WORKER_DIR) && $(GO) test -tags socktest ./sandbox/ -v

.PHONY: cachetest cachetest-config
cachetest-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/worker-config-cache.json)

# run go unit tests in initialized environment
cachetest : cachetest-config imgs/lambda imgs/cache-entry
	#cd $(WORKER_DIR) && $(GO) test ./handler -v
	cd $(WORKER_DIR) && $(GO) test ./server -v

.PHONY: clean
clean :
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda imgs/cache-entry imgs/olregistry
	rm -rf testing/test_worker testing/test_cache
	rm -f sock/sock-init
	${MAKE} -C lambda clean
	${MAKE} -C cache-entry clean

