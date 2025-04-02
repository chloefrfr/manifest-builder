package main

import (
	"fmt"
	"manifest-builder/pkg/cli"
	"manifest-builder/pkg/manifest"
	"os"
	"time"

	"github.com/fatih/color"
)


var (
    cyan   = color.New(color.FgCyan).SprintFunc()
    yellow = color.New(color.FgYellow).SprintFunc()
    green  = color.New(color.FgGreen).SprintFunc()
    red    = color.New(color.FgRed).SprintFunc()
)

func main() {
    config := cli.ParseFlags()
    
    if _, err := os.Stat(config.BuildPath); os.IsNotExist(err) {
        fmt.Printf("\n%s Directory does not exist: %s\n", red("✗"), yellow(config.BuildPath))
        os.Exit(1)
    }

    start := time.Now()
    fmt.Printf("\n%s Processing: %s\n", cyan("•"), yellow(config.BuildPath))

    generator := manifest.NewGenerator()
    if generator == nil {
        fmt.Printf("\n%s Failed to initialize generator\n", red("✗"))
        os.Exit(1)
    }

    m, err := generator.Generate(config.BuildPath)
    if err != nil || m == nil {
        fmt.Printf("\n%s Generation failed: %v\n", red("✗"), err)
        os.Exit(1)
    }

    if err := manifest.Write(m, config.OutputPath); err != nil {
        fmt.Printf("\n%s Write failed: %v\n", red("✗"), err)
        os.Exit(1)
    }

    fmt.Printf("\n%s Generated %s chunks in %s\n", 
        green("✓"),
        yellow(cli.Comma(len(m.Chunks))),
        yellow(time.Since(start).Round(time.Millisecond)),
    )
    fmt.Printf("%s Output: %s\n\n", cyan("•"), yellow(config.OutputPath))
}