package application

import (
	"errors"
	"fmt"
)

// ErrAssetNotSupported is returned as error if the taxi service does not handle the given assetHash
type ErrAssetNotSupported struct {
	AssetHash string
}

var _ error = (*ErrAssetNotSupported)(nil)

func (e ErrAssetNotSupported) Error() string {
	return "Asset not supported by taxi: " + e.AssetHash
}

// ErrUtxoNotFound is returned as error if the Utxo with value = targetedAmount and asset = targetedAsset is not found
type ErrUtxoNotFound struct {
	TargetedAmount uint64
	TargetedAsset  string
}

var _ error = (*ErrUtxoNotFound)(nil)

func (e ErrUtxoNotFound) Error() string {
	return "utxo with amount = " + fmt.Sprint(e.TargetedAmount) + " and asset = " + e.TargetedAsset + " not found."
}

// ErrCoinSelection is returned as error if we do not found enough coin to fill a transaction requirement
type ErrCoinSelection struct {
	TargetedAmount uint64
	TargetedAsset  string
}

var _ error = (*ErrCoinSelection)(nil)

func (e ErrCoinSelection) Error() string {
	return "not enough coin in wallet for " + fmt.Sprint(e.TargetedAmount) + " " + e.TargetedAsset
}

var (
	// ErrUtxoNotAvailable is returned if a selected utxo is already used (i.e locked by the taxi service)
	ErrUtxoNotAvailable = errors.New("a selected utxo is not available")
)
