WORKER:=worker
WORKER_SRC:=worker/*.go

SERVER_BIN:=worker/worker
CLIENT_BIN:=worker/prof/client/client
NODE_BIN:=node/bin
GO_FILES = $(shell find worker/ -name '*.go')
TEST_CLUSTER:=test_cluster

GO = $(abspath ./hack/go.sh)
GO_PATH = hack/go
WORKER_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker

.PHONY: all
all : .git/hooks/pre-commit imgs/lambda-node

.git/hooks/pre-commit: util/pre-commit
	cp util/pre-commit .git/hooks/pre-commit

# OL worker container, with OL server, Docker, and RethinkDB
imgs/lambda-node : bin/worker node/Dockerfile node/startup.py node/kill.py
	mkdir -p node/bin
	cp bin/worker node/bin/worker
	docker build -t lambda-node node
	touch imgs/lambda-node

# OL server
bin/worker : $(GO_FILES)
	cd $(WORKER_DIR) && $(GO) get
	cd $(WORKER_DIR) && $(GO) install
	mkdir -p bin
	cp $(GO_PATH)/bin/worker ./bin

.PHONY: test test-cluster

# create cluster for testing
test-cluster : imgs/lambda-node
	./util/stop-local-cluster.py -c $(TEST_CLUSTER) --if-running --force
	./util/start-local-cluster.py -c $(TEST_CLUSTER) --skip-db-wait

# run go unit tests in initialized environment
test : test-cluster
	$(eval export TEST_REGISTRY := localhost:$(shell jq -r '.host_port' ./util/$(TEST_CLUSTER)/registry.json))
	./testing/setup.py
	cd $(WORKER_DIR) && $(GO) get
	cd $(WORKER_DIR) && $(GO) test . ./handler -v
	./testing/pychat.py
	./util/stop-local-cluster.py -c $(TEST_CLUSTER)

.PHONY: clean
clean :
	rm -rf bin
	rm -f imgs/lambda-node
