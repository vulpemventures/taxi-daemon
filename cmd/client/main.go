package main

import (
	"context"
	"net/http"

	"github.com/vulpemventures/liquid-taxi/rpc/taxi"
)

func main() {
	client := taxi.NewTaxiProtobufClient("http://localhost:8080", &http.Client{})

	reply, err := client.TopupWithAsset(context.Background(), &taxi.TopupWithAssetRequest{
		AssetHash: "ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2",
		Unsigned: &taxi.Unsigned{
			Pset:       "foobar",
			SatPerByte: 0.1,
		},
	})
	if err != nil {
		panic(err.Error())
	}

	println(reply.Order.Partial.Pset)

}
