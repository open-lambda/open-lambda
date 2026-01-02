import os

SCRATCH_DIR = "/host/tmp"

# Set cache directories to writable scratch dir BEFORE importing pip-tools
os.environ["PIP_TOOLS_CACHE_DIR"] = SCRATCH_DIR
os.environ["XDG_CACHE_HOME"] = SCRATCH_DIR
os.environ["HOME"] = SCRATCH_DIR

from flask import Flask, request, Response
from piptools.scripts.compile import cli
from click.testing import CliRunner

app = Flask(__name__)


@app.route("/", methods=["POST"])
def compile_requirements():
    """
    Compile requirements.in to requirements.txt using pip-compile.

    Input (POST body): requirements.in content
    Output: compiled requirements.txt content (plain text)
    """
    requirements_in = request.get_data(as_text=True)

    if not requirements_in:
        return Response("No requirements provided", status=400, mimetype="text/plain")

    in_path = os.path.join(SCRATCH_DIR, "requirements.in")
    out_path = os.path.join(SCRATCH_DIR, "requirements.txt")

    with open(in_path, "w") as f:
        f.write(requirements_in)

    runner = CliRunner()
    result = runner.invoke(cli, [
        "--index-url", "https://pypi.org/simple/",
        "--output-file", out_path,
        in_path
    ])

    if result.exit_code != 0:
        return Response(
            result.output or str(result.exception),
            status=400,
            mimetype="text/plain"
        )

    with open(out_path, "r") as f:
        return Response(f.read(), mimetype="text/plain")
