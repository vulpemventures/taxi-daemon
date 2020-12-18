package fragmenter

import (
	"sync"
	"time"

	"github.com/tdex-network/tdex-daemon/pkg/crawler"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
)

const (
	psetQueueMaxSize  = 100
	errorQueueMaxSize = 10
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
	InputIndexToAddress map[int]string
}

type utxoFragmenter struct {
	crawlerSvc    crawler.Service
	errChan       chan error
	quitChan      chan int
	fragmentChan  chan Fragment
	ticker        *time.Ticker
	fragmentSize  uint64
	fragmentAsset string
	utxosMap      map[string][]explorer.Utxo
	locker        sync.Locker
}

// Opts defines the parameter needed to init an utxo fragmenter
type Opts struct {
	ExplorerEndpoint       string
	IntervalInMilliseconds int
	ErrorHandler           func(err error)
	FragmentSize           uint64
	FragmentAsset          string
}

// NewService is a factory function for fragmenter.Service interface
func NewService(opts Opts) Service {
	explorerSvc := explorer.NewService(opts.ExplorerEndpoint)
	crawlerSvc := crawler.NewService(crawler.Opts{
		ExplorerSvc:            explorerSvc,
		IntervalInMilliseconds: opts.IntervalInMilliseconds,
		Observables:            []crawler.Observable{},
		ErrorHandler:           opts.ErrorHandler,
	})

	return &utxoFragmenter{
		crawlerSvc:    crawlerSvc,
		errChan:       make(chan error, errorQueueMaxSize),
		quitChan:      make(chan int),
		fragmentChan:  make(chan Fragment, psetQueueMaxSize),
		ticker:        time.NewTicker(time.Duration(opts.IntervalInMilliseconds+100) * time.Millisecond),
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

func (f *utxoFragmenter) craftMergeTx(utxos []explorer.Utxo) Fragment {
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

			utxosWithAddresses = append(utxosWithAddresses, utxoWithAddress{
				utxo:    utxo,
				address: k,
			})
		}

	}
}
