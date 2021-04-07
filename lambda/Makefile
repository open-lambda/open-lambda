.PHONY: all
.PHONY: clean

all: spin

spin: spin.c
	gcc -O2 --static -o spin spin.c

clean:
	rm -rf spin
