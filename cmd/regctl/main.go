package main

import (
	"fmt"
	"os"

	"github.com/yukihamada/regctl/internal/cli"
)

var (
	version   = "dev"
	buildTime = "unknown"
)

func main() {
	rootCmd := cli.NewRootCmd(version)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
