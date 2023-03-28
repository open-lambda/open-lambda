import matplotlib.pyplot as plt
import json

names = []
for module in ["py", "pd"]:
    for num in ["64"]:
        for kind in ["seq", "par"]:
            names.append(f'{module}{num}-{kind}')

bench = [[],[],[],[]]
for i in range(1,5):
    data = json.load(open(f'benchmark-{i}.json', 'r'))
    for j, name in enumerate(names):
        bench[j].append(data[name]["ops/s"])

X = [1,2,3,4]
plt.figure(figsize=(10,7))
plt.plot(X, bench[0], label=names[0], marker="o")
plt.plot(X, bench[1], label=names[1], marker="o")
plt.xlabel("number of workers", fontsize=16)
plt.ylabel("ops/s", fontsize=16)
plt.xticks(X)
plt.legend(prop={'size': 12})
plt.savefig('py64.png')

plt.figure(figsize=(10,7))
plt.plot(X, bench[2], label=names[2], marker="o")
plt.plot(X, bench[3], label=names[3], marker="o")
plt.xlabel("number of workers", fontsize=16)
plt.ylabel("ops/s", fontsize=16)
plt.xticks(X)
plt.legend(prop={'size': 12})
plt.savefig('pd64.png')
