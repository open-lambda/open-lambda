KB = 1024.0
MB = 1024.0*KB
GB = 1024.0*MB

import collections, os, sys, math, json, subprocess

ddict = collections.defaultdict
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))
TRACE_RUN = True

def run(cmd):
    if TRACE_RUN:
        print 'EXEC ' + cmd
    return subprocess.check_output(cmd, shell=True)

def run_js(cmd):
    return json.loads(run(cmd))

def panic():
    assert(0)

def human_bytes(x):
    conv = [(1024.0**3,'GB'),
            (1024.0**2,'MB'),
            (1024.0**1,'KB'),
            (1,'B')]
    while len(conv):
        amt, unit = conv.pop(0)
        if amt <= x or len(conv) == 0:
            x = float(x) / amt
            if int(x*10) % 10 == 0:
                return '%d%s' % (x, unit)
            else:            
                return '%0.1f%s' % (x, unit)
    assert(0)

def only(lst):
    lst = list(lst)
    assert(len(lst) == 1)
    return lst[0]

def memoize(function):
    memo = {}
    def wrapper(*args, **kvargs):
        key = (args, tuple(sorted(kvargs.iteritems())))
        if key in memo:
            return memo[key]
        else:
            rv = function(*args, **kvargs)
            memo[key] = rv
            return rv
    return wrapper

def argsdict(**kvargs):
    return kvargs

def rdjs(path):
    return json.loads(readall(path))

def wrjs(path, data, atomic=False):
    if atomic:
        wrjs(path+'.tmp', data, atomic=False)
        os.rename(path+'.tmp', path)
    else:
        writeall(path, json.dumps(data, indent=2))

def readall(path):
    f = open(path)
    d = f.read()
    f.close()
    return d

def writeall(path, data):
    f = open(path, 'w')
    f.write(data)
    f.close()

def path_iter(path, skip_empty=True):
    f = open(path)
    for l in f:
        if skip_empty and l.strip() == '':
            continue
        yield l
    f.close()

# example: key1=val1, key2=val2, ...
def parse_comma_eq(data, typ=float):
    pairs = data.split(',')
    d = {}
    for pair in pairs:
        k,v = pair.strip().split('=')
        if typ != None:
            v = typ(v)
        d[k] = v
    return d

# example:
# node1
#     leaf1: val1
#     node2
#         leaf2: val2
def parse_tab_colon_tree(data, typ=float):
    def tab_count(l):
        return len(l) - len(l.lstrip('\t'))

    tree = {}
    levels = [tree]

    for l in data.split('\n'):
        if not l.strip():
            continue
        parts = map(str.strip, l.split(':'))
        key = parts[0]
        if len(parts) == 1:
            val = {}
        else:
            val = parts[1]
            if typ != None:
                val = typ(val)
        level_idx = tab_count(l)
        assert(level_idx < len(levels))
        levels[level_idx][key] = val
        if len(parts) == 1:
            if len(levels) <= level_idx+1:
                levels.append(None)
            levels[level_idx+1] = val
    return tree

def key_replace(orig={}, replace={}, recursive=False):
    for k1,k2 in replace.iteritems():
        if k1 in orig:
            v = orig.pop(k1)
            orig[k2] = v
    if recursive:
        for v in orig.values():
            if type(v) == dict:
                key_replace(v, replace, recursive)

class Sample:
    def __init__(self, vals=None):
        if vals == None:
            vals = []
        self.vals = vals

    def add(self, val):
        self.vals.append(val)

    def perc_under(self, threshold):
        count = len(filter(lambda v: v<threshold, self.vals))
        return count * 100.0 / len(self.vals)

    def sub_sample(self, lower=0, upper=None):
        vals = filter(lambda v: lower<=v<=upper, self.vals)
        return Sample(vals)

    def dump(self):
        vals = sorted(self.vals)
        for i, v in enumerate(vals):
            print '%d: %f' % (i, v)

    def median(self):
        vals = sorted(self.vals)
        if len(vals) % 2 == 0:
            return (vals[len(vals)/2] + vals[len(vals)/2-1]) / 2.0
        else:
            return vals[len(vals)/2]

    def avg(self):
        return sum(self.vals) / len(self.vals)

    def sum(self):
        return sum(self.vals)

    def max(self):
        return max(self.vals)

    def __str__(self):
        return ', '.join(map(str,self.vals))

def keylist_parse(keylist):
    if type(keylist) == str:
        return keylist.split(':')
    return list(keylist)

def tree_get(tree, keylist, default=None):
    keylist = keylist_parse(keylist)
    tmp = tree
    while len(keylist):
        key = keylist.pop(0)
        if not key in tmp:
            return default
        tmp = tmp[key]
    return tmp

def tree_put(tree, keylist, val):
    keylist = keylist_parse(keylist)
    tmp = tree
    while len(keylist) > 1:
        key = keylist.pop(0)
        if not key in tmp:
            tmp[key] = {}
        tmp = tmp[key]
    tmp[keylist.pop(0)] = val
    
def leaf_iter_callback(subtree, keylist=[], fn=None):
    for k,v in subtree.iteritems():
        if type(v) == dict:
            leaf_iter_callback(v, keylist+[k], fn)
        else:
            fn(keylist+[k])

def leaf_iter(subtree, keylist=[]):
    vals = []
    def add(val):
        vals.append(val)
    leaf_iter_callback(subtree, fn=add)
    return vals

def trees_sample(trees):
    sample_tree = {}
    for tree in trees:
        for keylist in leaf_iter(tree):
            sample = tree_get(sample_tree, keylist, Sample())
            sample.add(tree_get(tree, keylist))
            tree_put(sample_tree, keylist, sample)
    return sample_tree
    
