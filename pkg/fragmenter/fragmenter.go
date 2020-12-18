package fragmenter

import (
	"encoding/hex"
	"sync"
	"time"

	"github.com/tdex-network/tdex-daemon/pkg/crawler"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
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
}

// Fragment is the return type of the fragmenter
type Fragment struct {
	Pset                string
	Amount              uint64
	Change              uint64
	InputIndexToAddress map[int]string
}

type utxoFragmenter struct {
	crawlerSvc          crawler.Service
	explorerSvc         explorer.Service
	errChan             chan error
	quitChan            chan int
	fragmentChan        chan Fragment
	ticker              *time.Ticker
	fragmentSize        uint64
	fragmentAsset       string
	utxosMap            map[string][]explorer.Utxo
	locker              sync.Locker
	alreadyUsedUtxosMap map[explorer.Utxo]bool
}

// Opts defines the parameter needed to init an utxo fragmenter
type Opts struct {
	ExplorerSvc            explorer.Service
	IntervalInMilliseconds int
	ErrorHandler           func(err error)
	FragmentSize           uint64
	FragmentAsset          string
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
		crawlerSvc:    crawlerSvc,
		explorerSvc:   opts.ExplorerSvc,
		errChan:       make(chan error, errorQueueMaxSize),
		quitChan:      make(chan int),
		fragmentChan:  make(chan Fragment, psetQueueMaxSize),
		ticker:        time.NewTicker(time.Duration(3*opts.IntervalInMilliseconds) * time.Millisecond),
		fragmentSize:  opts.FragmentSize,
		fragmentAsset: opts.FragmentAsset,
	}
}

func (f *utxoFragmenter) Start() {
	f.crawlerSvc.Start()
	crawlerEventChan := f.crawlerSvc.GetEventChannel()

	for {
		select {
		case event := <-crawlerEventChan:
			go f.handleEvent(event)
		case <-f.ticker.C:
			fragmentUtxos()
		}
	}
}

func (f *utxoFragmenter) handleEvent(crawlerEvent crawler.Event) {
	addressEvent, isAddressEvent := crawlerEvent.(crawler.AddressEvent)
	if isAddressEvent {
		// addressEvent.
		f.safeSetUtxos(addressEvent.Address, addressEvent.Utxos)
	}
}

func (f *utxoFragmenter) Stop() {
	f.crawlerSvc.Stop()
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

func (f *utxoFragmenter) safeSetUtxos(address string, utxos []explorer.Utxo) {
	f.locker.Lock()
	defer f.locker.Unlock()

	f.utxosMap[address] = utxos
}

type utxoWithAddress struct {
	utxo    explorer.Utxo
	address string
}

func (f *utxoFragmenter) craftMergeTx(utxos []explorer.Utxo) []Fragment {
	f.locker.Lock()
	defer f.locker.Unlock()

	utxosWithAddresses := make([]utxoWithAddress, 0)

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

	amountNeeded := f.fragmentSize + defaultFeeValue
	for _, utxoWithAddress := range utxosWithAddresses {
		// first case, utxo's value is greater than fragmentSize
		if utxoWithAddress.utxo.Value() > f.fragmentSize {
			fragment, err := f.createFragment([]explorer.Utxo{utxoWithAddress.utxo})
			if err != nil {
				continue
			}

		}
	}

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

func (f *utxoFragmenter) createFragment(utxos []explorer.Utxo, changeAmount uint64) (*Fragment, error) {
	fragmentTx, err := pset.New([]*transaction.TxInput{}, []*transaction.TxOutput{}, 2, 0)
	if err != nil {
		return nil, err
	}

	updater, err := pset.NewUpdater(fragmentTx)
	if err != nil {
		return nil, err
	}

	for index, utxo := range utxos {
		buf, err := hex.DecodeString(utxo.Hash())
		if err != nil {
			return nil, err
		}
		hashBytes := elementsutil.ReverseBytes(buf)

		txInput := transaction.NewTxInput(hashBytes, utxo.Index())
		updater.AddInput(txInput)

		hex, err := f.explorerSvc.GetTransactionHex(utxo.Hash())
		if err != nil {
			return nil, err
		}

		tx, err := transaction.NewTxFromHex(hex)
		if err != nil {
			return nil, err
		}

		prevout := tx.Outputs[int(utxo.Index())]

		err = updater.AddInWitnessUtxo(prevout, index)
		if err != nil {
			return nil, err
		}
	}

	base64, err := fragmentTx.ToBase64()
	if err != nil {
		return nil, err
	}

	return &Fragment{
		Pset:   base64,
		Amount: f.fragmentSize,
		Change: changeAmount,
	}, nil
}
