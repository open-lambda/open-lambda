import os
import urllib.request
import urllib.error

SCRATCH_DIR = "/host/tmp"

# Set cache/temp directories to writable location BEFORE importing pip-tools
os.environ["HOME"] = SCRATCH_DIR
os.environ["TMPDIR"] = SCRATCH_DIR
os.environ["XDG_CACHE_HOME"] = SCRATCH_DIR
os.environ["PIP_CACHE_DIR"] = SCRATCH_DIR

from flask import Flask, request, Response
from piptools.scripts.compile import cli
from click.testing import CliRunner

app = Flask(__name__)


def do_compile(requirements_in, quiet=True):
    """Compile requirements.in content to requirements.txt."""
    if not requirements_in:
        return Response("No requirements provided", status=400, mimetype="text/plain")

    in_path = os.path.join(SCRATCH_DIR, "requirements.in")
    out_path = os.path.join(SCRATCH_DIR, "requirements.txt")

    with open(in_path, "w") as f:
        f.write(requirements_in)

    args = [
        "--output-file", out_path,
        "--pip-args", "--only-binary=:all:",
    ]
    if quiet:
        args.extend(["--no-header", "--no-annotate"])
    args.append(in_path)

    runner = CliRunner()
    result = runner.invoke(cli, args)

    if result.exit_code != 0:
        return Response(
            result.output or str(result.exception),
            status=400,
            mimetype="text/plain"
        )

    with open(out_path, "r") as f:
        lines = [l for l in f if not l.startswith("--")]
        return Response("".join(lines), mimetype="text/plain")


@app.route("/", methods=["GET"])
def docs():
    """Return documentation with curl examples."""
    return Response("""pip-compile Lambda Service
==========================

Compiles requirements.in files to pinned requirements.txt using pip-compile.

Endpoints
---------

POST /text
    Pass requirements.in content directly in the request body.

    curl -X POST -d $'flask>=2.0\\nrequests' http://localhost:5000/run/pip-compile/text

POST /url
    Pass a URL to fetch requirements.in from.

    curl -X POST -d 'https://example.com/requirements.in' http://localhost:5000/run/pip-compile/url
""", mimetype="text/plain")


@app.route("/text", methods=["POST"])
def compile_from_text():
    """Compile requirements.in from POST body text."""
    quiet = request.args.get("quiet", "1") == "1"
    return do_compile(request.get_data(as_text=True), quiet=quiet)


@app.route("/url", methods=["POST"])
def compile_from_url():
    """Fetch requirements.in from a URL and compile it."""
    url = request.get_data(as_text=True).strip()
    quiet = request.args.get("quiet", "1") == "1"

    if not url:
        return Response("No URL provided", status=400, mimetype="text/plain")

    try:
        with urllib.request.urlopen(url, timeout=30) as response:
            requirements_in = response.read().decode('utf-8')
    except urllib.error.URLError as e:
        return Response(f"Failed to fetch URL: {e}", status=400, mimetype="text/plain")

    return do_compile(requirements_in, quiet=quiet)
