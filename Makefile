WORKER:=worker
WORKER_SRC:=worker/*.go

SERVER_BIN:=worker/worker
CLIENT_BIN:=worker/prof/client/client
NODE_BIN:=node/bin
GO_FILES = $(shell find worker/ -name '*.go')
TEST_CLUSTER:=test_cluster

.PHONY: all
all : imgs/lambda-node

bin/worker : $(GO_FILES)
	cd hack && ./build.sh
	mkdir -p bin
	cp $(SERVER_BIN) bin/worker
	cp $(CLIENT_BIN) bin/client

imgs/lambda-node : bin/worker node/Dockerfile node/startup.py node/kill.py
	mkdir -p node/bin
	cp bin/worker node/bin/worker
	docker build -t lambda-node node
	touch imgs/lambda-node

clean :
	rm -rf bin
	rm -rf hack/.gopath
	rm -rf $(NODE_BIN)
	rm $(SERVER_BIN)
	rm $(CLIENT_BIN)

.PHONY: test test-cluster

test-cluster :
	./util/stop-local-cluster.py -c $(TEST_CLUSTER) --if-running
	./util/start-local-cluster.py -c $(TEST_CLUSTER) --skip-db-wait

test : test-cluster
	$(eval export TEST_REGISTRY := localhost:$(shell jq -r '.host_port' ./util/$(TEST_CLUSTER)/registry.json))
	./testing/setup.py
#	cd hack && ./build.sh test . ./handler -v # test
	cd hack && ./build.sh test . -v           # test1
	./util/stop-local-cluster.py -c $(TEST_CLUSTER)
