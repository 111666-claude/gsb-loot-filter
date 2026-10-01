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
	sample := flags.String("sample", "priority", "priority / default / ties / scan")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	switch *sample {
	case "priority":
		filter := lootfilter.NewFilter([]lootfilter.Rule{
			{Pattern: "epic_sword", Allow: false, Priority: 1},
			{Pattern: "epic", Allow: true, Priority: 2},
		})
		fmt.Fprintf(stdout, "allow=%v\n", filter.Allow("epic_sword"))
	case "default":
		filter := lootfilter.NewFilter([]lootfilter.Rule{{Pattern: "gold", Allow: true, Priority: 1}})
		fmt.Fprintf(stdout, "allow=%v\n", filter.Allow("stone"))
	case "ties":
		filter := lootfilter.NewFilter([]lootfilter.Rule{
			{Pattern: "gem", Allow: false, Priority: 1},
			{Pattern: "gem", Allow: true, Priority: 1},
		})
		fmt.Fprintf(stdout, "allow=%v\n", filter.Allow("gem"))
	case "scan":
		rules := make([]lootfilter.Rule, 0, 1000)
		for index := 0; index < 1000; index++ {
			rules = append(rules, lootfilter.Rule{Pattern: fmt.Sprintf("p%04d", index), Allow: true, Priority: index})
		}
		filter := lootfilter.NewFilter(rules)
		items := make([]string, 0, 1000)
		for index := 0; index < 1000; index++ {
			items = append(items, fmt.Sprintf("p%04d_x", index))
		}
		filter.AllowBatch(items)
		fmt.Fprintf(stdout, "scanned=%d\n", filter.Scanned())
	default:
		fmt.Fprintln(stderr, "需要 --sample priority|default|ties|scan")
		return 2
	}
	return 0
}

func main() {
	os.Exit(Run(os.Args[1:], os.Stdout, os.Stderr))
}
