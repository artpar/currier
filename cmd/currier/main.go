package main

import (
	"fmt"
	"os"

	"github.com/artpar/currier/internal/cli"
)

var version = "dev"

func main() {
	cmd := cli.NewRootCommand(version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
