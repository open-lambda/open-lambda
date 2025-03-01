#!/usr/bin/python

'''Apport package hook for shadow

(c) 2010 Canonical Ltd.
Contributors:
Marc Deslauriers <marc.deslauriers@canonical.com>

This program is free software; you can redistribute it and/or modify it
under the terms of the GNU General Public License as published by the
Free Software Foundation; either version 2 of the License, or (at your
option) any later version.  See http://www.gnu.org/copyleft/gpl.html for
the full text of the license.
'''

from apport.hookutils import *

def add_info(report):

    attach_file_if_exists(report, '/etc/login.defs', 'LoginDefs')

if __name__ == '__main__':
    report = {}
    add_info(report)
    for key in report:
        print('[%s]\n%s' % (key, report[key]))
