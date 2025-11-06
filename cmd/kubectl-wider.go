package main

import (
	"os"

	"github.com/spf13/pflag"
)

func main() {
	flags := pflag.NewFlagSet("kubectl-ns", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := NewRootCommand()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
