#include <unistd.h>

/* Lightweight "dummy" process to spin in containers */

int main() {
    while (1) {
        pause();
    }
}
