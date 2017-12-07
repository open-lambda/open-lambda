#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OLROOT="$( cd "$DIR/../../.." && pwd)"
ADMIN=$OLROOT/bin/admin
CLUSTER=/tmp/.oltest/sock-pkg-cache
CACHE_PKGS="jedi requests simplejson"

$ADMIN new -cluster $CLUSTER || exit 1
cp $DIR/template.json $CLUSTER/config/template.json || exit 1
for pkg in $CACHE_PKGS; do
    pip install -t $CLUSTER/packages/$pkg $pkg 2>&1 || exit 1
done
$ADMIN sock-container -cluster $CLUSTER || exit 1
$ADMIN workers -cluster $CLUSTER || exit 1

# wait for cluster creation
sleep 2
