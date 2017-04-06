import os

os.system('kill `lsof -t -i:9198` > /dev/null 2>&1')
os.system('kill `lsof -t -i:9199` > /dev/null 2>&1')