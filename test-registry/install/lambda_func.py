import os, sys

print(sys.path)
print(os.listdir("/packages"))

import jedi
import requests

def handler(event):
    return 'imported'
