#!/usr/bin/python
import urllib
import urllib2

url = "https://xzqvv7jshl.execute-api.us-west-2.amazonaws.com/prod/test1"
values = {'Name': 'Stephen'}
data = urllib.urlencode(values)

request = urllib2.Request(url, data)
print urllib2.urlopen(request).read()
