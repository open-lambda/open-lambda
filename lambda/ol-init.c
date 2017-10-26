#include <unistd.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>
#include <fcntl.h>
#include <sys/types.h>
#include <sys/wait.h>

char **params;

/*
 * Launch the lambda server.
 */
void signal_handler() {
	if (fork() == 0) {
		execv(params[0], params);
		exit(1);
	}
}

/*
 * Install the handler and block all other signals while handling
 * the signal. Reset the signal handler after caught to default.
 */
void install_handler() {
	struct sigaction setup_action;
	sigset_t block_mask;

	sigfillset(&block_mask);
	setup_action.sa_handler = signal_handler;
	setup_action.sa_mask = block_mask;
	setup_action.sa_flags = SA_RESETHAND;
	sigaction(SIGURG, &setup_action, NULL);
}

int main(int argc, char *argv[]) {
	int k;

	params = (char**)malloc((3+argc-1)*sizeof(char*));
	params[0] = "/usr/bin/python";
	params[1] = "/server.py";
	for (k = 1; k < argc; k++) {
		params[k+1] = argv[k];
	}
	params[argc+1] = NULL;

	install_handler();

	// notify worker server that signal handler is installed throught stdout
    int fd = open("/host/pipe", O_RDWR);
    if (fd < 0) {
        fprintf(stderr, "cannot open pipe\n");
        exit(1);
    }
    if (write(fd, "ready", 5) < 0) {
        perror("write");
        exit(1);
    }
    close(fd);

	while (1) {
		pause(); // sleep forever, we're init for the ns
	}

	return 0;
}
