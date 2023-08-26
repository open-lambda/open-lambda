import numpy
import pandas

def f(event):
    df = pandas.DataFrame(event)
    return {'result': int(df.values.sum()), 'numpy-version': numpy.__version__}
