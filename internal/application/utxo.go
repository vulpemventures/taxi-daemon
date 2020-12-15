package application

import (
	"encoding/hex"

	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// Utxo = unspent
type Utxo struct {
	TxId            string `json:"txid"`
	Vout            uint32 `json:"vout"`
	Value           uint64 `json:"value"`
	Asset           string `json:"asset"`
	ValueCommitment string `json:"valuecommitment"`
	AssetCommitment string `json:"assetcommitment"`
	NonceCommitment string `json:"noncecommitment"`
}

func (u Utxo) Key() domain.Lock {
	return domain.Lock{
		TxId: u.TxId,
		Vout: u.Vout,
	}
}

func (u Utxo) Hash() []byte {
	buf, _ := hex.DecodeString(u.TxId)
	return elementsutil.ReverseBytes(buf)
}

func (u Utxo) IsConfidential() bool {
	return u.ValueCommitment != "" &&
		u.AssetCommitment != "" &&
		u.NonceCommitment != "" &&
		u.Asset == "" &&
		u.Value == 0
}
