.PHONY: all

all: spin rust

rust:
	cd runtimes/rust && cargo build --release

spin: spin.c
	gcc -O2 --static -o spin spin.c

.PHONY: clean

clean:
	rm -rf spin
