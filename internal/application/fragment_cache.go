package application

import (
	"context"
	"math/rand"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// Outpoint is the unique utxo identifier
type Outpoint struct {
	TxID string
	Vout uint32
}

type fragmentCache struct {
	outpointToUtxo   map[Outpoint]explorer.Utxo
	locker           sync.Locker
	explorer         explorer.Service
	hdTreeRepository domain.HDTreeRepository
	lockRepository   domain.LockRepository
	fragmentAsset    string
	fragmentAmount   uint64
	interval         time.Duration
	tickerInterval   *time.Ticker
	stopSignal       chan bool
}

func newCache(taxiService *taxiService, interval time.Duration) *fragmentCache {
	return &fragmentCache{
		outpointToUtxo:   make(map[Outpoint]explorer.Utxo),
		locker:           &sync.Mutex{},
		explorer:         taxiService.explorer,
		hdTreeRepository: taxiService.hdTreeRepository,
		lockRepository:   taxiService.lockRepository,
		fragmentAsset:    taxiService.network.AssetID,
		fragmentAmount:   taxiService.fixedFeeAmount,
		interval:         interval,
		tickerInterval:   time.NewTicker(interval),
		stopSignal:       make(chan bool),
	}
}

func (f *fragmentCache) start() {
	log.Info("start utxos caching...")
	ctx := context.Background()
	// retrieve root path of the hd tree and instantiate the inmemory wallet
	hdTree, err := f.hdTreeRepository.GetOrCreateHDTree(ctx)
	if err != nil {
		log.Error(err)
		return
	}

	f.tickerInterval = time.NewTicker(f.interval)

	for range f.tickerInterval.C {
		// get all funding derived addresses asynchronously
		addresses, errors, _ := hdTree.AllDerivedAddressInfoForAcountAsync(domain.LBTCAccount)
		if err != nil {
			log.Error(err)
			return
		}

		const numOfWorker = 4

		utxos := make(chan explorer.Utxo)
		signals := make(chan bool, numOfWorker)

		for w := 0; w < numOfWorker; w++ {
			go f.getUnspentsWorker(addresses, utxos, errors, signals)
		}

		i := 0
		for {
			select {
			case <-f.stopSignal:
				return
			case utxo := <-utxos:
				lock := domain.UtxoToLock(utxo)
				if !f.lockRepository.IsLocked(ctx, lock) && !f.contains(utxo) {
					f.set(utxo)
					log.Info("new fragment utxo added to cache. (txid: ", utxo.Hash(), " index: ", utxo.Index(), ")")
				}
			case <-signals:
				i++
			case err := <-errors:
				log.Error(err)
			}

			if i == numOfWorker {
				close(utxos)
				close(signals)
				close(errors)
				break
			}
		}
	}
}

func (f *fragmentCache) stop() {
	log.Info("stop the fragment cache")
	f.tickerInterval.Stop()
	f.stopSignal <- true
}

func (f fragmentCache) getUnspentsWorker(
	jobs <-chan *domain.AddressInfo,
	results chan<- explorer.Utxo,
	errors chan<- error,
	finishSignals chan<- bool,
) {
	const maxErrors = 10

	nbError := 0
	for addr := range jobs {
		utxos, err := f.explorer.GetUnspents(addr.Address, [][]byte{})
		if err != nil {
			nbError++
			errors <- err

			if nbError >= maxErrors {
				break
			}

			continue
		}

		for _, utxo := range utxos {
			if utxo.Asset() == f.fragmentAsset && utxo.Value() == f.fragmentAmount {
				results <- utxo
			}
		}
	}

	finishSignals <- true
}

func (f *fragmentCache) contains(utxo explorer.Utxo) (yes bool) {
	f.locker.Lock()
	defer f.locker.Unlock()

	outpoint := Outpoint{
		TxID: utxo.Hash(),
		Vout: utxo.Index(),
	}

	_, yes = f.outpointToUtxo[outpoint]
	return
}

func (f *fragmentCache) set(utxo explorer.Utxo) {
	f.locker.Lock()
	defer f.locker.Unlock()

	outpoint := Outpoint{
		TxID: utxo.Hash(),
		Vout: utxo.Index(),
	}

	f.outpointToUtxo[outpoint] = utxo
}

func (f *fragmentCache) randomGetAndDelete() (explorer.Utxo, error) {
	f.locker.Lock()
	defer f.locker.Unlock()

	size := len(f.outpointToUtxo)
	if size == 0 {
		return nil, nil
	}

	rand.Seed(time.Now().UnixNano())
	r := rand.Intn(size)
	for outpoint, utxo := range f.outpointToUtxo {
		if r == 0 {
			delete(f.outpointToUtxo, outpoint)
			return utxo, nil
		}
		r--
	}
	return nil, ErrRandomIntegerUnreachable
}
