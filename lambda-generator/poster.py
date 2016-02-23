#!/usr/bin/python
import requests
import json

r = requests.post("http://localhost:5000", data=json.dumps({}))
print r.text
