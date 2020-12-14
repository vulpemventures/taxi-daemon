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

var (
	ErrUtxoNotAvailable = errors.New("a selected utxo is not available")
	ErrInvalidSignature = errors.New("at least one signature is invalid")
)
