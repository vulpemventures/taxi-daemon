package taxiserver

import (
	"context"
	"fmt"

	"github.com/twitchtv/twirp"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/storage"
	pb "github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

// GetTopup implements GetTopup rpc
func (s *Server) GetTopup(ctx context.Context, req *pb.GetTopupRequest) (*pb.GetTopupReply, error) {
	if req.GetOrderId() == "" {
		return nil, twirp.RequiredArgumentError("order_id")
	}

	// Open up the storage
	store, err := storage.NewStorage(config.GetString(config.MongoEndpoint))
	if err != nil {
		return nil, twirp.InternalError(fmt.Errorf("Storage: %w", err).Error())
	}
	order, notFound, err := store.GetOrderByID(req.GetOrderId())
	if !notFound && err != nil {
		return nil, twirp.InternalError(fmt.Errorf("Storage: %w", err).Error())
	}
	if notFound {
		return nil, twirp.NotFoundError("order does not exists")
	}

	return &pb.GetTopupReply{
		Order: order.ToProtobuf(req.GetOrderId()),
	}, nil
}
