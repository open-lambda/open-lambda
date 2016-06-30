# source of words is http://norvig.com/google-books-common-words.txt
import time, traceback, sys
import rethinkdb as r
AC = 'ac' # DB
WORDS = 'words' # TABLE
LINE = 'line' # COLUMN
WORD = 'word' # COLUMN
FREQ = 'freq' # COLUMN
PREFS = 'prefs' # TABLE
PREF = 'pref' # COLUMN
LOWER = 'lower' # COLUMN
UPPER = 'upper' # COLUMN
   
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


def keystroke(conn, event):
    prefix = event['pref']
    prefix = prefix.lower()
    prefEntry = r.db(AC).table(PREFS).get(prefix).run(conn)
    lower = int(prefEntry[LOWER])
    upper = int(prefEntry[UPPER])
    prefrange = [lower, upper]
    suggestions = []
    currMax = sys.maxint
    posswords = r.db(AC).table(WORDS).between(lower + 1, upper + 1).run(conn)
    poss = list(posswords)
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
