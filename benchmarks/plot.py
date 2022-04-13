#! /bin/env python3

import numpy as np
import copy
import matplotlib.pyplot as plt
from pandas import read_csv

def main():
    df = read_csv("bench-results.csv", header=0, skipinitialspace=True)

    ax = plt.subplot(1,1,1)

    bench_names = df.bench_name.unique()
    worker_types = df.worker_type.unique()

    worker_count = len(worker_types)
    x = np.arange(len(bench_names))
    width = 1.0 / (worker_count+1)

    baseline = None

    for (pos, worker_type) in enumerate(worker_types):
        means = []
        stdev = []

        for bench_name in bench_names:
            data = df[(df.worker_type == worker_type) & (df.bench_name == bench_name)]
            means.append(np.mean(data.elapsed))
            stdev.append(np.std(data.elapsed))

        abs_mean = None

        if pos == 0:
            baseline = copy.deepcopy(means)
            abs_mean = baseline

            for i in range(len(means)):
                means[i] = 1.0
        else:
            assert(len(means) == len(baseline))
            abs_mean = copy.deepcopy(means)

            for i in range(len(means)):
                means[i] = means[i] / baseline[i]

        ax.bar(x + (pos * width), means, width, label=worker_type)#, yerr=stdev)

        for (mpos, mean) in enumerate(means):
            label = "{:10.2f}ms".format(abs_mean[mpos])
            ax.text(x[mpos] + (pos * width) - 0.8*width, mean+0.01, label, color='black', fontsize='x-small') 

    ax.legend(loc="lower center")

    ax.set_xlabel("Benchmark")
    ax.set_ylabel("Normalized Execution Time")

    ax.set_xticks(x+(worker_count-1)*0.5*width)
    ax.set_xticklabels(bench_names)

    plt.savefig('bench-results.pdf')

 
if __name__ == "__main__":
    main()
