package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/admin"
)

var listFundingAddresses = cli.Command{
	Name:  "addresses",
	Usage: "list all the taxi LBTC account addresses",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:        "change",
			Aliases:     []string{"c"},
			Usage:       "optional. If set, return the internal chain addresses instead of external ones",
			DefaultText: "false",
			Value:       false,
		},
		&cli.Int64Flag{
			Name:        "start_index",
			Aliases:     []string{"s"},
			Usage:       "optional. Start index to begin generation",
			DefaultText: "0",
			Value:       0,
		},
		&cli.Int64Flag{
			Name:        "number",
			Aliases:     []string{"n"},
			Usage:       "optional. The number of addresses to fetch. 0 = all addresses.",
			DefaultText: "0",
			Value:       0,
		},
	},
	Action: listFundingAddressesAction,
}

func listFundingAddressesAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	nbOfAddresses := ctx.Int64("number")
	startIndex := ctx.Int64("start_index")
	isChange := ctx.Bool("change")

	reply, err := adminClient.ListFundingAddresses(context.Background(), &admin.ListFundingAddressesRequest{
		Start:  startIndex,
		Number: nbOfAddresses,
		Change: isChange,
	})
	if err != nil {
		return err
	}

	addressesInfos := reply.GetAddresses()

	fmt.Println()

	if len(addressesInfos) == 0 {
		fmt.Println("no funding addresses.")
		return nil
	}

	addresses := make([]string, len(addressesInfos), len(addressesInfos))
	for index, addr := range addressesInfos {
		addresses[index] = addr.Address
	}

	fmt.Println(strings.Join(addresses, " "))

	return nil
}
