package lootfilter

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
)

func TestDefaultDeny(t *testing.T) {
	if NewFilter(nil).Allow("p", "stone") {
		t.Fatal("没有规则命中时不该拾取")
	}
}

func TestSingleAllowRule(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	if !filter.Allow("p", "gold_coin") {
		t.Fatal("白名单前缀应该允许拾取")
	}
}

func TestExactDenyBeatsPrefixAllow(t *testing.T) {
	filter := NewFilter([]Rule{
		{Pattern: "gem", Allow: true, Priority: 2},
		{Pattern: "gem_ruby", Allow: false, Priority: 1},
	})
	if filter.Allow("p", "gem_ruby") {
		t.Fatal("更高优先级的禁止规则应该生效")
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if NewFilter(nil).Scanned() != 0 {
		t.Fatal("初始比较次数应该是 0")
	}
}

func TestPlayerRulesOverrideGlobal(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "stone", Allow: false, Priority: 1}})
	filter.SetPlayerRules("vip", []Rule{{Pattern: "stone", Allow: true, Priority: 1}})
	if !filter.Allow("vip", "stone") {
		t.Fatal("玩家例外规则应该盖过全局规则")
	}
	if filter.Allow("guest", "stone") {
		t.Fatal("没有例外的玩家应该只用全局规则")
	}
}

func TestPlayerRulesReplaceNotMix(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gem", Allow: true, Priority: 1}})
	filter.SetPlayerRules("vip", []Rule{{Pattern: "gem_ruby", Allow: false, Priority: 1}})
	if filter.Allow("vip", "gem_ore") {
		t.Fatal("两层不混用：有例外时全局的 gem 允许不该生效")
	}
	if filter.Allow("vip", "gem_ruby") {
		t.Fatal("例外层内部的 deny 应该生效")
	}
	if !filter.Allow("guest", "gem_ore") {
		t.Fatal("无例外玩家仍走全局规则")
	}
}

func TestEmptyPlayerRulesFallsBackToGlobal(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	filter.SetPlayerRules("vip", []Rule{{Pattern: "gold", Allow: false, Priority: 1}})
	if filter.Allow("vip", "gold_coin") {
		t.Fatal("例外生效时不该拾取")
	}
	filter.SetPlayerRules("vip", nil)
	if !filter.Allow("vip", "gold_coin") {
		t.Fatal("清空例外后应该回退到全局规则")
	}
}

func TestSameLayerOrder(t *testing.T) {
	filter := NewFilter([]Rule{
		{Pattern: "a", Allow: true, Priority: 3},
		{Pattern: "ab", Allow: false, Priority: 2},
		{Pattern: "abc", Allow: true, Priority: 1},
	})
	if !filter.Allow("p", "abc_x") {
		t.Fatal("优先级升序：Priority 1 的 abc 允许应该最先命中")
	}
	if filter.Allow("p", "ab_x") {
		t.Fatal("优先级升序：Priority 2 的 ab 禁止应该先于 Priority 3 的 a 允许")
	}
}

func TestPrefixLengthDescWithinPriority(t *testing.T) {
	filter := NewFilter([]Rule{
		{Pattern: "gem", Allow: true, Priority: 1},
		{Pattern: "gem_ruby", Allow: false, Priority: 1},
	})
	if filter.Allow("p", "gem_ruby") {
		t.Fatal("同优先级时更长的前缀应该优先")
	}
	if !filter.Allow("p", "gem_ore") {
		t.Fatal("长前缀不命中时短前缀兜底")
	}
}

func TestDenyFirstWithinSamePriorityAndLength(t *testing.T) {
	filter := NewFilter([]Rule{
		{Pattern: "gem", Allow: true, Priority: 1},
		{Pattern: "gem", Allow: false, Priority: 1},
	})
	if filter.Allow("p", "gem_ruby") {
		t.Fatal("同优先级同长度时 deny 应该优先")
	}
}

func TestUpdateRejectsStaleVersion(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	if !filter.Update(2, []Rule{{Pattern: "gold", Allow: false, Priority: 1}}) {
		t.Fatal("版本 2 应该生效")
	}
	if filter.Update(2, []Rule{{Pattern: "gold", Allow: true, Priority: 1}}) {
		t.Fatal("相同版本不该生效")
	}
	if filter.Update(1, []Rule{{Pattern: "gold", Allow: true, Priority: 1}}) {
		t.Fatal("更旧版本不该生效")
	}
	if filter.Allow("p", "gold_coin") {
		t.Fatal("被拒绝的更新不该改变规则集")
	}
}

func TestCacheInvalidatedOnUpdate(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	if !filter.Allow("p", "gold_coin") {
		t.Fatal("更新前应该允许")
	}
	filter.Update(1, []Rule{{Pattern: "gold", Allow: false, Priority: 1}})
	if filter.Allow("p", "gold_coin") {
		t.Fatal("Update 后旧缓存必须失效")
	}
}

func TestCacheInvalidatedOnSetPlayerRules(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	if !filter.Allow("vip", "gold_coin") {
		t.Fatal("设例外前应该允许")
	}
	filter.SetPlayerRules("vip", []Rule{{Pattern: "gold", Allow: false, Priority: 1}})
	if filter.Allow("vip", "gold_coin") {
		t.Fatal("SetPlayerRules 后旧缓存必须失效")
	}
}

func TestResultIndependentOfInsertionOrder(t *testing.T) {
	base := []Rule{
		{Pattern: "gem", Allow: true, Priority: 2},
		{Pattern: "gem_ruby", Allow: false, Priority: 1},
		{Pattern: "gem", Allow: false, Priority: 2},
		{Pattern: "g", Allow: true, Priority: 0},
	}
	reversed := make([]Rule, len(base))
	for i, rule := range base {
		reversed[len(base)-1-i] = rule
	}
	a, b := NewFilter(base), NewFilter(reversed)
	for _, item := range []string{"g", "gem", "gem_ruby", "gem_ore", "gold", "x"} {
		if a.Allow("p", item) != b.Allow("p", item) {
			t.Fatalf("插入顺序不该影响判定：%q", item)
		}
	}
}

func TestRepeatedQueryIsStable(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	first := filter.Allow("p", "gold_coin")
	for i := 0; i < 100; i++ {
		if filter.Allow("p", "gold_coin") != first {
			t.Fatal("同一输入重复判定结果必须相同")
		}
	}
}

func TestScannedDoesNotScaleWithQueries(t *testing.T) {
	rules := make([]Rule, 0, 1000)
	for i := 0; i < 1000; i++ {
		rules = append(rules, Rule{Pattern: fmt.Sprintf("p%04d", i), Allow: true, Priority: i})
	}
	filter := NewFilter(rules)
	for q := 0; q < 1000; q++ {
		filter.Allow("p", "p0999_x")
	}
	if filter.Scanned() > 4000 {
		t.Fatalf("Scanned 不该随判定次数×规则数放大，得到 %d", filter.Scanned())
	}
}

// reference 是题目口径的直白参考实现：生效规则集 + 同层排序 + 线性首命中。
type reference struct {
	global  []Rule
	players map[string][]Rule
	version int
}

func (r *reference) setPlayerRules(player string, rules []Rule) {
	if len(rules) == 0 {
		delete(r.players, player)
		return
	}
	r.players[player] = append([]Rule(nil), rules...)
}

func (r *reference) update(version int, rules []Rule) bool {
	if version <= r.version {
		return false
	}
	r.version = version
	r.global = append([]Rule(nil), rules...)
	return true
}

func (r *reference) allow(player, item string) bool {
	rules := r.global
	if playerRules, ok := r.players[player]; ok {
		rules = playerRules
	}
	ordered := append([]Rule(nil), rules...)
	sort.SliceStable(ordered, func(i, j int) bool { return less(ordered[i], ordered[j]) })
	for _, rule := range ordered {
		if strings.HasPrefix(item, rule.Pattern) {
			return rule.Allow
		}
	}
	return false
}

func TestDifferentialAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(20261001))
	alphabet := []byte("ab")
	randomPattern := func() string {
		n := rng.Intn(5)
		buf := make([]byte, n)
		for i := range buf {
			buf[i] = alphabet[rng.Intn(len(alphabet))]
		}
		return string(buf)
	}
	randomRules := func() []Rule {
		n := rng.Intn(12)
		rules := make([]Rule, n)
		for i := range rules {
			rules[i] = Rule{Pattern: randomPattern(), Allow: rng.Intn(2) == 0, Priority: rng.Intn(4)}
		}
		return rules
	}
	for trial := 0; trial < 200; trial++ {
		global := randomRules()
		filter := NewFilter(global)
		ref := &reference{global: append([]Rule(nil), global...), players: map[string][]Rule{}}
		players := []string{"vip", "guest", "bot"}
		version := 0
		for op := 0; op < 40; op++ {
			switch rng.Intn(4) {
			case 0: // 设/清玩家例外
				player := players[rng.Intn(len(players))]
				rules := randomRules()
				if rng.Intn(4) == 0 {
					rules = nil
				}
				filter.SetPlayerRules(player, rules)
				ref.setPlayerRules(player, rules)
			case 1: // 热更新（含过期版本）
				var next int
				switch rng.Intn(3) {
				case 0:
					version++
					next = version
				case 1:
					next = version // 同版本，应被拒绝
				default:
					next = version - 1 - rng.Intn(2) // 更旧，应被拒绝
				}
				rules := randomRules()
				if got := filter.Update(next, rules); got != ref.update(next, rules) {
					t.Fatalf("trial %d op %d: Update(%d) 返回值不一致", trial, op, next)
				}
			default: // 判定（含重复判定，验证缓存一致）
				player := players[rng.Intn(len(players))]
				item := randomPattern()
				for repeat := 0; repeat < 2; repeat++ {
					got, want := filter.Allow(player, item), ref.allow(player, item)
					if got != want {
						t.Fatalf("trial %d op %d: Allow(%q, %q)=%v，参考实现=%v", trial, op, player, item, got, want)
					}
				}
			}
		}
	}
}
