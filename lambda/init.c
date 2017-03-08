#include <unistd.h>

/* Lightweight "dummy" process to spin in containers */

int main() {
    pause();
}
