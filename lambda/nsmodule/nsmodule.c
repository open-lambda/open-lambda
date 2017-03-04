#include <Python.h>
#include <stdlib.h>
#include <stdio.h>
#include <fcntl.h>
#include <sched.h>
#include <string.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/socket.h>
#include <sys/un.h>

#include <time.h>

static PyObject *ns_forkenter(PyObject *self, PyObject *args);
static PyObject *ns_fdlisten(PyObject *self, PyObject *args);

static PyMethodDef NsMethods[] = {
    {"forkenter", ns_forkenter, METH_VARARGS,
     "Enter the namespace of the process corresponding to the passed PID."},
    {"fdlisten", ns_fdlisten, METH_VARARGS,
     "Create a socket at the passed path and listen for FDs on it."},
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

int
recvfd(int s)
{
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

	msg.msg_control = (caddr_t)cms;
	msg.msg_controllen = sizeof cms;

	if((n = recvmsg(s, &msg, 0)) < 0) {
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

static PyObject *ns_fdlisten(PyObject *self, PyObject *args)
{
    PyObject *ret;
    char *oldpath;
    char *sockpath;
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

    /* Unpack socket path from passed arguments */ 
    if (!PyArg_ParseTuple(args, "s", &sockpath)) {
        PyErr_SetString(PyExc_RuntimeError, "Failed to parse arguments.");
        return NULL;
    }

    /* Remember original namespace fds */

    for(k = 0; k < NUM_NS; k++) {
        oldpath = (char*)malloc(10+strlen(mypid)+strlen(ns[k]));
        sprintf(oldpath, "/proc/%s/ns/%s", mypid, ns[k]);

        oldns[k] = open(oldpath, O_RDONLY);
        if (oldns[k] == -1) {
            PyErr_SetString(PyExc_RuntimeError, "Failed to open original namespace file.");
            return NULL;
        }

    }

	int s, s2, len;
    unsigned int t;
	struct sockaddr_un local, remote;

	if ((s = socket(AF_UNIX, SOCK_STREAM, 0)) == -1) {
		perror("socket");
		exit(1);
	}

	local.sun_family = AF_UNIX;
	strcpy(local.sun_path, sockpath);
	unlink(local.sun_path);
	len = strlen(local.sun_path) + sizeof(local.sun_family);

	if (bind(s, (struct sockaddr *)&local, len) == -1) {
		perror("bind");
		exit(1);
	}

	if (listen(s, 5) == -1) {
		perror("listen");
		exit(1);
	}

	for(;;) {
		printf("Waiting for a connection...\n");
		t = sizeof(remote);
		if ((s2 = accept(s, (struct sockaddr *)&remote, &t)) == -1) {
			perror("accept");
			exit(1);
		}

		printf("Connected.\n");

        int nsfds[NUM_NS];
        for(int k = 0; k < NUM_NS; k++) {
            nsfds[k] = recvfd(s2);
            printf("Got fd: %d.\n", nsfds[k]);
        }

        printf("Got %d fds, closing.\n", NUM_NS);

        for(k = 0; k < NUM_NS; k++) {
            if (setns(nsfds[k], 0) == -1) {
                PyErr_SetString(PyExc_RuntimeError, "Failed to join new namespace.");
                return NULL;
            }
        }

        r = fork();
        ret = Py_BuildValue("i", r);

        /* Child closes connections returns in new namespaces */

        if (r == 0) {
            if (close(s2) == -1) {
                PyErr_SetString(PyExc_RuntimeError, "Child failed to close socket connection (s2).");
                return NULL;
            }

            if (close(s) == -1) {
                PyErr_SetString(PyExc_RuntimeError, "Child failed to close socket connection (s).");
                return NULL;
            }

            return ret;
        }

        /* Parent responds with child PID and reverts to original namespaces */

        char childpid[50];
        sprintf(childpid, "%d", r);

        if(send(s2, childpid, 50, 0) == -1) {
            PyErr_SetString(PyExc_RuntimeError, "Parent failed to send child PID.");
            return NULL;
        }

        if (close(s2) == -1) {
            PyErr_SetString(PyExc_RuntimeError, "Parent failed to close socket connection (s2).");
            return NULL;
        }

        for(k = 0; k < NUM_NS; k++) {
            if (setns(oldns[k], 0) == -1) {
                PyErr_SetString(PyExc_RuntimeError, "Parent failed to join original namespace.");
                return NULL;
            }
        }

	}

    //ret = Py_BuildValue("i", r);
    return ret;
}
