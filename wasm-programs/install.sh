#! /bin/bash

shopt -s nullglob

WASM_PREFIX=./wasm-programs/target/$1/release/
NATIVE_PREFIX=./wasm-programs/target/release/

mkdir -p test-registry

for f in ${WASM_PREFIX}*.wasm; do
	name=${f/${WASM_PREFIX}/}
	echo "Installing '$name' from '$f'"
	cp "$f" test-registry.wasm/

	name=${name/.wasm/}
	f2=${NATIVE_PREFIX}/$name
	echo "Installing '$name.bin' from '$f2'"
	cp $f2 test-registry/rust-$name.bin
done
