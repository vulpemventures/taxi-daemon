package application

import (
	"context"
	"errors"

	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

type TaxiService interface {
	TopupWithAsset(assetHash string, feeAmount uint64) (*domain.Topup, uint64, error)
	ListAssets() []string
}

type taxiService struct {
	utxosLocker        domain.UtxoKeyRepository
	topupRepository    domain.TopupRepository
	eplorer            ExplorerService
	taxiAccountAddress string
}

func NewTaxiService(explorerEndpoint string) TaxiService {
	return &taxiService{
		utxosLocker:     inmemory.NewInmemoryUtxoKeyRepository(),
		topupRepository: inmemory.NewInmemoryTopupRepository(),
		eplorer:         NewEsplorerService(explorerEndpoint),
	}
}

func (taxi *taxiService) TopupWithAsset(assetHash string, feeAmount uint64) (*domain.Topup, uint64, error) {
	// TO DO
	// estimate fee increase
	// lock utxo
	// create tx with taxi's utxo and outputs
	// set locktime on tx matching the unspent lock

	panic("not implemented")
}

func (taxi *taxiService) ListAssets() []string {
	panic("not implemented")
}

func (taxi *taxiService) fetchAvailableUtxos() ([]domain.Utxo, error) {
	utxos, err := taxi.eplorer.GetUnspents(taxi.taxiAccountAddress)
	if err != nil {
		return nil, err
	}

	availableUtxos := make([]domain.Utxo, 0)

	for _, utxo := range utxos {
		if !taxi.utxosLocker.Contains(context.Background(), utxo.Key()) {
			availableUtxos = append(availableUtxos, utxo)
		}
	}

	return availableUtxos, nil
}

func coinSelect(utxos []domain.Utxo, targetAmount uint64, targetAsset string) ([]domain.Utxo, uint64, error) {
	var totalAmount uint64 = 0
	selectedUtxos := make([]domain.Utxo, 0)

	for _, utxo := range utxos {
		if !utxo.IsConfidential() && utxo.Asset == targetAsset {
			totalAmount += utxo.Value
			selectedUtxos = append(selectedUtxos, utxo)
			if totalAmount >= targetAmount {
				break
			}
		}
	}

	if totalAmount < targetAmount {
		return nil, 0, errors.New(
			"error on target amount: total utxo amount does not cover target amount",
		)
	}

	changeAmount := totalAmount - targetAmount

	return selectedUtxos, changeAmount, nil
}
