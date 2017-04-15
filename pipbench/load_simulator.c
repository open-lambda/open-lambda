#include <Python.h>
#include <sys/time.h>
#include <sys/resource.h>

static PyObject *
simulate_load(PyObject *self, PyObject *args) {
    long cpu_time_ms;
    long mem_bytes;
    int is_import;
    char *p;
    struct rusage r;
    int j;

    if (!PyArg_ParseTuple(args, "LLi", &cpu_time_ms, &mem_bytes, &is_import)) {
        return NULL;
    }
    if (is_import) {
        p = malloc(mem_bytes);
    }

    int ONE_M = 1000000;
    while(1) {
        if (getrusage(RUSAGE_SELF, &r) == -1) {
            printf("getrusage() failed");
            exit(1);
        }
        long total_cpu_time_us = r.ru_utime.tv_sec * ONE_M + r.ru_utime.tv_usec +
            r.ru_stime.tv_sec * ONE_M + r.ru_stime.tv_usec;
        if (total_cpu_time_us / 1000 >= cpu_time_ms)
            break;
        for (long i = 0; i < 1000000; i++) {
            j++;
        }
    }

    return Py_BuildValue("s#", p, mem_bytes);
}

static PyMethodDef LoadSimulatorMethods[] =
{
    {"simulate_load", simulate_load, METH_VARARGS, "simulate load"},
    { NULL, NULL, 0, NULL }
};

PyMODINIT_FUNC
initload_simulator(void)
{
    (void) Py_InitModule("load_simulator", LoadSimulatorMethods);
}

