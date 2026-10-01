package lootfilter

import (
	"fmt"
	"testing"
)

func BenchmarkAllow100k(b *testing.B) {
	const n = 100000
	rules := make([]Rule, 0, n)
	for index := 0; index < n; index++ {
		rules = append(rules, Rule{Pattern: fmt.Sprintf("item%06d", index), Allow: true, Priority: index})
	}
	filter := NewFilter(rules)
	b.ReportAllocs()
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		filter.Allow("p", "item099999_x")
	}
}
