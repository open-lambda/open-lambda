#include <Python.h>
#include <stdlib.h>
#include <stdio.h>
#include <fcntl.h>
#include <sched.h>

#include <time.h>

static PyObject *ns_forkenter(PyObject *self, PyObject *args);

static PyMethodDef NsMethods[] = {
    {"forkenter", ns_forkenter, METH_VARARGS,
     "Enter the namespace of the process corresponding to the passed PID."},
     {NULL, NULL, 0, NULL}
};

PyMODINIT_FUNC initns(void)
{
    PyObject *m = Py_InitModule("ns", NsMethods);
    if (m == NULL)
        return;
}

static PyObject *ns_forkenter(PyObject *self, PyObject *args)
{
    PyObject *ret;
    char *pid;
    char *oldpath;
    char *newpath;
    int nsfd;
    int k;
    int r;

    int ipid = getpid();
    int pidlen = floor(log10(abs(ipid)));
    char mypid[pidlen];
    sprintf(mypid, "%d", ipid);

    /* Namespaces to be merged (all but 'user') - MUST merge 'mnt' last */

    const int NUM_NS = 6;
    int oldns[NUM_NS];
    const char *ns[NUM_NS];
    ns[0] = "cgroup";
    ns[1] = "ipc";
    ns[2] = "uts";
    ns[3] = "net";
    ns[4] = "pid";
    ns[5] = "mnt";

    /* Unpack pid from passed arguments */ 
    if (!PyArg_ParseTuple(args, "s", &pid)) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to parse arguments.");
        return NULL;
    }

    /* Merge each namespace, remembering original namespace fds */

    for(k = 0; k < NUM_NS; k++) {
        oldpath = (char*)malloc(10+strlen(mypid)+strlen(ns[k]));
        sprintf(oldpath, "/proc/%s/ns/%s", mypid, ns[k]);

        oldns[k] = open(oldpath, O_RDONLY);
        if (oldns[k] == -1) {
            PyErr_SetString(PyExc_RuntimeError, "Failed to open original namespace file.");
            return NULL;
        }

        newpath = (char*)malloc(10+strlen(pid)+strlen(ns[k]));
        sprintf(newpath, "/proc/%s/ns/%s", pid, ns[k]);

        nsfd = open(newpath, O_RDONLY);
        if (nsfd == -1) {
            PyErr_SetString(PyExc_RuntimeError, "Failed to open new namespace file.");
            return NULL;
        }

        if (setns(nsfd, 0) == -1) {
            PyErr_SetString(PyExc_RuntimeError, "Failed to join new namespace.");
            return NULL;
        }
    }

    r = fork();
    ret = Py_BuildValue("i", r);

    /* Child returns in new namespaces */

    if (r == 0) {
        return ret;
    }

    /* Parent reverts to original namespaces */

    for(k = 0; k < NUM_NS; k++) {
        if (setns(oldns[k], 0) == -1) {
            PyErr_SetString(PyExc_RuntimeError, "Failed to join original namespace.");
            return NULL;
        }
    }

    ret = Py_BuildValue("i", r);
    return ret;
}
