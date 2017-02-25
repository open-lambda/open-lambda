.PHONY: all

all: ns.so init

ns.so: nsmodule.c setup.py
	python setup.py build_ext --inplace

init: init.c
	gcc -O2 -o init init.c

.PHONY: clean

clean:
	rm -rf build/ ns.so
	rm -f init
