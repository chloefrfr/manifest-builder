package main

import "manifest-builder/pkg/cli"

func main() {
	println("Hello, World!")

	config := cli.ParseFlags()
	
	println(config.BuildPath)
	println(config.OutputPath)
}