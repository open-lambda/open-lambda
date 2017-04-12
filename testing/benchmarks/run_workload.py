import time
import os
import math
import json
import subprocess
import shutil
import re
import random
import argparse
import numpy
import grequests
import requests
import multiprocessing

TRACE_RUN = False
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))

class Distribution:
    def __init__(self, dist, dist_args):
        self.dist = dist
        self.dist_args = dist_args

    def sample(self):
        dist = getattr(numpy.random, self.dist)
        return abs(math.ceil(dist(**self.dist_args)))

def distribution_factory(dist_spec):
    dist = dist_spec['dist']
    dist_spec.pop('dist')
    return Distribution(dist, dist_spec)

def run(cmd, quiet=False):
    if TRACE_RUN:
        print('EXEC ' + cmd)
    if quiet:
        err = open('/dev/null')
        rv = subprocess.check_output(cmd, shell=True, stderr=err)
        err.close()
    else:
        rv = subprocess.check_output(cmd, shell=True)
    return rv

def copy_handlers(cluster_name, handler_dir):
    shutil.rmtree('%s/%s/registry' % (SCRIPT_DIR, cluster_name))
    shutil.copytree('%s/%s' % (SCRIPT_DIR, handler_dir), '%s/%s/registry' % (SCRIPT_DIR, cluster_name))

def handle_async_request_response(res):
    print('Received res')

def get_async_request_obj(handler_name):
    headers = {'Content-Type': 'application/json'}
    return grequests.post('http://localhost:8080/runLambda/%s' % handler_name, headers=headers, data=json.dumps({
        "name": "Alice"
    }), hooks={'response': handle_async_request_response})

def make_blocking_request(handler_name):
    print('PID %d' % os.getpid())
    headers = {'Content-Type': 'application/json'}
    return requests.post('http://localhost:8080/runLambda/%s' % handler_name, headers=headers, data=json.dumps({
        "name": "Alice"
    }))

def fork_and_make_request(handler_name):
    p = multiprocessing.Process(target=make_blocking_request, args=(handler_name,))
    p.start()
    return p

def get_time_millis():
    return round(time.clock() * 1000)

def parse_config(config_file_name, handler_dir):
    if config_file_name is None:
        return {
            "cycles": 10, # amount
            "cycleInterval": 5, # time between cycle starts in ms
            "handlerGroups": [
                {
                    "groupName": "hello",
                    "runSample": {
                        "dist": "normal",
                        "loc": 5.0,
                        "scale": 1.0
                    },
                    "runAmount": {
                        "dist": "normal",
                        "loc": 10.0,
                        "scale": 5.0
                    },
                    # handlerRegex
                    "handlers": [

                    ]
                }
            ]
        }
    else:
        f = open(config_file_name, 'r')
        config = json.load(f)
        # if handlerRegex present populate handlers[] from it
        for handler_group in config["handlerGroups"]:
            # construct distributions
            handler_group["runSample"] = distribution_factory(handler_group["runSample"])
            handler_group["runAmount"] = distribution_factory(handler_group["runAmount"])
            # find handlers that belong to group if necessary
            if "handlerRegex" in handler_group:
                matched_handlers = []
                ro = re.compile(handler_group["handlerRegex"])
                present_handlers = os.listdir('%s/%s' % (SCRIPT_DIR, handler_dir))
                for handler_name in present_handlers:
                    if ro.match(handler_name):
                        matched_handlers.append(handler_name)
                print(len(matched_handlers))
                handler_group["handlers"] = matched_handlers

        return config

def cleanup_children(children):
    for k in range(0, len(children)):
        children[k].join()

def benchmark(config, verbose):
    if verbose:
        print('Running for %d cycles with an interval of time %d ms' % (config["cycles"], config["cycleInterval"]))
    children = []
    num_requests = 0
    for i in range(0, config["cycles"]):
        if verbose:
            print('Cycle %d:' % (i + 1))
        start_time = get_time_millis()
        for handler_group in config["handlerGroups"]:
            if numpy.random.random() < handler_group["runSample"].sample():
                num_to_run = handler_group["runAmount"].sample()
                for j in range(0, num_to_run):
                    handlers = handler_group["handlers"]
                    num_hanlders = len(handlers)
                    handler_to_run = handlers[random.randint(0, num_hanlders - 1)]
                    if verbose:
                        print('Create request to handler %s' % handler_to_run)
                    if num_requests % 100 == 0 and num_requests != 0:
                        cleanup_children(children)
                        children = []
                    children.append(fork_and_make_request(handler_to_run))
                    num_requests += 1
                    print(num_requests)
        end_time = get_time_millis()
        if verbose:
            print('Cycle took %d ms' % (end_time - start_time))
        if end_time - start_time < config["cycleInterval"]:
            time.sleep((end_time - start_time) / 1000)
        cleanup_children(children)


parser = argparse.ArgumentParser(description='Start a cluster')
parser.add_argument('-cluster', default=None)
parser.add_argument('-config', default=None)
parser.add_argument('-handler-dir', default=None)
parser.add_argument('--copy-handlers', action='store_true')
parser.add_argument('--verbose', action='store_true')
args = parser.parse_args()

if __name__ == '__main__':
    if args.copy_handlers:
        if not args.cluster:
            print('Must specify cluster name if copying handlers')
            exit()
        if args.verbose:
            print('Copying handlers')
        if not args.handler_dir:
            print('Must specify handler directory')
        copy_handlers(args.cluster, args.handler_dir)

    if args.verbose:
        print('Parsing config')
    config = parse_config(args.config, args.handler_dir)
    if args.verbose:
        print(config)
        print('Benchmarking...')
    benchmark(config, args.verbose)
