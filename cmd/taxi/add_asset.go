package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi-daemon/v1"
)

const (
	assetKey      = "asset"
	priceKey      = "price"
	basisPointKey = "basis_point"
)

var addAsset = cli.Command{
	Name:  "add-asset",
	Usage: "add or update an asset/price",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     assetKey,
			Usage:    "the asset hash to add/update (hex string)",
			Required: true,
		},
		&cli.StringFlag{
			Name:     priceKey,
			Usage:    "the quantity of asset needed to get 1 satoshi of LBTC",
			Required: true,
		},
		&cli.IntFlag{
			Name:     basisPointKey,
			Aliases:  []string{"bp"},
			Usage:    "the spread used in basis point",
			Required: true,
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

	assetHash := ctx.String(assetKey)
	price := ctx.Float64(priceKey)
	basisPoints := ctx.Uint(basisPointKey)

	if _, err := adminClient.AddAsset(
		context.Background(),
		&pb.AddAssetRequest{
			AssetHash:   assetHash,
			Price:       price,
			BasisPoints: uint32(basisPoints),
		},
	); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("price has been updated")
	return nil
}
