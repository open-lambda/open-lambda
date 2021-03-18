#! /bin/env python3

import numpy as np
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

    for (pos, worker_type) in enumerate(worker_types):
        means = []
        stdev = []

        for bench_name in bench_names:
            data = df[(df.worker_type == worker_type) & (df.bench_name == bench_name)]
            means.append(np.mean(data.elapsed))
            stdev.append(np.std(data.elapsed))

        ax.bar(x + (pos * width), means, width, label=worker_type, yerr=stdev)

    ax.legend()

    ax.set_xlabel("Benchmark")
    ax.set_ylabel("Time (ms)")

    ax.set_xticks(x+(worker_count-1)*0.5*width)
    ax.set_xticklabels(bench_names)

    plt.savefig('bench-results.pdf')

 
if __name__ == "__main__":
    main()
