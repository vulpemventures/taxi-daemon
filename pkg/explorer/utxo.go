package explorer

import (
	"encoding/hex"

	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

// Utxo = unspent
// it is json decodable and encodable
type Utxo interface {
	Key() domain.Lock
	Hash() []byte
	IsConfidential() bool
	Value() uint64
	Asset() string
}

type utxo struct {
	Txid            string `json:"txid"`
	Vout            uint32 `json:"vout"`
	UnblindedValue  uint64 `json:"value"`
	UnblindedAsset  string `json:"asset"`
	ValueCommitment string `json:"valuecommitment"`
	AssetCommitment string `json:"assetcommitment"`
	NonceCommitment string `json:"noncecommitment"`
}

var _ Utxo = (*utxo)(nil)

func (u utxo) Value() uint64 {
	return u.UnblindedValue
}

func (u utxo) Asset() string {
	return u.UnblindedAsset
}

// Key returns the unique identifier of the Utxo
// it returns a Lock domain type
func (u utxo) Key() domain.Lock {
	return domain.Lock{
		TxID: u.Txid,
		Vout: u.Vout,
	}
}

// Hash returns the TxID as bytes
func (u utxo) Hash() []byte {
	buf, _ := hex.DecodeString(u.Txid)
	return elementsutil.ReverseBytes(buf)
}

// IsConfidential equals true if the utxo references a confidential output, false otherwise
func (u utxo) IsConfidential() bool {
	return u.ValueCommitment != "" &&
		u.AssetCommitment != "" &&
		u.NonceCommitment != "" &&
		u.UnblindedAsset == "" &&
		u.UnblindedValue == 0
}
