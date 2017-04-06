#include <stdio.h>
#include <stdlib.h>

#define MEM_B   1024*1024
#define CPU_C   100000000 // 100M

int main()
{
    char *p;
    p = (char *) malloc(MEM_B);
    free(p);
    int j = 0;
    for (int i = 0; i < CPU_C; i++) {
        j++;
    }
    return 0;
}