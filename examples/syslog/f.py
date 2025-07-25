import subprocess
from ctypes import *

def f(event):
	libc = CDLL("/lib/x86_64-linux-gnu/libc.so.6")
	return libc.syscall(103, 0, "")
