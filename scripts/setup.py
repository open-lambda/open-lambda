''' Installer for the OpenLambda python bindings '''

from setuptools import setup

setup(
    name='open_lambda',
    version='0.1.0',
    py_modules=['open_lambda'],
    install_requires=["requests"]
)
