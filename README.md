# loot-filter

拾取过滤：全局规则 + 玩家例外，同层按优先级判定，规则带版本热更新，只用 Go 标准库。

```
go test ./...
go run ./cmd/loot-filter --sample priority
go run ./cmd/loot-filter --sample player
go run ./cmd/loot-filter --sample stale
```

## 口径（README 为准）

- **生效规则集**：玩家有例外规则就只用例外，否则用全局规则；两层不混用。
- **同层判定顺序**：优先级升序 → 前缀长度降序 → `deny` 优先，第一条命中决定结果；
  没有任何规则命中时返回 `false`。
- **版本单调**：`Update` 只在版本严格大于当前版本时生效（返回 `true`），否则返回 `false`
  且规则集不变。
- **缓存一致**：判定结果可以缓存，但 `Update` 与 `SetPlayerRules` 之后旧缓存必须失效。
- **查询代价**：`Allow` 不许随规则数线性扫描（前缀索引或缓存），
  `Scanned` 不随「判定次数 × 规则数」放大。
- **规模**：10 万条规则、每秒 100 万次判定，单次摊还 O(物品名长度)，内存 O(规则数)。

## 输出契约（不改格式）

```
allow=false
allow=true
ok=false allow=true
scanned<=4000
```

## 目录

```
filter.go               规则、例外与热更新
cmd/loot-filter         命令行入口
filter_test.go          go test 用例
```
