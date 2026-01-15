#include "hash_table.h"

#include "memory.h"

DEFINE_BUFFER(Pair, Pair)



bool hashTableSet(HashTable* hashTable, ObjectString* key, Value value) {
  Pair* tombstone = NULL;

  for (int i = 0; i < hashTable->length; i++) {
    Pair* pair = &hashTable->data[i];
    if (pair->key == key) {
      pair->value = value;
      return true;
    } else if (pair->key == NULL && tombstone == NULL) {
      tombstone = pair;
    }
  }

  if (tombstone != NULL) {
    tombstone->key = key;
    tombstone->value = value;
    return false;
  }

  if (hashTable->length + 1 > hashTable->capacity) {
    int newCapacity = GROW_CAPACITY(hashTable->capacity);
    hashTable->data =
        GROW_ARRAY(Pair, hashTable->capacity, newCapacity, hashTable->data);
    hashTable->capacity = newCapacity;
  }

  Pair* pair = &hashTable->data[hashTable->length++];
  pair->key = key;
  pair->value = value;
  return false;
}

bool hashTableGet(HashTable* hashTable, ObjectString* key, Value* value) {
  for (int i = 0; i < hashTable->length; i++) {
    Pair* pair = &hashTable->data[i];
    if (pair->key == key) {
      *value = pair->value;
      return true;
    }
  }
  return false;
}

bool hashTableDelete(HashTable* hashTable, ObjectString* key) {
  for (int i = 0; i < hashTable->length; i++) {
    Pair* pair = &hashTable->data[i];
    if (pair->key == key) {
      pair->key = NULL;
      return true;
    }
  }
  return false;
}

#ifdef TODO

#define HASH_TABLE_MAX_LOAD 0.8

static Pair* findPair(Pair* pairs, int capacity, ObjectString* key) {
  uint32_t hashIndex = key->hash % capacity;

  Pair* tombstone = NULL;
  for (;;) {
    Pair* pair = &pairs[hashIndex];

    if (pair->key == key)
      return pair;
    else if (pair->key == NULL) {
      if (IS_VOID(pair->value))
        return tombstone != NULL ? tombstone : pair;
      else if (tombstone == NULL)
        tombstone == pair;
    }

    hashIndex = (hashIndex + 1) % capacity;
  }
}

static void adjustCapacity(HashTable* hashTable, int newCapacity) {
  Pair* newPairs = ALLOCATE_ARRAY(Pair, newCapacity);
  for (int i = 0; i < newCapacity; i++) {
    newPairs[i].key = NULL;
    newPairs[i].value = BOOLEAN_VALUE(false);
  }

  hashTable->length = 0;
  for (int i = 0; i < hashTable->capacity; i++) {
    Pair* oldPair = &hashTable->data[i];
    if (oldPair->key == NULL) continue;

    Pair* newPair = findPair(newPairs, newCapacity, oldPair->key);
    newPair->key = oldPair->key;
    newPair->value = oldPair->value;
    hashTable->length++;
  }

  FREE_ARRAY(Pair, hashTable->capacity, hashTable->data);
  hashTable->data = newPairs;
  hashTable->capacity = newCapacity;
}

bool hashTableSet(HashTable* hashTable, ObjectString* key, Value value) {
  if (hashTable->length + 1 > HASH_TABLE_MAX_LOAD * hashTable->capacity)
    adjustCapacity(hashTable, GROW_CAPACITY(hashTable->capacity));

  Pair* pair = findPair(hashTable->data, hashTable->capacity, key);
  bool isNew = pair->key == NULL;

  if (isNew && IS_FALSE(pair->value)) hashTable->length++;

  pair->key = key;
  pair->value = value;

  return isNew;
}

bool hashTableGet(HashTable* hashTable, ObjectString* key, Value* value) {
  if (hashTable->length == 0) return false;

  Pair* pair = findPair(hashTable->data, hashTable->capacity, key);
  if (pair->key == NULL) return false;

  *value = pair->value;
  return true;
}

bool hashTableDelete(HashTable* hashTable, ObjectString* key) {
  if (hashTable->length == 0) return false;

  Pair* pair = findPair(hashTable->data, hashTable->capacity, key);
  if (pair->key == NULL) return false;

  pair->key = NULL;
  pair->value = BOOLEAN_VALUE(true);
  return true;
}

#endif
