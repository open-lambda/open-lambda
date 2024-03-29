# Package Management

## Specifying Dependencies

You can specify lambda package requirements in a `requirements.txt`
file.  OpenLambda requires that you:

1. specify exact package versions
2. specify both direct and indirect packages (for example, pandas depends on numpy, but you must explicitly include numpy in your requirements.txt file)


The easiest way to do this is with `pip-compile`.  You can get this
tool with `pip install pip-tools`.

Then, you specify just direct dependencies (with or without versions)
in a `requirements.in` file.  For example, you might have this:

```
numpy==1.24
pandas==1.5
```

Then, run `pip-compile requirements.in` to generate a
`requirements.txt` file that looks like this (note that you might get
different versions):

```
#
# This file is autogenerated by pip-compile with Python 3.10
# by the following command:
#
#    pip-compile requirements.in
#
numpy==1.24.0
    # via
    #   -r requirements.in
    #   pandas
pandas==1.5.0
    # via -r requirements.in
python-dateutil==2.8.2
    # via pandas
pytz==2023.3
    # via pandas
six==1.16.0
    # via python-dateutil
```

## Try It

Start an OpenLambda worker (if not already started).  For example, you
could run this (to use the `pkg-demo` as the worker directory and run
the worker in the background):

```
ol worker worker -p pkg-demo -d
```

Create/edit "pkg-demo/registry/scraper/f.py" (or similar if you're using
a different worker directory).  Paste the following:

```python
import requests

def f(event):
    r = requests.get(event)
    r.raise_for_status()
    return r.text[:500] + "... (from requests version %s)" % requests.__version__
```

In "pkg-demo/registry/scraper/requirements.in", paste this:

```
requests==2.20
```

Run `pip-compile requirements.in` in the "scraper" directory.

Try invoking your lambda:

```
curl -X POST localhost:5000/run/scraper -d '"https://raw.githubusercontent.com/open-lambda/open-lambda/main/README.md"'
```

The first time this will be slow the first time.  If you watch the
tail of the log file (`tail -f pkg-demo/worker.out`), you'll see it is
running several pip install commands.  Subsequent calls should be
fast.

## Exploring the Implementation

### Package Directory

The [base directory](base-image.md) at "pkg-demo/lambda" contains the
root file system used by all lambda instances; sharing is OK because
they have read-only access.  "pkg-demo/lambda/packages" contains
installs of PyPI packages.  Different versions installs of the
packages will go in different directories under "packages", so getting
the versions needed by a particular lambda is accomplished by
configuring `sys.path` (see below).

If you `ls pkg-demo/lambda/packages`, you'll see several directories
here (certifi, chardet, etc.) -- the lambda you wrote explicitly
depends on `requests`, which in turn has its own dependencies
(as determined by `pip-compile`).

Now run this:

```
ls "pkg-demo/lambda/packages/requests==2.20/files"/
```

You'll see a couple directories:
* requests
* requests-2.20.0.dist-info

PyPI packages often involve multiple modules/resources.  If you had
pip installed `requests` in a regular environment, both these modules
would have been created for you in "/usr/lib/python3/dist-packages"
(or similar) instead of this "files" directory.

### `sys.path`

To understand how package versions are selected, modify your
`scraper.py` to just return the `sys.path`, without doing anything more:

```python
import requests, sys

def f(event):
    return sys.path
```

Invoke it -- you should see something like this:

```
["/packages/certifi/files", "/packages/idna/files", "/packages/chardet/files", "/packages/urllib3/files", "/packages/requests==2.20/files", "/runtimes/python", "/lib/python310.zip", "/lib/python3.10", "/lib/python3.10/lib-dynload", "/lib/python3/dist-packages", "/usr/local/lib/python3.10/dist-packages", "/handler"]
```

Notice the "/packages/requests==2.20/files" entry -- the lambda
instance runs in a chroot'd environment, so this is EXACTLY the same
directory as "pkg-demo/lambda/packages/requests==2.20/files" as seen
from the host.

If you look at "pkg-demo/worker/scratch/", you'll see the writable
scratch directories for the SOCK containers.  You'll notice multiple
"????-scraper" directories since the code has changed and new
directories have been created.  `ls` inside the one with the biggest
number, and you'll find a "bootstrap.py" file.  If you look inside
that, you'll see the modifications to `sys.path`.  Take a look at
`SOCKPool.Create` in the code to see where this file is created for
each new sandbox.

### Untrusted Install (`PackagePuller`)

We assume PyPI contains may be malicious.  This means that we need to
install packages in containers, and we shouldn't do multiples installs
in the same container (even in the case of implicit dependencies).
OpenLambda does not currently limit the time for an install, so a
malicious package could still tie up resources indefinitely even
though it is containerized (this is future work).  We don't limit the
space it can consume either.

The `PackagePuller` in
github.com/open-lambda/open-lambda/ol/worker/lambda is responsible for
doing pip installs in sandboxes and recursively finding dependencies.

Normally, "/host" (from inside a SOCK container) maps to a directory
inside "worker/scratch" (on the host).  For install containers,
"/host" maps to a package-specific directory under
"pkg-demo/lambda/packages" so that pip can install the package in a
directory visible to all lambda instances (but cannot interfere with
other files).

`PackagePuller` uses an install like this to install in "packages" on the host:

```
pip3 install --no-deps PACKAGE_NAME --cache-dir /tmp/.cache -t /host/files
```

The `--no-deps` flag means it will not attempt to install implicit
dependencies.  Instead, the `PackagePuller` will inspect the
`METADATA` file for dependencies (looking for lines like
`Requires-Dist:`).  These will be passed back so the PackagePuller can
install the implicit dependencies individually in other containers.
Note that the `PackagePuller` parsing of `METADATA` is not as
sophisticated as `pip` itself -- it probably misses some dependencies
in more complicated cases (like platform specific dependencies or
conditional dependencies in general).
