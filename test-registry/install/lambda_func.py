import os, sys

# ol-install: parso,jedi,idna,chardet,certifi,requests
# ol-import: parso,jedi,idna,chardet,certifi,requests,urllib3

print(sys.path)
print(os.listdir("/packages"))

import jedi
import requests

def handler(event):
    return 'imported'
