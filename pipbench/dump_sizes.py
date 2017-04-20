import json

with open('new-graph-100000.json') as fd:
    spec = json.load(fd)

with open('package_sizes.txt', 'w') as fd:
    for pkg in spec:
        fd.write('%s:%s\n' % (pkg['name'], pkg['uncompressed']))
