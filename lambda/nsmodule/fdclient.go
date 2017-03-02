package main

/*
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
#include "fdclient.h"

int
sendfd(int s, int fd)
{
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
        perror("sendmsg");
        exit(1);
    }

	return 0;
}

int
sendFds(char *sockPath, char *pid)
{
    char *path;
    int k;

    // Namespaces to be merged (all but 'user') - MUST merge 'mnt' last

    const int NUM_NS = 6;
    int nsfds[NUM_NS];
    const char *ns[NUM_NS];
    ns[0] = "cgroup";
    ns[1] = "ipc";
    ns[2] = "uts";
    ns[3] = "net";
    ns[4] = "pid";
    ns[5] = "mnt";

    // Get fds for all namespaces.

    for(k = 0; k < NUM_NS; k++) {
        path = (char*)malloc(10+strlen(pid)+strlen(ns[k]));
        sprintf(path, "/proc/%s/ns/%s", pid, ns[k]);

        nsfds[k] = open(path, O_RDONLY);
        if (nsfds[k] == -1) {
            perror("open");
            exit(1);
        }
    }

    // Connect to server via socket.

    int s, len, ret;
    struct sockaddr_un remote;

    if ((s = socket(AF_UNIX, SOCK_STREAM, 0)) == -1) {
        perror("socket");
        exit(1);
    }

    printf("Trying to connect...\n");

    remote.sun_family = AF_UNIX;
    strcpy(remote.sun_path, sockPath);
    len = strlen(remote.sun_path) + sizeof(remote.sun_family);
    if (connect(s, (struct sockaddr *)&remote, len) == -1) {
        perror("connect");
        exit(1);
    }

    printf("Connected.\n");

    // Send fds to server.

    printf("Sending fds.\n");

    for(k = 0; k < NUM_NS; k++) {
        if ((ret = sendfd(s, nsfds[k])) == -1) {
            perror("sendfd");
            //exit(1);
        }
    }

    int buf_len = 50;
    char buf[50];
    printf("Listening...\n");
    if((len = recv(s, buf, 50, 0)) == -1) {
        printf("Failed.\n");
        perror("recv pid");
    }

    printf("buffer: %s\n", buf);

    if(close(s) == -1) {
        perror("close socket");
    }

    return 0;
}
*/
import "C"

import (
	"fmt"
	"os"
)

const NUM_NS = 6

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <sockfile> <pid>\n", os.Args[0])
	}

	sockPath := os.Args[1]
	targetPid := os.Args[2]

	csock := C.CString(sockPath)
	cpid := C.CString(targetPid)

	C.sendFds(csock, cpid)

}
