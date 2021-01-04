package domain

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

const (
	// ExternalChain is used to built derivation path
	ExternalChain uint = 0
	// LBTCAccount is the index of the LBTC account
	LBTCAccount uint = iota
	// PayoutAccount is the index of the Payout account
	PayoutAccount
)

// Account represents a wallet account
type Account struct {
	AccountIndex           uint
	NextExternalIndex      uint
	DerivationPathByScript map[string]string
}

// NewAccount a factory function for the Account type
func NewAccount(accountIndex uint) *Account {
	return &Account{
		AccountIndex:           accountIndex,
		DerivationPathByScript: map[string]string{},
		NextExternalIndex:      0,
	}
}

// incrementExternalIndex increments and returns the last external index
func (a *Account) incrementExternalIndex() (next uint) {
	// restart from 0 if index has reached the its max value
	next = 0
	if a.NextExternalIndex != hdkeychain.HardenedKeyStart-1 {
		next = a.NextExternalIndex + 1
	}
	a.NextExternalIndex = next
	return
}

// addDerivationPath adds a new entry to the map: script -> derivation_path
func (a *Account) addDerivationPath(script, derivationPath string) {
	if _, ok := a.DerivationPathByScript[script]; !ok {
		a.DerivationPathByScript[script] = derivationPath
	}
}

// HDTree represents a HD wallet
type HDTree struct {
	RootDerivationPath string
	Accounts           map[uint]*Account
}

// NewHDTree is the factory function for the HDTree domain
func NewHDTree(rootPath wallet.DerivationPath) (*HDTree, error) {
	if len(config.GetMnemonic()) <= 0 {
		return nil, errors.New("mnemonic must be set in env")
	}
	return &HDTree{
		RootDerivationPath: rootPath.String(),
		Accounts:           make(map[uint]*Account, 0),
	}, nil
}

// AddressInfo defines all the address data
type AddressInfo struct {
	Address        string
	Script         string
	DerivationPath string
	Index          uint
	PublicKey      string
}

// DeriveNextAddressForLBTCAccount returns a new address for the LBTCAccount
func (t *HDTree) DeriveNextAddressForLBTCAccount() (*AddressInfo, error) {
	return t.deriveNextAddressForAccount(LBTCAccount)
}

// DeriveNextAddressForPayoutAccount returns a new address for the Payout account
func (t *HDTree) DeriveNextAddressForPayoutAccount() (*AddressInfo, error) {
	return t.deriveNextAddressForAccount(PayoutAccount)
}

// DeriveAddressInRangeForAccount derive a set of addresses of a given account: from `start` to `start + numOfAddresses - 1`
func (t *HDTree) DeriveAddressInRangeForAccount(account, start, numOfAddresses uint) ([]AddressInfo, error) {
	if start == 0 && numOfAddresses == 0 {
		return t.allDerivedAddressInfoForAccount(account)
	}

	if numOfAddresses == 0 {
		numOfAddresses := t.Accounts[account].NextExternalIndex - 1
		if numOfAddresses < 0 {
			return nil, nil
		}
	}

	return t.deriveAddressInRangeForAccount(account, start, numOfAddresses)
}

// DerivationPathForScript returns, if exist in HDTree accounts, the derivation path of the given script
func (t *HDTree) DerivationPathForScript(script string) (string, bool) {
	for _, account := range t.Accounts {
		if derivationPath, ok := account.DerivationPathByScript[script]; ok {
			return derivationPath, true
		}
	}
	return "", false
}

// DerivationPathForAddress returns, if exist in HDTree accounts, the derivation path of the given address
func (t *HDTree) DerivationPathForAddress(addr string) (string, bool) {
	script, err := address.ToOutputScript(addr)
	if err != nil {
		return "", false
	}

	for _, account := range t.Accounts {
		derivationPath, ok := account.DerivationPathByScript[hex.EncodeToString(script)]
		if ok {
			return derivationPath, true
		}
	}
	return "", false
}

// AllDerivedAddressInfoForLBTCAccount returns all the AddressInfo for the LBTCAccount
func (t *HDTree) AllDerivedAddressInfoForLBTCAccount() ([]AddressInfo, error) {
	return t.allDerivedAddressInfoForAccount(LBTCAccount)
}

// AllDerivedAddressInfoForPayoutAccount returns all the AddressInfo for the PayoutAccount
func (t *HDTree) AllDerivedAddressInfoForPayoutAccount() ([]AddressInfo, error) {
	return t.allDerivedAddressInfoForAccount(PayoutAccount)
}

func (t *HDTree) deriveNextAddressForAccount(accountIndex uint) (*AddressInfo, error) {
	account := t.initOrGetAccount(accountIndex)
	addrInfo, derivationPath, err := t.deriveAddressInfo(accountIndex, account.NextExternalIndex)
	if err != nil {
		return nil, err
	}

	account.addDerivationPath(addrInfo.Script, derivationPath.String())
	account.incrementExternalIndex()

	return addrInfo, nil
}

func (t *HDTree) deriveAddressInfo(
	accountIndex, addressIndex uint,
) (*AddressInfo, wallet.DerivationPath, error) {
	rootPath, _ := wallet.ParseDerivationPath(t.RootDerivationPath)
	w, err := wallet.NewWallet(
		config.GetMnemonic(),
		rootPath,
		*config.GetNetwork(),
	)
	if err != nil {
		return nil, nil, err
	}

	derivationPath := wallet.DerivationPath{
		uint32(accountIndex),
		uint32(ExternalChain),
		uint32(addressIndex),
	}

	p, err := w.DerivePath(derivationPath.String())
	if err != nil {
		return nil, nil, err
	}
	addr, _ := p.WitnessPubKeyHash()
	script := hex.EncodeToString(p.WitnessScript)
	fullDerivationPath := append(rootPath, derivationPath...)
	pubkey := hex.EncodeToString(p.PublicKey.SerializeCompressed())

	return &AddressInfo{
		Address:        addr,
		Script:         script,
		DerivationPath: fullDerivationPath.String(),
		Index:          addressIndex,
		PublicKey:      pubkey,
	}, derivationPath, nil
}

func (t *HDTree) deriveAddressInRangeForAccount(accountIndex, start, numOfAddresses uint) ([]AddressInfo, error) {
	addrInfoList := make([]AddressInfo, numOfAddresses, numOfAddresses)
	j := 0
	for i := start; i < start+numOfAddresses; i++ {
		addrInfo, _, err := t.deriveAddressInfo(accountIndex, i)
		if err != nil {
			return nil, err
		}

		addrInfoList[j] = *addrInfo
		j++
	}
	return addrInfoList, nil
}

func (t *HDTree) allDerivedAddressInfoForAccount(accountIndex uint) ([]AddressInfo, error) {
	account := t.initOrGetAccount(accountIndex)
	addrInfoList := make([]AddressInfo, account.NextExternalIndex, account.NextExternalIndex)

	for i := uint(0); i < account.NextExternalIndex; i++ {
		addrInfo, _, err := t.deriveAddressInfo(accountIndex, i)
		if err != nil {
			return nil, err
		}
		addrInfoList[i] = *addrInfo
	}

	return addrInfoList, nil
}

func (t *HDTree) initOrGetAccount(accountIndex uint) *Account {
	account, ok := t.Accounts[accountIndex]
	if !ok {
		account = NewAccount(accountIndex)
		t.Accounts[accountIndex] = account
	}
	return account
}
