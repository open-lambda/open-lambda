import os, sys, json
import time, datetime
import requests, grequests
import math, random, statistics
import threading, multiprocessing

global config
global handlers
global log_queue
global async_lock
global outstanding

HEADERS = {'Content-Type': 'application/json'}
SCRIPT_DIR = os.path.dirname(os.path.realpath(__file__))

def async_response(r, **kwargs):
    global outstanding
    global log_queue

    split = r.request.url.split('/')
    handler_name = split[len(split)-1]
    ms = r.elapsed.seconds*1000.0 + r.elapsed.microseconds/1000.0
    log_queue.put({"time": datetime.datetime.now(), "handler_name": handler_name, "status_code": r.status_code, "latency": ms})

    with async_lock:
        outstanding -= 1

    return

def async_request(handler_name):
    global outstanding

    r = grequests.post('http://%s:8080/runLambda/%s' % (config['host'], handler_name), headers=HEADERS, data=json.dumps({
        "name": "Alice"
    }), hooks=dict(response=async_response))

    job = grequests.send(r, grequests.Pool(1))

    with async_lock:
        outstanding += 1

    return

def sync_request(handler_name):
    r = requests.post('http://%s:8080/runLambda/%s' % (config['host'], handler_name), headers=HEADERS, data=json.dumps({
        "name": "Alice"
    }))

    ms = r.elapsed.seconds*1000.0 + r.elapsed.microseconds/1000.0
    result = {"time": datetime.datetime.now(), "handler_name": handler_name, "status_code": r.status_code, "latency": ms}
    log_queue.put(result)

    return


def handlers_slice(start, end):
    if end < start:
        start_ind = math.floor((start / 100) * len(handlers))
        end_ind = math.floor((end / 100) * len(handlers))
        return  handlers[:end_ind] + handlers[start_ind:]
    else:
        start_ind = math.floor((start / 100) * len(handlers))
        end_ind = math.floor((end / 100) * len(handlers))
        return handlers[start_ind:end_ind]


def runner(i, flag, request_func):
    global outstanding
    global async_lock
    global log_queue

    async_lock = threading.Lock()
    with async_lock:
        outstanding = 0

    wait = float(config['wait'])/1000.0
    rot_start = None
    rot_cur = None
    rot_handlers = None
    print('runner %d started' % i)
    while flag.value != 1:
        if config['wait'] > 0:
            time.sleep(wait)

        if not 'handler_choice' in 'rotation' or config['handler_choice'] != 'rotation':
            handler = random.choice(handlers)
        else:
            if rot_start == None:
                rot_cur = 0
                rot_handlers = handlers_slice(rot_cur, (rot_cur + config['rotation_slice']) % 100)
                rot_start = time.time()
            elif (time.time() - rot_start) * 1000 > config['rotation_period']:
                rot_cur = (rot_cur +  config['rotation_slice']) % 100
                rot_handlers = handlers_slice(rot_cur, (rot_cur + config['rotation_slice']) % 100)
                rot_start = time.time()
            handler = random.choice(rot_handlers)

        request_func(handler)

    log_queue.put(None)
    log_queue = None

    return


def log_consumer(log_path):
    global log_queue

    results = []

    log = open(log_path, 'w')

    num_500s = 0
    finished = 0
    while finished < config['runners']:
        entry = log_queue.get()
        if entry == None:
            finished += 1
            continue

        if type(entry) is str:
            print("Could not find server")
            log.write("Could not find server")
            break
        else:
            results.append(entry)
            log_str = '[%s] handler: %s status: %d in: %d ms' % ( datetime.datetime.isoformat(entry["time"]), entry["handler_name"], entry["status_code"], entry["latency"])
            print(log_str)
            log.write(log_str + '\n')
            if entry["status_code"] == 500:
                num_500s += 1

    log.write('\n')

    # compute handler results
    handlers = []
    for l in results:
        if not l["handler_name"] in handlers:
            handlers.append(l["handler_name"])
    for h in handlers:
        log.write("Handler: '%s'\n" % h)
        reqs = []
        for l in results:
            if l["handler_name"] == h and l["status_code"] == 200:
                reqs.append(l)
        r = [req['latency'] for req in reqs]
        if len(r) != 0:
            log.write("\tAvg latency: %d\n" % statistics.mean(r))
        log.write("\tTotal requests: %d\n\n" % len(r))

    # compute total results
    only_200s = []
    for l in results:
        if l["status_code"] == 200:
            only_200s.append(l)

    log.write('Aggregate: \n')
    if len(only_200s) != 0:
        log.write("\tAvg latency: %d\n" % (statistics.mean([ent['latency'] for ent in only_200s])))
    log.write("\tThroughput: %d\n" % len(only_200s))
    log.write("\tFailed requests: %d\n" % num_500s)
    log.close()

    return

def run_benchmark():
    if config['type'] == 'sync':
        request_func = sync_request
    elif config['type'] == 'async':
        request_func = async_request
    else:
        print('benchmark type must be either sync or async')
        sys.exit(1)

    minutes = config['minutes']
    runners = []
    flag = multiprocessing.Value('b')

    wait = float(config['wait'])/1000.0
    inc = wait / config['runners']
    for i in range(config['runners']):
        time.sleep(inc)
        p = multiprocessing.Process(target=runner, args=(i, flag, request_func))
        p.start()
        runners.append(p)

    start = time.time()
    while time.time() - start < minutes * 60:
        time.sleep(0.1)

    flag.value = 1

    for p in runners:
        p.join()

    return

def get_handlers():
    global handlers

    handler_path = os.path.join(SCRIPT_DIR, config['handlers'])
    if not os.path.exists(handler_path):
        print('handler file not found (%s)' % handler_path)
        sys.exit(1)

    handlers = []
    with open(handler_path) as fd:
        for line in fd:
            handlers.append(line.strip())

    return

def main(log_path):
    global log_queue
    log_queue = multiprocessing.Queue(10000)

    logger = multiprocessing.Process(target=log_consumer, args=(log_path,))
    logger.start()

    get_handlers()
    run_benchmark()

    logger.join()

if __name__ == '__main__':
    global config 

    if len(sys.argv) != 2:
        print('Usage: %s <config.json>' % sys.argv[0])
        sys.exit(1)

    with open(sys.argv[1]) as fd:
        config = json.load(fd)

    log_path = sys.argv[1].split('.json')[0] + '.log'
    if os.path.exists(log_path):
        print('log file exists - please move it')
        sys.exit(1)

    main(log_path)

