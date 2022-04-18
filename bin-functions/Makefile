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
	cargo build --target=${WASM_TARGET} --release

native-functions:
	cargo build --release
