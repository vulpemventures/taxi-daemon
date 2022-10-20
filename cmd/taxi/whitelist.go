package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi-daemon/v1"
)

const (
	addressKey = "address"
)

var whitelist = cli.Command{
	Name:  "whitelist",
	Usage: "manage taxi's whitelist of known payouts",
	Subcommands: []*cli.Command{
		{
			Name:   "list",
			Usage:  "list all addresses of the whitelist",
			Action: listAddressesAction,
		},
		{
			Name:  "add",
			Usage: "add one or more addresses to the whitelist",
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:     addressKey,
					Usage:    "list of addresses to add",
					Required: true,
				},
			},
			Action: addAddressesAction,
		},
		{
			Name:  "delete",
			Usage: "add one or more addresses from the whitelist",
			Flags: []cli.Flag{
				&cli.StringSliceFlag{
					Name:     addressKey,
					Usage:    "list of addresses to delete",
					Required: true,
				},
			},
			Action: deleteAddressesAction,
		},
	},
}

func listAddressesAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	whitelist, err := adminClient.ListWhitelistedScripts(
		context.Background(), &pb.ListWhitelistedScriptsRequest{},
	)
	if err != nil {
		return err
	}

	fmt.Println()
	printRespJSON(whitelist)
	return nil
}

func addAddressesAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	addresses := ctx.StringSlice(addressKey)

	if _, err := adminClient.AddWhitelistedAddresses(
		context.Background(),
		&pb.AddWhitelistedAddressesRequest{Addresses: addresses},
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("whitelist has been updated")
	return nil
}

func deleteAddressesAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	addresses := ctx.StringSlice(addressKey)

	if _, err := adminClient.DeleteWhitelistedAddresses(
		context.Background(),
		&pb.DeleteWhitelistedAddressesRequest{Addresses: addresses},
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("whitelist has been updated")
	return nil
}
