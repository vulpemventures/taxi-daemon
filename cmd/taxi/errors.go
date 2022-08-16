package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

type invalidUsageError struct {
	ctx     *cli.Context
	command string
}

var _ error = (*invalidUsageError)(nil)

func (e *invalidUsageError) Error() string {
	return fmt.Sprintf("invalid usage of command %s", e.command)
}

func fatal(err error) {
	var e *invalidUsageError
	if errors.As(err, &e) {
		cli.ShowCommandHelp(e.ctx, e.command)
	} else {
		_, _ = fmt.Fprintf(os.Stderr, "[taxi-cli] %v\n", err)
	}
	os.Exit(1)
}
