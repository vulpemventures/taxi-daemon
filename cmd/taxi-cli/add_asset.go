package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

var addAsset = cli.Command{
	Name:  "add",
	Usage: "add (update if exists) an asset/price",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset hash to add/update (string)",
		},
		&cli.StringFlag{
			Name:  "price",
			Usage: "the number of assets (in satoshis) needed to get 1 LBTC (float)",
		},
	},
	Action: addAssetAction,
}

func addAssetAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	assetHash := ctx.String("asset")
	price := float32(ctx.Float64("price"))

	_, err = adminClient.AddAsset(context.Background(), &pb.AddAssetRequest{
		AssetHash: assetHash,
		Price:     price,
	})

	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("price has been updated")
	return nil
}
