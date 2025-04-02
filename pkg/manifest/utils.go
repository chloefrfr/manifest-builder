package manifest

import (
	"github.com/fatih/color"
)

var (
    cyan = color.New(color.FgCyan).SprintFunc()
    yellow = color.New(color.FgYellow).SprintFunc()
)

func max(a, b int64) int64 {
    if a > b { return a }
    return b
}

func min(a, b int64) int64 {
    if a < b { return a }
    return b
}
