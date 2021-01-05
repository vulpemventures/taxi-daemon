package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
)

var fundingAddress = cli.Command{
	Name:  "fund",
	Usage: "fund generates one or more address of the taxi's LBTC account",
	Flags: []cli.Flag{
		&cli.Int64Flag{
			Name:        "nb_addresses",
			Aliases:     []string{"n"},
			Usage:       "optional. The number of addresses to generate",
			DefaultText: "1",
			Value:       1,
		},
	},
	Action: fundingAddressAction,
}

func fundingAddressAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	nbOfAddresses := ctx.Int64("nb_addresses")
	reply, err := adminClient.FundingAddress(
		context.Background(),
		&admin.FundingAddressRequest{
			NumOfAddresses: nbOfAddresses,
		},
	)

	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Send LBTC to the following address(es):")
	fmt.Println(strings.Join(reply.GetAddresses(), " "))

	return nil
}
