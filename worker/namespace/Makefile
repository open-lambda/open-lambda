all: nsmodule.c setup.py
	python setup.py build_ext --inplace

test: all test.py
	python test.py

clean:
	rm -rf build/ ns.so 
