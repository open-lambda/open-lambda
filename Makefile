GOPATH:=$(PWD)
WORKER:=lambdaManager

SERVER_BIN:=lambdaManager/lambdaManager
CLIENT_BIN:=lambdaManager/prof/client/client
worker :
	cd hack && ./build.sh
	mkdir -p bin
	cp $(SERVER_BIN) bin/lambdaWorker
	cp $(CLIENT_BIN) bin/client

clean :
	rm -rf bin
	rm -rf hack/.gopath
	rm $(SERVER_BIN)
	rm $(CLIENT_BIN)

