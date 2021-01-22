package domain

import (
	"github.com/shopspring/decimal"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
)

// Price is the data structure that represents the amount of a certain asset that equals the value of 1 satoshis BTC
type Price struct {
	AssetHash  string
	Amount     decimal.Decimal
	BasisPoint uint32
}

func (price *Price) ToAssetDetailsMessage() *pb.AssetDetails {
	p, _ := price.Amount.Float64()

	return &pb.AssetDetails{
		AssetHash:  price.AssetHash,
		BasisPoint: price.BasisPoint,
		AssetPrice: float32(p),
	}
}
