package sandbox

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

int
sendRootFD(char *sockPath, char *rootdir) {
    int chrootFD = open(rootdir, O_RDONLY);
    if (chrootFD == -1) {
        return -1;
    }

    // Connect to server via socket.
    int s, len, ret;
    struct sockaddr_un remote;

    if ((s = socket(AF_UNIX, SOCK_STREAM, 0)) == -1) {
        return -1;
    }

    remote.sun_family = AF_UNIX;
    strcpy(remote.sun_path, sockPath);
    len = strlen(remote.sun_path) + sizeof(remote.sun_family);
    if (connect(s, (struct sockaddr *)&remote, len) == -1) {
        return -1;
    }

    printf("send chrootFD=%d\n", chrootFD);
    if (sendfd(s, chrootFD) == -1) {
        return -1;
    }

    int status;
    if (read(s, &status, sizeof status) != sizeof status) {
        return -1;
    }

    if(close(chrootFD) == -1) {
        return -1;
    }

    if(close(s) == -1) {
        return -1;
    }

    return status;
}

*/
import "C"

import (
	"fmt"
	"unsafe"
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
func (c *SOCKContainer) forkRequest(rootDir string) error {
	cSock := C.CString(fmt.Sprintf("%s/ol.sock", c.HostDir()))
	cRoot := C.CString(rootDir)

	defer C.free(unsafe.Pointer(cSock))
	defer C.free(unsafe.Pointer(cRoot))

	_, err := C.sendRootFD(cSock, cRoot)
	return err
}
