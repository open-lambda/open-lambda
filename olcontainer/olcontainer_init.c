#define _GNU_SOURCE

#include <stdio.h>
#include <stdlib.h>
#include <getopt.h>
#include <fcntl.h>
#include <unistd.h>
#include <sched.h>
#include <string.h>
#include <sys/wait.h>
#include <sys/mount.h>

void errExit(char *msg) {
  perror(msg);
  exit(EXIT_FAILURE);
}

void usage() {
  printf("Usage: init <root> <program> [ARGS...]\n");
  exit(EXIT_FAILURE);
}

static unsigned long parse_propagation(const char *str)
{
	size_t i;
	static const struct prop_opts {
		const char *name;
		unsigned long flag;
	} opts[] = {
		{ "slave",	MS_REC | MS_SLAVE },
		{ "private",	MS_REC | MS_PRIVATE },
		{ "shared",     MS_REC | MS_SHARED },
		{ "unchanged",        0 }
	};

	for (i = 0; i < sizeof(opts)/sizeof(opts[0]); i++) {
		if (strcmp(opts[i].name, str) == 0)
			return opts[i].flag;
	}

	exit(EXIT_FAILURE);
}

int main(int argc, char *argv[]) {
  int res, status, pid;
  int unshare_flags = 0;
	int propagation = 0;
  char c;

  enum {
    OPT_PROPAGATION
  };
  static const struct option longopts[] = {
    { "mount",         optional_argument, NULL, 'm'             },
    { "uts",           optional_argument, NULL, 'u'             },
    { "ipc",           optional_argument, NULL, 'i'             },
    { "net",           optional_argument, NULL, 'n'             },
    { "pid",           optional_argument, NULL, 'p'             },
    { "user",          optional_argument, NULL, 'U'             },
    { "propagation",   required_argument, NULL, OPT_PROPAGATION },
    { NULL, 0, NULL, 0 }
  };

  while ((c = getopt_long(argc, argv, "+muinpCU", longopts, NULL)) != -1) {
    switch (c) {
    case 'm':
      unshare_flags |= CLONE_NEWNS;
      break;
    case 'u':
      unshare_flags |= CLONE_NEWUTS;
      break;
    case 'i':
      unshare_flags |= CLONE_NEWIPC;
      break;
    case 'n':
      unshare_flags |= CLONE_NEWNET;
      break;
    case 'p':
      unshare_flags |= CLONE_NEWPID;
      break;
    case 'U':
      unshare_flags |= CLONE_NEWUSER;
      break;
		case OPT_PROPAGATION:
			propagation = parse_propagation(optarg);
			break;
    default:
      exit(EXIT_FAILURE);
    }
  }

  if (argc - optind < 2) {
    usage();
  }

  res = unshare(unshare_flags);
  if (res != 0) {
    errExit("unshare failed");
  }

  int pipefd[2];
  if (pipe(pipefd) < 0) {
      errExit("fork");
  }

  if ((pid = fork()) == -1) {
    errExit("fork failed");
  } else if (pid != 0) { // parent
    close(pipefd[0]);
    if (write(pipefd[1], (void *) &pid, sizeof(pid)) < 0) {
        errExit("write");
    }
    close(pipefd[1]);

    if (waitpid(pid, &status, 0) == -1)
      errExit("waitpid failed");
    else if (WIFEXITED(status))
      return WEXITSTATUS(status);
    else if (WIFSIGNALED(status))
      kill(getpid(), WTERMSIG(status));
    errExit("child exit failed");
  }

  if ((unshare_flags & CLONE_NEWNS) && propagation) {
    res = mount("none", "/", NULL, propagation, NULL);
    if (res != 0) {
        errExit("mount failed");
    }
  }

  // use new root
  res = chroot(argv[optind]);
  if (res != 0) {
    errExit("chroot failed");
  }

  res = chdir("/");
  if (res != 0) {
    errExit("chdir failed");
  }

  // notify worker our pid
  close(pipefd[1]);
  if (read(pipefd[0], (void *) &pid, sizeof(pid)) < 0) {
      errExit("read");
  }
  close(pipefd[1]);

  int fd = open("/host/pipe", O_RDWR);
  if (fd < 0) {
      fprintf(stderr, "cannot open pipe\n");
      exit(1);
  }
  char buf[6];
  snprintf(buf, 6, "%d", pid);
  if (write(fd, buf, 5) < 0) {
      perror("write");
      exit(1);
  }
  close(fd);

  // start user proc
  res = execve(argv[optind+1], &argv[optind+1], environ);
  fprintf(stderr, "cgroup_init: failed\n");
  if (res != 0) {
    errExit("failed to do execve");
  }
  
  return 0;
}
