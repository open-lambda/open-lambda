import os

SCRATCH_DIR = "/host/tmp"

# Set cache directories to writable scratch dir BEFORE importing pip-tools
os.environ["PIP_TOOLS_CACHE_DIR"] = SCRATCH_DIR
os.environ["XDG_CACHE_HOME"] = SCRATCH_DIR
os.environ["HOME"] = SCRATCH_DIR

from piptools.scripts.compile import cli
from click.testing import CliRunner


def f(event):
    """
    Compile requirements.in to requirements.txt using pip-compile.

    Input: requirements.in content (string or dict with 'requirements' key)
    Output: compiled requirements.txt content (string)
    """
    # Handle different input formats
    if isinstance(event, dict):
        requirements_in = event.get('requirements', event.get('body', ''))
    else:
        requirements_in = str(event) if event else ''

    if not requirements_in:
        return {"error": "No requirements provided"}

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
        return {"error": result.output or str(result.exception)}

    with open(out_path, "r") as f:
        return f.read()
