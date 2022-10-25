package application

import (
	"encoding/hex"
	"errors"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type UnblindedInput struct {
	Index         uint32
	Asset         string
	Amount        uint64
	AssetBlinder  string
	AmountBlinder string
}

func (u *UnblindedInput) validate() error {
	return validation.ValidateStruct(
		u,
		validation.Field(&u.Asset, validation.By(validateAssetString)),
		validation.Field(&u.Amount, validation.Required),
		validation.Field(&u.AssetBlinder, validation.By(validateAssetString)),
		validation.Field(&u.AmountBlinder, validation.By(validateAssetString)),
	)
}

func validateAssetString(asset interface{}) error {
	assetString, _ := asset.(string)

	buf, err := hex.DecodeString(assetString)
	if err != nil {
		return errors.New("asset is not in hex format")
	}

	if len(buf) != 32 {
		return errors.New("asset length is invalid")
	}

	return nil
}
