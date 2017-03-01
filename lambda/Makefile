.PHONY: all

all: ns.so init

ns.so: nsmodule/nsmodule.c nsmodule/setup.py
	cd nsmodule && python setup.py build_ext --inplace
	mv nsmodule/ns.so .
	cd nsmodule && export GOPATH=$(pwd) && go build fdclient.go

init: init.c
	gcc -O2 -o init init.c

.PHONY: clean

clean:
	rm -rf nsmodule/build/ ns.so
	rm -f init fdclient
