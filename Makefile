PWD = $(shell pwd)

CLIENT_BIN:=worker/prof/client/client
LAMBDA_BIN=lambda/bin
REG_BIN:=registry/bin

WORKER_GO_FILES = $(shell find worker/ -name '*.go')

GO = $(abspath ./hack/go.sh)
GO_PATH = hack/go
WORKER_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker
ADMIN_DIR = $(GO_PATH)/src/github.com/open-lambda/open-lambda/worker/admin

.PHONY: all
all : .git/hooks/pre-commit imgs/lambda bin/admin

.git/hooks/pre-commit: util/pre-commit
	cp util/pre-commit .git/hooks/pre-commit

imgs/lambda : lambda/Dockerfile lambda/server.py
	docker build lambda -t lambda
	touch imgs/lambda

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

.PHONY: clean
clean :
	rm -rf bin
	rm -rf registry/bin
	rm -f imgs/lambda
	rm -f imgs/olregistry
