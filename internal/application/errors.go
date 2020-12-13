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
	// ErrMalformedDerivationPath ...
	ErrMalformedDerivationPath = errors.New(
		"path must not start or end with a '/' and " +
			"can optionally start with 'm/' for absolute paths",
	)
	// ErrNullDerivationPath ...
	ErrNullDerivationPath    = errors.New("derivation path must not be null")
	ErrInvalidDerivationPath = errors.New("invalid derivation path")
)
