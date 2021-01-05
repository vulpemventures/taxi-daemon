package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()

	app.Version = "0.0.1"
	app.Name = "taxi admin CLI"
	app.Usage = "CLI for taxi-server administration"
	app.EnableBashCompletion = true
	app.Commands = append(
		app.Commands,
		&addAsset,
		&config,
		&fundingAddress,
	)

	err := app.Run(os.Args)
	if err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	var e *invalidUsageError
	if errors.As(err, &e) {
		_ = cli.ShowCommandHelp(e.ctx, e.command)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "[taxi-cli] %v\n", err)
	}
	os.Exit(1)
}
