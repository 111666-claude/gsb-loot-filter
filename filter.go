// Package lootfilter 是拾取过滤：玩家有例外规则时只用例外，否则用全局规则；
// 同层按「优先级升序 → 前缀长度降序 → deny 优先」取第一条命中，无命中默认不拾取；
// 全局规则带版本号热更新，判定结果按玩家+物品缓存，更新后失效。
//
// 为了不随规则数线性扫描，每层规则构造成一棵字节前缀树（trie）：
// 沿物品名向下走最多 len(item) 条边，途中优先级最小的规则即为答案，
// 单次摊还 O(物品名长度)，内存 O(规则总长度)。
package lootfilter

// Rule 是一条拾取规则，按物品名前缀匹配。
type Rule struct {
	Pattern  string
	Allow    bool
	Priority int // 越小越先
}

// trieNode 是前缀树的一个节点；节点自身带规则表示「该节点对应前缀」上有规则。
type trieNode struct {
	children map[byte]*trieNode
	allow    bool
	priority int
	set      bool
}

// cacheKey 是判定缓存的键。
type cacheKey struct {
	player string
	item   string
}

// Filter 是过滤器。
type Filter struct {
	global  *trieNode
	players map[string]*trieNode
	version int
	cache   map[cacheKey]bool
	scanned int
}

// NewFilter 建过滤器，全局规则版本从 0 开始。
func NewFilter(global []Rule) *Filter {
	return &Filter{
		global:  buildTrie(global),
		players: map[string]*trieNode{},
		cache:   map[cacheKey]bool{},
	}
}

// SetPlayerRules 给某个玩家设例外规则。玩家例外存在即覆盖全局规则（两层不混用）；
// 设空切片表示该玩家没有任何可用规则（一切不拾取）。设置后旧判定缓存失效。
func (f *Filter) SetPlayerRules(player string, rules []Rule) {
	f.players[player] = buildTrie(rules)
	f.cache = map[cacheKey]bool{}
}

// Update 热更新全局规则：只在 version 严格大于当前版本时生效（返回 true）；
// 否则返回 false，规则集、版本与缓存都不变。生效后旧判定缓存失效。
func (f *Filter) Update(version int, rules []Rule) bool {
	if version <= f.version {
		return false
	}
	f.global = buildTrie(rules)
	f.version = version
	f.cache = map[cacheKey]bool{}
	return true
}

// Allow 判断某个玩家的某件物品是否要被拾取：
// 玩家有例外规则时只查例外，否则查全局；无任何命中返回 false。
// 同一 (player, item) 的重复判定直接走缓存，不再产生比较次数。
func (f *Filter) Allow(player, item string) bool {
	key := cacheKey{player: player, item: item}
	if cached, ok := f.cache[key]; ok {
		return cached
	}
	root := f.global
	if playerTree, ok := f.players[player]; ok {
		root = playerTree
	}
	allow := matchTrie(root, item, &f.scanned)
	f.cache[key] = allow
	return allow
}

// Scanned 是累计规则比较次数（规模观测）：缓存命中不计，
// 未命中只按前缀树遍历的字符边数计数，即 O(物品名长度)，不随规则数放大。
func (f *Filter) Scanned() int { return f.scanned }

// buildTrie 把一组规则插入前缀树；同一前缀上的多条规则只保留判定序最靠前的一条。
func buildTrie(rules []Rule) *trieNode {
	root := &trieNode{}
	for _, rule := range rules {
		node := root
		for index := 0; index < len(rule.Pattern); index++ {
			ch := rule.Pattern[index]
			next := node.children[ch]
			if next == nil {
				if node.children == nil {
					node.children = map[byte]*trieNode{}
				}
				next = &trieNode{}
				node.children[ch] = next
			}
			node = next
		}
		if !node.set || ruleWins(rule.Priority, rule.Allow, node.priority, node.allow) {
			node.set = true
			node.priority = rule.Priority
			node.allow = rule.Allow
		}
	}
	return root
}

// ruleWins 判断候选规则 (priority, allow) 是否排在在位规则之前。
// 判定序：优先级升序 → 前缀长度降序（长度在调用处由 trie 深度天然保证）
// → 同优先级同前缀时 deny 优先。
func ruleWins(priority int, allow bool, oldPriority int, oldAllow bool) bool {
	if priority != oldPriority {
		return priority < oldPriority
	}
	return !allow && oldAllow
}

// matchTrie 沿物品名逐字符走前缀树，返回命中规则中判定序最靠前者的结论；
// 每经过一条边记一次比较。无命中（含空前缀树）返回 false。
func matchTrie(root *trieNode, item string, scanned *int) bool {
	bestSet := false
	bestPriority := 0
	bestAllow := false

	node := root
	if root.set {
		bestSet = true
		bestPriority = root.priority
		bestAllow = root.allow
	}
	for index := 0; index < len(item); index++ {
		next := node.children[item[index]]
		if next == nil {
			*scanned++
			return bestAllow
		}
		node = next
		*scanned++
		if node.set && (!bestSet || node.priority <= bestPriority) {
			bestSet = true
			bestPriority = node.priority
			bestAllow = node.allow
		}
	}
	return bestAllow
}
