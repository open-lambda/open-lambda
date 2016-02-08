GOPATH:=$(PWD)
WORKER:=src/github.com/tylerharter/open-lambda/lambdaManager

worker :
	GOPATH=$(GOPATH) cd $(WORKER)/server; go build
	GOPATH=$(GOPATH) cd $(WORKER)/prof/client; go build
