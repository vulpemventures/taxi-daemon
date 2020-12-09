package explorer

// Utxo = unspent
type Utxo struct {
	TxId            string `json:"txid"`
	Vout            int    `json:"vout"`
	Value           int    `json:"value"`
	Asset           string `json:"asset"`
	ValueCommitment string `json:"valuecommitment"`
	AssetCommitment string `json:"assetcommitment"`
	NonceCommitment string `json:"noncecommitment"`
}

func (u Utxo) IsConfidential() bool {
	return u.ValueCommitment != "" &&
		u.AssetCommitment != "" &&
		u.NonceCommitment != "" &&
		u.Asset == "" &&
		u.Value == 0
}
