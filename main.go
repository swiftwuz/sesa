package main

import (
	"os"

	"sesa/internal/cli"
)

func main() {
	os.Exit(cli.New(os.Stdin, os.Stdout, os.Stderr).Run(os.Args[1:]))
}
