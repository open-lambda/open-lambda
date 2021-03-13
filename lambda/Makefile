.PHONY: all
.PHONY: clean
.PHONY: rust

all: spin rust

spin: spin.c
	gcc -O2 --static -o spin spin.c

clean:
	rm -rf spin
