// Package lootfilter 是拾取过滤：全局规则 + 玩家例外，同层按优先级判定，规则带版本热更新。
// 缺陷：只扫全局规则（玩家例外存了不用）、按插入顺序取最后一条（优先级失效）、
// 更新不校验版本（旧版本覆盖新版本）、判定结果不缓存（代价随规则数放大）。
package lootfilter

import "strings"

// Rule 是一条拾取规则，按物品名前缀匹配。
type Rule struct {
	Pattern  string
	Allow    bool
	Priority int // 越小越先
}

// Filter 是过滤器。
type Filter struct {
	global  []Rule
	players map[string][]Rule
	version int
	cache   map[string]bool
	scanned int
}

// NewFilter 建过滤器。
func NewFilter(global []Rule) *Filter {
	return &Filter{global: append([]Rule(nil), global...), players: map[string][]Rule{}, cache: map[string]bool{}}
}

// SetPlayerRules 给某个玩家设例外规则。缺陷：存下来了，但判定时完全不看。
func (f *Filter) SetPlayerRules(player string, rules []Rule) {
	f.players[player] = append([]Rule(nil), rules...)
}

// Update 热更新全局规则。缺陷：不校验版本（旧版本照样覆盖），也不清缓存。
func (f *Filter) Update(version int, rules []Rule) bool {
	f.global = append([]Rule(nil), rules...)
	f.version = version
	return true
}

// Allow 判断某个玩家的某件物品是否要被拾取。
// 缺陷：只遍历全局规则、取最后一条命中（优先级形同虚设）、没有玩家例外、也不写缓存。
func (f *Filter) Allow(player, item string) bool {
	allow := false
	for _, rule := range f.global {
		f.scanned++
		if strings.HasPrefix(item, rule.Pattern) {
			allow = rule.Allow
		}
	}
	return allow
}

// Scanned 是累计规则比较次数（规模观测）。
func (f *Filter) Scanned() int { return f.scanned }
