package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
)

var listLocks = cli.Command{
	Name:   "locks",
	Usage:  "locks lists all the utxo key currently locked",
	Action: listLocksAction,
}

func listLocksAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	reply, err := adminClient.ListLocks(context.Background(), &admin.ListLocksRequest{})
	if err != nil {
		return err
	}

	locksJSON, err := json.Marshal(reply.GetLocks())
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(locksJSON)

	return nil
}
