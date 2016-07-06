#!/usr/bin/env python
import os, sys, requests, json, base64, time, random, collections, argparse
from multiprocessing import Process, Pipe
from common import *

# TODO: use conf
ARRIVAL_INTERVAL_MEAN = 1
ARRIVAL_INTERVAL_DEV = 0.5

def conf():
    if conf.val == None:
        with open('static/config.json') as f:
            conf.val = json.loads(f.read())
    return conf.val
conf.val = None


class User():
    def __init__(self, endtime, linenum, typingSpeedDelay):
        self.url = conf()['url']
        self.endtime = endtime
        self.linenum = linenum
        self.line = lines[linenum].strip()
        self.typingSpeedDelay = typingSpeedDelay
        self.stats = {'ops': 0, 
                      'latency-sum': 0.0}
        print self.line

    def post(self, data):
        print "pref: " + data['pref']
        t0 = time.time()
        # TODO: support skip mode to make sure client isn't overwhelmed
        r = requests.post(self.url, data=json.dumps(data))
        t1 = time.time()

        self.stats['ops'] += 1
        self.stats['latency-sum'] += (t1-t0)

        return r.text

    # TODO: verify results
    def run(self):
        count = 1
        word = ''
        while True:
            if time.time() + self.typingSpeedDelay >= self.endtime:
                break
            # TODO: subtract out time spent on last req
            beforePref = time.time()
            pref, count, word = self.getPref(count, word)
            #print count
            #print pref
            afterPref = time.time()
            elapsed = afterPref - beforePref
            realDelay = self.typingSpeedDelay - elapsed
            if realDelay > 0:
                time.sleep(realDelay)
            else:
                print "TOO SLOW"
            post = self.post({"op":"keystroke", "pref":pref})
            print post
            words = post[12:].split(', ')
            print words
            word  = unicode('"' + word + '"')
            print "word: " + word
            print "words[0] " + words[0]
            if word == words[0]:
                print "*******************FOUND"
                count = self.line.find(' ', count)
                if count == -1:
                    count = len(self.line)
            word = word[1:-2]
        return self.stats

    def getPref(self, count, currword):
        if count == len(self.line):
            self.linenum = random.randint(0, len(lines) - 2)
            self.line = lines[self.linenum]
            count = 1
        if  count == 1 or self.line[count] == ' ':
            endword = self.line.find(' ', count + 1)
            currword = self.line[count:endword]
            currword = currword.strip()
            currword = currword.lower()

        #print self.line
        substr = self.line[0:count]
        #print substr
        subsplit = substr.split(' ')
        currpref = subsplit[len(subsplit) - 1]

        return currpref, count + 1, currword

class UserProcess:
    def __init__(self, endtime, linenum, typingSpeedDelay):
        self.parent_conn = None
        self.child = None
        self.endtime = endtime
        self.linenum = linenum
        self.typingSpeedDelay = typingSpeedDelay

    def run(self, conn):
        u = User(self.endtime, self.linenum, self.typingSpeedDelay)
        results = u.run()
        conn.send(results)
        conn.close()

    def start(self):
        self.parent_conn, child_conn = Pipe()
        self.child = Process(target=self.run, args=(child_conn,))
        self.child.start()

    def wait(self):
        result = self.parent_conn.recv()
        self.child.join()
        return result

# child
def run(conn):
    u = User()
    results = u.run()
    conn.send(results)
    conn.close()
# parent
def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--users', '-u', metavar='u', default=1, type=int)
    parser.add_argument('--seconds', '-s', metavar='s', default=10, type=int)
    args = parser.parse_args()
    endtime = time.time() + args.seconds
    procs = []
    global lines
    lines = []
    textsPath = os.path.join(SCRIPT_DIR, "texts")
    textfiles = [a for a in os.listdir(textsPath) if os.path.isfile(os.path.join(textsPath, a))]
    #textfiles = ["Frankenstein.txt"]
    #textfiles = ["SOTU.txt"]
    for t in textfiles:
        f = open(os.path.join(textsPath, t), 'r')
        for line in f:
            lines.append(line)
        f.close()
#typing speed delays based on avg 40 wpm +/- 20 wpm. Average word length is five
#characters, so typing between 20 * 5 = 100 to 60 * 5 = 300 characters/min
#so, typing chars at rate of somewhere between 1 char every 0.2 s to 0.6 s.

    for i in range(args.users):
        linenum = random.randint(0, len(lines) - 2)
        typingSpeedDelay = random.randint(2000, 6000)
        typingSpeedDelay = typingSpeedDelay / 10000.0
        procs.append(UserProcess(endtime, linenum, typingSpeedDelay))

    for proc in procs:
        proc.start()

    totals = {'latency-sum': 0.0, 'ops': 0.0}
    for proc in procs:
        results = proc.wait()
        for k in totals.keys():
            totals[k] += results[k]
    print 'Average latency: %.3f seconds' % (totals['latency-sum'] / totals['ops'])

if __name__ == '__main__':
    main()
