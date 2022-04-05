#! /bin/bash

shopt -s nullglob

REGISTRY_PATH_WASM=$1
WASM_TARGET=$2

WASM_PREFIX=./native-programs/target/${WASM_TARGET}/release/

mkdir -p ${REGISTRY_PATH_WASM}

echo "Searching for programs ins ${WASM_PREFIX}"

for f in ${WASM_PREFIX}*.wasm; do
    name=${f/${WASM_PREFIX}/}
    echo "Installing wasm program '$name' from '$f' to ${REGISTRY_PATH_WASM}/${name}"
    rsync --checksum "$f" ${REGISTRY_PATH_WASM}/

    type_path=./programs/${name/.wasm/}/${name/.wasm/.type}
    rsync --checksum "$type_path" ${REGISTRY_PATH_WASM}/
done
