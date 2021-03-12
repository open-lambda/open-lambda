.PHONY: all
.PHONY: clean
.PHONY: rust

all: spin rust

rust:
	cd runtimes/rust && cargo build --release

spin: spin.c
	gcc -O2 --static -o spin spin.c

clean:
	rm -rf spin
