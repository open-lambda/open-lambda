#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <sched.h>

void errExit(char *msg) {
  perror(msg);
  exit(EXIT_FAILURE);
}

int main(int argc, char *argv[]) {
  int res;

  if (argc < 3) {
    printf("Usage: init <root> <program> [ARGS...]\n");
    exit(EXIT_FAILURE);
  }

  // use new namespaces
  int flags = CLONE_NEWIPC|CLONE_NEWNS|CLONE_NEWNET|CLONE_NEWPID|CLONE_NEWUTS|CLONE_NEWUSER;
  res = unshare(flags);
  if (res != 0) {
    errExit("unshare failed");
  }

  // use new root
  res = chroot(argv[1]);
  if (res != 0) {
    errExit("chroot failed");
  }

  res = chdir("/");
  if (res != 0) {
    errExit("chdir failed");
  }

  // start user proc
  res = execve(argv[2], &argv[2], environ);
  printf("cgroup_init: failed\n");
  if (res != 0) {
    errExit("failed to do execve");
  }
  
  return 0;
}
