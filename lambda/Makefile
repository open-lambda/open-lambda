.PHONY: all

all: spin ol-init ns.so

spin: spin.c
	gcc -O2 -o spin spin.c

ol-init: ol-init.c
	gcc -Wall -O2 ol-init.c -o ol-init

ns.so: nsmodule/nsmodule.c nsmodule/setup.py 
	cd nsmodule && python3 setup.py build_ext --inplace
	mv nsmodule/ns.*.so ./ns.so

# for testing only
fdclient: nsmodule/fdclient.go
	cd nsmodule && export GOPATH=$(pwd) && go build fdclient.go
	mv nsmodule/fdclient .

.PHONY: clean

clean:
	rm -rf nsmodule/build ol-init spin fdclient ns.*.so
