package fragmenter

import (
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/tdex-network/tdex-daemon/pkg/crawler"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
)

const (
	psetQueueMaxSize  = 100
	errorQueueMaxSize = 10
	defaultFeeValue   = 3000
)

// Service is the interface for Fragmenter
type Service interface {
	Start()
	Stop()
	AddAddress(address string)
	RemoveAddress(address string)
	GetFragmentPSETChan() chan string
}

// Fragment is the return type of the fragmenter
type Fragment struct {
	Inputs   []*transaction.TxInput
	Prevouts []*transaction.TxOutput
	Change   uint64
}

type utxoFragmenter struct {
	crawlerSvc          crawler.Service
	explorerSvc         explorer.Service
	errChan             chan error
	quitChan            chan int
	ticker              *time.Ticker
	fragmentSize        uint64
	fragmentAsset       string
	utxosMap            map[string][]explorer.Utxo
	locker              sync.Locker
	alreadyUsedUtxosMap map[explorer.Utxo]bool
	errorHandler        func(err error)
	fragmentPSETChan    chan string
	receiveAddress      string
	changeAddress       string
}

// Opts defines the parameter needed to init an utxo fragmenter
type Opts struct {
	ExplorerSvc            explorer.Service
	IntervalInMilliseconds int
	ErrorHandler           func(err error)
	FragmentSize           uint64
	FragmentAsset          string
	ReceiveAddress         string
	ChangeAddress          string
}

// NewService is a factory function for fragmenter.Service interface
func NewService(opts Opts) Service {
	crawlerSvc := crawler.NewService(crawler.Opts{
		ExplorerSvc:            opts.ExplorerSvc,
		IntervalInMilliseconds: opts.IntervalInMilliseconds,
		Observables:            []crawler.Observable{},
		ErrorHandler:           opts.ErrorHandler,
	})

	return &utxoFragmenter{
		crawlerSvc:          crawlerSvc,
		explorerSvc:         opts.ExplorerSvc,
		errChan:             make(chan error, errorQueueMaxSize),
		quitChan:            make(chan int),
		ticker:              time.NewTicker(3 * time.Minute),
		fragmentSize:        opts.FragmentSize,
		fragmentAsset:       opts.FragmentAsset,
		fragmentPSETChan:    make(chan string, psetQueueMaxSize),
		utxosMap:            make(map[string][]explorer.Utxo),
		locker:              &sync.Mutex{},
		alreadyUsedUtxosMap: make(map[explorer.Utxo]bool),
		receiveAddress:      opts.ReceiveAddress,
		changeAddress:       opts.ChangeAddress,
	}
}

func (f *utxoFragmenter) GetFragmentPSETChan() chan string {
	return f.fragmentPSETChan
}

func (f *utxoFragmenter) Start() {
	go f.crawlerSvc.Start()
	crawlerEventChan := f.crawlerSvc.GetEventChannel()

	for {
		select {
		case event := <-crawlerEventChan:
			go f.handleEvent(event)
		case <-f.ticker.C:
			fragments := f.createFragments()
			if len(fragments) > 0 {
				psetBase64, err := f.createTxFromFragments(fragments)
				if err != nil {
					f.errorHandler(err)
					continue
				}
				f.fragmentPSETChan <- psetBase64
			}
		case <-f.quitChan:
			return
		}
	}
}

func (f *utxoFragmenter) Stop() {
	f.crawlerSvc.Stop()
	f.quitChan <- 1
	close(f.fragmentPSETChan)
}

func (f *utxoFragmenter) AddAddress(address string) {
	f.crawlerSvc.AddObservable(&crawler.AddressObservable{
		Address: address,
		// AccountIndex & Blinding not used --> taxi expect only unblinded tx
		AccountIndex: -1,
		BlindingKey:  []byte{},
	})
}

func (f *utxoFragmenter) RemoveAddress(address string) {
	f.crawlerSvc.RemoveObservable(&crawler.AddressObservable{
		Address:      address,
		AccountIndex: -1,
		BlindingKey:  []byte{},
	})
}

func (f *utxoFragmenter) handleEvent(crawlerEvent crawler.Event) {
	addressEvent, isAddressEvent := crawlerEvent.(crawler.AddressEvent)
	if isAddressEvent {
		// addressEvent.
		f.safeSetUtxos(addressEvent.Address, addressEvent.Utxos)
	}
}

func (f *utxoFragmenter) safeSetUtxos(address string, utxos []explorer.Utxo) {
	f.locker.Lock()
	defer f.locker.Unlock()

	f.utxosMap[address] = utxos
}

type utxoWithAddress struct {
	utxo    explorer.Utxo
	address string
}

func (f *utxoFragmenter) createFragments() []Fragment {
	f.locker.Lock()
	defer f.locker.Unlock()

	utxosWithAddresses := make([]utxoWithAddress, 0)

	// filter utxos
	for k, value := range f.utxosMap {
		for _, utxo := range value {
			if utxo.IsConfidential() {
				continue
			}

			if utxo.Asset() != f.fragmentAsset {
				continue
			}

			if utxo.Value() == f.fragmentSize {
				continue
			}

			_, locked := f.alreadyUsedUtxosMap[utxo]
			if locked {
				continue
			}

			utxosWithAddresses = append(utxosWithAddresses, utxoWithAddress{
				utxo:    utxo,
				address: k,
			})
		}
	}

	fragments := make([]Fragment, 0)

	toCumulate := make([]utxoWithAddress, 0)
	var sumValuesToCumulate uint64

	for _, utxoWithAddr := range utxosWithAddresses {
		// first case, utxo's value is greater than fragmentSize
		diff := utxoWithAddr.utxo.Value() - f.fragmentSize
		if diff >= 0 {
			fragment, err := f.createFragment([]explorer.Utxo{utxoWithAddr.utxo}, diff)
			if err != nil {
				continue
			}

			fragments = append(fragments, *fragment)
			continue
		}

		// else: the utxo value < fragmentSize
		// cumulate before
		if sumValuesToCumulate >= f.fragmentSize {
			fragment, err := f.cumulate(toCumulate)
			if err == nil {
				fragments = append(fragments, *fragment)
				toCumulate = make([]utxoWithAddress, 0)
			}
		}

		// append to toCumulate array
		toCumulate = append(toCumulate, utxoWithAddr)
		sumValuesToCumulate += utxoWithAddr.utxo.Value()
	}

	return fragments
}

func (f *utxoFragmenter) lockUtxoFor5Minutes(utxo explorer.Utxo) {
	f.alreadyUsedUtxosMap[utxo] = true
	go func() {
		f.locker.Lock()
		defer f.locker.Unlock()

		time.Sleep(5 * time.Minute)
		delete(f.alreadyUsedUtxosMap, utxo)
	}()
}

func (f *utxoFragmenter) cumulate(utxosWithAddresses []utxoWithAddress) (*Fragment, error) {
	var sum uint64

	utxos := make([]explorer.Utxo, 0)

	for _, utxoWithAddr := range utxosWithAddresses {
		sum += utxoWithAddr.utxo.Value()
		utxos = append(utxos, utxoWithAddr.utxo)
	}

	change := sum - f.fragmentSize
	return f.createFragment(utxos, change)
}

func (f *utxoFragmenter) createFragment(utxos []explorer.Utxo, changeAmount uint64) (*Fragment, error) {
	inputs := make([]*transaction.TxInput, 0)
	prevouts := make([]*transaction.TxOutput, 0)

	for _, utxo := range utxos {
		txInput, TxOutput, err := utxo.Parse()
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, txInput)
		prevouts = append(prevouts, TxOutput)
	}

	return &Fragment{
		Inputs:   inputs,
		Prevouts: prevouts,
		Change:   changeAmount,
	}, nil
}

func (f *utxoFragmenter) createTxFromFragments(fragments []Fragment) (string, error) {
	fragmentSizeElementsValue, err := elementsutil.SatoshiToElementsValue(f.fragmentSize)
	if err != nil {
		return "", err
	}

	fragmentAssetBytes, err := assetHashBytes(f.fragmentAsset)
	if err != nil {
		return "", err
	}

	outputScript, err := address.ToOutputScript(f.receiveAddress)
	if err != nil {
		return "", err
	}

	fragmentTx, err := pset.New([]*transaction.TxInput{}, []*transaction.TxOutput{}, 2, 0)
	if err != nil {
		return "", err
	}

	updater, err := pset.NewUpdater(fragmentTx)
	if err != nil {
		return "", err
	}

	inputIndex := 0
	var changeAmount uint64

	for _, fragment := range fragments {
		// add the inputs
		for i, input := range fragment.Inputs {
			updater.AddInput(input)
			err = updater.AddInWitnessUtxo(fragment.Prevouts[i], inputIndex)
			if err != nil {
				return "", err
			}
			inputIndex++
		}

		// add a fragment output
		fragmentOutput := transaction.NewTxOutput(fragmentAssetBytes, fragmentSizeElementsValue, outputScript)
		updater.AddOutput(fragmentOutput)
		// increase the change total amount
		changeAmount += fragment.Change
	}

	// TODO estimate the fee
	feeValue := uint64(5000)
	changeAmount -= feeValue

	if changeAmount < 0 {
		return "", errors.New("not enough fund to pay the fees")
	}

	feeElementsValue, err := elementsutil.SatoshiToElementsValue(feeValue)
	if err != nil {
		return "", err
	}

	feeOutput := transaction.NewTxOutput(fragmentAssetBytes, feeElementsValue, []byte{})
	updater.AddOutput(feeOutput)

	if changeAmount > 0 {
		changeScript, err := address.ToOutputScript(f.changeAddress)
		if err != nil {
			return "", err
		}

		changeElementsValue, err := elementsutil.SatoshiToElementsValue(changeAmount)
		if err != nil {
			return "", err
		}

		changeOutput := transaction.NewTxOutput(fragmentAssetBytes, changeElementsValue, changeScript)
		updater.AddOutput(changeOutput)
	}

	fragmentTxBase64, err := fragmentTx.ToBase64()
	if err != nil {
		return "", err
	}

	return fragmentTxBase64, nil
}

func assetHashBytes(asset string) ([]byte, error) {
	buf, err := hex.DecodeString(asset)
	if err != nil {
		return nil, err
	}
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf, nil
}
