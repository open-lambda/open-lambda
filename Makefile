PWD = $(shell pwd)

CLIENT_BIN:=worker/prof/client/client
LAMBDA_BIN=lambda/bin
REG_BIN:=registry/bin

WORKER_GO_FILES = $(shell find worker/ -name '*.go')
LAMBDA_FILES = $(shell find lambda)
POOL_FILES = $(shell find cache-entry)

GO = $(abspath ./hack/go.sh)
GO_PATH = hack/go
WORKER_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker
ADMIN_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker/admin

LAMBDA_DIR = $(abspath ./lambda)

.PHONY: all
all : .git/hooks/pre-commit imgs/lambda imgs/cache-entry bin/admin

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

bin/admin : $(WORKER_GO_FILES)
	cd $(ADMIN_DIR) && $(GO) install
	mkdir -p bin
	cp $(GO_PATH)/bin/admin ./bin

.PHONY: test test-config

test-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/worker-config.json)

# run go unit tests in initialized environment
test : test-config imgs/lambda
	cd $(WORKER_DIR) && $(GO) test ./handler -v
	cd $(WORKER_DIR) && $(GO) test ./server -v

cgroup/cgroup_init : cgroup/cgroup_init.c
	${MAKE} -C cgroup

# TODO: eventually merge this with default tests
.PHONY: cgrouptest cgrouptest-config
cgrouptest-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/worker-config-cgroup.json)

cgrouptest : cgrouptest-config imgs/lambda cgroup/cgroup_init
	cd $(WORKER_DIR) && $(GO) test ./sandbox -v

.PHONY: pooltest pooltest-config
pooltest-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/worker-config-pool.json)

# run go unit tests in initialized environment
pooltest : pooltest-config imgs/lambda imgs/cache-entry
	cd $(WORKER_DIR) && $(GO) test ./handler -v
	cd $(WORKER_DIR) && $(GO) test ./server -v

.PHONY: clean
clean :
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda imgs/cache-entry imgs/olregistry
	rm -rf testing/test_worker testing/test_pool
	rm -f cgroup/cgroup_init
	${MAKE} -C lambda clean
	${MAKE} -C cache-entry clean
