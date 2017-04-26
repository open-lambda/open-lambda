#!/bin/bash

if [ $# -ne 4 ]; then
    echo "run.sh <keys> <depth> <length> <iterations>"
    exit
fi


keys=$1
depth=$2
length=$3
iterations=$4

out="results/result_${keys}_${depth}_${length}_${iterations}.out"

aws lambda invoke --function-name server --payload "{\"num_keys\":$keys, \"depth\":$depth, \"value_len\":$length, \"iterations\":$iterations}" $out > /dev/null
echo
cat $out
echo 
echo

exit
