package interceptor

import (
	"errors"
	middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"runtime/debug"
)

type Service interface {
	CreateServerOpts() []grpc.ServerOption
}

type interceptorChain struct{}

func NewService() (Service, error) {
	return &interceptorChain{}, nil
}

func (i *interceptorChain) CreateServerOpts() []grpc.ServerOption {
	var unaryInterceptors []grpc.UnaryServerInterceptor
	var streamInterceptors []grpc.StreamServerInterceptor

	//logger
	unaryInterceptors = append(
		unaryInterceptors, i.unaryLogger,
	)
	streamInterceptors = append(
		streamInterceptors, i.streamLogger,
	)

	recoveryOpts := grpc_recovery.WithRecoveryHandler(
		func(p interface{}) error {
			log.Errorf("panic recovered: %v", p)
			log.Debugf("stack trace: %v", string(debug.Stack()))

			return errors.New("internal error, please try again")
		},
	)

	//panic handler
	unaryInterceptors = append(
		unaryInterceptors, grpc_recovery.UnaryServerInterceptor(recoveryOpts),
	)
	streamInterceptors = append(
		streamInterceptors, grpc_recovery.StreamServerInterceptor(recoveryOpts),
	)

	chainedUnary := middleware.WithUnaryServerChain(
		unaryInterceptors...,
	)
	chainedStream := middleware.WithStreamServerChain(
		streamInterceptors...,
	)

	serverOpts := []grpc.ServerOption{chainedUnary, chainedStream}

	return serverOpts
}
