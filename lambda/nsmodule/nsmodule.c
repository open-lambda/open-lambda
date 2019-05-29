#include <Python.h>
#include <arpa/inet.h>
#include <stdlib.h>
#include <stdio.h>
#include <fcntl.h>
#include <sched.h>
#include <string.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/wait.h>
#include <signal.h>

/* Python C wrapper declarations */

static PyObject *ns_reset(PyObject *self, PyObject *args);
static PyObject *ns_fdlisten(PyObject *self, PyObject *args);
static PyObject *ns_forkenter(PyObject *self, PyObject *args);

static PyMethodDef NsMethods[] = {
    {"fdlisten", ns_fdlisten, METH_VARARGS,
     "Create a socket at the passed path, listen for FDs on it, and forkenter."},
    {"forkenter", ns_forkenter, METH_VARARGS,
     "Fork a child into the namespace defined by the global namespace file descriptor array."},
    {"reset", ns_reset, METH_VARARGS,
     "Reset global variables."},
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

/* Global variables */

int sock, conn, initialized;
const int NUM_NS = 4;
int newns[4], root_fd;

/* Helper functions */
/*
 * Initializes and binds a unix socket at the passed path. The
 * socket is not returned, but rather stored in the global
 * variable, "sock."
 *
 * Returns 0 on success, -1 on error.
 */
int initSock(char *sockpath) {
    int len;
    struct sockaddr_un local;

    if ((sock = socket(AF_UNIX, SOCK_STREAM, 0)) == -1) {
	perror("socket");
	exit(1);
    }

    local.sun_family = AF_UNIX;
    strcpy(local.sun_path, sockpath);
    unlink(local.sun_path);
    len = strlen(local.sun_path) + sizeof(local.sun_family);

    if (bind(sock, (struct sockaddr*)&local, len) == -1) {
	perror("bind");
	exit(1);
    }

    // notify worker that the socket is ready
    int pipefd = open("/host/server_pipe", O_WRONLY);
    if (pipefd < 0) {
        perror("open");
        exit(1);
    }
    if (write(pipefd, "ready", strlen("ready")) < 0) {
        perror("write");
        exit(1);
    }
    close(pipefd);

    return 0;
}

/*  
 * Listens on unix domain socket passed as a file descriptor
 * for a single file descriptor sent using the corresponding
 * sendmsg system call.
 *
 * Returns the received file descriptor or -1 on error.
 */
int recvfd(int sock) {
    int n, fd;
    char cms[CMSG_SPACE(sizeof(int))], buf[1];

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
	return -1;
    }

    if(n == 0){
	perror("unexpected EOF");
	return -1;
    }

    cmsg = CMSG_FIRSTHDR(&msg);
    memmove(&fd, CMSG_DATA(cmsg), sizeof(int));

    return fd;
}

/* Python C wrapper functions */

/*  
 * Listens on unix domain socket at the passed path, receiving
 * 6 file descriptors followed by a string with a whitespace-
 * delimited list of packages to import.
 *
 * Returns the package list for importing in Python interpreter.
 */
static PyObject *ns_reset(PyObject *self, PyObject *args) {
    PyObject *ret;
    if (!initialized) {
        PyErr_SetString(PyExc_RuntimeError, "Trying to reset but never initialized.");
        return NULL;
    } else {
        sock = 0;
        conn = 0;
        memset(newns, 0, sizeof(newns));
        initialized = 0;
    }

    ret = Py_BuildValue("i", 0);
    return ret;
}

int recv_all(int sockfd, void *buf, int len, int flags) {
    int rc;
    while (len > 0) {
        rc = recv(sockfd, buf, len, flags);
        if (rc < 0)
            return rc;
        buf += rc;
        len -= rc;
    }
    return 0;
}

/*  
 * Listens on unix domain socket at the passed path, receiving
 * 6 file descriptors followed by a string with a whitespace-
 * delimited list of packages to import.
 *
 * Returns the package list for importing in Python interpreter.
 */
static PyObject *ns_fdlisten(PyObject *self, PyObject *args) {
    char *buf;
    int read_size = 0;
    struct sockaddr_un remote;
    int k, len;
    unsigned int t;
    PyObject *ret;

    if (!initialized) {
        char *sockpath;
        /* Unpack socket path from arguments */ 
        if (!PyArg_ParseTuple(args, "s", &sockpath)) {
            PyErr_SetString(PyExc_RuntimeError, "Failed to parse arguments.");
            return NULL;
        }

        /* Bind socket */
        PySys_WriteStdout("ns_fdlisten: BIND SOCKET\n");
        fflush(stdout);
        if(initSock(sockpath) == -1) {
            PyErr_SetString(PyExc_RuntimeError, "Failed to initialize socket.");
            return NULL;
        }

	if (listen(sock, 1) == -1) {
	    PyErr_SetString(PyExc_RuntimeError, "Listen on socket failed.");
	    return NULL;
	}

        initialized = 1;
    }

    t = sizeof(remote);
    if ((conn = accept(sock, (struct sockaddr *)&remote, &t)) == -1) {
        PyErr_SetString(PyExc_RuntimeError, "Accepting connection from socket failed.");
        return NULL;
    }

    PySys_WriteStdout("ns_fdlisten: Receive fds:");
    for(k = 0; k < NUM_NS; k++) {
        newns[k] = recvfd(conn);
        PySys_WriteStdout(" %d", newns[k]);
    }
    PySys_WriteStdout("\n");


    root_fd = recvfd(conn);
    PySys_WriteStdout("root_fd: %d\n", root_fd);

    // how many bytes to receive?

    if((len = recv(conn, &read_size, sizeof(int), 0)) == -1) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to read rootdir size.");
        return NULL;
    }
    read_size = ntohl(read_size);

    buf = malloc(read_size);
    if (buf == NULL) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to allocation memory.");
        return NULL;
    }
    if((len = recv_all(conn, buf, read_size, 0)) == -1) {
        free(buf);
        PyErr_SetString(PyExc_RuntimeError, "Receiving package string from socket failed.");
        return NULL;
    }

    PySys_WriteStdout("ns_fdlisten: Receive packages string: \"%s\"\n", buf);

    ret = Py_BuildValue("s", buf);
    free(buf);
    return ret;
}

/*  
 * Enter the namespace specified by the file descriptors stored
 * in "newns," then fork.
 * 
 * Child closes socket connections and returns in the new namespaces.
 * Parent responds with the PID of the child and reverts to its
 * original namespaces.
 *
 * Assumes that ns_fdlisten has been called previously to initialize
 * the global sockets and "newns" file descriptors.
 *
 * Returns 0 if child, PID of the child if parent.
 */
static PyObject *ns_forkenter(PyObject *self, PyObject *args) {
    PyObject *ret;
    int pipefd[2];
    char childpid[50];
    int k, r, status, gc_pid;

    /* Prevent zombie child processes by double-forking */

    gc_pid = 0;
    if(pipe(pipefd) < 0) {
        PyErr_SetString(PyExc_RuntimeError, "Create pipe failed.");
        return NULL;
    }

    fflush(stdout);
    fflush(stderr);
    /* Original process */
    if ((r = fork())) {
        close(pipefd[1]); // close pipe write fd

        // read grandchild pid from pipe (blocks)
        if(read(pipefd[0], &gc_pid, sizeof(gc_pid)) < 0) {
            PyErr_SetString(PyExc_RuntimeError, "Read from PID pipe failed.");
            return NULL;
        }

        // close pipe read fd and wait for child to die
        close(pipefd[0]); // close pipe read fd
        waitpid(r, &status, 0); // wait for child to die

	/* Child process */
    } else if (!r) {
        // join the passed namespaces
        for(k = 0; k < NUM_NS; k++) {
            if (setns(newns[k], 0) == -1) {
                PyErr_SetString(PyExc_RuntimeError, "setns failed. Couldn't join new namespace.");
                return NULL;
            }
        }

        /* Child process */
        if ((gc_pid = fork())) {
            close(pipefd[0]); // close pipe read fd

            // write grandchild pid to pipe
            if(write(pipefd[1], &gc_pid, sizeof(gc_pid)) < 0) {
                PyErr_SetString(PyExc_RuntimeError, "Write to PID pipe failed.");
                return NULL;
            }

            // close pipe write fd and die
            close(pipefd[1]);
            exit(0);

	    /* Grandchild process */
        } else if (!gc_pid) {
            // chroot to new root directory
            if (fchdir(root_fd) != 0) {
                PyErr_SetString(PyExc_RuntimeError, "Fchdir failed.");
		return NULL;
	    }
            if(chroot(".") != 0 ) {
                PyErr_SetString(PyExc_RuntimeError, "Chroot failed.");
                return NULL;
            }

            // just close pipe fds
            close(pipefd[0]);
            close(pipefd[1]);
        } else {
            PyErr_SetString(PyExc_RuntimeError, "Second fork failed.");
            return NULL;
        }
    } else {
        PyErr_SetString(PyExc_RuntimeError, "First fork failed.");
        return NULL;
    }

    /* Close the passed file descriptors */
    for(k = 0; k < NUM_NS; k++) {
        if (close(newns[k]) == -1) {
            PyErr_SetString(PyExc_RuntimeError, "close passed ns file descriptor failed.");
            return NULL;
        }
    }

    if (close(root_fd) != 0) {
	PyErr_SetString(PyExc_RuntimeError, "close passed root file descriptor failed.");
	return NULL;
    }

    ret = Py_BuildValue("i", gc_pid);

    /* Grandchild closes connections returns in new namespaces */

    if (!gc_pid) {
        /*
	  if (close(conn) == -1) {
	  PyErr_SetString(PyExc_RuntimeError, "Child failed to close socket connection (s2).");
	  return NULL;
	  }

	  if (close(sock) == -1) {
	  PyErr_SetString(PyExc_RuntimeError, "Child failed to close socket connection (s).");
	  return NULL;
	  }
        */

        return ret;
    }

    /* Parent responds with grandchild PID and returns in original namespaces */

    sprintf(childpid, "%d", gc_pid);
    PySys_WriteStdout("ns_forkenter: child pid: %s\n", childpid);

    if(send(conn, childpid, 50, 0) == -1) {
        PyErr_SetString(PyExc_RuntimeError, "Parent failed to send child PID.");
        return NULL;
    }

    if (close(conn) == -1) {
        PyErr_SetString(PyExc_RuntimeError, "Parent failed to close socket connection (s2).");
        return NULL;
    }

    return ret;
}
