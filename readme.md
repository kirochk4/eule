# Eule - little scripting language

## Example

```js
let iterate(a) {
  let i = 0
  return def => a[i++]
}

let add(...a) {
  unless (a) return 0

  let result = 0
  foreach (elem in iterate(a))
    result += elem

  return result
}

assert(add(1, 2, 3, 4) == 10)
```
