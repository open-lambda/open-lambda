# Base Directory

By default, WORKER_DIR/lambda will contain a base image file system
for every OpenLambda container (it depends on how you initialize the
worker, but this directory generally contains all the files for an
Ubuntu install, other dependencies, and some OpenLambda-specific
code).  WORKER_DIR/lambda will be read only inside the container but writable
outside.

When calling "./ol worker init", you can specify what Docker image you
want to use for the base.  It is `ol-wasm` by default (for a full
deployment with Python and WASM support).  To save space/time, you
might use `ol-min` if you just need Python.  Or, you could extend this
if there are resources needed by all your lambdas.

The `initOLBaseDir` function in the
github.com/open-lambda/open-lambda/ol/worker package initializes the
"lambda" directory.  In addition dumping the Docker image (which takes
99% of the time), there are some other things that get created:

* /host directory (for scratch files) -- the only writeable directory inside a SOCK container
* /handler (for code specific to the lambda instance being run)
* /packages (a read-only view of ALL packages used by any SOCK container).  Some of these may be malicious since we don't trust PyPI.  That's OK as long a lambda instance doesn't import packages the programmer didn't ask for.  [Read more here](pypi-packages.md).
* resolve.conf (specifying the DNS server as 8.8.8.8)
* /dev/(null,random,urandom) -- otherwise many packages requiring randomness don't work

## Future Work

There might be more things that need to be added to the base over
time.  For example, we didn't originally have /dev/random -- adding
that enabled more containers to work.  The most notable thing missing
is a procfs at /proc.  It's not clear how often there are good
(non-malicuous) use cases for this.  One thing we should think about
is /proc/cpuinfo.  Do packages use this (or something similar) to
decide how many threads to launch?  And if so, should it be the real
number of CPUs, or based on the resources allocated to the container?