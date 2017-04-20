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

def async_response(res):
    global outstanding
    global log_queue

    print('response')
    split = r.request.url.split('/')
    handler_name = split[len(split)-1]
    ms = r.elapsed.seconds*1000.0 + r.elapsed.microseconds/1000.0
    log_queue.put({"time": datetime.datetime.now(), "handler_name": handler_name, "status_code": r.status_code, "latency": ms})

    with async_lock:
        outstanding -= 1

    return

def async_request(handler_name):
    global outstanding

    r = grequests.post('http://localhost:8080/runLambda/%s' % handler_name, headers=HEADERS, data=json.dumps({
        "name": "Alice"
    }), hooks={'response': async_response})

    grequests.map([r])

    with async_lock:
        outstanding += 1

    return

def sync_request(handler_name):
    r = requests.post('http://localhost:8080/runLambda/%s' % handler_name, headers=HEADERS, data=json.dumps({
        "name": "Alice"
    }))

    ms = r.elapsed.seconds*1000.0 + r.elapsed.microseconds/1000.0
    result = {"time": datetime.datetime.now(), "handler_name": handler_name, "status_code": r.status_code, "latency": ms}
    log_queue.put(result)

    return 

def runner(i, flag, request_func):
    global outstanding
    global async_lock
    global log_queue

    async_lock = threading.Lock()
    with async_lock:
        outstanding = 0

    wait = float(config['wait'])/1000.0

    print('runner %d started' % i)
    while flag.value != 1:
        if config['wait'] > 0:
            time.sleep(wait)

        request_func(random.choice(handlers))

    while True:
        with async_lock:
            if outstanding == 0:
                break

        time.sleep(0.1)

    log_queue.put(None)

def log_consumer(log_path):
    global log_queue

    start = time.clock()
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

    for i in range(config['runners']):
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

def main():
    global log_queue
    log_queue = multiprocessing.Queue(10000)

    log_path = os.path.join(SCRIPT_DIR, config['log'])
    if os.path.exists(log_path):
        print('log file exists - please move it')
        sys.exit(1)

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

    main()

