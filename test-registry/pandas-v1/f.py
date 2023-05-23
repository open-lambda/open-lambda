# ol-install: numpy==1.24,pandas==1.5

import numpy
import pandas

def f(event):
    df = pandas.DataFrame(event)
    return {'result': int(df.values.sum()), 'numpy-version': numpy.__version__}
