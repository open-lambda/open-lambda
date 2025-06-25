#! /bin/bash

set -e
shopt -s nullglob

REGISTRY_PATH=$1
NATIVE_PREFIX=./bin-functions/target/x86_64-unknown-linux-gnu/release

mkdir -p ${REGISTRY_PATH}

echo "Searching for functions in ${NATIVE_PREFIX}"

for f in ${NATIVE_PREFIX}/*; do
    name=$(basename "$f")

    # Ignore subdirectories, libraries, and non-executable files
    if [[ $name != *".so" && -f "$f" && -x "$f" ]]; then
        echo "Installing native function '$name.bin' from '$f' to ${REGISTRY_PATH}/${name}.bin"
        func_name="${name}"
        archive_name="${func_name}.tar.gz"
        tmp_dir=$(mktemp -d)

        echo "Packaging native function '${func_name}' → ${archive_name}"

        cp "$f" "${tmp_dir}/f.bin"
        tar -czf "${REGISTRY_PATH}/${archive_name}" -C "$tmp_dir" f.bin
        rm -r "$tmp_dir"
    fi
done
