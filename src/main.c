#include <stdio.h>
#include <math.h>

void add(int a, int b, int c) {
    printf("%d + %d + %d = %d\n", a, b, c, a + b + c);
}

int main() {
    add(5, 3, 2);
    printf("Hello, World!\n");
    printf("sqrt(16) = %f\n", sqrt(16));
    return 0;
}