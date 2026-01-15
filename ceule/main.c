#include <stdio.h>
#include <string.h>

int main(int argc, char** argv) {
  if (argc == 1) {
    printf("Usage: eule [script]\n");
  } else if (argc == 2 && strcmp(argv[1], "--help") == 0) {
    printf("Usage: eule [script]\n");
  }
  int a = 0;
  return 0;
}