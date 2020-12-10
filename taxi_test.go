package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/internal/explorer"
)

const defaultExplorer = "http://localhost:3001"

func TestEndToEnd(t *testing.T) {
	explorerURL, ok := os.LookupEnv("API_URL")
	if !ok {
		explorerURL = defaultExplorer
	}
	//Explorer
	e := explorer.NewEsplorerService(explorerURL)

	// Treasury
	treasuryKeyPair, _ := btcec.NewPrivateKey(btcec.S256())
	treasury := payment.FromPublicKey(treasuryKeyPair.PubKey(), &network.Regtest, nil)
	treasuryAddress, _ := treasury.WitnessPubKeyHash()
	// Alice
	aliceKeyPair, _ := btcec.NewPrivateKey(btcec.S256())
	alice := payment.FromPublicKey(aliceKeyPair.PubKey(), &network.Regtest, nil)
	aliceAddress, _ := alice.WitnessPubKeyHash()
	//Bob
	bobKeyPair, _ := btcec.NewPrivateKey(btcec.S256())
	bob := payment.FromPublicKey(bobKeyPair.PubKey(), &network.Regtest, nil)
	bobAddress, _ := bob.WitnessPubKeyHash()
	println("alice: "+aliceAddress, "bob: "+bobAddress, "taxi: "+treasuryAddress)

	_, err := faucet(aliceAddress, explorerURL)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Second)
	_, err = faucet(treasuryAddress, explorerURL)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Second)

	LBTC := assetHash(network.Regtest.AssetID)

	// Retrieve Alice utxos
	aliceAllUtxos, err := e.GetUnspents(aliceAddress)
	if err != nil {
		t.Fatal(err)
	}

	// filter --> use only unconfidential utxos
	aliceUtxos := make([]explorer.Utxo, 0)
	for _, utxo := range aliceAllUtxos {
		if !utxo.IsConfidential() {
			aliceUtxos = append(aliceUtxos, utxo)
		}
	}

	aliceUtxoValue := aliceUtxos[0].Value
	bobValue := uint64(5000000)

	aliceInput := transaction.NewTxInput(aliceUtxos[0].Hash(), aliceUtxos[0].Vout)
	bobElementsValue, err := elementsutil.SatoshiToElementsValue(bobValue)
	if err != nil {
		t.Error(err)
	}
	aliceOutput := transaction.NewTxOutput(LBTC, bobElementsValue, bob.WitnessScript)
	changeValue, err := elementsutil.SatoshiToElementsValue(aliceUtxoValue - bobValue)
	if err != nil {
		t.Error(err)
	}
	aliceChangeOutput := transaction.NewTxOutput(LBTC, changeValue, alice.WitnessScript)

	inputs := []*transaction.TxInput{aliceInput}
	outputs := []*transaction.TxOutput{aliceOutput, aliceChangeOutput}
	p, err := pset.New(inputs, outputs, 2, 0)
	if err != nil {
		t.Error(err)
	}

	updater, err := pset.NewUpdater(p)
	if err != nil {
		t.Error(err)
	}

	aliceWitnessUtxo, err := aliceUtxos[0].FetchPrevout(e)
	if err != nil {
		t.Error(err)
	}

	if err := updater.AddInWitnessUtxo(aliceWitnessUtxo, 0); err != nil {
		t.Error(err)
	}

	b64, err := p.ToBase64()
	if err != nil {
		t.Fatalf("base64: %s", err)
	}
	println(b64)

	// Retrieve Alice utxos
	taxiAllUtxos, err := e.GetUnspents(treasuryAddress)
	if err != nil {
		t.Fatal(err)
	}

	// filter --> use only unconfidential utxos
	taxiUtxos := make([]explorer.Utxo, 0)
	for _, utxo := range taxiAllUtxos {
		if !utxo.IsConfidential() {
			taxiUtxos = append(taxiUtxos, utxo)
		}
	}

	taxiInput := transaction.NewTxInput(taxiUtxos[0].Hash(), taxiUtxos[0].Vout)

	taxiUtxoValue := taxiUtxos[0].Value
	feeValue := uint64(500)
	taxiOutputValue, err := elementsutil.SatoshiToElementsValue(taxiUtxoValue - feeValue)
	if err != nil {
		t.Error(err)
	}

	feeValueElements, err := elementsutil.SatoshiToElementsValue(feeValue)
	if err != nil {
		t.Error(err)
	}

	taxiOutput := transaction.NewTxOutput(LBTC, taxiOutputValue, treasury.WitnessScript)
	feeOutput := transaction.NewTxOutput(LBTC, feeValueElements, []byte{})

	pTaxi, err := pset.NewPsetFromBase64(b64)
	if err != nil {
		t.Error(err)
	}

	updaterTaxi, err := pset.NewUpdater(pTaxi)
	if err != nil {
		t.Error(err)
	}

	updaterTaxi.AddInput(taxiInput)

	taxiWitnessUtxo, err := taxiUtxos[0].FetchPrevout(e)
	if err != nil {
		t.Error(err)
	}

	if err := updaterTaxi.AddInWitnessUtxo(taxiWitnessUtxo, 1); err != nil {
		t.Error(err)
	}

	updaterTaxi.AddOutput(taxiOutput)
	updaterTaxi.AddOutput(feeOutput)

	sigHashes := make([][32]byte, len(pTaxi.Inputs))
	scripts := [][]byte{
		alice.Script,
		treasury.Script,
	}
	for i, in := range pTaxi.Inputs {
		if err := updaterTaxi.AddInSighashType(txscript.SigHashAll, i); err != nil {
			t.Error(err)
		}

		sigHash := pTaxi.UnsignedTx.HashForWitnessV0(
			i,
			scripts[i],
			in.WitnessUtxo.Value,
			txscript.SigHashAll,
		)

		sigHashes[i] = sigHash
	}

	sig, err := aliceKeyPair.Sign(sigHashes[0][:])
	if err != nil {
		t.Error(err)
	}

	sigWithHashType := append(sig.Serialize(), byte(txscript.SigHashAll))
	if _, err := updaterTaxi.Sign(
		0,
		sigWithHashType,
		aliceKeyPair.PubKey().SerializeCompressed(),
		nil,
		nil,
	); err != nil {
		t.Error(err)
	}

	sig, err = treasuryKeyPair.Sign(sigHashes[1][:])
	if err != nil {
		t.Error(err)
	}
	sigWithHashType = append(sig.Serialize(), byte(txscript.SigHashAll))
	if _, err := updaterTaxi.Sign(
		1,
		sigWithHashType,
		treasuryKeyPair.PubKey().SerializeCompressed(),
		nil,
		nil,
	); err != nil {
		t.Error(err)
	}

	valid, err := pTaxi.ValidateAllSignatures()
	if err != nil {
		t.Error(err)
	}

	if !valid {
		t.Error(errors.New("Signature is invalid"))
	}

	err = pset.FinalizeAll(pTaxi)
	if err != nil {
		t.Error(err)
	}

	// Extract the final signed transaction from the Pset wrapper.
	finalTx, err := pset.Extract(pTaxi)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize the transaction and try to broadcast.
	txHex, err := finalTx.ToHex()
	if err != nil {
		t.Fatal(err)
	}
	res, err := e.Broadcast(txHex)
	if err != nil {
		t.Fatal(err)
	}

	println(res)
}

// Faucet ...
func faucet(address string, baseURL string) (string, error) {
	url := baseURL + "/faucet"
	payload := map[string]string{"address": address}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "appliation/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respBody := map[string]string{}
	err = json.Unmarshal(data, &respBody)
	if err != nil {
		return "", err
	}

	return respBody["txId"], nil
}

func assetHash(asset string) []byte {
	buf, _ := hex.DecodeString(asset)
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf
}
