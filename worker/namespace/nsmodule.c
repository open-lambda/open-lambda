#include <Python.h>
#include <stdlib.h>
#include <stdio.h>
#include <fcntl.h>
#include <sched.h>

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
    int pidfd;
    int mntfd;

    /* Fork first, parent returns */

    int r = fork();
    if (r < 0) {
        PyErr_SetString(PyExc_RuntimeError, "Fork failed.");
        return NULL;
    } else if (r > 0) {
        ret = Py_BuildValue("i", r);
        return ret;
    }

    /* Unpack pid from passed arguments */ 
    if (!PyArg_ParseTuple(args, "s", &pid)) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to parse arguments.");
        return NULL;
    }

    /* Format namespaces filepaths */

    char *pidPath = (char*)malloc(13+strlen(pid));
    char *mntPath = (char*)malloc(13+strlen(pid));

    sprintf(pidPath, "/proc/%s/ns/pid", pid);
    sprintf(mntPath, "/proc/%s/ns/mnt", pid);

    /* Open namespace fds */

    pidfd = open(pidPath, O_RDONLY);
    if (pidfd == -1) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to open pid file.");
        return NULL;
    }

    mntfd = open(mntPath, O_RDONLY);
    if (mntfd == -1) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to open mnt file.");
        return NULL;
    }

    /* Join namespaces */

    if (setns(pidfd, 0) == -1) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to join pid ns.");
        return NULL;
    }

    if (setns(mntfd, 0) == -1) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to join mnt ns.");
        return NULL;
    }

    /* Second fork, parent returns -1 to die */

    r = fork();
    if (r != 0) {
        ret = Py_BuildValue("i", -1);
        return ret;
    }

    ret = Py_BuildValue("i", r);
    return ret;
}
