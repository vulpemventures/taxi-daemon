package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi-daemon/v1"
)

var listTopups = cli.Command{
	Name:  "topups",
	Usage: "list all the topups",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  assetKey,
			Usage: "optional. asset hash filter",
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

	assetFilter := ctx.String(assetKey)
	topups, err := adminClient.ListTopups(
		context.Background(), &pb.ListTopupsRequest{
			AssetHash: assetFilter,
		},
	)
	if err != nil {
		return err
	}

	fmt.Println()
	printRespJSON(topups)

	return nil
}
