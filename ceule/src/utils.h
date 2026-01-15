#ifndef UTILS_H
#define UTILS_H

#include "common.h"
#include "memory.h"

#define DECLARE_BUFFER(name, type)               \
  typedef struct {                               \
    int length;                                  \
    int capacity;                                \
    type* data;                                  \
  } name##Buffer;                                \
  void init##name##Buffer(name##Buffer* buffer); \
  void free##name##Buffer(name##Buffer* buffer);

#define DEFINE_BUFFER(name, type)                     \
  void init##name##Buffer(name##Buffer* buffer) {     \
    buffer->length = 0;                               \
    buffer->capacity = 0;                             \
    buffer->data = NULL;                              \
  }                                                   \
                                                      \
  void free##name##Buffer(name##Buffer* buffer) {     \
    FREE_ARRAY(type, buffer->capacity, buffer->data); \
    init##name##Buffer(buffer);                       \
  }

DECLARE_BUFFER(Byte, uint8_t)
DECLARE_BUFFER(Int, int)

#endif  // UTILS_H