package main

import (
	"fmt"

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
