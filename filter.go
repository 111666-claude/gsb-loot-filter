// Package lootfilter 是拾取过滤：全局规则 + 玩家例外，同层按优先级判定，规则带版本热更新。
//
// 口径（README 为准）：
//   - 生效规则集：玩家有例外规则就只用例外，否则用全局规则，两层不混用。
//   - 同层判定顺序：优先级升序 → 前缀长度降序 → deny 优先，第一条命中决定结果，无命中返回 false。
//   - 版本单调：Update 只在版本严格大于当前版本时生效。
//   - 缓存一致：Update 与 SetPlayerRules 之后旧判定缓存失效。
//   - 查询代价：前缀索引（trie）+ 判定缓存，单次摊还 O(物品名长度)。
package lootfilter

import "sort"

// Rule 是一条拾取规则，按物品名前缀匹配。
type Rule struct {
	Pattern  string
	Allow    bool
	Priority int // 越小越先
}

// less 是同层判定顺序：优先级升序 → 前缀长度降序 → deny 优先。
// 完全并列只可能是同 Pattern 同 Allow 的重复规则，故判定结果与插入顺序无关。
func less(a, b Rule) bool {
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	if len(a.Pattern) != len(b.Pattern) {
		return len(a.Pattern) > len(b.Pattern)
	}
	return !a.Allow && b.Allow
}

// index 是一层规则的前缀索引（trie）。节点记录以该前缀结尾、
// 同层判定名次最靠前的规则；查询沿物品名下行，取沿途最小名次。
type index struct {
	root   *node
	allows []bool // 名次 -> 该名次规则的 Allow
}

type node struct {
	children map[byte]*node
	rank     int // -1 表示没有规则以此前缀结尾
}

func newIndex(rules []Rule) *index {
	ordered := append([]Rule(nil), rules...)
	sort.SliceStable(ordered, func(i, j int) bool { return less(ordered[i], ordered[j]) })
	idx := &index{root: &node{rank: -1}}
	for rank, rule := range ordered {
		idx.allows = append(idx.allows, rule.Allow)
		cur := idx.root
		for i := 0; i < len(rule.Pattern); i++ {
			c := rule.Pattern[i]
			if cur.children == nil {
				cur.children = map[byte]*node{}
			}
			child, ok := cur.children[c]
			if !ok {
				child = &node{rank: -1}
				cur.children[c] = child
			}
			cur = child
		}
		if cur.rank < 0 { // 按名次升序插入，先到的名次最小，保留
			cur.rank = rank
		}
	}
	return idx
}

// match 返回命中规则的名次，steps 是走过的 trie 边数（规模观测）。
func (idx *index) match(item string) (rank int, steps int) {
	best := -1
	cur := idx.root
	if cur.rank >= 0 { // 空前缀规则命中一切
		best = cur.rank
	}
	for i := 0; i < len(item); i++ {
		child, ok := cur.children[item[i]]
		if !ok {
			break
		}
		steps++
		cur = child
		if cur.rank >= 0 && (best < 0 || cur.rank < best) {
			best = cur.rank
		}
	}
	return best, steps
}

// Filter 是过滤器。
type Filter struct {
	global  []Rule
	players map[string][]Rule
	version int
	cache   map[string]map[string]bool // 玩家 -> 物品 -> 判定结果
	scanned int

	globalIdx *index
	playerIdx map[string]*index
}

// NewFilter 建过滤器。
func NewFilter(global []Rule) *Filter {
	return &Filter{
		global:    append([]Rule(nil), global...),
		players:   map[string][]Rule{},
		cache:     map[string]map[string]bool{},
		globalIdx: newIndex(global),
		playerIdx: map[string]*index{},
	}
}

// SetPlayerRules 给某个玩家设例外规则；玩家有例外规则时判定只用例外。
// 传空规则等于清除例外（回退到全局规则）。调用后旧判定缓存失效。
func (f *Filter) SetPlayerRules(player string, rules []Rule) {
	if len(rules) == 0 {
		delete(f.players, player)
		delete(f.playerIdx, player)
	} else {
		f.players[player] = append([]Rule(nil), rules...)
		f.playerIdx[player] = newIndex(rules)
	}
	f.cache = map[string]map[string]bool{}
}

// Update 热更新全局规则。版本严格大于当前版本才生效（返回 true），
// 否则返回 false 且规则集不变。生效后旧判定缓存失效。
func (f *Filter) Update(version int, rules []Rule) bool {
	if version <= f.version {
		return false
	}
	f.version = version
	f.global = append([]Rule(nil), rules...)
	f.globalIdx = newIndex(rules)
	f.cache = map[string]map[string]bool{}
	return true
}

// Allow 判断某个玩家的某件物品是否要被拾取。
func (f *Filter) Allow(player, item string) bool {
	if byItem, ok := f.cache[player]; ok {
		if allow, ok := byItem[item]; ok {
			return allow
		}
	}
	idx := f.globalIdx
	if _, ok := f.players[player]; ok {
		idx = f.playerIdx[player]
	}
	rank, steps := idx.match(item)
	f.scanned += steps
	allow := rank >= 0 && idx.allows[rank]
	byItem, ok := f.cache[player]
	if !ok {
		byItem = map[string]bool{}
		f.cache[player] = byItem
	}
	byItem[item] = allow
	return allow
}

// Scanned 是累计前缀索引比较次数（规模观测）。
func (f *Filter) Scanned() int { return f.scanned }
