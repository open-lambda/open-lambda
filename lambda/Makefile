.PHONY: all

all: ns.so spin

ns.so: nsmodule/nsmodule.c nsmodule/setup.py 
	cd nsmodule && python3 setup.py build_ext --inplace
	mv nsmodule/ns.*.so ./ns.so

spin: spin.c
	gcc -O2 -o spin spin.c

.PHONY: clean

clean:
	rm -rf nsmodule/build ol-init spin fdclient ns.*.so
