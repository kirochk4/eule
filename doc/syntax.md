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
var name() {}
var name {}
var name() => void
var name => void

func() {}
func {}
func() => void
func => void

var tbl = {
  .prop() {},
  .prop {},
  .prop() => void,
  .prop => void,
}
```
