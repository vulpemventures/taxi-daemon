package oceanwallet

import (
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ocean struct {
	conn    *grpc.ClientConn
	wallet  ports.WalletManager
	account ports.AccountManager
	tx      ports.TransactionManager
	notify  ports.NotificationManager
}

func NewInsecureOceanWallet(addr string) (ports.OceanWallet, error) {
	conn, err := grpc.Dial(
		addr, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	notifyManager, err := newNotifyManager(conn)
	if err != nil {
		return nil, err
	}

	return &ocean{
		conn,
		newWalletManager(conn),
		newAccountManager(conn),
		newTxManager(conn),
		notifyManager,
	}, nil
}

func (o *ocean) WalletManager() ports.WalletManager {
	return o.wallet
}

func (o *ocean) AccountManager() ports.AccountManager {
	return o.account
}

func (o *ocean) TransactionManager() ports.TransactionManager {
	return o.tx
}

func (o *ocean) NotificationManager() ports.NotificationManager {
	return o.notify
}

func (o *ocean) Close() {
	o.conn.Close()
}
