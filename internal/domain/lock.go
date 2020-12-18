package domain

// Lock represents the unique identifier of an Utxo
type Lock struct {
	// TxID is the transaction hash of the utxo
	TxID string
	// Vout is the output index of the prevout utxo transaction
	Vout uint32
}
