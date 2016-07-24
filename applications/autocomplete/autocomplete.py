# source of words is http://norvig.com/google-books-common-words.txt
import time, traceback, sys
import rethinkdb as r
AC = 'ac' # DB
WORDS = 'words' # TABLE
WORD = 'word' # COLUMN
FREQ = 'freq' # COLUMN

#get the most frequent words arising from a given prefix
def findMaxFreq(prefrange, currmax, conn, poss):
    maxFreqInd = -1
    maxFreq = -1
    count = 0
    maxWord = ""
    for ent in poss:
        if int(ent[FREQ]) > maxFreq and int(ent[FREQ]) < currmax:
            maxFreqInd = count
            maxFreq = int(ent[FREQ])
            maxWord = ent[WORD]
        count = count + 1
    return maxWord, maxFreq

#handle keystroke events
def keystroke(conn, event):
    prefix = event['pref']
    prefix = prefix.lower()
    punc = '"!.,?;:-0123456789()[]{}'
    if prefix == '' or prefix[-1:] in punc:
        return []
    elif prefix[0:1] in punc or prefix[0:1] == "'":
        prefix = prefix[1:]
        if prefix == '':
            return []
        elif prefix[0:1] in punc or prefix[0:1] == "'":
            prefix = prefix[1:]
    if prefix == '':
        return []
    lower = prefix + "a"
    upper = prefix + "zzzzzzzzzzzzzz"
    loweru = unicode(lower)
    upperu = unicode(upper)
    suggestions = []
    currMax = sys.maxint
    posswords = r.db(AC).table(WORDS).between(loweru, upperu, right_bound='closed').run(conn)
    poss = list(posswords)
    prefrange = []
    for i in range(5):
        suggestion, currMax = findMaxFreq(prefrange, currMax, conn, poss)
        if currMax != -1:
            suggestions.append(suggestion)
        else:
            break
    return suggestions

def handler(conn, event):
    fn = {'keystroke':     keystroke}.get(event['op'], None)
    if fn != None:
        try:
            result = fn(conn, event)
            return {'result':result}
        except Exception:
            return {'error': traceback.format_exc()}
    else:
        return {'error': 'bad op'}

