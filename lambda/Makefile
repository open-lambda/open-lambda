.PHONY: all

all: spin ol-init

spin: spin.c
	gcc -Wall -O2 spin.c -o spin

ol-init: ol-init.c
	gcc -Wall -O2 ol-init.c -o ol-init

.PHONY: clean

clean:
	rm -f ol-init spin
