import requests
import sys
import json
import os
import time
from subprocess import call


def child():
	time.sleep(5)
	payload = json.dumps({"name":"Neha"})
	r = requests.post("http://172.17.0.1:8081/runLambda/hello", data=payload)
        #fg = open("hello_output", 'wb')	
	#fg.write(str(r.status_code))	
	#fg.write(str(r.content))
	#fg.close()
	os._exit(0)	

def handler(conn, event):
#def handler():
    try:
	newpid = os.fork()
	if newpid == 0:
	   child()
	else:
	  f = open("x", 'wb')
	  g = open("y", 'wb')
	  call(["perf","record", "-e", "syscalls:sys_*", "-e", "net:*", "-e", "skb:*" ,"-e", "sock:*" ,"-e", "cpu-clock","-F", "99","--output=perf_ver4.data", "-a", "-g", "-p", str(newpid) ], stdout=f, stderr=g)
	  #call(["perf","record", "-e", "syscalls:sys_*", "-e", "net:*", "-e", "skb:*" ,"-e", "sock:*" ,"-e", "cpu-clock","-F", "99","--output=perf_ver2.data", "-a", "-g", "-s", "sleep", "10" ], stdout=f, stderr=g)
	  done = os.waitpid(newpid,0)
	  f.write("child pid" + str(newpid))
	  f.write(str(done))
	  perf_output = open("request_perf_output", 'wb')
	  call(["perf", "script", "--input=perf_ver4.data"], stdout=perf_output, stderr=g)
	  perf_output.close()
	  #pid, sts = os.waitpid(newpid)
	  f.close()
	  g.close()
	  f = open("x", 'rb')
          g = open("y", 'rb')
	  return str(f.read()) + str(g.read()) + "How are you, %s!" % event['name']
	  #return "How are you, %s!" % event['name']
	  #os._exit(0)
    except Exception as e:
        return {'error': str(e)}

#def main():
#	handler()
#
#if __name__== "__main__":
#  main()
