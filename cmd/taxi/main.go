package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/urfave/cli/v2"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	app := cli.NewApp()

	app.Version = formatVersion()
	app.Name = "taxi admin CLI"
	app.Usage = "CLI for taxi-server administration"
	app.EnableBashCompletion = true
	app.Commands = append(
		app.Commands,
		&addAsset,
		&config,
		&fundingAddress,
		&listTopups,
		&balance,
		&listFundingAddresses,
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

/*
Modified from https://github.com/lightninglabs/pool/blob/master/cmd/pool/main.go
Original Copyright 2017 Oliver Gugger. All Rights Reserved.
*/
func printRespJSON(res interface{}) {
	jsonMarshaler := &jsonpb.Marshaler{
		EmitDefaults: true,
		OrigName:     true,
		Indent:       "\t", // Matches indentation of printJSON.
	}

	jsonStr, err := jsonMarshaler.MarshalToString(res.(proto.Message))
	if err != nil {
		fmt.Println("unable to decode response: ", err)
		return
	}

	fmt.Println(jsonStr)
}

func formatVersion() string {
	return fmt.Sprintf(
		"\nVersion: %s\nCommit: %s\nDate: %s",
		version, commit, date,
	)
}
