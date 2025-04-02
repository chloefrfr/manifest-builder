package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type Config struct {
    BuildPath  string
    OutputPath string
}

func ParseFlags() *Config {
    var config Config
    flag.StringVar(&config.BuildPath, "input", "", "Path to build directory (required)")
    flag.StringVar(&config.OutputPath, "output", "build.manifest", "Output file path")
    flag.Parse()
    
    if config.BuildPath == "" {
        fmt.Println("Error: Input directory is required")
        os.Exit(1)
    }
    return &config
}

func Comma(n int) string {
    s := fmt.Sprintf("%d", n)
    parts := strings.Split(s, ".")
    integer := parts[0]
    var result string
    for i, c := range integer {
        if i > 0 && (len(integer)-i)%3 == 0 {
            result += ","
        }
        result += string(c)
    }
    return result
}