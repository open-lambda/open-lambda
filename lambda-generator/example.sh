#!/bin/bash

# Create container to run function defined in hello.py
sudo ./builder.py -n pylambda:hello -l hello.py
sudo docker run -dp 5000:8080 --name hello_lambda pylambda:hello ./server.py

# Should print "Hello, world!"
./poster.py

# Clean up (output hidden to better show output from above)
sudo docker rm -f hello_lambda > /dev/null
sudo docker rmi pylambda:hello > /dev/null
