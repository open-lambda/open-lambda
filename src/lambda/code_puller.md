# Code Puller

An OL worker maintains a `lambda_code` directory within the worker
directory.  `lambda_code` containers unpacked directories with lambda
code that may be mapped into lambda instance sandboxes.

`lambda_code` is populated by a CodePuller (`codePuller.go`), from
either local files or web resources.

## Configuration

The worker JSON config file contains two fields related to CodePuller:

* `registry`: this is a local path or URL prefix, and is contatenated with a lambda name (and perhaps a suffix) to get a location for the resource to pull
* `registry_cache_ms`: this is how long lambda code in `lambda_code` can be used without checking its staleness

## Formats

**Remote:** If `registry` starts with "http://" or "https://", then
lambdas may be represented as (1) a standalone .py file (with any
name) or (2) a .tar.gz (which must contain a f.py file may
contain other files).  If `registry` is `http://localhost:5000` and
the worker is trying to pull a lambda named `runme`, then the
CodePuller will first try to download the code (via a GET HTTP
request) from `http://localhost:5000/runme.tar.gz`.  If that doesn't
exist (i.e., a 404 is returned), the worker will next try
`http://localhost:5000/runme.py`.  If that doesn't exist either, the
pull fails.

**Local:** If `registry` is a path to a directory in the local file
  system, then lambdas may be represented as a (1) a standalone .py
  file, (2) a .tar.gz containing a f.py file, or (3) a
  directory containing a f.py file.  If `registry` is
  `/var/local/registry` and a lambda named `runme` is being pulled,
  the worker will check for these resources (in this order):

1. file named /var/local/registry/runme.tar.gz
2. file named /var/local/registry/runme.py
3. dir named /var/local/registry/runme

Remote mode is layered over local mode (i.e., the file is fetched,
then local unpacking is used).

## Caching

OL reuses lambda code when possible rather than re-pulling every time.
If the time since the last pull for a given lambda function is less
than `registry_cache_ms`, OL uses the code without checking whether it
is current.

If more time has elapsed, OL checks whether there is a newer version.
If `registry` is local, the CodePuller checks the timestamp of the .py
or .tar.gz file to see if it as changed.  If the lambda was
represented as a local directory, OL does NOT check the timestamp of
each file in that directory, and it is recursively re-copied every
time (thus, this format is only recommended for debugging).

If `registry` is a URL prefix, CodePuller sends a request for the
latest code.  However, it also passes a `If-Modified-Since` header
based on the `Last-Modified` header of the previous request.  If HTTP
server hosting the code understands this header, it will return a 403
status (`http.StatusNotModified`) instead of returning all the file
data, and the CodePuller will know that previously-cached directory
containing the code is still fresh.

## Known Issues

* local directory format: there is no caching supported here, so it's probably slow.  Avoid it if it matters.
* garbage collection: if lambdas change over time, the previously downloaded code becomes obsolete.  Currently, we never delete this, and we don't take care to make sure all handlers running older versions are killed.
* authentication: currently, CodePuller only works with HTTP servers that publicly host the lambdas.  In the future, we should support an HTTP-based access key (https://en.wikipedia.org/wiki/Basic_access_authentication).
