package application

import (
	"encoding/hex"
	"errors"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type UnblindedInput struct {
	Index         int
	Asset         string
	Amount        int
	AssetBlinder  string
	AmountBlinder string
}

func (u *UnblindedInput) validate() error {
	return validation.ValidateStruct(
		u,
		validation.Field(&u.Asset, validation.By(validateAssetString)),
		validation.Field(&u.Amount, validation.Required),
		validation.Field(&u.AssetBlinder, validation.By(validateHex)),
		validation.Field(&u.AmountBlinder, validation.By(validateHex)),
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

func validateHex(value interface{}) error {
	valueString, _ := value.(string)

	_, err := hex.DecodeString(valueString)
	if err != nil {
		return errors.New("value is not in hex format")
	}

	return nil
}
