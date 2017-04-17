import requests
#import sys
import json
from subprocess import call

def handler(conn, event):
    try:
        call(["perf","record", "-e", "cpu-clock","-F", "99", "--output=perf.data", "-a", "-g", "--",
              "curl", "-X", "POST", "http://172.17.0.1:8081/runLambda/hello", "-d", "'{}'" ])
	f = open("f", 'wb')
	g = open("g", 'wb')
        call(["perf", "script", "--input=perf.data"], stdout=f, stderr=g)
        f.close()
        g.close()
        f = open("f", "rb")	
        g = open("g", "rb")	
    	return f.read() + g.read()
    except Exception as e:
        return {'error': str(e)}
