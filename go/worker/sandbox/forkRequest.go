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
sendfds(int s, int *fds, int fdcount) {
	char buf[1];
	struct iovec iov;
	struct msghdr header;
	struct cmsghdr *cmsg;
	int n;
	char cms[CMSG_SPACE(sizeof(int) * fdcount)];

	buf[0] = 0;
	iov.iov_base = buf;
	iov.iov_len = 1;

	memset(&header, 0, sizeof header);
	header.msg_iov = &iov;
	header.msg_iovlen = 1;
	header.msg_control = (caddr_t)cms;
	header.msg_controllen = CMSG_LEN(sizeof(int) * fdcount);

	cmsg = CMSG_FIRSTHDR(&header);
	cmsg->cmsg_len = CMSG_LEN(sizeof(int) * fdcount);
	cmsg->cmsg_level = SOL_SOCKET;
	cmsg->cmsg_type = SCM_RIGHTS;
	memmove(CMSG_DATA(cmsg), fds, sizeof(int) * fdcount);

	if((n = sendmsg(s, &header, 0)) != iov.iov_len) {
		return -1;
	}

	return 0;
}

int
sendRootFD(char *sockPath, int chrootFD, int memFD) {
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
	int fds[2];
	fds[0] = chrootFD;
	fds[1] = memFD;
	if (sendfds(s, fds, 2) == -1) {
		return -1;
	}

	int status;
	if (read(s, &status, sizeof status) != sizeof status) {
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
	"os"
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
func (_ *SOCKContainer) forkRequest(fileSockPath string, rootDir *os.File, memCG *os.File) error {
	cSock := C.CString(fileSockPath)
	defer C.free(unsafe.Pointer(cSock))

	_, err := C.sendRootFD(cSock, C.int(rootDir.Fd()), C.int(memCG.Fd()))
	return err
}
