package application_test

import (
	"context"
	"github.com/vulpemventures/taxi-daemon/internal/core/application"

	"github.com/stretchr/testify/mock"
	"github.com/vulpemventures/taxi-daemon/internal/core/domain"
	"github.com/vulpemventures/taxi-daemon/internal/core/ports"
)

type mockWallet struct {
	mock.Mock
}

func (m *mockWallet) UpdatePartialWithFees(
	ctx context.Context, partial string, millisatPerByte uint64,
	userUnblindedInput []application.UnblindedInput,
) (string, uint64, error) {
	args := m.Called(ctx, partial, millisatPerByte, userUnblindedInput)
	var res string
	if a := args.Get(0); a != nil {
		res = a.(string)
	}
	var res1 uint64
	if a := args.Get(1); a != nil {
		res1 = a.(uint64)
	}
	return res, res1, args.Error(2)
}

func (m *mockWallet) CreatePartialWithFeesAndPayout(
	ctx context.Context, price *domain.Price,
	estimatedSize, millisatPerByte uint64,
) (string, []ports.Utxo, uint64, uint64, uint64, error) {
	args := m.Called(ctx, price, estimatedSize, millisatPerByte)
	var res string
	if a := args.Get(0); a != nil {
		res = a.(string)
	}
	var res1 []ports.Utxo
	if a := args.Get(1); a != nil {
		res1 = a.([]ports.Utxo)
	}
	var res2 uint64
	if a := args.Get(2); a != nil {
		res2 = a.(uint64)
	}
	var res3 uint64
	if a := args.Get(3); a != nil {
		res3 = a.(uint64)
	}
	var res4 uint64
	if a := args.Get(4); a != nil {
		res4 = a.(uint64)
	}
	return res, res1, res2, res3, res4, args.Error(5)
}

func (m *mockWallet) PartialContainsKnownPayout(
	partial string, scripts []string,
) (bool, error) {
	args := m.Called(partial, scripts)
	var res bool
	if a := args.Get(0); a != nil {
		res = a.(bool)
	}
	return res, args.Error(1)
}

func (m *mockWallet) PartialContainsKnownUtxos(
	partial string, utxos []ports.Utxo,
) (bool, error) {
	args := m.Called(partial, utxos)
	var res bool
	if a := args.Get(0); a != nil {
		res = a.(bool)
	}
	return res, args.Error(1)
}

func (m *mockWallet) OnUtxosUnlocked(func([]ports.Utxo)) {}

func (m *mockWallet) OnUtxosSpent(func([]ports.Utxo)) {}
