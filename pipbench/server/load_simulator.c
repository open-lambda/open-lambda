#include <Python.h>

static PyObject *
simulate(PyObject *self, PyObject *args) {
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
    printf("Load Simulator Completed\n");
    return Py_BuildValue("");
}

static PyMethodDef LoadSimulatorMethods[] =
{
    {"simulate", simulate, METH_VARARGS, "simulate usage"},
    { NULL, NULL, 0, NULL }
};

PyMODINIT_FUNC
initload_simulator(void)
{
    (void) Py_InitModule("load_simulator", LoadSimulatorMethods);
}

