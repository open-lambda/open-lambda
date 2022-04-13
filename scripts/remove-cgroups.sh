#! /bin/bash

# USE WITH CAUTION!

DIR_NAME=$1

for name in `ls /sys/fs/cgroup/${DIR_NAME}-sandboxes | grep cg-`; do
    rmdir /sys/fs/cgroup/${DIR_NAME}-sandboxes/$name
done

rmdir /sys/fs/cgroup/${DIR_NAME}-sandboxes
