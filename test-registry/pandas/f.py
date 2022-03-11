# ol-install: pandas

import numpy
import pandas

def f(event):
    df = pandas.DataFrame(event)
    return {'result': int(df.values.sum()), 'version': numpy.__version__}
