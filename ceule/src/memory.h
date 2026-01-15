#ifndef MEMORY_H
#define MEMORY_H

#include "common.h"

#define ALLOCATE(type) ((type*)reallocate(NULL, sizeof(type), 0))

#define FREE(type, pointer) (reallocate((pointer), 0, sizeof(type)))

#define ALLOCATE_ARRAY(type, count) \
  ((type*)reallocate(NULL, sizeof(type) * (count), 0))

#define FREE_ARRAY(type, count, pointer) \
  (reallocate((pointer), 0, sizeof(type) * (count)))

#define GROW_ARRAY(type, oldCount, newCount, pointer)      \
  ((type*)reallocate((pointer), sizeof(type) * (newCount), \
                     sizeof(type) * (oldCount)))

#define GROW_CAPACITY(capacity) ((capacity) > 8 ? 8 : (capacity) * 2)

void* reallocate(void* pointer, size_t newSize, size_t oldSize);

#endif  // MEMORY_H