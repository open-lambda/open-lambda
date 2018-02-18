package cache

/*
#include <arpa/inet.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <string.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <sys/socket.h>
#include <sys/un.h>

char errmsg[1024];

int
sendfd(int s, int fd) {
	char buf[1];
	struct iovec iov;
	struct msghdr msg;
	struct cmsghdr *cmsg;
	int n;
	char cms[CMSG_SPACE(sizeof(int))];

	buf[0] = 0;
	iov.iov_base = buf;
	iov.iov_len = 1;

	memset(&msg, 0, sizeof msg);
	msg.msg_iov = &iov;
	msg.msg_iovlen = 1;
	msg.msg_control = (caddr_t)cms;
	msg.msg_controllen = CMSG_LEN(sizeof(int));

	cmsg = CMSG_FIRSTHDR(&msg);
	cmsg->cmsg_len = CMSG_LEN(sizeof(int));
	cmsg->cmsg_level = SOL_SOCKET;
	cmsg->cmsg_type = SCM_RIGHTS;
	memmove(CMSG_DATA(cmsg), &fd, sizeof(int));

	if((n = sendmsg(s, &msg, 0)) != iov.iov_len) {
        return -1;
    }

	return 0;
}

int sendAll(int sockfd, const void *buf, int len, int flags) {
	int rc;
	while (len > 0) {
		rc = send(sockfd, buf, len, flags);
		if (rc < 0)
			return rc;
		buf += rc;
		len -= rc;
	}
	return 0;
}

char*
sendFds(char *sockPath, char *pid, char *rootdir, int rootdirLen, char *pkgs, int pkgsLen) {
    char *path;
    int k;

    const int NUM_NS = 4;
    int nsfds[NUM_NS];
    const char *ns[NUM_NS];
    ns[0] = "ipc";
    ns[1] = "uts";
    ns[2] = "pid";
    ns[3] = "mnt";

    // Get fds for all namespaces.

    for(k = 0; k < NUM_NS; k++) {
        path = (char*)malloc(10+strlen(pid)+strlen(ns[k]));
        sprintf(path, "/proc/%s/ns/%s", pid, ns[k]);

        nsfds[k] = open(path, O_RDONLY);
        if (nsfds[k] == -1) {
            sprintf(errmsg, "open %s failed: %s\n", path, strerror(errno));
            return errmsg;
        }
    }

    int chrootFD = open(rootdir, O_RDONLY);
    if (chrootFD == -1) {
        sprintf(errmsg, "open %s failed: %s\n", rootdir, strerror(errno));
        return errmsg;
    }

    // Connect to server via socket.

    int s, len, ret;
    struct sockaddr_un remote;

    if ((s = socket(AF_UNIX, SOCK_STREAM, 0)) == -1) {
        sprintf(errmsg, "socket: %s\n", strerror(errno));
        return errmsg;
    }

    remote.sun_family = AF_UNIX;
    strcpy(remote.sun_path, sockPath);
    len = strlen(remote.sun_path) + sizeof(remote.sun_family);
    if (connect(s, (struct sockaddr *)&remote, len) == -1) {
        sprintf(errmsg, "connect: %s\n", strerror(errno));
        return errmsg;
    }

    // Send fds to server.

    for(k = 0; k < NUM_NS; k++) {
        if (sendfd(s, nsfds[k]) == -1) {
            sprintf(errmsg, "sendfd: %s\n", strerror(errno));
            return errmsg;
        }
    }

    if (sendfd(s, chrootFD) == -1) {
        sprintf(errmsg, "sendfd: %s\n", strerror(errno));
        return errmsg;
    }

    // Send package string to server.
    int conv_int = htonl(pkgsLen);
    if(send(s, &conv_int, sizeof(int), 0) == -1) {
        sprintf(errmsg, "send pkgs: %s\n", strerror(errno));
        return errmsg;
    }
    if(sendAll(s, pkgs, pkgsLen, 0) == -1) {
        sprintf(errmsg, "send pkgs: %s\n", strerror(errno));
        return errmsg;
    }

    // Receive spawned PID from server.

    char *buf = malloc(10);

    if((len = recv(s, buf, 10, 0)) == -1) {
        sprintf(errmsg, "recv: %s\n", strerror(errno));
        return errmsg;
    }

    if(close(s) == -1) {
        sprintf(errmsg, "close: %s\n", strerror(errno));
        return errmsg;
    }

    // Close fds for all namespaces.

    for(k = 0; k < NUM_NS; k++) {
        if(close(nsfds[k]) == -1) {
            sprintf(errmsg, "close: %s\n", strerror(errno));
            return errmsg;
        }
    }

    if(close(chrootFD) == -1) {
        sprintf(errmsg, "close: %s\n", strerror(errno));
        return errmsg;
    }

    return buf;
}
*/
import "C"

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unsafe"

	"github.com/open-lambda/open-lambda/worker/benchmarker"
)

/*
 * Send the namespace file descriptors for the targetPid process
 * and the passed package list to a lambda server listening on the
 * unix socket at sockPath.
 *
 * The packages in pkgList are assumed to be whitespace-delimited.
 *
 * Returns the PID of the spawned process upon success.
 */

func forkRequest(sockPath, targetPid, rootDir string, imports []string, handler bool) (string, error) {
	b := benchmarker.GetBenchmarker()
	var t *benchmarker.Timer
	if b != nil {
		t = b.CreateTimer("send fds", "us")
	}

	if t != nil {
		t.Start()
	}

	var signal string
	if handler {
		signal = "serve"
	} else {
		signal = "cache"
	}

	signalStr := strings.Join(append(imports, signal), " ")

	cSock := C.CString(sockPath)
	cPid := C.CString(targetPid)
	cRoot := C.CString(rootDir)
	cSignal := C.CString(signalStr)

	defer C.free(unsafe.Pointer(cSock))
	defer C.free(unsafe.Pointer(cPid))
	defer C.free(unsafe.Pointer(cRoot))
	defer C.free(unsafe.Pointer(cSignal))

	var err error
	for k := 0; k < 5; k++ {
		ret, err := C.sendFds(cSock, cPid, cRoot, C.int(len(rootDir)+1), cSignal, C.int(len(signalStr)+1))
		if err == nil {
			pid := C.GoString(ret)
			C.free(unsafe.Pointer(ret))
			if err != nil || pid == "" {
				err = fmt.Errorf("sendFds failed :: %s", err)
			} else {
				return pid, nil
			}
		} else {
			log.Printf("forkRequest: sendFds error: %v %v", ret, err)
		}
		time.Sleep(50 * time.Microsecond)
	}

	return "", err
}
