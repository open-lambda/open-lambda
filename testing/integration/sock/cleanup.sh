#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OLROOT="$( cd "$DIR/../../.." && pwd)"
ADMIN=$OLROOT/bin/admin
CLUSTER=/tmp/.oltest/sock

$ADMIN kill -cluster=$CLUSTER || exit 1
# wait for unmount
sleep 1
rm -r $CLUSTER || exit 1
