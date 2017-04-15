import math
import numpy


class Distribution:
    def __init__(self, dist, dist_args):
        self.dist = dist
        self.dist_args = dist_args

    def sample(self):
        if self.dist == 'exact_distribution':
            r = numpy.random.randint(0, 1)
            total = 0
            for v in self.dist_args['values']:
                if v['weight'] + total > r and total < r:
                     return int(v['value'])
        elif self.dist == 'exact_value':
            return self.dist_args['value']
        else:
            dist = getattr(numpy.random, self.dist)
            return abs(math.ceil(dist(**self.dist_args)))


def distribution_factory(dist_spec):
    dist = dist_spec['dist']
    dist_spec.pop('dist')
    return Distribution(dist, dist_spec)