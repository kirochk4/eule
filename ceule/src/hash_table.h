#ifndef HASH_TABLE_H
#define HASH_TABLE_H

#include "utils.h"
#include "value.h"

typedef struct {
  ObjectString* key;
  Value value;
} Pair;

DECLARE_BUFFER(Pair, Pair)

typedef PairBuffer HashTable;

inline void initHashTable(HashTable* hashTable) { initPairBuffer(hashTable); }
inline void freeHashTable(HashTable* hashTable) { freePairBuffer(hashTable); }

bool hashTableSet(HashTable* hashTable, ObjectString* key, Value value);
bool hashTableGet(HashTable* hashTable, ObjectString* key, Value* value);
bool hashTableDelete(HashTable* hashTable, ObjectString* key);

#endif  // HASH_TABLE_H