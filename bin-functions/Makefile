# This uses two different target dirs, one for native and one for wasm, to avoid recompiles

.PHONY: lint cargo-clean update-dependencies cargo-clean udeps wasm-functions native-functions

CLIPPY_FLAGS=-- --deny "warnings"
WASM_TARGET=wasm32-unknown-unknown

all: wasm-functions native-functions

lint:
	cargo clippy ${CLIPPY_FLAGS}
	cargo clippy --target=${WASM_TARGET} ${CLIPPY_FLAGS}

update-dependencies:
	cargo update

cargo-clean:
	cargo clean

udeps:
	cargo udeps
	cargo udeps --target=${WASM_TARGET}

wasm-functions:
	env CARGO_TARGET_DIR=wasm-target cargo build --target=${WASM_TARGET} --release

native-functions:
	cargo build --release
