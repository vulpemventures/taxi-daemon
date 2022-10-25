package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
)

const (
	minMillisatPerByte     = 100
	defaultMillisatPerByte = 100
)

type TaxiService interface {
	TopupWithAsset(
		ctx context.Context, assetHash string,
		estimatedTxSize, millisatPerByte uint64,
	) (*domain.Topup, []ports.Utxo, error)
	TopupWithKnownPayout(
		ctx context.Context, psetBase64 string, millisatPerByte uint64,
		userUnblindedInput []UnblindedInput,
	) (*domain.Topup, error)
	ListAssets(ctx context.Context) ([]*domain.Price, error)
}

type taxiService struct {
	wallet      Wallet
	repoManager ports.RepoManager
}

func NewTaxiService(
	wallet Wallet, repoManager ports.RepoManager,
) TaxiService {
	return newTaxiService(wallet, repoManager)
}

func newTaxiService(
	wallet Wallet, repoManager ports.RepoManager,
) *taxiService {
	svc := &taxiService{wallet, repoManager}
	svc.registerHandlerForSpentUtxos()
	svc.registerHandlerForExpiredUtxos()
	return svc
}

// TopupWithAsset returns a transaction letting to pay a given amount of fees with assetHash token.
func (s *taxiService) TopupWithAsset(
	ctx context.Context, assetHash string,
	estimatedTxSize, millisatPerByte uint64,
) (*domain.Topup, []ports.Utxo, error) {
	price, err := s.repoManager.PriceRepository().GetPrice(ctx, assetHash)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil, nil, fmt.Errorf("asset %s not supported", assetHash)
		}
		return nil, nil, err
	}

	partial, selectedUtxos, expireDate, feeAmount, payoutAmount, err := s.wallet.
		CreatePartialWithFeesAndPayout(
			ctx, price, estimatedTxSize, millisatPerByte,
		)
	if err != nil {
		return nil, nil, err
	}

	spreadAmount := getSpreadAmount(price, feeAmount)

	topup := domain.NewTopupWithAsset(
		assetHash, feeAmount, price.Amount,
		time.Unix(int64(expireDate), 0),
	)
	topup.Complete(partial, payoutAmount, spreadAmount)

	go func() {
		s.repoManager.TopupRepository().AddTopup(ctx, topup)
	}()

	return topup, selectedUtxos, nil
}

func (s *taxiService) TopupWithKnownPayout(
	ctx context.Context, partial string, millisatPerByte uint64,
	userUnblindedInput []UnblindedInput,
) (*domain.Topup, error) {
	if len(partial) == 0 {
		return nil, fmt.Errorf("missing partial transaction")
	}
	if millisatPerByte < minMillisatPerByte {
		millisatPerByte = defaultMillisatPerByte
	}

	for _, v := range userUnblindedInput {
		if err := v.validate(); err != nil {
			return nil, err
		}
	}

	scripts, err := s.repoManager.WhitelistRepository().GetScripts(ctx)
	if err != nil {
		return nil, err
	}

	containsKnownPayout, err := s.wallet.PartialContainsKnownPayout(
		partial, scripts,
	)
	if err != nil {
		return nil, err
	}
	if !containsKnownPayout {
		return nil, fmt.Errorf("known payout not found in partial transaction")
	}

	updatedPartial, expirationDate, err := s.wallet.UpdatePartialWithFees(
		ctx, partial, millisatPerByte, userUnblindedInput,
	)
	if err != nil {
		return nil, err
	}

	topup := domain.NewTopupWithPartial(
		updatedPartial, time.Unix(int64(expirationDate), 0),
	)
	topup.CompletedAt = time.Now()
	topup.Status = domain.Completed

	go func() {
		s.repoManager.TopupRepository().AddTopup(ctx, topup)
	}()

	return topup, nil
}

// ListAssets return all the available assets of the taxi service
func (s *taxiService) ListAssets(ctx context.Context) ([]*domain.Price, error) {
	return s.repoManager.PriceRepository().GetAllPrices(ctx)
}

func getSpreadAmount(price *domain.Price, feeAmount uint64) uint64 {
	feeAmountAsDecimal := decimal.NewFromInt(int64(feeAmount))
	assetAmountAsDecimal := price.Amount.Mul(feeAmountAsDecimal)

	spreadRate := decimal.NewFromFloat(float64(price.BasisPoint) * 0.01 / 100)
	serviceFeeAmountAsDecimal := assetAmountAsDecimal.Mul(spreadRate)

	spreadAmount := serviceFeeAmountAsDecimal.BigInt().Uint64()
	return spreadAmount
}

func (s *taxiService) registerHandlerForSpentUtxos() {
	s.wallet.OnUtxosSpent(func(utxos []ports.Utxo) {
		ctx := context.Background()
		topups, _ := s.repoManager.TopupRepository().GetAllTopups(ctx)
		for _, topup := range topups {
			if topup.Status == domain.Settled || topup.Status == domain.Expired {
				continue
			}

			ok, _ := s.wallet.PartialContainsKnownUtxos(topup.Partial, utxos)
			if ok {
				if err := s.repoManager.TopupRepository().UpdateTopup(
					ctx, topup.ID, func(t *domain.Topup) (*domain.Topup, error) {
						if err := t.Settle(); err != nil {
							return nil, err
						}
						return t, nil
					},
				); err != nil {
					log.WithError(err).Warnf(
						"failed to update status of topup %s to settled", topup.ID,
					)
				}
				log.Debugf("topup %s updated to status settled", topup.ID)
				return
			}
		}
	})
}

func (s *taxiService) registerHandlerForExpiredUtxos() {
	s.wallet.OnUtxosUnlocked(func(utxos []ports.Utxo) {
		ctx := context.Background()
		topups, _ := s.repoManager.TopupRepository().GetAllTopups(ctx)
		for _, topup := range topups {
			if topup.Status == domain.Settled || topup.Status == domain.Expired {
				continue
			}

			ok, _ := s.wallet.PartialContainsKnownUtxos(topup.Partial, utxos)
			if ok {
				if err := s.repoManager.TopupRepository().UpdateTopup(
					ctx, topup.ID, func(t *domain.Topup) (*domain.Topup, error) {
						if err := t.Expire(); err != nil {
							return nil, err
						}
						return t, nil
					},
				); err != nil {
					log.WithError(err).Warnf(
						"failed to update status of topup %s to settled", topup.ID,
					)
				}
				log.Debugf("topup %s updated to status expired", topup.ID)
				return
			}
		}
	})
}
