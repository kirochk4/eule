|           old           |              new              |
| :---------------------: | :---------------------------: |
|    `if (!condition)`    |     `unless (condition)`      |
|      `!condition`       |        `not condition`        |
| `condition ? one : two` | `condition then one else two` |
|  `while (!condition)`   |      `until (condition)`      |
|     `goto endloop`      |         `break loop`          |
|    `// linecomment`     |        `# linecomment`        |

#### function

```eul
let name() {}
let name {}
let name() => nil
let name => nil

def() {}
def {}
def() => nil
def => nil

let tbl = {
  .prop() {},
  .prop {},
  .prop() => nil,
  .prop => nil,
}
```
