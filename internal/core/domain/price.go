package domain

import (
	"github.com/shopspring/decimal"
)

// Price is the data structure that represents the amount of a certain asset that equals the value of 1 satoshis BTC
type Price struct {
	AssetHash  string
	Amount     decimal.Decimal
	BasisPoint uint32
}
