package wallet

import "errors"

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
