package lootfilter

import (
	"fmt"
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

func TestPlayerExceptionExclusive(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "stone", Allow: false, Priority: 1}})
	filter.SetPlayerRules("vip", []Rule{{Pattern: "stone", Allow: true, Priority: 1}})
	if !filter.Allow("vip", "stone") {
		t.Fatal("玩家例外规则应该覆盖全局规则")
	}
	if filter.Allow("p", "stone") {
		t.Fatal("没有例外的玩家仍按全局规则判定")
	}
}

func TestPlayerEmptyExceptionDeniesAll(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	filter.SetPlayerRules("vip", nil)
	if filter.Allow("vip", "gold_coin") {
		t.Fatal("玩家例外存在即为生效规则集，空例外应全部不拾取")
	}
	if !filter.Allow("p", "gold_coin") {
		t.Fatal("其他玩家不受空例外影响")
	}
}

func TestSameLayerOrder(t *testing.T) {
	// 优先级升序：priority 小的先命中。
	filter := NewFilter([]Rule{
		{Pattern: "epic", Allow: true, Priority: 2},
		{Pattern: "epic_sword", Allow: false, Priority: 1},
	})
	if filter.Allow("p", "epic_sword") {
		t.Fatal("priority 1 的精确禁止应压过 priority 2 的前缀允许")
	}
	// 前缀长度降序：同优先级时长前缀先命中。
	filter = NewFilter([]Rule{
		{Pattern: "epic", Allow: true, Priority: 1},
		{Pattern: "epic_sword", Allow: false, Priority: 1},
	})
	if filter.Allow("p", "epic_sword") {
		t.Fatal("同优先级下更长的前缀应先命中")
	}
	if !filter.Allow("p", "epic_shield") {
		t.Fatal("短前缀规则对非精确物品仍应生效")
	}
	// deny 优先：同优先级同前缀时禁止胜出。
	filter = NewFilter([]Rule{
		{Pattern: "gem", Allow: true, Priority: 1},
		{Pattern: "gem", Allow: false, Priority: 1},
	})
	if filter.Allow("p", "gem_ruby") {
		t.Fatal("同优先级同前缀时 deny 应优先于 allow")
	}
}

func TestUpdateVersionMonotonic(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	if !filter.Update(2, []Rule{{Pattern: "gold", Allow: true, Priority: 1}}) {
		t.Fatal("版本 2 应该生效")
	}
	if filter.Update(1, []Rule{{Pattern: "gold", Allow: false, Priority: 1}}) {
		t.Fatal("旧版本 1 不该生效")
	}
	if !filter.Allow("p", "gold_coin") {
		t.Fatal("旧版本被拒绝后规则集应保持版本 2 的内容")
	}
	if filter.Update(2, []Rule{{Pattern: "gold", Allow: false, Priority: 1}}) {
		t.Fatal("相同版本不算严格大于，不该生效")
	}
	if !filter.Allow("p", "gold_coin") {
		t.Fatal("同版本被拒绝后规则集应保持不变")
	}
	if !filter.Update(3, []Rule{{Pattern: "gold", Allow: false, Priority: 1}}) {
		t.Fatal("更高版本应该生效")
	}
	if filter.Allow("p", "gold_coin") {
		t.Fatal("版本 3 生效后应按新规则判定")
	}
}

func TestCacheInvalidation(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	if !filter.Allow("p", "gold_coin") {
		t.Fatal("初始规则应允许拾取")
	}
	scannedAfterFirst := filter.Scanned()
	if !filter.Allow("p", "gold_coin") {
		t.Fatal("缓存命中结果应与首次一致")
	}
	if filter.Scanned() != scannedAfterFirst {
		t.Fatal("缓存命中不该再产生比较次数")
	}
	// Update 生效后旧缓存必须失效。
	if !filter.Update(1, []Rule{{Pattern: "gold", Allow: false, Priority: 1}}) {
		t.Fatal("版本 1 应该生效")
	}
	if filter.Allow("p", "gold_coin") {
		t.Fatal("Update 后旧缓存必须失效，应按新规则判定")
	}
	// 被拒绝的 Update 不应触碰缓存。
	scannedBeforeReject := filter.Scanned()
	if filter.Update(1, []Rule{{Pattern: "gold", Allow: true, Priority: 1}}) {
		t.Fatal("同版本不该生效")
	}
	if filter.Allow("p", "gold_coin") {
		t.Fatal("被拒绝的 Update 不该改变判定结果")
	}
	if filter.Scanned() != scannedBeforeReject {
		t.Fatal("被拒绝的 Update 不该让缓存失效")
	}
	// SetPlayerRules 后旧缓存必须失效。
	filter.SetPlayerRules("p", []Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	if !filter.Allow("p", "gold_coin") {
		t.Fatal("SetPlayerRules 后旧缓存必须失效，应按例外规则判定")
	}
}

func TestInsertionOrderIndependent(t *testing.T) {
	rules := []Rule{
		{Pattern: "gem", Allow: true, Priority: 3},
		{Pattern: "gem_ruby", Allow: false, Priority: 1},
		{Pattern: "gem_ruby", Allow: true, Priority: 2},
		{Pattern: "gem_ruby_star", Allow: false, Priority: 2},
	}
	forward := NewFilter(rules)
	reversed := make([]Rule, len(rules))
	for index, rule := range rules {
		reversed[len(rules)-1-index] = rule
	}
	backward := NewFilter(reversed)
	for _, item := range []string{"gem", "gem_ruby", "gem_ruby_star", "gem_onyx", "other"} {
		if forward.Allow("p", item) != backward.Allow("p", item) {
			t.Fatalf("物品 %q 的判定不应依赖规则插入顺序", item)
		}
	}
}

func TestQueryCost(t *testing.T) {
	rules := make([]Rule, 0, 1000)
	for index := 0; index < 1000; index++ {
		rules = append(rules, Rule{Pattern: fmt.Sprintf("p%04d", index), Allow: true, Priority: index})
	}
	filter := NewFilter(rules)
	for query := 0; query < 1000; query++ {
		if !filter.Allow("p", "p0999_x") {
			t.Fatal("p0999_x 应命中规则 p0999")
		}
	}
	if scanned := filter.Scanned(); scanned > 4000 {
		t.Fatalf("1000 次判定只产生 %d 次比较，不应随规则数放大", scanned)
	}
	// 单次未命中判定的代价是 O(物品名长度)，与规则数无关。
	fresh := NewFilter(rules)
	before := fresh.Scanned()
	fresh.Allow("p", "p0999_x")
	if cost := fresh.Scanned() - before; cost > len("p0999_x") {
		t.Fatalf("单次判定比较 %d 次，超过物品名长度 %d", cost, len("p0999_x"))
	}
	before = fresh.Scanned()
	fresh.Allow("p", "zz")
	if cost := fresh.Scanned() - before; cost > 1 {
		t.Fatalf("首字符即分叉的判定比较 %d 次，应为 1", cost)
	}
}
