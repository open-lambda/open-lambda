import math
import numpy


class Distribution:
    def __init__(self, dist, transform, dist_args):
        self.dist = dist
        self.dist_args = dist_args
        self.transform = transform

    def sample(self):
        val = None
        if self.dist == 'exact_distribution':
            r = numpy.random.randint(0, 1)
            total = 0
            for v in self.dist_args['values']:
                if total < r and total < r + v['weight']:
                    val = v['value']
                    break
                total += v['weight']
        elif self.dist == 'exact_distribution_uniform':
            i = numpy.random.randint(0, 20)
            val = self.dist_args['values'][i]
        elif self.dist == 'exact_value':
            val = self.dist_args['value']
        else:
            dist = getattr(numpy.random, self.dist)
            val = dist(**self.dist_args)

        if self.transform:
            if self.transform == 'float_s_to_int_ms':
                return round(val * 1000)
        else:
            return abs(math.ceil(val))


def distribution_factory(dist_spec):
    dist = dist_spec['dist']
    dist_spec.pop('dist')
    transform = None
    if 'transform' in dist_spec:
        transform = dist_spec['transform']
        dist_spec.pop('transform')
    return Distribution(dist, transform, dist_spec)