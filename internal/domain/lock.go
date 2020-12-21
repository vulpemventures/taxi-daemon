package domain

import "github.com/tdex-network/tdex-daemon/pkg/explorer"

// Lock represents the unique identifier of an Utxo
type Lock struct {
	// TxID is the transaction hash of the utxo
	TxID string
	// Vout is the output index of the prevout utxo transaction
	Vout uint32
}

// UtxoToLock maps an explorer.utxo to a Lock
func UtxoToLock(utxo explorer.Utxo) Lock {
	return Lock{
		TxID: utxo.Hash(),
		Vout: utxo.Index(),
	}
}
