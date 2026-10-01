// Command loot-filter 跑拾取过滤样例。
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"example.com/lootfilter"
)

// Run 执行一次命令行调用，返回退出码。
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("loot-filter", flag.ContinueOnError)
	flags.SetOutput(stderr)
	sample := flags.String("sample", "priority", "priority / player / stale / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *sample == "priority" {
		filter := lootfilter.NewFilter([]lootfilter.Rule{
			{Pattern: "epic_sword", Allow: false, Priority: 1},
			{Pattern: "epic", Allow: true, Priority: 2},
		})
		fmt.Fprintf(stdout, "allow=%v\n", filter.Allow("p", "epic_sword"))
		return 0
	}
	if *sample == "player" {
		filter := lootfilter.NewFilter([]lootfilter.Rule{{Pattern: "stone", Allow: false, Priority: 1}})
		filter.SetPlayerRules("vip", []lootfilter.Rule{{Pattern: "stone", Allow: true, Priority: 1}})
		fmt.Fprintf(stdout, "allow=%v\n", filter.Allow("vip", "stone"))
		return 0
	}
	if *sample == "stale" {
		filter := lootfilter.NewFilter([]lootfilter.Rule{{Pattern: "gold", Allow: true, Priority: 1}})
		filter.Update(2, []lootfilter.Rule{{Pattern: "gold", Allow: true, Priority: 1}})
		ok := filter.Update(1, []lootfilter.Rule{{Pattern: "gold", Allow: false, Priority: 1}})
		fmt.Fprintf(stdout, "ok=%v allow=%v\n", ok, filter.Allow("p", "gold_coin"))
		return 0
	}
	if *sample == "scan" {
		rules := make([]lootfilter.Rule, 0, 1000)
		for index := 0; index < 1000; index++ {
			rules = append(rules, lootfilter.Rule{Pattern: fmt.Sprintf("p%04d", index), Allow: true, Priority: index})
		}
		filter := lootfilter.NewFilter(rules)
		for query := 0; query < 1000; query++ {
			filter.Allow("p", "p0999_x")
		}
		fmt.Fprintf(stdout, "scanned=%d\n", filter.Scanned())
		return 0
	}
	fmt.Fprintln(stderr, "需要 --sample priority|player|stale|scan")
	return 2
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
