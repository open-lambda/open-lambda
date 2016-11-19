PWD = $(shell pwd)

CLIENT_BIN:=worker/prof/client/client
NODE_BIN:=node/bin
LAMBDA_BIN=lambda/bin
REG_BIN:=registry/bin

WORKER_GO_FILES = $(shell find worker/ -name '*.go')
REG_GO_FILES = $(shell find registry/ -name '*.go') 
TEST_CLUSTER:=test_cluster

GO = $(abspath ./hack/go.sh)
GO_PATH = hack/go
UTIL_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/util
WORKER_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker
REG_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/registry
ADMIN_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker/admin

.PHONY: all
all : .git/hooks/pre-commit bin/regpush imgs/olregistry imgs/lambda-node
	docker pull eoakes/lambda:latest

.git/hooks/pre-commit: util/pre-commit
	cp util/pre-commit .git/hooks/pre-commit

# OL worker container, with OL server, Docker, and RethinkDB
imgs/lambda-node : bin/worker node/Dockerfile node/startup.py node/kill.py node/lambda/Dockerfile
	mkdir -p $(NODE_BIN)
	cp bin/worker $(NODE_BIN)/worker
	docker build -t lambda-node node
	touch imgs/lambda-node

imgs/olregistry : bin/pushserver registry/Dockerfile registry/pushserver.go
	mkdir -p $(REG_BIN)
	cp bin/pushserver $(REG_BIN)/pushserver
	docker build -t olregistry registry
	touch imgs/olregistry

# OL server
bin/worker : $(WORKER_GO_FILES)
	cd $(WORKER_DIR) && $(GO) get
	cd $(WORKER_DIR) && $(GO) install
	mkdir -p bin
	cp $(GO_PATH)/bin/worker ./bin

bin/admin : $(WORKER_GO_FILES)
	cd $(ADMIN_DIR) && $(GO) get
	cd $(ADMIN_DIR) && $(GO) install
	mkdir -p bin
	cp $(GO_PATH)/bin/admin ./bin

# OL registry server
bin/pushserver : $(REG_GO_FILES)
	cd $(REG_DIR) && $(GO) get -tags 'pushserver'
	cd $(REG_DIR) && $(GO) build pushserver.go
	mkdir -p registry/bin
	cp $(REG_DIR)/pushserver ./bin

bin/regpush : registry/regpush.go
	cd $(REG_DIR) && $(GO) get -tags 'regpush'
	cd $(REG_DIR) && $(GO) build regpush.go
	mkdir -p bin
	cp $(REG_DIR)/regpush ./bin

.PHONY: test test-cluster

# create cluster for testing
test-cluster : imgs/lambda-node
	./util/stop-cluster.py -c $(TEST_CLUSTER) --if-running --force
	./util/start-cluster.py -c $(TEST_CLUSTER) --skip-db-wait

# run go unit tests in initialized environment
test : test-cluster
	$(eval export WORKER_CONFIG := $(PWD)/testing/worker-config.json) ./testing/setup.py --cluster=$(TEST_CLUSTER)
	cd $(WORKER_DIR) && $(GO) test ./server ./handler -v
	./util/stop-cluster.py -c $(TEST_CLUSTER)

.PHONY: clean
clean :
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda-node
	rm -f imgs/olregistry
