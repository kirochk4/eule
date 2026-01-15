#include "memory.h"

void* reallocate(void* pointer, size_t newSize, size_t oldSize) {
  if (newSize == 0) {
    free(pointer);
    return NULL;
  }
  if ((pointer = realloc(pointer, newSize)) == NULL) exit(1);
  return pointer;
}