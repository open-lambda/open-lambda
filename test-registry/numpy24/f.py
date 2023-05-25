# ol-install: numpy==1.24
import numpy

def f(event):
    return {'result': int(numpy.array(event).sum()), 'numpy-version': numpy.__version__}
