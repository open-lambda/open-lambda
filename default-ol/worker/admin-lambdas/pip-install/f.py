
#!/usr/bin/env python
import os, sys, platform, re

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
extra = '' # TODO: support extras

def matches(markers):
    return eval(markers)

def top(dirname):
    path = None
    for name in os.listdir(dirname):
        if name.endswith('-info'):
            path = os.path.join(dirname, name, "top_level.txt")
    if path == None or not os.path.exists(path):
        return []
    with open(path) as f:
        return f.read().strip().split("\n")

def deps(dirname):
    path = None
    for name in os.listdir(dirname):
        if name.endswith('-info'):
            path = os.path.join(dirname, name, "METADATA")
    if path == None or not os.path.exists(path):
        return []

    rv = set()
    with open(path, encoding='utf-8') as f:
        for line in f:
            prefix = 'Requires-Dist: '
            if line.startswith(prefix):
                line = line[len(prefix):].strip()
                parts = line.split(';')
                if len(parts) > 1:
                    match = matches(parts[1])
                else:
                    match = True
                if match:
                    name = re.split(' \(', parts[0])[0]
                    rv.add(name)
    return list(rv)

def f(event):
    pkg = event["pkg"]
    alreadyInstalled = event["alreadyInstalled"]
    if not alreadyInstalled:
        rc = os.system('pip3 install --no-deps %s -t /host/files' % pkg)
        print('pip install returned code %d' % rc)
        assert(rc == 0)
    name = pkg.split("==")[0]
    d = deps("/host/files")
    t = top("/host/files")
    return {"Deps":d, "TopLevel":t}
