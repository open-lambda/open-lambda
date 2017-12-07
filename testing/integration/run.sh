#!/bin/bash

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
OLROOT="$( cd "$DIR/../.." && pwd)"
ADMIN=$OLROOT/bin/admin

# setup directories and resources
mkdir -p /tmp/.oltest

run_lambda() {
    HANDLER=$1
    JSON=$2
    ASSERT=$3

    time out=$(curl --silent -XPOST localhost:8080/runLambda/$HANDLER -d "$JSON")
    if [ "$out" != "$ASSERT" ]; then
        echo "expect $ASSERT but found $out"
        exit 1
    fi
}

run_worker() {
    CLUSTER=/tmp/.oltest/$1

    echo "*** setting up cluster $CLUSTER ***"
    $DIR/$1/setup.sh >> $DIR/$1.log || exit 1

    cp -r $OLROOT/testing/handlers/hello $CLUSTER/handlers/hello
    cp -r $OLROOT/testing/handlers/hello2 $CLUSTER/handlers/hello2
    echo "*** run lambda 'hello' ***"
    run_lambda hello '{}' '"hello"'
    echo "*** run lambda 'hello2' ***"
    run_lambda hello2 '{}' '"hello"'

    cp -r $OLROOT/testing/handlers/install $CLUSTER/handlers/install
    cp -r $OLROOT/testing/handlers/install2 $CLUSTER/handlers/install2
    cp -r $OLROOT/testing/handlers/install3 $CLUSTER/handlers/install3
    echo "*** run lambda 'install' ***"
    run_lambda install '{}' '"imported"'
    echo "*** run lambda 'install2' ***"
    run_lambda install2 '{}' '"imported"'
    echo "*** run lambda 'install3' ***"
    run_lambda install3 '{}' '"imported"'
}

if [ "$#" -eq 0 ]; then
    # all subdirectories
    tests="$DIR/*/"
else
    tests="$@"
fi

for test in $tests; do
    cluster=$(basename "$test")
    echo "$test $cluster"
    run_worker "$cluster"
    $cluster/cleanup.sh >> $DIR/$1.log || exit 1
done
