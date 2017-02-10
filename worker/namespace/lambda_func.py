import os, datetime

def handler(conn, event):
    os.system(event['arg'])
    return str(datetime.datetime.today())
