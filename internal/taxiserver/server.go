package taxiserver

import (
	"context"

	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/liquid-taxi/config"

	pb "github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

//ASSETS ...
var ASSETS = map[string]map[string]bool{
	"liquid": {
		"ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2": true,
	},
	"regtest": {
		"": true,
	},
}

//NETWORK ...
var NETWORK = map[string]network.Network{
	"liquid":  network.Liquid,
	"regtest": network.Regtest,
}

// Server implements the Taxi service
type Server struct{}

// Topup implements Topup rpc
func (s *Server) Topup(ctx context.Context, req *pb.TopupRequest) (res *pb.TopupReply, err error) {
	return &pb.TopupReply{}, nil
}

// TopupWithKey implements TopupWithKey rpc
func (s *Server) TopupWithKey(ctx context.Context, req *pb.TopupWithKeyRequest) (res *pb.TopupWithKeyReply, err error) {
	return &pb.TopupWithKeyReply{}, nil
}

func notAcceptedAsset(asset string) bool {
	net := config.GetString(config.NetworkKey)
	if net != "liquid" {
		return true
	}

	if _, ok := ASSETS[net][asset]; ok {
		return true
	}

	return false
}
