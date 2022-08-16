package oceanwallet

import (
	"context"

	pb "github.com/vulpemventures/ocean/api-spec/protobuf/gen/go/ocean/v1alpha"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
	"google.golang.org/grpc"
)

type walletManager struct {
	client pb.WalletServiceClient
}

func newWalletManager(conn *grpc.ClientConn) ports.WalletManager {
	return &walletManager{pb.NewWalletServiceClient(conn)}
}

func (wm *walletManager) Status(
	ctx context.Context,
) (ports.WalletStatus, error) {
	res, err := wm.client.Status(ctx, &pb.StatusRequest{})
	if err != nil {
		return nil, err
	}

	return walletStatus{res}, nil
}

func (wm *walletManager) GetInfo(
	ctx context.Context,
) (ports.WalletInfo, error) {
	res, err := wm.client.GetInfo(ctx, &pb.GetInfoRequest{})
	if err != nil {
		return nil, err
	}

	return walletInfo{res}, nil
}

// walletStatus is a concrete impl of ports.WalletStatus
type walletStatus struct {
	*pb.StatusResponse
}

func (ws walletStatus) IsInitialized() bool {
	return ws.GetInitialized()
}
func (ws walletStatus) IsSynced() bool {
	return ws.GetSynced()
}
func (ws walletStatus) IsUnlocked() bool {
	return ws.GetUnlocked()
}

// walletInfo is a concrete impl of ports.WalletInfo
type walletInfo struct {
	*pb.GetInfoResponse
}

func (wi walletInfo) Network() ports.Network {
	return walletNetwork(wi.GetNetwork())
}
func (wi walletInfo) NativeAsset() string {
	return wi.GetNativeAsset()
}
func (wi walletInfo) Accounts() []ports.WalletAccount {
	accounts := wi.GetAccounts()
	portableAccounts := make([]ports.WalletAccount, 0, len(accounts))
	for _, a := range accounts {
		portableAccounts = append(portableAccounts, walletAccount{a})
	}
	return portableAccounts
}

// walletNetwork is a concrete impl of ports.WalletNetwork
type walletNetwork pb.GetInfoResponse_Network

func (wn walletNetwork) IsUnknown() bool {
	return int(wn) == int(pb.GetInfoResponse_NETWORK_UNSPECIFIED)
}
func (wn walletNetwork) IsMainnet() bool {
	return int(wn) == int(pb.GetInfoResponse_NETWORK_MAINNET)
}
func (wn walletNetwork) IsTestnet() bool {
	return int(wn) == int(pb.GetInfoResponse_NETWORK_TESTNET)
}
func (wn walletNetwork) IsRegtest() bool {
	return int(wn) == int(pb.GetInfoResponse_NETWORK_REGTEST)
}

// walletAccount is a concrete impl of ports.WalletAccount
type walletAccount struct {
	*pb.AccountInfo
}

func (wa walletAccount) Index() uint32 {
	return wa.GetIndex()
}
func (wa walletAccount) Name() string {
	return wa.GetName()
}
func (wa walletAccount) DerivationPath() string {
	return wa.GetDerivationPath()
}
func (wa walletAccount) Xpub() string {
	return wa.GetXpub()
}
