#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OLROOT="$( cd "$DIR/../.." && pwd)"
CLUSTER=/tmp/.oltest/$1

$OLROOT/bin/admin kill -cluster=$CLUSTER
rm -r $CLUSTER
