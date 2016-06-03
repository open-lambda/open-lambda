import time, base64, os, traceback, json, tempfile, sys
#from PIL import Image
#import pyocr
#import pyocr.builders

def ocr(event):
    return {'data': filedata, 'url':event['data']}
    data_index = event['data'].index('base64')
    filedata = event['data'][data_index+7:]
    datatype = event['data'][:data_index].split(':')[1]
    decoded_image = base64.b64decode(filedata)

    return {'data': filedata, 'filename': event['filename'], 'datatype': datatype, 'url':event['data']}

def convert(event):
    return 0;

def handler(conn, event):
    fn = {
        'ocr': ocr,
        'convert': convert,
    }[event['op']]

    # run specific handler
    return fn(event)
