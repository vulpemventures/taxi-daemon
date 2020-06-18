package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vulpemventures/liquid-taxi/rpc/taxi"
)

func topupAction(psetBase64 string, satPerByte float32, taxiURL string) error {
	if psetBase64 == "" {
		return fmt.Errorf("missing required flag")
	}
	client := taxi.NewTaxiProtobufClient(taxiURL, &http.Client{})
	reply, err := client.TopupWithAsset(context.Background(), &taxi.TopupWithAssetRequest{
		Unsigned: &taxi.Unsigned{
			Pset:       psetBase64,
			SatPerByte: 0.1,
		},
		AssetHash: "ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2",
	})
	if err != nil {
		return fmt.Errorf("taxi: %w", err)
	}

	fmt.Println()
	fmt.Println(reply.Order.Partial.Pset)

	return nil
}
