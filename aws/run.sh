#!/bin/bash

keys=1
depth=1
iterations=100

# zero size
aws lambda invoke --function-name server --payload "{\"num_keys\":0, \"depth\":0, \"value_len\":0, \"iterations\":$iterations}" results/result_0_0_0_${iterations}.out


for (( length=64; length<2000000; length*=4 )) 
do 
    # echo $length;
    aws lambda invoke --function-name server --payload "{\"num_keys\":$keys, \"depth\":$depth, \"value_len\":$length, \"iterations\":$iterations}" results/result_${keys}_${depth}_${length}_${iterations}.out
done
