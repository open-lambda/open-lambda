#include <Python.h>
#include <sys/time.h>
#include <sys/resource.h>

#define MILLION 1000000

static PyObject *
simulate_load(PyObject *self, PyObject *args) {
    long cpu_time_ms;
    long mem_bytes;
    char *p;
    struct rusage r;
    int j;

    if (!PyArg_ParseTuple(args, "LL", &cpu_time_ms, &mem_bytes)) {
        return NULL;
    }

    p = malloc(mem_bytes);

    while(1) {
        if (getrusage(RUSAGE_SELF, &r) == -1) {
            printf("getrusage() failed");
            exit(1);
        }
        long total_cpu_time_us = r.ru_utime.tv_sec * MILLION + r.ru_utime.tv_usec +
            r.ru_stime.tv_sec * MILLION + r.ru_stime.tv_usec;
        if (total_cpu_time_us / 1000 >= cpu_time_ms)
            break;
        for (long i = 0; i < MILLION; i++) {
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

