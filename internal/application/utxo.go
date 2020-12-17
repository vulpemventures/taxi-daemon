package application

import (
	"encoding/hex"

	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// Utxo = unspent
// it is json decodable and encodable
type Utxo struct {
	TxID            string `json:"txid"`
	Vout            uint32 `json:"vout"`
	Value           uint64 `json:"value"`
	Asset           string `json:"asset"`
	ValueCommitment string `json:"valuecommitment"`
	AssetCommitment string `json:"assetcommitment"`
	NonceCommitment string `json:"noncecommitment"`
}

// Key returns the unique identifier of the Utxo
// it returns a Lock domain type
func (u Utxo) Key() domain.Lock {
	return domain.Lock{
		TxID: u.TxID,
		Vout: u.Vout,
	}
}

// Hash returns the TxID as bytes
func (u Utxo) Hash() []byte {
	buf, _ := hex.DecodeString(u.TxID)
	return elementsutil.ReverseBytes(buf)
}

// IsConfidential equals true if the utxo references a confidential output, false otherwise
func (u Utxo) IsConfidential() bool {
	return u.ValueCommitment != "" &&
		u.AssetCommitment != "" &&
		u.NonceCommitment != "" &&
		u.Asset == "" &&
		u.Value == 0
}
