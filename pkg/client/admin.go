package client

import (
	"fmt"

	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
	"google.golang.org/grpc"
)

var (
	// maxMsgRecvSize is the largest message our client will receive. We
	// set this to 200MiB atm.
	maxMsgRecvSize = grpc.MaxCallRecvMsgSize(1 * 1024 * 1024 * 200)
)

// NewAdminClient returns an adminGRPCclient for admin service
func NewAdminClient(address string) (admin.AdminClient, func() error, error) {
	conn, err := getClientConn(address)
	if err != nil {
		return nil, nil, err
	}

	return admin.NewAdminClient(conn), conn.Close, nil
}

func getClientConn(address string) (*grpc.ClientConn, error) {

	opts := []grpc.DialOption{grpc.WithDefaultCallOptions(maxMsgRecvSize), grpc.WithInsecure()}

	conn, err := grpc.Dial(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to RPC server: %v",
			err)
	}

	return conn, nil
}
