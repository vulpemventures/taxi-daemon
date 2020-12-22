package wallet

import "errors"

var (
	// ErrMalformedDerivationPath ...
	ErrMalformedDerivationPath = errors.New(
		"path must not start or end with a '/' and " +
			"can optionally start with 'm/' for absolute paths",
	)
	// ErrNullDerivationPath ...
	ErrNullDerivationPath = errors.New("derivation path must not be null")
	// ErrInvalidDerivationPath is returned if the derivation path is invalid
	ErrInvalidDerivationPath = errors.New("invalid derivation path")
	// ErrInvalidSignature is returned if there is at least one invalid signature in a given PSET
	ErrInvalidSignature = errors.New("at least one signature is invalid")
)
