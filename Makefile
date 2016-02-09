GOPATH:=$(PWD)
WORKER:=lambdaManager

worker :
	cd hack && ./build.sh
	mkdir -p bin
	cp lambdaManager/server/server bin/lambdaWorker
	cp lambdaManager/prof/client/client bin/client

clean :
	rm -rf bin
	rm -rf hack/.gopath

