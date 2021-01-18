package application

import (
	"context"
	"math/rand"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/taxi-daemon/internal/domain"
)

// Outpoint is the unique utxo identifier
type Outpoint struct {
	TxID string
	Vout uint32
}

// FragmentCache is a service used to manage fragment utxo caching
type FragmentCache interface {
	Start()
	Stop()
	Delete(outpoint Outpoint) error
	RandomGetAndDelete() (explorer.Utxo, error)
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

// NewCacheArgs is a wrapper struct for NewCache constructor
type NewCacheArgs struct {
	Explorer         explorer.Service
	HdTreeRepository domain.HDTreeRepository
	LockRepository   domain.LockRepository
	FragmentAsset    string
	FragmentAmount   uint64
	Interval         time.Duration
}

// NewCache is a factory function for FragmentCache service
func NewCache(args NewCacheArgs) FragmentCache {
	return newCache(args)
}

func newCache(args NewCacheArgs) FragmentCache {
	return &fragmentCache{
		outpointToUtxo:   make(map[Outpoint]explorer.Utxo),
		locker:           &sync.Mutex{},
		explorer:         args.Explorer,
		hdTreeRepository: args.HdTreeRepository,
		lockRepository:   args.LockRepository,
		fragmentAsset:    args.FragmentAsset,
		fragmentAmount:   args.FragmentAmount,
		interval:         args.Interval,
		tickerInterval:   time.NewTicker(args.Interval),
		stopSignal:       make(chan bool),
	}
}

func (f *fragmentCache) Delete(outpoint Outpoint) error {
	f.locker.Lock()
	defer f.locker.Unlock()

	_, ok := f.outpointToUtxo[outpoint]
	if !ok {
		return ErrUtxoNotFoundInCache
	}

	return nil
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

func (f *fragmentCache) RandomGetAndDelete() (explorer.Utxo, error) {
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

func (f *fragmentCache) Start() {
	log.Info("start utxos caching...")
	ctx := context.Background()

	f.tickerInterval = time.NewTicker(f.interval)

	for range f.tickerInterval.C {
		// retrieve root path of the hd tree and instantiate the inmemory wallet
		hdTree, err := f.hdTreeRepository.GetOrCreateHDTree(ctx)
		if err != nil {
			log.Error(err)
			return
		}

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

func (f *fragmentCache) Stop() {
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
