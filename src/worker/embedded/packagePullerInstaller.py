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
    packages_path = '/packages/'
    if packages_path not in sys.path:
        sys.path.append(packages_path)

    pkg = event["pkg"]
    alreadyInstalled = event["alreadyInstalled"]
    pip_mirror = event.get("pip_mirror", "")
    if not alreadyInstalled:
        try:
            if pip_mirror == "":
                subprocess.check_output(
                    ['pip3', 'install', '--no-deps', pkg, '--cache-dir', '/tmp/.cache', '-t', '/host/files'])
            else:
                pip_mirror = pip_mirror.rstrip('/') + '/simple/' # make sure it ends with / and has simple at the end
                host_start_index = pip_mirror.find('://') + 3
                host_end_index = pip_mirror.find('/', host_start_index)
                mirror_host = pip_mirror[host_start_index:host_end_index]
                cmds = ['pip3', 'install', '--no-deps', pkg, '--cache-dir', '/tmp/.cache', '-t', '/host/files', f"--trusted-host={mirror_host}", f"--index-url={pip_mirror}",  "-vvv"]
                print(f"[packaagePullerInstaller.py] attempting install with command: {cmds}")
                out = subprocess.check_output(cmds)
                print(f"[packagePullerInstaller.py] Install output: {out}")
        except subprocess.CalledProcessError as e:
            print(f'[packagePullerInstaller.py] pip install failed with error code {e.returncode}')
            print(f'[packagePullerInstaller.py] Output: {e.output}')

    name = pkg.split("==")[0]
    d = deps("/host/files")
    t = top("/host/files")
    return {"Deps": d, "TopLevel": t}
