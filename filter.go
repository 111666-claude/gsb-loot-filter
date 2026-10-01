// Package lootfilter 是拾取过滤：按规则优先级判定是否拾取，支持批量判定与结果缓存。
// 缺陷：取最后一条匹配的规则（不看优先级）、未匹配时默认拾取、批量判定每条都全扫规则表。
package lootfilter

// Rule 是一条拾取规则，按物品名前缀匹配。
type Rule struct {
	Pattern  string
	Allow    bool
	Priority int // 越小越先
}

// Filter 是过滤器。
type Filter struct {
	rules   []Rule
	cache   map[string]bool
	scanned int
}

// NewFilter 建过滤器。
func NewFilter(rules []Rule) *Filter {
	return &Filter{rules: append([]Rule(nil), rules...), cache: map[string]bool{}}
}

// Allow 判断某件物品是否要被拾取。
// 缺陷：遍历所有规则取最后一条匹配（优先级形同虚设）、没有规则匹配时默认允许、也不写缓存。
func (f *Filter) Allow(item string) bool {
	allow := true
	for _, rule := range f.rules {
		f.scanned++
		if len(item) >= len(rule.Pattern) && item[:len(rule.Pattern)] == rule.Pattern {
			allow = rule.Allow
		}
	}
	return allow
}

// AllowBatch 批量判定，返回允许拾取的数量。
func (f *Filter) AllowBatch(items []string) int {
	count := 0
	for _, item := range items {
		if f.Allow(item) {
			count++
		}
	}
	return count
}

// Scanned 是累计规则比较次数（规模观测）。
func (f *Filter) Scanned() int { return f.scanned }
