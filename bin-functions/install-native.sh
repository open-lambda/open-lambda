#! /bin/bash

shopt -s nullglob

REGISTRY_PATH=$1
NATIVE_PREFIX=./bin-functions/target/x86_64-unknown-linux-gnu/release

mkdir -p ${REGISTRY_PATH}

echo "Searching for functions in ${NATIVE_PREFIX}"

for f in ${NATIVE_PREFIX}/*; do
    name=${f/${NATIVE_PREFIX}/}

    # Ignore subdirectories, libraries, and non-executable files
    if [[ $name != *".so" && -f "$f" && -x "$f" ]]; then
        echo "Installing native function '$name.bin' from '$f' to ${REGISTRY_PATH}/${name}.bin"
        rsync -c $f ${REGISTRY_PATH}/$name.bin
    fi
done
