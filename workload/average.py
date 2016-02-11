from __future__ import print_function

import boto3
import json

def lambda_handler(event, context):
    dynamo = boto3.resource('dynamodb').Table(event['tableName'])
    column = event['column']
    
    request = dynamo.scan(AttributesToGet=[column])
    
    # Undefined for no values
    if request['Count'] is 0:
        return None
    
    total = 0
    for item in request['Items']:
        total += item[column]
    
    return total/request['Count']
