package main

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/pkg/partial"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

const treasuryHex = "1cc080a4cd371eafcad489a29664af6a7276b362fe783443ce036552482b971d"
const aliceHex = "bfb96a215dfb07d1a193464174b9ea8e91f2a15bba79800dea838add330f6d86"
const bobHex = "1804e76aa3016013bc9969103554668913cf697c03c23aecb28136d0e0ac16f0"

func TestEndToEnd(t *testing.T) {
	//Explorer
	e := &explorer.Explorer{BaseURL: "https://nigiri.network/liquid/api"}
	// BTC
	assetBytes, _ := hex.DecodeString(network.Regtest.AssetID)
	assetBytes = append([]byte{0x01}, reverseBytes(assetBytes)...)

	// Treasury
	treasury, treasuryAddress, treasuryPrivKey := wallet.FromPrivateKey(treasuryHex)
	// Alice
	alice, aliceAddress, alicePrivKey := wallet.FromPrivateKey(aliceHex)
	//Bob
	bob, bobAddress, _ := wallet.FromPrivateKey(bobHex)
	println(aliceAddress, bobAddress, treasuryAddress)

	_, err := e.Faucet(aliceAddress)
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.Faucet(treasuryAddress)
	if err != nil {
		t.Fatal(err)
	}
	// Retrieve Alice utxos
	aliceUtxos, err := e.Unspents(aliceAddress)
	if err != nil {
		panic(err.Error())
	}
	aliceUtxoValue := uint64(aliceUtxos[0]["value"].(float64))
	bobValue := uint64(5000000)

	emptyPset, _ := pset.New([]*transaction.TxInput{}, []*transaction.TxOutput{}, 2, 0)
	p := &partial.Partial{Data: emptyPset}

	p.AddInput(aliceUtxos[0]["txid"].(string), uint32(aliceUtxos[0]["vout"].(float64)), &partial.WitnessUtxo{assetBytes, aliceUtxoValue, alice.Script}, nil)
	p.AddOutput(assetBytes, bobValue, bob.Script)
	//Alice's change
	p.AddOutput(assetBytes, aliceUtxoValue-bobValue, alice.Script)

	b64, _ := p.Data.ToBase64()
	println(b64)
	println()

	// Retrieve Taxi utxos
	taxiUtxos, err := e.Unspents(treasuryAddress)
	if err != nil {
		panic(err.Error())
	}

	feeValue := uint64(500)
	taxiUtxoValue := uint64(taxiUtxos[0]["value"].(float64))
	// Add input for fee
	p.AddInput(taxiUtxos[0]["txid"].(string), uint32(taxiUtxos[0]["vout"].(float64)), &partial.WitnessUtxo{assetBytes, taxiUtxoValue, treasury.Script}, nil)
	// Taxi's change
	p.AddOutput(assetBytes, taxiUtxoValue-feeValue, treasury.Script)
	// Taxi adds Fee output
	p.AddOutput(assetBytes, feeValue, []byte{})

	// Taxi signs last added input with his own private key
	err = p.SignWithPrivateKey(1, treasury, treasuryPrivKey)
	if err != nil {
		panic(err.Error())
	}

	// back to Alice now signs with his privatekey
	err = p.SignWithPrivateKey(0, alice, alicePrivKey)
	if err != nil {
		panic(err.Error())
	}
	pFinalized := p.Data
	err = pset.FinalizeAll(pFinalized)
	if err != nil {
		panic(err.Error())
	}

	// Extract the final signed transaction from the Pset wrapper.
	finalTx, err := pset.Extract(pFinalized)
	if err != nil {
		panic(err.Error())
	}

	// Serialize the transaction and try to broadcast.
	txHex, err := finalTx.ToHex()
	if err != nil {
		panic(err.Error())
	}
	res, err := e.Broadcast(txHex)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) <= 0 || strings.Contains(res, "sendrawtransaction") {
		t.Fatalf("Expected transaction to be broadcasted, failed for reason: %s", res)
	}

	println(res)
}

// reverseBytes returns the given byte slice with elems in reverse order.
func reverseBytes(buf []byte) []byte {
	if len(buf) < 1 {
		return buf
	}
	for i := len(buf)/2 - 1; i >= 0; i-- {
		j := len(buf) - 1 - i
		buf[i], buf[j] = buf[j], buf[i]
	}
	return buf
}
