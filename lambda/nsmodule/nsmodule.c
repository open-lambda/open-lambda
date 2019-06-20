#include <Python.h>
#include <arpa/inet.h>
#include <stdlib.h>
#include <stdio.h>
#include <sched.h>
#include <sys/socket.h>
#include <sys/un.h>

static PyObject *ns_unshare(PyObject *module) {
    int res = unshare(CLONE_NEWUTS|CLONE_NEWPID|CLONE_NEWUTS);
    return Py_BuildValue("i", res);
}

static PyObject *ns_open_sock_file(PyObject *module, PyObject *arg) {
    char *path = PyUnicode_AsUTF8(arg);
    printf("ns_open_sock_file(%s)\n", path);

    int sock = socket(AF_UNIX, SOCK_STREAM, 0);
    if (sock == -1) {
	perror("socket");
	exit(1);
    }

    struct sockaddr_un local;
    local.sun_family = AF_UNIX;
    strcpy(local.sun_path, path);
    unlink(local.sun_path);
    int len = strlen(local.sun_path) + sizeof local.sun_family;

    if (bind(sock, (struct sockaddr*)&local, len) == -1) {
	perror("bind");
	exit(1);
    }

    if (listen(sock, 1) == -1) {
	perror("listen");
	exit(1);
    }

    return Py_BuildValue("i", sock);
}

static PyObject *ns_read_fd(PyObject *module, PyObject *arg) {
    int sock = (int)PyLong_AsLong(arg);
    printf("ns_read_fd(%d)\n", sock);

    // wait for connection
    struct sockaddr_un remote;
    unsigned int t = sizeof remote;
    int conn = accept(sock, (struct sockaddr *)&remote, &t);
    if (conn == -1) {
	perror("accept");
	exit(1);
    }

    // get FD over connection
    int n, fd;
    char cms[CMSG_SPACE(sizeof(int))];
    char buf[1];

    struct iovec iov;
    struct msghdr msg;
    struct cmsghdr *cmsg;

    iov.iov_base = buf;
    iov.iov_len = 1;

    memset(&msg, 0, sizeof msg);
    msg.msg_name = 0;
    msg.msg_namelen = 0;
    msg.msg_iov = &iov;
    msg.msg_iovlen = 1;

    msg.msg_control = cms;
    msg.msg_controllen = sizeof cms;

    if((n = recvmsg(sock, &msg, 0)) < 0) {
        perror("recvmsg");
	exit(1);
    }

    if(n == 0){
	perror("unexpected EOF");
	exit(1);
    }

    cmsg = CMSG_FIRSTHDR(&msg);
    memmove(&fd, CMSG_DATA(cmsg), sizeof(int));

    // close connection
    close(conn);

    return Py_BuildValue("i", fd);
}

static PyMethodDef NsMethods[] = {
    {"unshare", (PyCFunction)ns_unshare, METH_NOARGS, "unshare"},
    {"open_sock_file", (PyCFunction)ns_open_sock_file, METH_O, "open_sock_file"},
    {"read_fd", (PyCFunction)ns_read_fd, METH_O, "read_fd"},
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
