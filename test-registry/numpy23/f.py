# ol-install: numpy==1.23
import numpy

def f(event):
    return {'result': int(numpy.array(event).sum()), 'numpy-version': numpy.__version__}
