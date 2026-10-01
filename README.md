# loot-filter

拾取过滤：按规则优先级判定是否拾取，支持批量判定与结果缓存，只用 Go 标准库。

```
go test ./...
go run ./cmd/loot-filter --sample priority
go run ./cmd/loot-filter --sample default
go run ./cmd/loot-filter --sample ties
```

## 口径（README 为准）

- **按优先级取第一条匹配**：按 `Priority` 升序（同优先级按 `Pattern` 字典序，再按 `Allow`
  取 `false` 优先）匹配，第一条命中的规则决定结果。
- **默认不拾取**：没有任何规则匹配时返回 `false`。
- **批量判定要省**：`AllowBatch` 允许复用缓存/前缀索引，`Scanned` 不随「掉落数 × 规则数」放大。
- **不变量**：同一件物品重复判定的结果一致；规则的顺序打乱不改变判定结果。
- **规模**：1000 条规则、每秒 10 万次判定，单次摊还 O(1)（命中缓存）或 O(前缀深度)，内存 O(规则数)。

## 输出契约（不改格式）

```
allow=false
allow=false
allow=false
scanned<=2000
```

## 目录

```
filter.go               规则匹配
cmd/loot-filter         命令行入口
filter_test.go          go test 用例
```
