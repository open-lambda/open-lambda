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

  printf("cgroup_init: unshare\n");

  // use new namespaces
  int flags = CLONE_NEWIPC|CLONE_NEWNS|CLONE_NEWNET|CLONE_NEWPID|CLONE_NEWUTS|CLONE_NEWUSER;
  res = unshare(flags);
  if (res != 0) {
    errExit("unshare failed");
  }

  // TODO: unmount stuff?
  
  // TODO: use new cgroups

  // use new root
  printf("cgroup_init: chroot\n");
  res = chroot(argv[1]);
  if (res != 0) {
    errExit("chroot failed");
  }

  printf("cgroup_init: chdir\n");
  res = chdir("/");
  if (res != 0) {
    errExit("chdir failed");
  }

  // start user proc
  printf("cgroup_init: execve\n");
  res = execve(argv[2], &argv[2], environ);
  printf("cgroup_init: failed\n");
  if (res != 0) {
    errExit("failed to do execve");
  }
  
  return 0;
}
