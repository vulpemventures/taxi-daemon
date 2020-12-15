package application

import (
	"errors"
	"fmt"

	"github.com/shopspring/decimal"
)

type errStatusIsNotOk struct {
	httpCode int
}

var _ error = (*errStatusIsNotOk)(nil)

func (e errStatusIsNotOk) Error() string {
	return "http status code != OK: " + fmt.Sprint(e.httpCode)
}

type ErrAssetNotSupported struct {
	AssetHash string
}

var _ error = (*ErrAssetNotSupported)(nil)

func (e ErrAssetNotSupported) Error() string {
	return "Asset not supported by taxi: " + e.AssetHash
}

type ErrSatsPerByteIsInvalid struct {
	SatsPerByte decimal.Decimal
}

var _ error = (*ErrSatsPerByteIsInvalid)(nil)

func (e ErrSatsPerByteIsInvalid) Error() string {
	return "Sats per byte value " + e.SatsPerByte.String() + " is lower than 0.1"
}

type ErrUtxoNotFound struct {
	TargetedAmount uint64
	TargetedAsset  string
}

var _ error = (*ErrUtxoNotFound)(nil)

func (e ErrUtxoNotFound) Error() string {
	return "utxo with amount = " + fmt.Sprint(e.TargetedAmount) + " and asset = " + e.TargetedAsset + " not found."
}

var (
	ErrUtxoNotAvailable = errors.New("a selected utxo is not available")
	ErrInvalidSignature = errors.New("at least one signature is invalid")
)
