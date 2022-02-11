# ol-install: numpy==1.19
import numpy

def f(event):
    return {'result': int(numpy.array(event).sum()), 'version': numpy.__version__}
