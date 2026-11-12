package main

import (
	"fmt"
	"os"

	"github.com/ebrgr/TuneTrail/engine"
	"github.com/ebrgr/TuneTrail/internal/version"
)

var Version = version.Current

func main() {
	if len(os.Args) > 1 && os.Args[1] == "reset-password" {
		if err := engine.ResetPassword(os.Getenv, os.Stdout, Version); err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			os.Exit(1)
		}
		return
	}
	if err := engine.Run(os.Getenv, os.Stdout, Version); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}
