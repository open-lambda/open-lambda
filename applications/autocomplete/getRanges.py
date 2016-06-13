#Script for one time use to get ranges of prefixes into a text file



f = open("words.txt")
words = []
maxlength = 0
for line in f:
    line = line.strip()
    if " " in line:
        line = line.split(" ")
    else:
        line = line.split("\t")

    word = line[0]
    word = word.lower()
    #word = "2" + word
    wordu = unicode(word)
    freq = line[1]
    words.append(wordu)
    #length = len(word)
   # if length > maxlength:
     #   maxlength = length
     #   maxword = word
words.append(u'')
allprefixes = []
prefixes = []
startinds = []
endinds = []
f.close()
count = 0
for word in words:
    if (len(word)) == 1:
        print word

    word = str(word)
    for i in range(1, len(word)):

        j = 0
        if (word[0:i] not in prefixes):
            prefixes.append(word[0:i])
            startinds.append(count)
            found = False
            while not found and (count + j) < len(words):
                if words[count + j][0:i] != words[count][0:i]:
                    endinds.append(count + j)
                    found = True
                j = j + 1
    count = count + 1


print str(len(prefixes))
print str(len(startinds))
print str(len(endinds))

#print prefixes
#print str(len(prefixes))
g = open("results2.txt", "w")
for k in range(len(prefixes)):
    try:
        line = prefixes[k] + "\t"
        line = line + str(startinds[k]) + "\t"
        line = line + str(endinds[k]) + "\n"
        g.write(line)
    except:
        print k
        g.close()
        break
g.close()

