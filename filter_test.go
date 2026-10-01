package lootfilter

import "testing"

func TestExactDenyWinsOverPrefixAllow(t *testing.T) {
	filter := NewFilter([]Rule{
		{Pattern: "gem", Allow: true, Priority: 2},
		{Pattern: "gem_ruby", Allow: false, Priority: 1},
	})
	if filter.Allow("gem_ruby") {
		t.Fatal("更高优先级的禁止规则应该生效")
	}
}

func TestSingleAllowRule(t *testing.T) {
	filter := NewFilter([]Rule{{Pattern: "gold", Allow: true, Priority: 1}})
	if !filter.Allow("gold_coin") {
		t.Fatal("白名单前缀应该允许拾取")
	}
}

func TestBatchCounts(t *testing.T) {
	filter := NewFilter([]Rule{
		{Pattern: "gold", Allow: true, Priority: 1},
		{Pattern: "silver", Allow: false, Priority: 1},
	})
	if got := filter.AllowBatch([]string{"gold_coin", "silver_coin"}); got != 1 {
		t.Fatalf("批量判定数不对：%d", got)
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	if NewFilter(nil).Scanned() != 0 {
		t.Fatal("初始比较次数应该是 0")
	}
}
