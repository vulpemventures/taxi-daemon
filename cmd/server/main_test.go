package main

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
	g "google.golang.org/grpc"
)

func TestTaxiServer(t *testing.T) {
	go main()

	conn, err := g.Dial("localhost:8080", g.WithInsecure())
	if err != nil {
		t.Error(err)
	}

	client := pb.NewTaxiClient(conn)
	log.Info(client)

	time.Sleep(30 * time.Second)
}
