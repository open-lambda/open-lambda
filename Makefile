PWD = $(shell pwd)

LAMBDA_BIN=lambda/bin

OL_GO_FILES = $(shell find ol/ -name '*.go')
LAMBDA_FILES = $(shell find lambda)
POOL_FILES = $(shell find cache-entry)

TEST_CLUSTER=testing/test-cluster
KILL_WORKER=./bin/ol kill -cluster=$(TEST_CLUSTER);rm -rf $(TEST_CLUSTER)/workers/*
RUN_LAMBDA=curl -XPOST localhost:8080/runLambda

STARTUP_PKGS='{"startup_pkgs": ["parso", "jedi", "urllib3", "idna", "chardet", "certifi", "requests", "simplejson"]}'
REGISTRY_DIR='{"registry": "$(abspath testing/registry)"}'

SOCK_NOCACHE='{"sandbox": "sock", "handler_cache_size": 0, "import_cache_size": 0, "cg_pool_size": 10}'
SOCK_HANDLER='{"sandbox": "sock", "handler_cache_size": 10000000, "import_cache_size": 0, "cg_pool_size": 10}'
SOCK_IMPORT='{"sandbox": "sock", "handler_cache_size": 0, "import_cache_size": 10000000, "cg_pool_size": 10}'
SOCK_BOTH='{"sandbox": "sock", "handler_cache_size": 10000000, "import_cache_size": 10000000, "cg_pool_size": 10}'

DOCKER_NOCACHE='{"sandbox": "docker", "handler_cache_size": 0, "import_cache_size": 0, "cg_pool_size": 0}'
DOCKER_HANDLER='{"sandbox": "docker", "handler_cache_size": 10000000, "import_cache_size": 0, "cg_pool_size": 0}'
DOCKER_IMPORT='{"sandbox": "docker", "handler_cache_size": 0, "import_cache_size": 10000000, "cg_pool_size": 0}'
DOCKER_BOTH='{"sandbox": "docker", "handler_cache_size": 10000000, "import_cache_size": 10000000, "cg_pool_size": 0}'

WORKER_TIMEOUT=60

define RUN_TEST=
	@echo "Killing worker if running..."
	-$(KILL_WORKER)
	@echo
	@echo "Starting worker..."
	./bin/ol setconf -cluster=$(TEST_CLUSTER) CONDITION
	./bin/ol workers -cluster=$(TEST_CLUSTER)
	@echo
	@echo "Waiting for worker to initialize..."
	@for i in $$(seq 1 $(WORKER_TIMEOUT)); \
	do \
		[ $$i -gt 1 ] && sleep 2; \
		./bin/ol status -cluster=$(TEST_CLUSTER) 1>/dev/null && s=0 && break || s=$$?; \
	done; ([ $$s -eq 0 ] || (echo "Worker failed to initialize after $(WORKER_TIMEOUT)s" && exit 1))
	@echo "Worker ready. Requesting lambdas..."
	$(RUN_LAMBDA)/echo -d '{}'
	@echo
	$(RUN_LAMBDA)/install -d '{}'
	@echo
	$(RUN_LAMBDA)/install2 -d '{}'
	@echo
	$(RUN_LAMBDA)/install3 -d '{}'
	@echo
	@echo
endef

GO = go
OL_DIR = $(abspath ./ol)

LAMBDA_DIR = $(abspath ./lambda)
PIPBENCH_DIR = $(abspath ./pipbench)

.PHONY: all
all: bin/ol clean-test sock/sock-init imgs/lambda

sock/sock-init: sock/sock-init.c
	${MAKE} -C sock

imgs/lambda: $(LAMBDA_FILES)
	${MAKE} -C lambda
	docker build -t lambda lambda
	touch imgs/lambda

bin/ol: $(OL_GO_FILES)
	cd $(OL_DIR) && $(GO) build -mod vendor
	mkdir -p bin
	cp $(OL_DIR)/ol ./bin

.PHONY: test-all test-sock-all test-docker-all test-cluster

test-all: test-sock-all test-docker-all

test-sock-all: test-sock-nocache test-sock-handler test-sock-import test-sock-both

test-docker-all: test-docker-nocache test-docker-handler

test-cluster: imgs/test-cluster

imgs/test-cluster: 
	@echo "Starting test cluster..."
	./bin/ol new -cluster=$(TEST_CLUSTER)
	./bin/ol setconf -cluster=$(TEST_CLUSTER) $(REGISTRY_DIR)
	./bin/ol setconf -cluster=$(TEST_CLUSTER) $(STARTUP_PKGS)
	@echo
	touch imgs/test-cluster

clean-test:
	@echo "Killing worker if running..."
	-$(KILL_WORKER)
	@echo
	@echo "Cleaning up test cluster..."
	rm -rf $(TEST_CLUSTER) imgs/test-cluster
	@echo

test-sock-nocache: bin/ol imgs/lambda test-cluster
	$(subst CONDITION, $(SOCK_NOCACHE), $(RUN_TEST))

test-sock-handler: bin/ol imgs/lambda test-cluster
	$(subst CONDITION, $(SOCK_HANDLER), $(RUN_TEST))

test-sock-import: bin/ol imgs/lambda test-cluster
	$(subst CONDITION, $(SOCK_IMPORT), $(RUN_TEST))

test-sock-both: bin/ol imgs/lambda test-cluster
	$(subst CONDITION, $(SOCK_BOTH), $(RUN_TEST))

test-docker-nocache: bin/ol imgs/lambda test-cluster
	$(subst CONDITION, $(DOCKER_NOCACHE), $(RUN_TEST))

test-docker-handler: bin/ol imgs/lambda test-cluster
	$(subst CONDITION, $(DOCKER_HANDLER), $(RUN_TEST))

test-docker-import: bin/ol imgs/lambda test-cluster
	$(subst CONDITION, $(DOCKER_IMPORT), $(RUN_TEST))

test-docker-both: bin/ol imgs/lambda test-cluster
	$(subst CONDITION, $(DOCKER_BOTH), $(RUN_TEST))

.PHONY: clean
clean: clean-test
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda
	rm -rf testing/test_worker
	${MAKE} -C lambda clean
	${MAKE} -C sock clean

FORCE:
