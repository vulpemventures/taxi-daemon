package application

import (
	"errors"
	"fmt"
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

var (
	ErrUtxoNotAvailable = errors.New("a selected utxo is not available")
	ErrInvalidSignature = errors.New("at least one signature is invalid")
)
