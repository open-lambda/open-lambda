# ol-install: numpy==1.18.5,pandas==1.4

# newer versions of pandas do not support numpy 1.18 so we are setting pandas to 1.4 here

import numpy
import pandas

def f(event):
    df = pandas.DataFrame(event)
    return {'result': int(df.values.sum()), 'version': numpy.__version__}
