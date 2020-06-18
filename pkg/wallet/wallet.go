package wallet

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/liquid-taxi/pkg/partial"
)

// FromPrivateKey takes a hex encoded private key  and return a WitnessPubKeyHash payment struct and the encoded bech32 address and the EC private key
func FromPrivateKey(hexPriv string) (*payment.Payment, string, *btcec.PrivateKey) {
	privateKeyBytes, _ := hex.DecodeString(hexPriv)
	privateKey, publicKey := btcec.PrivKeyFromBytes(btcec.S256(), privateKeyBytes)
	wpk := payment.FromPublicKey(publicKey, &network.Regtest, nil)
	address, _ := wpk.WitnessPubKeyHash()
	return wpk, address, privateKey
}

//AssetHashToBytes reverse decode from hex string and reverse it adding a 0x01 byte for ublinded asset
func AssetHashToBytes(hash string, blinded bool) []byte {
	firstByte := byte(0x01)
	if blinded {
		firstByte = byte(0x00)
	}
	assetBytes, _ := hex.DecodeString(hash)
	assetBytes = append([]byte{firstByte}, partial.ReverseBytes(assetBytes)...)
	return assetBytes
}

// Utxo defines the utxo from the explorer
type Utxo struct {
	Hash  string
	Index uint32
	Value uint64
	Asset string
}

// CoinSelect returns the utxos that satisfies the target amount.
func CoinSelect(utxos []map[string]interface{}, amount uint64, asset string) (unspents []*Utxo, change uint64, err error) {
	change = 0
	unspents = []*Utxo{}
	availableSats := uint64(0)

	for _, unspent := range utxos {
		if asset == unspent["asset"].(string) {

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
	}

	if availableSats < amount {
		return nil, 0, errors.New("You do not have enough coins")
	}

	change = availableSats - amount

	return unspents, change, nil
}
