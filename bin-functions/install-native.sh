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
        echo "Installing native function '$name.tar.gz' from '$f' to ${REGISTRY_PATH}/${name}.tar.gz"
        
        # Create temporary directory for tar.gz creation
        temp_dir=$(mktemp -d)
        
        # Copy binary to temp directory as f.bin
        cp "$f" "$temp_dir/f.bin"
        
        # Create tar.gz file containing f.bin
        tar -czf "${REGISTRY_PATH}/${name}.tar.gz" -C "$temp_dir" f.bin
        
        # Clean up temporary directory
        rm -rf "$temp_dir"
    fi
done
