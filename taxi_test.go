package main

import (
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/vulpemventures/go-elements/confidential"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/pkg/explorer"
)

const treasuryHex = "1cc080a4cd371eafcad489a29664af6a7276b362fe783443ce036552482b971d"
const aliceHex = "bfb96a215dfb07d1a193464174b9ea8e91f2a15bba79800dea838add330f6d86"
const bobHex = "1804e76aa3016013bc9969103554668913cf697c03c23aecb28136d0e0ac16f0"

func TestEndToEnd(t *testing.T) {
	// BTC
	assetBytes, _ := hex.DecodeString(network.Regtest.AssetID)
	assetBytes = append([]byte{0x01}, reverseBytes(assetBytes)...)
	// Treasury
	treasury, treasuryAddress, treasuryPrivKey := fromPrivateKey(treasuryHex)
	// Alice
	alice, aliceAddress, alicePrivKey := fromPrivateKey(aliceHex)
	//Bob
	bob, bobAddress, _ := fromPrivateKey(bobHex)
	println(aliceAddress, bobAddress)

	_, err := explorer.Faucet(aliceAddress)
	if err != nil {
		t.Fatal(err)
	}
	_, err = explorer.Faucet(treasuryAddress)
	if err != nil {
		t.Fatal(err)
	}
	// Retrieve Alice utxos
	aliceUtxos, err := explorer.Unspents(aliceAddress)
	if err != nil {
		panic(err.Error())
	}
	aliceUtxoValue := uint64(aliceUtxos[0]["value"].(float64))
	bobValue := uint64(5000000)

	emptyPset, _ := pset.New([]*transaction.TxInput{}, []*transaction.TxOutput{}, 2, 0)
	p := &Partial{emptyPset}

	p.addInput(aliceUtxos[0]["txid"].(string), aliceUtxos[0]["vout"].(float64), &WitnessUtxo{assetBytes, aliceUtxoValue, alice.Script}, nil)
	p.addOutput(assetBytes, bobValue, bob.Script)
	//Alice's change
	p.addOutput(assetBytes, aliceUtxoValue-bobValue, alice.Script)

	size := p.data.UnsignedTx.SerializeSize(true, true)
	println(size)

	// Retrieve Taxi utxos
	taxiUtxos, err := explorer.Unspents(treasuryAddress)
	if err != nil {
		panic(err.Error())
	}

	feeValue := uint64(500)
	taxiUtxoValue := uint64(taxiUtxos[0]["value"].(float64))
	// Add input for fee
	p.addInput(taxiUtxos[0]["txid"].(string), taxiUtxos[0]["vout"].(float64), &WitnessUtxo{assetBytes, taxiUtxoValue, treasury.Script}, nil)
	// Taxi's change
	p.addOutput(assetBytes, taxiUtxoValue-feeValue, treasury.Script)
	// Taxi adds Fee output
	p.addOutput(assetBytes, feeValue, []byte{})

	// Taxi signs last added input with his own private key
	err = p.signWithPrivateKey(1, treasury, treasuryPrivKey)
	if err != nil {
		panic(err.Error())
	}

	// back to Alice now signs with his privatekey
	err = p.signWithPrivateKey(0, alice, alicePrivKey)
	if err != nil {
		panic(err.Error())
	}

	println(len(p.data.Inputs))
	println(len(p.data.Outputs))

	pFinalized := p.data
	err = pset.FinalizeAll(pFinalized)
	if err != nil {
		panic(err.Error())
	}

	b64, _ := pFinalized.ToBase64()
	println(b64)
	println()

	// Extract the final signed transaction from the Pset wrapper.
	finalTx, err := pset.Extract(pFinalized)
	if err != nil {
		panic(err.Error())
	}

	size = finalTx.SerializeSize(true, true)
	println(size)

	// Serialize the transaction and try to broadcast.
	txHex, err := finalTx.ToHex()
	if err != nil {
		panic(err.Error())
	}
	res, err := explorer.Broadcast(txHex)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) <= 0 || strings.Contains(res, "sendrawtransaction") {
		t.Fatalf("Expected transaction to be broadcasted, failed for reason: %s", res)
	}

	println(res)
}

// Partial defines a Partial Signed Elements Transaction
type Partial struct {
	data *pset.Pset
}

// WitnessUtxo defines a witness utxo
type WitnessUtxo struct {
	asset  []byte
	value  uint64
	script []byte
}

func (p *Partial) addInput(hash string, index float64, witnessUtxo *WitnessUtxo, nonWitnessUtxo []byte) error {
	updater, err := pset.NewUpdater(p.data)
	if err != nil {
		return err
	}

	inputHash, _ := hex.DecodeString(hash)
	inputHash = reverseBytes(inputHash)
	inputIndex := uint32(index)
	input := transaction.NewTxInput(inputHash, inputIndex)

	updater.AddInput(input)
	lastAdded := len(updater.Upsbt.Inputs) - 1

	err = updater.AddInSighashType(txscript.SigHashAll, lastAdded)
	if err != nil {
		return err
	}

	if witnessUtxo != nil {
		elementsValue, err := confidential.SatoshiToElementsValue(witnessUtxo.value)
		if err != nil {
			return err
		}
		witnessUtxo := transaction.NewTxOutput(witnessUtxo.asset, elementsValue[:], witnessUtxo.script)
		updater.AddInWitnessUtxo(witnessUtxo, lastAdded)
		p.data = updater.Upsbt
		return nil
	}

	return errors.New("Either witnessUtxo or nonWitnessUtxo is missing")
}

func (p *Partial) addOutput(asset []byte, value uint64, script []byte) error {
	updater, err := pset.NewUpdater(p.data)
	if err != nil {
		return err
	}

	elementsValue, _ := confidential.SatoshiToElementsValue(value)
	output := transaction.NewTxOutput(asset, elementsValue[:], script)

	updater.AddOutput(output)
	p.data = updater.Upsbt
	return nil
}

func (p *Partial) signWithPrivateKey(index int, pay *payment.Payment, privKey *btcec.PrivateKey) error {
	updater, err := pset.NewUpdater(p.data)
	if err != nil {
		return err
	}
	// legacy Script
	legacyScript := append(append([]byte{0x76, 0xa9, 0x14}, pay.Hash...), []byte{0x88, 0xac}...)
	witValue := updater.Upsbt.Inputs[index].WitnessUtxo.Value
	witHash := updater.Upsbt.UnsignedTx.HashForWitnessV0(index, legacyScript, witValue[:], txscript.SigHashAll)
	sig, err := privKey.Sign(witHash[:])
	if err != nil {
		return err
	}

	sigWithHashType := append(sig.Serialize(), byte(txscript.SigHashAll))
	if err != nil {
		return err
	}

	// Update the pset adding the input signature script and the pubkey.
	_, err = updater.Sign(index, sigWithHashType, pay.PublicKey.SerializeCompressed(), nil, nil)
	if err != nil {
		return err
	}

	p.data = updater.Upsbt
	return nil
}

// fromPrivateKey
func fromPrivateKey(hexPriv string) (*payment.Payment, string, *btcec.PrivateKey) {
	privateKeyBytes, _ := hex.DecodeString(hexPriv)
	privateKey, publicKey := btcec.PrivKeyFromBytes(btcec.S256(), privateKeyBytes)
	wpk := payment.FromPublicKey(publicKey, &network.Regtest, nil)
	address, _ := wpk.WitnessPubKeyHash()
	return wpk, address, privateKey
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
