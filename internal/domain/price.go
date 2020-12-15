package domain

import "github.com/shopspring/decimal"

// Price maps an assethash to an amount = the value for 1 BTC satoshi
type Price struct {
	AssetHash string
	Amount    decimal.Decimal
}
