#!/usr/bin/env python
import os, sys, platform, re
import subprocess
import pkgutil

import pkg_resources
from pkg_resources import parse_requirements


def format_full_version(info):
    version = '{0.major}.{0.minor}.{0.micro}'.format(info)
    kind = info.releaselevel
    if kind != 'final':
        version += kind[0] + str(info.serial)
    return version


# as specified here: https://www.python.org/dev/peps/pep-0508/#environment-markers
os_name = os.name
sys_platform = sys.platform
platform_machine = platform.machine()
platform_python_implementation = platform.python_implementation()
platform_release = platform.release()
platform_system = platform.system()
platform_version = platform.version()
python_version = platform.python_version()[:3]
python_full_version = platform.python_version()
implementation_name = sys.implementation.name
if hasattr(sys, 'implementation'):
    implementation_version = format_full_version(sys.implementation.version)
else:
    implementation_version = "0"


# top_level.txt cannot be trusted, use pkgutil to get top level packages
def top(dirname):
    return [name for _, name, _ in pkgutil.iter_modules([dirname])]


def deps(dirname):
    path = None
    for name in os.listdir(dirname):
        if name.endswith('-info'):
            path = os.path.join(dirname, name, "METADATA")
    if path == None or not os.path.exists(path):
        return []

    rv = set()
    with open(path, encoding='utf-8') as f:
        metadata = f.read()

    dist_lines = [line for line in metadata.splitlines() if line.startswith("Requires-Dist: ")]
    dependencies = "\n".join(line[len("Requires-Dist: "):] for line in dist_lines)

    for dependency in parse_requirements(dependencies):
        try:
            if dependency.marker is None or (dependency.marker is not None and dependency.marker.evaluate()):
                rv.add(dependency.project_name)
        # TODO: 'extra' would causes UndefinedEnvironmentName, simply ignore it for now
        #  except "extra", is there anything else cause UndefinedEnvironmentName?
        except pkg_resources.extern.packaging.markers.UndefinedEnvironmentName:
            continue
    return list(rv)


def f(event):
    pkg = event["pkg"]
    alreadyInstalled = event["alreadyInstalled"]
    if not alreadyInstalled:
        try:
            subprocess.check_output(
                ['pip3', 'install', '--no-deps', pkg, '--cache-dir', '/tmp/.cache', '-t', '/host/files'])
        except subprocess.CalledProcessError as e:
            print(f'pip install failed with error code {e.returncode}')
            print(f'Output: {e.output}')

    name = pkg.split("==")[0]
    d = deps("/host/files")
    t = top("/host/files")
    return {"Deps": d, "TopLevel": t}
