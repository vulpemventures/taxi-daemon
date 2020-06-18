package wallet

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
)

// FromPrivateKey takes a hex encoded private key  and return a WitnessPubKeyHash payment struct and the encoded bech32 address and the EC private key
func FromPrivateKey(hexPriv string) (*payment.Payment, string, *btcec.PrivateKey) {
	privateKeyBytes, _ := hex.DecodeString(hexPriv)
	privateKey, publicKey := btcec.PrivKeyFromBytes(btcec.S256(), privateKeyBytes)
	wpk := payment.FromPublicKey(publicKey, &network.Regtest, nil)
	address, _ := wpk.WitnessPubKeyHash()
	return wpk, address, privateKey
}

// Utxo defines the utxo from the explorer
type Utxo struct {
	Hash  string
	Index uint32
	Value uint64
	Asset string
}

// CoinSelect returns the utxos that satisfies the target amount.
func CoinSelect(utxos []map[string]interface{}, amount uint64) (unspents []*Utxo, change uint64, err error) {
	change = 0
	unspents = []*Utxo{}
	availableSats := uint64(0)

	for _, unspent := range utxos {
		utxo := &Utxo{
			Hash:  unspent["txid"].(string),
			Index: uint32(unspent["vout"].(float64)),
			Value: uint64(unspent["value"].(float64)),
			Asset: unspent["asset"].(string),
		}
		unspents = append(unspents, utxo)
		availableSats += utxo.Value

		if availableSats >= amount {
			break
		}
	}

	if availableSats < amount {
		return nil, 0, errors.New("You do not have enough coins")
	}

	change = availableSats - amount

	return unspents, change, nil
}
