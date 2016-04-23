from __future__ import print_function

import boto3
import json
import math

def lambda_handler(event, context):
    dynamo = boto3.resource('dynamodb').Table(event['tableName'])
    column = event['column']
    
    request = dynamo.scan(AttributesToGet=[column])
    
    # Undefined for < 2 values
    if request['Count'] < 2:
        return None
    
    # Get average
    total = 0
    for item in request['Items']:
        total += item[column]
    
    avg = total/request['Count']
    
    # Sum of squared deviations
    dev_sq = 0
    for item in request['Items']:
        dev_sq += (item[column] - avg)**2

    # Return sample stdev
    return math.sqrt(dev_sq/(request['Count']-1))
