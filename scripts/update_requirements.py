#! /bin/env python3

'''
Re-runs pip-compile to bump minor versions of packages easily
'''

import os

from os import path
from subprocess import check_call

REGISTRY_DIR = "./test-registry"

for entry in os.scandir(REGISTRY_DIR):
    if not entry.is_dir():
        continue

    folder = entry.path
    if not path.isfile(f'{folder}/requirements.in'):
        continue

    # TODO seems to not work?
    if "pandas" in folder:
        continue

    print(f'Refreshing requirements for "{folder}"')
    check_call(['pip-compile', 'requirements.in'],
               cwd=f'{folder}')
