import argparse
import json
import numpy


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('spec_file', help='json specification file of the pipbench mirror')
    #parser.add_argument('zipf_arg', help='parameter for the zipfian distribution')
    parser.add_argument('dist_file', help='handler import dist')
    args = parser.parse_args()

    #zipf_arg = int(args.zipf_arg)

    with open(args.spec_file, 'r') as f:
        spec = json.load(f)

    with open(args.dist_file, 'r') as df:
        dist = [x.strip() for x in df]


    with open(args.spec_file, 'w') as sf:
        with open(args.dist_file, 'r') as df:
            for entry in spec:
                #entry['handler_popularity'] = numpy.random.zipf(zipf_arg)
                entry['handler_popularity'] = int(dist[numpy.random.randint(0, len(dist))])

        json.dump(spec, sf, indent=4)


if __name__ == '__main__':
    main()
