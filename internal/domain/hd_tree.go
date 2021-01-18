package domain

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/taxi-daemon/pkg/wallet"
)

const (
	// ExternalChain is used to built derivation path
	ExternalChain uint = 0
	// InternalChain is used to build derivation path (for change addresses)
	InternalChain uint = 1
	// LBTCAccount is the index of the LBTC account
	LBTCAccount uint = 0
	// PayoutAccount is the index of the Payout account
	PayoutAccount uint = 1
)

// Account represents a wallet account
type Account struct {
	AccountIndex           uint
	NextExternalIndex      uint
	NextInternalIndex      uint
	DerivationPathByScript map[string]string
}

// NewAccount a factory function for the Account type
func NewAccount(accountIndex uint) *Account {
	return &Account{
		AccountIndex:           accountIndex,
		DerivationPathByScript: map[string]string{},
		NextExternalIndex:      0,
		NextInternalIndex:      0,
	}
}

func nextIndexAfter(index uint) (next uint) {
	// restart from 0 if index has reached the its max value
	next = 0
	if index != hdkeychain.HardenedKeyStart-1 {
		next = index + 1
	}
	return
}

// incrementExternalIndex increments and returns the last external index
func (a *Account) incrementExternalIndex() (next uint) {
	next = nextIndexAfter(a.NextExternalIndex)
	a.NextExternalIndex = next
	return
}

// incrementInternalIndex increments and returns the last external index
func (a *Account) incrementInternalIndex() (next uint) {
	next = nextIndexAfter(a.NextInternalIndex)
	a.NextInternalIndex = next
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
	Mnemonic           []string
	RootDerivationPath string
	Accounts           map[uint]*Account
	Network            network.Network
}

// NewHDTree is the factory function for the HDTree domain
func NewHDTree(mnemonic []string, rootPath wallet.DerivationPath, network network.Network) (*HDTree, error) {
	if len(mnemonic) <= 0 {
		return nil, errors.New("mnemonic can't be zero length string")
	}
	hdTree := &HDTree{
		Mnemonic:           mnemonic,
		Network:            network,
		RootDerivationPath: rootPath.String(),
		Accounts:           make(map[uint]*Account, 0),
	}

	return hdTree, nil
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
func (t *HDTree) DeriveNextAddressForLBTCAccount(isChange bool) (*AddressInfo, error) {
	return t.deriveNextAddressForAccount(LBTCAccount, isChange)
}

// DeriveNextAddressForPayoutAccount returns a new address for the Payout account
func (t *HDTree) DeriveNextAddressForPayoutAccount(isChange bool) (*AddressInfo, error) {
	return t.deriveNextAddressForAccount(PayoutAccount, isChange)
}

// DeriveAddressInRangeForAccount derive a set of addresses of a given account: from `start` to `start + numOfAddresses - 1`
func (t *HDTree) DeriveAddressInRangeForAccount(account, start, numOfAddresses uint, isChange bool) ([]AddressInfo, error) {
	selectedAccount := t.initOrGetAccount(account)
	if numOfAddresses == 0 {
		next := selectedAccount.NextExternalIndex
		if isChange {
			next = selectedAccount.NextInternalIndex
		}

		if next < 1 {
			return nil, nil
		}

		return t.deriveAddressInRangeForAccount(account, start, next, isChange)
	}

	return t.deriveAddressInRangeForAccount(account, start, numOfAddresses, isChange)
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

func (t *HDTree) deriveNextAddressForAccount(accountIndex uint, isChange bool) (*AddressInfo, error) {
	account := t.initOrGetAccount(accountIndex)
	addressIndex := account.NextExternalIndex
	if isChange {
		addressIndex = account.NextInternalIndex
	}

	addrInfo, derivationPath, err := t.deriveAddressInfo(accountIndex, addressIndex, isChange)
	if err != nil {
		return nil, err
	}

	account.addDerivationPath(addrInfo.Script, derivationPath.String())

	if isChange {
		account.incrementInternalIndex()
	} else {
		account.incrementExternalIndex()
	}

	return addrInfo, nil
}

func (t *HDTree) deriveAddressInfo(
	accountIndex, addressIndex uint,
	isChange bool,
) (*AddressInfo, wallet.DerivationPath, error) {
	rootPath, _ := wallet.ParseDerivationPath(t.RootDerivationPath)
	w, err := wallet.NewWallet(
		t.Mnemonic,
		rootPath,
		t.Network,
	)
	if err != nil {
		return nil, nil, err
	}

	chain := ExternalChain
	if isChange {
		chain = InternalChain
	}

	derivationPath := wallet.DerivationPath{
		uint32(accountIndex),
		uint32(chain),
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

func (t *HDTree) deriveAddressInRangeForAccount(
	accountIndex, start, numOfAddresses uint,
	isChange bool,
) ([]AddressInfo, error) {
	addrInfoList := make([]AddressInfo, numOfAddresses, numOfAddresses)
	j := 0
	for i := start; i < start+numOfAddresses; i++ {
		addrInfo, _, err := t.deriveAddressInfo(accountIndex, i, isChange)
		if err != nil {
			return nil, err
		}

		addrInfoList[j] = *addrInfo
		j++
	}
	return addrInfoList, nil
}

// AllDerivedAddressInfoForAcountAsync ...
func (t *HDTree) AllDerivedAddressInfoForAcountAsync(accountIndex uint) (chan *AddressInfo, chan error, uint) {
	const numWorkers = 4

	account := t.initOrGetAccount(accountIndex)
	size := account.NextExternalIndex + account.NextInternalIndex

	jobs := make(chan derivationJob, size)
	addresses := make(chan *AddressInfo, size)
	errorsChan := make(chan error)
	finishFlags := make(chan bool, numWorkers)

	for w := 0; w < numWorkers; w++ {
		go t.workerAddressDerivation(jobs, addresses, errorsChan, finishFlags)
	}

	go func() {
		i := 0
		for {
			select {
			case <-finishFlags:
				i++
				if i == numWorkers {
					close(finishFlags)
					close(addresses)
					return
				}
			}
		}
	}()

	indexesExternal := make([]uint, 0, int(account.NextExternalIndex))
	for indexExternal := uint(0); indexExternal < account.NextExternalIndex; indexExternal++ {
		indexesExternal = append(indexesExternal, indexExternal)
	}

	for i := range indexesExternal {
		job := derivationJob{
			accountIndex: accountIndex,
			addressIndex: uint(i),
			isChange:     false,
		}
		jobs <- job
	}

	indexesInternal := make([]uint, 0, int(account.NextInternalIndex))
	for indexInternal := uint(0); indexInternal < account.NextInternalIndex; indexInternal++ {
		indexesInternal = append(indexesInternal, indexInternal)
	}

	for i := range indexesInternal {
		job := derivationJob{
			accountIndex: accountIndex,
			addressIndex: uint(i),
			isChange:     true,
		}
		jobs <- job
	}

	close(jobs)

	return addresses, errorsChan, size
}

func (t *HDTree) allDerivedAddressInfoForAccount(accountIndex uint) ([]AddressInfo, error) {
	addresses, errorsChan, size := t.AllDerivedAddressInfoForAcountAsync(accountIndex)
	addrInfoList := make([]AddressInfo, 0, size)

	for i := 0; i < int(size); i++ {
		select {
		case addr := <-addresses:
			addrInfoList = append(addrInfoList, *addr)
		case err := <-errorsChan:
			return nil, err
		}
	}

	return addrInfoList, nil
}

type derivationJob struct {
	accountIndex uint
	addressIndex uint
	isChange     bool
}

func (t HDTree) workerAddressDerivation(
	jobs <-chan derivationJob,
	resultAddresses chan<- *AddressInfo,
	errors chan<- error,
	finishFlags chan<- bool,
) {
	for job := range jobs {
		addrInfo, _, err := t.deriveAddressInfo(job.accountIndex, job.addressIndex, job.isChange)
		if err != nil {
			errors <- err
		}
		resultAddresses <- addrInfo
	}

	finishFlags <- true
}

func (t *HDTree) initOrGetAccount(accountIndex uint) *Account {
	account, ok := t.Accounts[accountIndex]
	if !ok {
		account = NewAccount(accountIndex)
		t.Accounts[accountIndex] = account
	}
	return account
}
