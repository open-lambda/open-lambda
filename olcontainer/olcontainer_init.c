#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <fcntl.h>
#include <unistd.h>
#include <sched.h>
#include <string.h>
#include <sys/wait.h>

void errExit(char *msg) {
  perror(msg);
  exit(EXIT_FAILURE);
}

int main(int argc, char *argv[]) {
  int res, pid, status, fd;
  char buf[256] = { 0 };
  uid_t uid = geteuid();
  gid_t gid = getegid();

  if (argc < 3) {
    printf("Usage: init <root> <program> [ARGS...]\n");
    exit(EXIT_FAILURE);
  }

  res = unshare(CLONE_NEWIPC|CLONE_NEWNET|CLONE_NEWNS|CLONE_NEWPID|CLONE_NEWUSER|
      CLONE_NEWUTS);
  if (res != 0) {
    errExit("unshare failed");
  }

  if ((pid = fork()) == -1) {
    errExit("fork failed");
  } else if (pid != 0) { // parent
    if (waitpid(pid, &status, 0) == -1)
      errExit("waitpid failed");
    else if (WIFEXITED(status))
      return WEXITSTATUS(status);
    else if (WIFSIGNALED(status))
      kill(getpid(), WTERMSIG(status));
    errExit("child exit failed");
  }

  fd = open("/proc/self/setgroups", O_WRONLY);
  if (fd < 0)
      errExit("open failed");
  if (write(fd, "deny", 4) != 4)
      errExit("write failed");
  close(fd);

  fd = open("/proc/self/uid_map", O_WRONLY);
  if (fd < 0)
      errExit("open failed");
  snprintf(buf, sizeof(buf), "2000 %u 1", uid);
  if (write(fd, buf, strlen(buf)) != strlen(buf))
      errExit("write failed");
  close(fd);

  fd = open("/proc/self/gid_map", O_WRONLY);
  if (fd < 0)
      errExit("open failed");
  snprintf(buf, sizeof(buf), "2000 %u 1", gid);
  if (write(fd, buf, strlen(buf)) != strlen(buf))
      errExit("write failed");
  close(fd);

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
