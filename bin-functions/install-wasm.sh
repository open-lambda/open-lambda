#! /bin/bash

shopt -s nullglob

REGISTRY_PATH_WASM=$1
WASM_TARGET=$2

WASM_PREFIX=./bin-functions/target/${WASM_TARGET}/release/

mkdir -p ${REGISTRY_PATH_WASM}

echo "Searching for function ins ${WASM_PREFIX}"

for f in ${WASM_PREFIX}*.wasm; do
    name=${f/${WASM_PREFIX}/}
    echo "Installing wasm program '$name' from '$f' to ${REGISTRY_PATH_WASM}/${name}"
    rsync --checksum "$f" ${REGISTRY_PATH_WASM}/
done
