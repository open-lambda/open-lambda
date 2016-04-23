import boto3
import json

# Simple least squares regression
def lambda_handler(event, context):
    dynamo = boto3.resource('dynamodb').Table('test')
    request = dynamo.scan(AttributesToGet=['values'])
    
    xarr = request["Items"][0]["values"]["x"]
    yarr = request["Items"][0]["values"]["y"]
    
    x_bar = 0
    y_bar = 0
    xy_bar = 0
    xx_bar = 0
    
    for x, y in zip(xarr, yarr):
        x_bar += x
        y_bar += y
        xy_bar += x*y
        xx_bar += x*x
    
    beta = (xy_bar - (x_bar*y_bar/len(xarr)))/(xx_bar-((x_bar**2)/len(xarr)))
    alpha = (y_bar - beta*x_bar)/len(xarr)
    
    return {"Beta":beta, "Alpha":alpha}
