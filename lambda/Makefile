.PHONY: all

all: ol.so spin

ol.so: setup.py ol.c
	python3 setup.py build_ext --inplace
	mv ol.*.so ./ol.so

spin: spin.c
	gcc -O2 -o spin spin.c

.PHONY: clean

clean:
	rm -rf spin ol.so
