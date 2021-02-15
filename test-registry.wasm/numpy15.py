import numpy

return {'result': int(numpy.array(event).sum()), 'version': numpy.__version__}
