#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

int
main(int argc, char *argv[]) {
    if (argc != 2) {
        printf("Usage: %s <comm_name>\n", argv[0]);
        exit(1);
    }
    FILE *comm;

    comm = fopen("/proc/self/task/1/comm", "w");
    if (fputs(argv[1], comm) < 0) {
        printf("Failed to change command name\n");
        exit(1);
    }
    pause();
    return 1;
}
