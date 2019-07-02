# ol-install: numpy

def handler(event):
    import numpy
    return str(dir(numpy))
    return numpy.array(event).sum()
