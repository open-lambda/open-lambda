PWD = $(shell pwd)

LAMBDA_BIN=lambda/bin

WORKER_GO_FILES = $(shell find worker/ -name '*.go')
LAMBDA_FILES = $(shell find lambda)
POOL_FILES = $(shell find cache-entry)

TEST_CLUSTER=testing/test-cluster
KILL_WORKER=./bin/admin kill -cluster=$(TEST_CLUSTER);rm -rf $(TEST_CLUSTER)/workers/*

GO = $(abspath ./hack/go.sh)
GO_PATH = hack/go
ADMIN_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker/admin

LAMBDA_DIR = $(abspath ./lambda)
PIPBENCH_DIR = $(abspath ./pipbench)

.PHONY: all
all: clean-test .git/hooks/pre-commit sock/sock-init imgs/lambda bin/admin

.git/hooks/pre-commit: util/pre-commit
	cp util/pre-commit .git/hooks/pre-commit

sock/sock-init: sock/sock-init.c
	${MAKE} -C sock

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

bin/admin: $(WORKER_GO_FILES)
	cd $(ADMIN_DIR) && $(GO) install
	mkdir -p bin
	cp $(GO_PATH)/bin/admin ./bin

.PHONY: test-all test-sock-all test-docker-all test-cluster

test-all: bin/admin imgs/lambda	
	python -m unittest discover testing/integration-tests/ -p "*_test.py"

test-sock-all: bin/admin imgs/lambda
	python testing/integration-tests/sock_test.py

test-docker-all: bin/admin imgs/lambda
	python testing/integration-tests/docker_test.py


clean-test:
	@echo "Killing worker if running..."
	-$(KILL_WORKER)
	@echo
	@echo "Cleaning up test cluster..."
	rm -rf $(TEST_CLUSTER) imgs/test-cluster
	@echo

.PHONY: clean
clean: clean-test
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda
	rm -rf testing/test_worker
	${MAKE} -C lambda clean

FORCE:
