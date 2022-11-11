package ports

import (
	"context"

	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/ocean/v1"
)

type OceanWallet interface {
	WalletManager() WalletManager
	AccountManager() AccountManager
	TransactionManager() TransactionManager
	NotificationManager() NotificationManager
	Close()
}

type WalletManager interface {
	Status(ctx context.Context) (status WalletStatus, err error)
	GetInfo(ctx context.Context) (info WalletInfo, err error)
}

type AccountManager interface {
	CreateAccount(
		ctx context.Context, name string,
	) (accountIndex uint32, xpub string, err error)
	DeriveAddressesForAccount(
		ctx context.Context, account string, numOfAddresses uint64,
	) (addresses []string, err error)
	DeriveChangeAddressForAccount(
		ctx context.Context, account string, numOfAddresses uint64,
	) (addresses []string, err error)
}

type TransactionManager interface {
	SelectUtxosForAccount(
		ctx context.Context, account string,
		targetAsset string, targetAmount uint64, strategy Strategy,
	) (utxos []Utxo, change, expirationDate uint64, err error)
	EstimateFees(
		ctx context.Context, inputs []Input, outputs []Output,
		millisatsPerByte uint64,
	) (feeAmount uint64, err error)
	CreateTransaction(
		ctx context.Context, inputs []Input, outputs []Output,
	) (psetBase64 string, err error)
	UpdateTransaction(
		ctx context.Context, psetBase64 string, inputs []Input, outputs []Output,
	) (updatedPset string, err error)
	BlindTransaction(
		ctx context.Context, psetBase64 string, lastBlinder bool,
		userUnblindedInputs []*pb.UnblindedInput,
	) (blindedPsetBase64 string, err error)
	SignTransaction(
		ctx context.Context, psetBase64 string, sighash uint32,
	) (signedPsetBase64 string, err error)
}

type NotificationManager interface {
	UtxoChannel() chan UtxoNotification
}

type WalletInfo interface {
	Network() Network
	NativeAsset() string
	Accounts() []WalletAccount
}

type WalletAccount interface {
	Index() uint32
	Name() string
	DerivationPath() string
	Xpub() string
}

type Network interface {
	IsUnknown() bool
	IsMainnet() bool
	IsTestnet() bool
	IsRegtest() bool
}

type WalletStatus interface {
	IsInitialized() bool
	IsSynced() bool
	IsUnlocked() bool
}

type InitWalletMsg interface {
	Error() error
	Message() string
}

type Balance interface {
	Unconfirmed() uint64
	Confirmed() uint64
	Total() uint64
}

type TxNotification interface {
	TxID() string
	EventType() TxEventType
	BlockDetails() BlockDetails
}

type UtxoNotification interface {
	Utxos() []Utxo
	EventType() UtxoEventType
}

type TxEventType interface {
	IsUnknown() bool
	IsTxBroadcasted() bool
	IsTxUnconfirmed() bool
	IsTxConfirmed() bool
}

type UtxoEventType interface {
	IsUnknown() bool
	IsNew() bool
	IsConfirmed() bool
	IsLocked() bool
	IsUnlocked() bool
	IsSpent() bool
}

type BlockDetails interface {
	Hash() string
	Height() uint32
	Timestamp() int64
}

type TxDetails interface {
	Hash() string
	Hex() string
}

type Strategy interface {
	IsUnknown() bool
	IsBranchBound() bool
	IsFragment() bool
}

type UtxoKey interface {
	TxID() string
	Index() uint32
}

type Utxo interface {
	TxID() string
	Index() uint32
	Asset() string
	Value() uint64
	Script() []byte
	AssetBlinder() []byte
	ValueBlinder() []byte
	AccountName() string
}

type Input interface {
	UtxoKey
	Script() string
	ScriptSigSize() int
	WitnessSize() int
}

type Output interface {
	Asset() string
	Value() uint64
	Script() string
	BlindingKey() string
}
