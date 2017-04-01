import os, statistics

files = {
    'ninc': None,
    'inc': None,
    'nic': None,
    'ic': None
}

results_all = {
    'ninc': None,
    'inc': None,
    'nic': None,
    'ic': None
}

for name, _ in files.items():
    try:
        filename = 'perf/' + name + '.perf'
        print filename
        f = open(filename, 'r')
        lines = f.readlines()
        lines = [line.strip() for line in lines]
        files[name] = lines
        print 'success'
    except Exception, e:
        print e

for name, lines in files.items():
    if lines is None:
        continue
    results = {}
    for line in lines:
        line_as_list = line.split(':')
        value_unit = line_as_list[1].split()
        if line_as_list[0] not in results:
            results[line_as_list[0]] = ([], value_unit[1])
        results[line_as_list[0]][0].append(int(value_unit[0]))

    results_all[name] = results

print results_all

for name, results in results_all.items():
    print name + '\n'
    if results is None:
        continue
    for result in results.items():
        print result[0]
        value_list = result[1][0]
        unit = result[1][1]
        maxv = str(max(value_list))
        minv = str(min(value_list))
        avg = str(statistics.mean(value_list))
        stddev = None
        var = None
        try:
            stddev = str(statistics.stdev(value_list))
            var = str(statistics.variance(value_list))
        except Exception:
            stddev = 'NA'
            var = 'NA'
        print 'max: ' +  maxv + ' min: ' + minv + ' avg: ' + avg  + ' stddev: ' + stddev + ' var: ' + var + ' ' + unit \
              + ' entries: ' + str(len(value_list))
