#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OLROOT="$( cd "$DIR/../../.." && pwd)"
ADMIN=$OLROOT/bin/admin
CLUSTER=/tmp/.oltest/sock

$ADMIN new -cluster $CLUSTER || exit 1
cp $DIR/template.json $CLUSTER/config/template.json || exit 1
$ADMIN sock-container -cluster $CLUSTER
$ADMIN workers -cluster $CLUSTER || exit 1

# wait for cluster creation
sleep 1
