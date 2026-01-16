#ifndef VM_H
#define VM_H

#include "common.h"
#include "value.h"

typedef struct {
  int ip;
  Function* fn;
  Upvalue* upvals;
} CallFrame;

typedef struct {
  Value* stack;
  Value* st;
  CallFrame* callStack;
  CallFrame* cst;
  Upvalue* opened;
} VM;

int interpret(VM* vm, char* source);

#endif  // VM_H