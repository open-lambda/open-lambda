The *NEW and IMPROVED* lambda-worker!

Finds a container, either in the registry or locally, runs it.

Lambdas must be designed to accept http requests on port 8080. The worker proxy's requests directly to this port

Will never shut down the container.

### Important notes for running

The environment variable `OL_DOCKER_HOST` must be set to the ip address of the docker daamon! I.E.
