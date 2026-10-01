package lootfilter

import (
	"math/rand"
	"sort"
	"testing"
)

// refFilter 是 README 口径的朴素参考实现：每次判定都排序后线性扫描，
// 用 memo 按 (player, item) 记忆化判定结果，Update/SetPlayerRules 生效时整体失效，
// 用于和被测实现比对每次判定与缓存失效行为。
type refFilter struct {
	global  []Rule
	players map[string][]Rule
	version int
	memo    map[cacheKey]bool
}

func newRefFilter(global []Rule) *refFilter {
	return &refFilter{
		global:  append([]Rule(nil), global...),
		players: map[string][]Rule{},
		memo:    map[cacheKey]bool{},
	}
}

func (r *refFilter) setPlayerRules(player string, rules []Rule) {
	r.players[player] = append([]Rule(nil), rules...)
	r.memo = map[cacheKey]bool{}
}

func (r *refFilter) update(version int, rules []Rule) bool {
	if version <= r.version {
		return false
	}
	r.global = append([]Rule(nil), rules...)
	r.version = version
	r.memo = map[cacheKey]bool{}
	return true
}

func (r *refFilter) allow(player, item string) bool {
	key := cacheKey{player: player, item: item}
	if cached, ok := r.memo[key]; ok {
		return cached
	}
	r.sortRules()
	rules := r.global
	if playerRules, ok := r.players[player]; ok {
		rules = playerRules
	}
	for _, rule := range rules {
		if len(item) >= len(rule.Pattern) && item[:len(rule.Pattern)] == rule.Pattern {
			r.memo[key] = rule.Allow
			return rule.Allow
		}
	}
	r.memo[key] = false
	return false
}

func (r *refFilter) sortRules() {
	sort.SliceStable(r.global, func(i, j int) bool { return ruleLess(r.global[i], r.global[j]) })
	for player := range r.players {
		sort.SliceStable(r.players[player], func(i, j int) bool { return ruleLess(r.players[player][i], r.players[player][j]) })
	}
}

// ruleLess 是同层判定序：优先级升序 → 前缀长度降序 → deny 优先。
func ruleLess(a, b Rule) bool {
	if a.Priority != b.Priority {
		return a.Priority < b.Priority
	}
	if len(a.Pattern) != len(b.Pattern) {
		return len(a.Pattern) > len(b.Pattern)
	}
	return a.Allow != b.Allow && !a.Allow
}

// TestDifferentialReference 随机生成 200 组「全局规则 + 玩家例外 + 热更新 + 物品序列」，
// 与参考实现逐次比对判定结果与缓存失效行为，差异必须为 0。
func TestDifferentialReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20261002))
	atoms := []string{"", "a", "ab", "b", "gem", "gem_ruby", "gold", "p0", "p00", "石"}
	players := []string{"p", "vip", "guest"}

	randomRules := func() []Rule {
		count := rng.Intn(8)
		rules := make([]Rule, 0, count)
		for index := 0; index < count; index++ {
			rules = append(rules, Rule{
				Pattern:  atoms[rng.Intn(len(atoms))],
				Allow:    rng.Intn(2) == 0,
				Priority: rng.Intn(4),
			})
		}
		return rules
	}

	for group := 0; group < 200; group++ {
		initial := randomRules()
		filter := NewFilter(initial)
		ref := newRefFilter(initial)
		for step := 0; step < 40; step++ {
			switch rng.Intn(5) {
			case 0:
				rules := randomRules()
				player := players[rng.Intn(len(players))]
				filter.SetPlayerRules(player, rules)
				ref.setPlayerRules(player, rules)
			case 1:
				rules := randomRules()
				version := rng.Intn(6) - 1
				if got, want := filter.Update(version, rules), ref.update(version, rules); got != want {
					t.Fatalf("第 %d 组第 %d 步：Update(%d) 生效性 got=%v want=%v", group, step, version, got, want)
				}
			default:
				player := players[rng.Intn(len(players))]
				item := atoms[rng.Intn(len(atoms))] + atoms[rng.Intn(len(atoms))]
				_, filterCached := filter.cache[cacheKey{player: player, item: item}]
				_, refCached := ref.memo[cacheKey{player: player, item: item}]
				if got, want := filter.Allow(player, item), ref.allow(player, item); got != want {
					t.Fatalf("第 %d 组第 %d 步：Allow(%q, %q) got=%v want=%v", group, step, player, item, got, want)
				}
				if filterCached != refCached {
					t.Fatalf("第 %d 组第 %d 步：Allow(%q, %q) 缓存命中行为 got=%v want=%v", group, step, player, item, filterCached, refCached)
				}
			}
		}
	}
}
