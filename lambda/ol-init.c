#include <unistd.h>
#include <signal.h>
#include <stdio.h>
#include <stdlib.h>

/* Lightweight "dummy" process to spin in containers */

char **params;

void signal_handler() {
	int ret = fork();
	if (ret == 0) {
		execv(params[0], params);
	}

	return;
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

	signal(SIGUSR1, signal_handler);
	pause(); // wait for SIGUSR1 and handle
	pause(); // sleep forever, we're init for the ns

	return 0;
}
