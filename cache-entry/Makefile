.PHONY: all

all: init ns.so

init: init.c
	gcc -O2 -o init init.c

ns.so: nsmodule/nsmodule.c nsmodule/setup.py 
	cd nsmodule && python setup.py build_ext --inplace
	mv nsmodule/ns.so .

# for testing only
fdclient: nsmodule/fdclient.go
	cd nsmodule && export GOPATH=$(pwd) && go build fdclient.go
	mv nsmodule/fdclient .

.PHONY: clean

clean:
	rm -rf nsmodule/build
	rm -f fdclient ns.so
