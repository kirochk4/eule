#ifndef VALUE_H
#define VALUE_H

#include "common.h"
#include "hash_table.h"
#include "utils.h"

#define AS_BOOLEAN(value) ((bool)value.as.boolean)
#define AS_NUMBER(value) ((double)value.as.number)
#define AS_OBJECT(value) ((Object*)value.as.object)

#define IS_VOID(value) (value.type == eValueVoid)
#define IS_BOOLEAN(value) (value.type == eValueBoolean)
#define IS_NUMBER(value) (value.type == eValueNumber)
#define IS_OBJECT(value) (value.type == eValueObject)

#define IS_TRUE(value) (IS_BOOLEAN(value) && AS_BOOLEAN(value))
#define IS_FALSE(value) (IS_BOOLEAN(value) && !AS_BOOLEAN(value))

#define VOID_VALUE ((Value){eValueVoid, {.number = 0}})
#define BOOLEAN_VALUE(value) \
  ((Value){eValueBoolean, {.boolean = (value) == 0 ? false : true}})
#define NUMBER_VALUE ((Value){eValueNumber, {.number = (value)}})

#define OBJECT_TYPE(value) (AS_OBJECT(value)->type)

#define AS_STRING(value) ((ObjectString*)AS_OBJECT(value))
#define AS_TABLE(value) ((ObjectTable*)AS_OBJECT(value))
#define AS_FUNCTION(value) ((ObjectFunction*)AS_OBJECT(value))
#define AS_CLOSURE(value) ((ObjectClosure*)AS_OBJECT(value))
#define AS_UPVALUE(value) ((ObjectUpvalue*)AS_OBJECT(value))

#define IS_STRING(value) (isObjectType((value), eObjectString))
#define IS_TABLE(value) (isObjectType((value), eObjectTable))
#define IS_FUNCTION(value) (isObjectType((value), eObjectFunction))
#define IS_CLOSURE(value) (isObjectType((value), eObjectClosure))
#define IS_UPVALUE(value) (isObjectType((value), eObjectUpvalue))

typedef enum {
  eObjectString,
  eObjectTable,
  eObjectFunction,
  eObjectClosure,
  eObjectUpvalue,
} ObjectType;

typedef struct sObject {
  ObjectType type;
  struct sObject* next;
} Object;

typedef struct {
  int length;
  char* string;
  uint32_t hash;
} String;

typedef struct sTable {
  HashTable pairs;
  struct sTable* proto;
} Table;

typedef struct {
  ByteBuffer code;
  IntBuffer lines;
  ValueBuffer constants;
  int params;
} Function;

typedef struct {
  Function* fn;
  Upvalue* upvals;
} Closure;

typedef struct {
  Value* location;
  Value closed;
} Upvalue;

typedef enum {
  eValueVoid,
  eValueBoolean,
  eValueNumber,
  eValueObject,
} ValueType;

typedef struct sValue {
  ValueType type;
  union {
    bool boolean;
    double number;
    Object* object;
  } as;
} Value;

DECLARE_BUFFER(Value, Value)

static inline bool isObjectType(Value value, ObjectType type) {
  return IS_OBJECT(value) && OBJECT_TYPE(value) == type;
}

#endif  // VALUE_H