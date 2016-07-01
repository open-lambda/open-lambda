#!/usr/bin/env python
import os, sys, random, string
from common import *

NGINX_EXAMPLE = 'docker run -d -p 80:80 -v %s:/usr/share/nginx/html:ro nginx'
def main():
    if  len(sys.argv) == 1 or (len(sys.argv) == 2 and (sys.argv[1] == 'help' or sys.argv[1] == '-h' or sys.argv[1] == '--help')):
        print "setup.py takes two arguments: the application directory of the app you want to start and the lambda function of that app. For example, python setup.py pychat chat.py"
        print "Additional arguments can be used to specify other python scripts that need to run to setup an app. For example, the autocomplete app needs a script to set up the word database, so run python setup.py autocomplete autocomplete.py makeDB.py. "
        print "These scripts will be run after the lambda is made and pushed"
        print "APP" + '\t\t\t' + "COMMAND"
        print "pychat" + '\t       ' + " ./setup.py pychat chat.py"
        print "pyocr" + '\t\t' + "./setup.py pyocr ocr.py"
        print "pytix" + '\t\t' + "./setup.py pytix tix.py"
        print "autocomplete" + '\t' + "./setup.py autocomplete autocomplete.py makeDB.py"
        sys.exit()
    if len(sys.argv) < 3:
        print "You need to specify an application directory and lambda function"
        sys.exit()
    appNames = os.listdir(os.path.join(SCRIPT_DIR, "..",  "applications"))
    if sys.argv[1] not in appNames:
        print "That is not an application directory. Go to /applications."
        sys.exit()
    app_dir = os.path.join( SCRIPT_DIR, "..", "applications", sys.argv[1])
    app_files =  [z for z in os.listdir(app_dir) if os.path.isfile(os.path.join(app_dir, z))]   
    if sys.argv[2] not in app_files:
        print "That file is not in this directory"
        sys.exit()
    for i in range(3, len(sys.argv)):
        if sys.argv[i] not in app_files:
            print "One of your additional files is not the app directory"
            sys.exit()

    lambdaFn = sys.argv[2]
    app_name = ''.join(random.choice(string.ascii_lowercase) for _ in range(12))
    static_dir = os.path.join(app_dir, 'static')
    root_dir = os.path.join(app_dir, '..', '..')
    cluster_dir = os.path.join(root_dir, 'util', 'cluster')
    builder_dir = os.path.join(root_dir, 'lambda-generator')
    if not os.path.exists(cluster_dir):
        return 'cluster not running'

    # build image
    print '='*40
    print 'Building image'
    builder = os.path.join(builder_dir, 'builder.py')
    builder = builder + ' -n %s -l %s' %(app_name, os.path.join(app_dir, lambdaFn))
    if 'lambda-config.json' in app_files:
        builder = builder + ' -c %s' %(os.path.join(app_dir, 'lambda-config.json'))
    if 'environment.json' in app_files:
        builder = builder + ' -e %s' %(os.path.join(app_dir, 'environment.json'))

    run(builder, True)

    # push image
    print '='*40
    print 'Pushing image'
    registry = rdjs(os.path.join(cluster_dir, 'registry.json'))
    img = 'localhost:%s/%s' % (registry['host_port'], app_name)
    run('docker tag -f %s %s' % (app_name, img), True)
    run('docker push ' + img, True)

    # setup config
    balancer = rdjs(os.path.join(cluster_dir, 'loadbalancer-1.json'))
    config_file = os.path.join(static_dir, 'config.json')
    url = ("http://%s:%s/runLambda/%s" %
           (balancer['host_ip'], balancer['host_port'], app_name))
    wrjs(config_file, {'url': url})

    #run additional scripts
    for j in range(3, len(sys.argv)):
        spath = os.path.join(app_dir, sys.argv[j])
        spath = "python " + spath
        print '='*40
        print "running " + sys.argv[j]
        run(spath, True)

            
    # directions
    print '='*40
    print 'Consider serving the app with nginx as follows:'
    print NGINX_EXAMPLE % static_dir
    print "If that fails, try changing the port statement 80:80 to 81:80"
    return None

if __name__ == '__main__':
    rv = main()
    if rv != None:
        print 'ERROR: ' + rv
        sys.exit(1)
    sys.exit(0)
