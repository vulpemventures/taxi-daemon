package interceptor

import (
	"context"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

func (i *interceptorChain) unaryLogger(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	log.Debug(info.FullMethod)
	return handler(ctx, req)
}

func (i *interceptorChain) streamLogger(
	srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) error {
	log.Debug(info.FullMethod)
	return handler(srv, stream)
}
