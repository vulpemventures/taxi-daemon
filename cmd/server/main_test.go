package main

import (
	"context"
	"testing"
	"time"

	"github.com/tj/assert"
	"github.com/vulpemventures/go-elements/pset"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
	g "google.golang.org/grpc"
)

var emptyContext = context.Background()

func TestTaxiServer(t *testing.T) {
	go main()

	// give time to server to serve
	time.Sleep(10 * time.Second)

	conn, err := g.Dial("localhost:8080", g.WithInsecure())
	if err != nil {
		t.Error(err)
	}
	defer conn.Close()

	client := pb.NewTaxiClient(conn)
	reply, err := client.TopupWithAsset(emptyContext, &pb.TopupWithAssetRequest{
		AssetHash: "2e4c430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c4798bc4f",
	})

	assert.Nil(t, err)
	assert.NotNil(t, reply)

	b64Tx := reply.Topup.GetPartial()
	p, err := pset.NewPsetFromBase64(b64Tx)

	assert.Nil(t, err)
	assert.NotNil(t, p)

	signatureIsValid, err := p.ValidateAllSignatures()

	assert.Nil(t, err)
	assert.Equal(t, true, signatureIsValid)
}
