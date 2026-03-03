package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"

	"github.com/dl-alexandre/abc/internal/cli"
)

var (
	version   = "dev"
	gitCommit = "unknown"
	buildTime = "unknown"
)

func main() {
	var c cli.CLI
	ctx := kong.Parse(&c,
		kong.Name("abc"),
		kong.Description("CLI for Apple Business Connect API v3.0"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			Compact: true,
		}),
		kong.Vars{
			"version": version,
		},
	)

	if ctx.Command() == "version" {
		fmt.Printf("abc %s (commit: %s) built %s\n", version, gitCommit, buildTime)
		os.Exit(0)
	}

	if err := ctx.Run(&c.Globals); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
