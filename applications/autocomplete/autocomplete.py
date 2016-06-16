# source of words is http://norvig.com/google-books-common-words.txt
import time, traceback, sys, marisa_trie
global words
global freqs
global prefs
global trie
def setup():
    #make array of words objects with word and frequency
    f = open("words.txt")
    words = []
    freqs = []
    for line in f:
        line = line.strip()
        line = line.lower()
        if " " in line:
            line = line.split(" ")
        else:
            line = line.split("\t")
        word = line[0]
        freq = line[1]
        freqInt = int(freq)
        words.append(word)
        freqs.append(freqInt)

    f.close()

    #make trie of ranges in word array
    g = open("ranges.txt")
    prefixes = []
    ranges = []
    for line in g:
        line = line.strip()
        line = line.split("\t")
        lineu = unicode(line[0])
        prefixes.append(lineu)
        rangeind = (int(line[1]), int(line[2]))
        ranges.append(rangeind)
    fmt = "<LL"
    trie = marisa_trie.RecordTrie(fmt, zip(prefixes, ranges))
    g.close()
    return words, freqs, prefixes, trie

def init(event):

    words, freqs, prefs, trie = setup()
    return 'initialized'

def findMaxFreq(prefrange, currmax):
    maxFreqInd = prefrange[0][0]
    maxFreq = freqs[maxFreqInd]
    for i in range(prefrange[0][0], prefrange[0][1] + 1):
        if freqs[i] > maxFreq and freqs[i] < currmax:
            maxFreqInd = i
            maxFreq = freqs[i]
    return words[maxFreqInd], freqs[maxFreqInd]


def keystroke(event):
    prefix = event['pref']
    global trie
    prefixu = unicode(prefix)
    prefrange = trie.get(prefixu)
    suggestions = []
    currMax = sys.maxint
    for i in range(5):
        suggestion, currMax = findMaxFreq(prefrange, currMax)
        suggestions.append(suggestion)
        
    return suggestions




def handler(conn, event):
    fn = {'init':    init,
          'keystroke':     keystroke}.get(event['op'], None)
    if fn != None:
        try:
            result = fn(event)
            return {'result':result}
        except Exception:
            return {'error': traceback.format_exc()}
    else:
        return {'error': 'bad op'}
"""
init(5)
t = time.time()
suggestions = keystroke("a")
print (time.time() - t)
print suggestions
res = handler(5, {"op":"init"})
print (res)
t = time.time()
res = handler(4, {"op":"keystroke", "pref": "ab"})
print (time.time() - t)
print res
"""
