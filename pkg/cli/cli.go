package cli

import (
	"flag"
	"fmt"
	"os"
)

type Config struct {
    BuildPath  string
    OutputPath string
}

func ParseFlags() *Config {
    var config Config

    flag.StringVar(&config.BuildPath, "input", "./build", "Path to build directory")
    flag.StringVar(&config.OutputPath, "output", "./build.manifest", "Output manifest path")
    
    flag.Usage = func() {
        fmt.Printf("Usage: %s [options]\n\nOptions:\n", os.Args[0])
        flag.PrintDefaults()
    }
    
    flag.Parse()
    return &config
}