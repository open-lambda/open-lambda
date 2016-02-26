GOPATH:=$(PWD)
WORKER:=worker
WORKER_SRC:=$(cd worker $$ ls *.go)

SERVER_BIN:=worker/worker
CLIENT_BIN:=worker/prof/client/client

worker : $(WORKER_SRC)
	cd hack && ./build.sh
	mkdir -p bin
	cp $(SERVER_BIN) bin/worker
	cp $(CLIENT_BIN) bin/client

clean :
	rm -rf bin
	rm -rf hack/.gopath
	rm $(SERVER_BIN)
	rm $(CLIENT_BIN)

