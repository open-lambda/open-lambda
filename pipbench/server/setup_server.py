import os


os.system('gcc -shared -I/usr/include/python2.7 -lpython2.7  load_simulator.c -o load_simulator.so')
