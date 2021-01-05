package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
)

const (
	listTopupsAssetParam = "asset"
)

var listTopups = cli.Command{
	Name:  "topups",
	Usage: "list all the topups",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        listTopupsAssetParam,
			Aliases:     []string{"a"},
			Usage:       "optional. asset hash filter",
			DefaultText: "",
			Value:       "",
		},
	},
	Action: listTopupsAction,
}

func listTopupsAction(ctx *cli.Context) error {
	adminClient, close, err := getAdminClient()
	if err != nil {
		return err
	}

	defer close()

	assetFilter := ctx.String(listTopupsAssetParam)

	req := &admin.ListTopupsRequest{}
	if assetFilter != "" {
		req.AssetHash = assetFilter
	}

	topups, err := adminClient.ListTopups(context.Background(), req)
	if err != nil {
		return err
	}

	topupsDetails := topups.GetTopups()
	topupsJSON, err := json.Marshal(topupsDetails)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(topupsJSON)

	return nil
}
