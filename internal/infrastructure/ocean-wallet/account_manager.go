package oceanwallet

import (
	"context"

	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/ocean/v1"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
	"google.golang.org/grpc"
)

type accountManager struct {
	client pb.AccountServiceClient
}

func newAccountManager(conn *grpc.ClientConn) ports.AccountManager {
	return &accountManager{pb.NewAccountServiceClient(conn)}
}

func (am *accountManager) CreateAccount(
	ctx context.Context, name string,
) (uint32, string, error) {
	res, err := am.client.CreateAccountBIP44(ctx, &pb.CreateAccountBIP44Request{
		Name: name,
	})
	if err != nil {
		return 0, "", err
	}
	return res.GetAccountIndex(), res.GetAccountName(), nil
}

func (am *accountManager) DeriveAddressesForAccount(
	ctx context.Context, account string, numOfAddresses uint64,
) ([]string, error) {
	res, err := am.client.DeriveAddresses(ctx, &pb.DeriveAddressesRequest{
		AccountName:    account,
		NumOfAddresses: numOfAddresses,
	})
	if err != nil {
		return nil, err
	}
	return res.GetAddresses(), nil
}

func (am *accountManager) DeriveChangeAddressForAccount(
	ctx context.Context, account string, numOfAddresses uint64,
) ([]string, error) {
	res, err := am.client.DeriveChangeAddresses(
		ctx, &pb.DeriveChangeAddressesRequest{
			AccountName:    account,
			NumOfAddresses: numOfAddresses,
		},
	)
	if err != nil {
		return nil, err
	}
	return res.GetAddresses(), nil
}
