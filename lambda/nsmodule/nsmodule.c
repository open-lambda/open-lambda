#include <Python.h>
#include <arpa/inet.h>
#include <stdlib.h>
#include <stdio.h>
#include <sched.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/types.h>
#include <sys/wait.h>

static PyObject *ns_unshare(PyObject *module) {
    int res = unshare(CLONE_NEWUTS|CLONE_NEWPID|CLONE_NEWIPC);
    return Py_BuildValue("i", res);
}

static PyMethodDef NsMethods[] = {
    {"unshare", (PyCFunction)ns_unshare, METH_NOARGS, "unshare"},
    {NULL, NULL, 0, NULL}
};

static struct PyModuleDef nsMod = {
    PyModuleDef_HEAD_INIT,
    "ns",
    NULL,
    -1,
    NsMethods
};

PyMODINIT_FUNC
PyInit_ns(void)
{
    return PyModule_Create(&nsMod);
}
