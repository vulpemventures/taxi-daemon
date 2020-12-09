package explorer

import (
	"encoding/hex"

	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/transaction"
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

func (u Utxo) FetchPrevout(explorer ExplorerService) (*transaction.TxOutput, error) {
	hex, err := explorer.FetchTx(u.TxId)
	if err != nil {
		return nil, err
	}

	prevTx, err := transaction.NewTxFromHex(hex)
	if err != nil {
		return nil, err
	}

	return prevTx.Outputs[u.Vout], nil
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
