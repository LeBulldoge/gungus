package main

import "github.com/LeBulldoge/gungus/cmd"

var (
	version string
	build   string
)

func main() {
	gungus.Run(version, build)
}
