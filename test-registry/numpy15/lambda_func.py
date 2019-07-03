# ol-install: numpy==1.15
import numpy

def handler(event):
    return {'result': int(numpy.array(event).sum()), 'version': numpy.__version__}
