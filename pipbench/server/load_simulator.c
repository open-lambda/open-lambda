#include <Python.h>

static PyObject *
simulate_install(PyObject *self, PyObject *args) {
    long cpu_units;
    long mem_bytes;

    if (!PyArg_ParseTuple(args, "LL", &cpu_units, &mem_bytes)) {
        return NULL;
    }

    char *p;
    p = (char *) malloc(mem_bytes);

    int j = 0;
    for (long i = 0; i < cpu_units; i++) {
        j++;
    }

    free(p);
    return Py_BuildValue("i", 0);
}

static PyObject *
simulate_import(PyObject *self, PyObject *args) {
    long cpu_units;
    long mem_bytes;

    if (!PyArg_ParseTuple(args, "LL", &cpu_units, &mem_bytes)) {
        return NULL;
    }

    int j = 0;
    for (long i = 0; i < cpu_units; i++) {
        j++;
    }
    PyObject * p = (PyObject *) PyMem_Malloc(mem_bytes);
    return p;
}

static PyMethodDef LoadSimulatorMethods[] =
{
    {"simulate_install", simulate_install, METH_VARARGS, "simulate install"},
    {"simulate_import", simulate_import, METH_VARARGS, "simulate import"},
    { NULL, NULL, 0, NULL }
};

PyMODINIT_FUNC
initload_simulator(void)
{
    (void) Py_InitModule("load_simulator", LoadSimulatorMethods);
}

