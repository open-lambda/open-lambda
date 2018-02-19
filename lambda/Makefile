.PHONY: all

all: spin sock-init ns.so

spin: spin.c
	gcc -O2 -o spin spin.c

sock-init: sock-init.c
	gcc -Wall -O2 sock-init.c -o sock-init

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
	rm -f sock-init spin fdclient ns.so
