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
all : .git/hooks/pre-commit bin/regpush imgs/olregistry imgs/lambda

.git/hooks/pre-commit: util/pre-commit
	cp util/pre-commit .git/hooks/pre-commit

# OL worker container, with OL server, Docker, and RethinkDB
imgs/lambda-node : bin/worker node/Dockerfile node/startup.py node/kill.py node/lambda/Dockerfile
	mkdir -p $(NODE_BIN)
	cp bin/worker $(NODE_BIN)/worker
	docker build -t lambda-node node
	touch imgs/lambda-node

imgs/lambda :
	docker pull eoakes/lambda:latest
	touch imgs/lambda

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

.PHONY: test test-config

test-config :
	$(eval export WORKER_CONFIG := $(PWD)/testing/worker-config.json)

# run go unit tests in initialized environment
test : test-config imgs/lambda
	cd $(WORKER_DIR) && $(GO) test ./handler -v
	cd $(WORKER_DIR) && $(GO) test ./server -v

.PHONY: clean
clean :
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda-node
	rm -f imgs/olregistry
