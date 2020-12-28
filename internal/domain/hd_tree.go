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
	ExternalChain uint = 0

	LBTCAccount uint = iota
	PayoutAccount
)

type Account struct {
	AccountIndex           uint
	NextExternalIndex      uint
	DerivationPathByScript map[string]string
}

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

type HDTree struct {
	RootDerivationPath string
	Accounts           map[uint]*Account
}

func NewHDTree(rootPath wallet.DerivationPath) (*HDTree, error) {
	if len(config.GetMnemonic()) <= 0 {
		return nil, errors.New("mnemonic must be set in env")
	}
	return &HDTree{
		RootDerivationPath: rootPath.String(),
		Accounts:           make(map[uint]*Account, 0),
	}, nil
}

type AddressInfo struct {
	Address string
	Script  string
}

func (t *HDTree) DeriveNextAddressForLBTCAccount() (*AddressInfo, error) {
	return t.deriveNextAddressForAccount(LBTCAccount)
}

func (t *HDTree) DeriveNextAddressForPayoutAccount() (*AddressInfo, error) {
	return t.deriveNextAddressForAccount(PayoutAccount)
}

func (t *HDTree) DerivationPathForScript(script string) (string, bool) {
	for _, account := range t.Accounts {
		if derivationPath, ok := account.DerivationPathByScript[script]; ok {
			return derivationPath, true
		}
	}
	return "", false
}

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

func (t *HDTree) AllDerivedAddressInfoForLBTCAccount() ([]AddressInfo, error) {
	return t.allDerivedAddressInfoForAccount(LBTCAccount)
}

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

	p, err := w.DerivePath(derivationPath.String(), config.GetNetwork())
	if err != nil {
		return nil, nil, err
	}
	addr, _ := p.WitnessPubKeyHash()
	script := hex.EncodeToString(p.WitnessScript)
	return &AddressInfo{
		Address: addr,
		Script:  script,
	}, derivationPath, nil
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
