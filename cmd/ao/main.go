package main

import (
	"os"

	"github.com/uesugitorachiyo/ao-foundry/internal/cli"
)

func main() {
	args := append([]string{"ao"}, os.Args[1:]...)
	os.Exit(cli.Run(args, os.Stdout, os.Stderr))
}
