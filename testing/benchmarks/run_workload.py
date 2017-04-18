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
import datetime
import statistics

TRACE_RUN = False
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))

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
    headers = {'Content-Type': 'application/json'}
    start = get_time_millis()
    r = requests.post('http://localhost:8080/runLambda/%s' % handler_name, headers=headers, data=json.dumps({
        "name": "Alice"
    }))
    end = get_time_millis()
    now = datetime.datetime.now()
    result = {"time": datetime.datetime.now(), "handler_name": handler_name, "status_code": r.status_code, "latency": end-start}
    return result

def fork_and_make_request(handler_name):
    p = multiprocessing.Process(target=make_blocking_request, args=(handler_name,))
    p.start()
    return p

def get_time_millis():
    return time.time() * 1000

def parse_config(config_file_name, handler_dir):
    if config_file_name is None:
        config = {
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
                    "handlerRegex": ".*"
                },
            ],
            "workload": {
                "type": "latency",
                "latencyDist": {
                    "dist": "exact_value",
                    "value": 1000 # ms
                },
                "500_tolerance": 1 # a percentile, out of int 100
            }
        }
    else:
        f = open(config_file_name, 'r')
        config = json.load(f)
    if config["workload"]["type"] == 'latency':
        config["workload"]["latencyDist"] = distribution_factory(config["workload"]["latencyDist"])
    # if handlerRegex present populate handlers[] from it
    for hg in (config["handlerGroups"]):
        # construct distributions
        hg["runSample"] = distribution_factory(hg["runSample"])
        hg["runAmount"] = distribution_factory(hg["runAmount"])
        # find handlers that belong to group if necessary
        if "handlerRegex" in hg:
            matched_handlers = []
            ro = re.compile(hg["handlerRegex"])
            present_handlers = os.listdir('%s/%s' % (SCRIPT_DIR, handler_dir))
            for handler_name in present_handlers:
                if ro.match(handler_name):
                    matched_handlers.append(handler_name)
            hg["handlers"] = matched_handlers

    return config

def cleanup_children(children):
    for k in range(0, len(children)):
        children[k].join()

def request_runner(config, id, log_queue, shared_bool):
    print('Request runner %d started' % id)
    while shared_bool.value != 1:
        for handler_group in config["handlerGroups"]:
            if numpy.random.random() < handler_group["runSample"].sample():
                num_to_run = handler_group["runAmount"].sample()
                for j in range(0, num_to_run):
                    handlers = handler_group["handlers"]
                    num_hanlders = len(handlers)
                    start = time.clock()
                    handler_to_run = handlers[random.randint(0, num_hanlders - 1)]
                    try:
                        res_str = make_blocking_request(handler_to_run)
                    except Exception as e:
                        log_queue.put("lost_connection")
                        return
                    log_queue.put(res_str)
                    if config["workload"]["type"] == 'latency':
                        elapsed_time = time.clock() - start
                        latency =  config["workload"]["latencyDist"].sample()
                        if elapsed_time / 1000 < latency:
                            time.sleep((latency - elapsed_time) / 1000)
    log_queue.put(None)

def log_queue_consumer(config, stats_queue, log_file_name, num_minutes, num_producers, end_on_error):
    start = time.clock()
    results = []
    f = None
    if log_file_name:
        f = open(log_file_name, 'w')
    end_count = 0
    num_500s = 0
    while time.clock() - start < num_minutes * 60:
        log_entry = stats_queue.get()
        if log_entry == None:
            end_count += 1
            if end_count == num_producers:
                break
            continue
        elif type(log_entry) is str:
            print("Could not find server")
            f.write("Could not find server")
            break
        else:
            results.append(log_entry)
            log_str = '[%s] handler: %s status: %d in: %d ms' % ( datetime.datetime.isoformat(log_entry["time"]), log_entry["handler_name"], log_entry["status_code"], log_entry["latency"])
            print(log_str)
            if f:
                f.write(log_str + '\n')
            # check if above 500 tolerance
            if log_entry["status_code"] == 500:
                num_500s += 1
                if float(num_500s) / len(results) > config["workload"]["500_tolerance"] / 100:
                    print("500 Tolerance Reached, stopping benchmark...")
                    f.write("500 Tolerance Reached, stopping benchmark...")
                    break
    # compute handler results
    handlers = []
    for l in results:
        if not l["handler_name"] in handlers:
            handlers.append(l["handler_name"])
    for h in handlers:
        reqs = []
        for l in results:
            if l["handler_name"] == h and l["status_code"] == 200:
                reqs.append(l)
        r = [req['latency'] for req in reqs]
        if len(r) != 0:
            f.write("Hanlder %s avg latency: %d\n" % (h, statistics.mean(r)))
        f.write("Total num reqs to handler %s: %d\n" % (h, len(r)))
    # compute total results
    only_200s = []
    for l in results:
        if l["status_code"] == 200:
            only_200s.append(l)

    if len(only_200s) != 0:
        f.write("Overall Average latency(200s only): %d\n" % (statistics.mean([ent['latency'] for ent in only_200s])))
    f.write("Overall throughput(200s only): %d\n" % len(only_200s))
    f.close()
    # tell parent we're done
    end_on_error.value = 1

def get_latency(le):
    return le["latency"]

def benchmark(config, num_minutes, num_req_runners, log_file_name):
    req_runners = []
    log_queue = multiprocessing.Queue(10000)
    runner_flag = multiprocessing.Value('b')
    end_on_error = multiprocessing.Value('b')

    print('Creating log queue consumer')
    queue_consumer = multiprocessing.Process(target=log_queue_consumer, args=(config, log_queue, log_file_name, num_minutes, num_req_runners, end_on_error))
    queue_consumer.start()

    print('Creating %d request runners' % num_req_runners)
    for i in range(0, num_req_runners):
        p = multiprocessing.Process(target=request_runner, args=(config, i, log_queue, runner_flag))
        p.start()
        req_runners.append(p)

    start = time.clock()
    while time.clock() - start < num_minutes * 60:
        if end_on_error.value == 1:
            break
    runner_flag.value = 1
    for p in req_runners:
        p.join()

    queue_consumer.join()


parser = argparse.ArgumentParser(description='Start a cluster')
parser.add_argument('-cluster', default=None)
parser.add_argument('-config', default=None)
parser.add_argument('-handler-dir', default=None)
parser.add_argument('-request-runners', type=int, default=2)
parser.add_argument('-run-minutes', type=int, default=5)
parser.add_argument('-log-file', default=None)
parser.add_argument('--copy-handlers', action='store_true')
parser.add_argument('--verbose', action='store_true')
args = parser.parse_args()

def main():
    if args.copy_handlers:
        if not args.cluster:
            print('Must specify cluster name if copying handlers')
            exit()
        print('Copying handlers')
        if not args.handler_dir:
            print('Must specify handler directory')
        copy_handlers(args.cluster, args.handler_dir)

    print('Parsing config')
    config = parse_config(args.config, args.handler_dir)
    if args.verbose:
        print(config)
    print('Benchmarking...')
    benchmark(config, args.run_minutes, args.request_runners, args.log_file)

if __name__ == '__main__':
    main()

