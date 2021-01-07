package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
)

var balance = cli.Command{
	Name:   "balance",
	Usage:  "get the balance",
	Action: balanceAction,
}

func balanceAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	req := &admin.BalanceRequest{}
	reply, err := adminClient.Balance(context.Background(), req)
	if err != nil {
		return err
	}

	if len(reply.Balance) == 0 {
		fmt.Println()
		fmt.Println("No funds on taxi accounts, try `fund` command to get new LBTC funding addresses.")
	}

	for asset, balanceInfo := range reply.Balance {
		fmt.Println()
		fmt.Println("Balance for ", asset)
		fmt.Println("	confirmed: ", balanceInfo.ConfirmedBalance)
		fmt.Println("	unconfirmed: ", balanceInfo.UnconfirmedBalance)
		fmt.Println("	total: ", balanceInfo.TotalBalance)
	}

	return nil
}
