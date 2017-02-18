.PHONY: all, clean

all: nsmodule.c setup.py
	python setup.py build_ext --inplace

clean:
	rm -rf build/ ns.so 
