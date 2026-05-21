package main

import (
	"os"

	"github.com/baptistax/rdlxd/internal/app"
)

func main() {
	cli := app.New(os.Stdout, os.Stderr)
	if err := cli.Run(os.Args[1:]); err != nil {
		cli.PrintError(err)
		os.Exit(1)
	}
}
