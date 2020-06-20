package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/tiero/ocean/pkg/explorer/blockstream"
	"github.com/tiero/ocean/pkg/keypair"
	"github.com/tiero/ocean/pkg/partial"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
)

const treasuryHex = "1cc080a4cd371eafcad489a29664af6a7276b362fe783443ce036552482b971d"
const aliceHex = "bfb96a215dfb07d1a193464174b9ea8e91f2a15bba79800dea838add330f6d86"
const bobHex = "1804e76aa3016013bc9969103554668913cf697c03c23aecb28136d0e0ac16f0"
const defaultExplorer = "http://localhost:3001"

func TestEndToEnd(t *testing.T) {
	explorerURL, ok := os.LookupEnv("API_URL")
	if !ok {
		explorerURL = defaultExplorer
	}
	//Explorer
	e := blockstream.NewExplorer(explorerURL)

	// Treasury
	treasuryKeyPair, _ := keypair.FromPrivateKey(treasuryHex)
	treasury := payment.FromPublicKey(treasuryKeyPair.PublicKey, &network.Regtest, nil)
	treasuryAddress, _ := treasury.WitnessPubKeyHash()
	// Alice
	aliceKeyPair, _ := keypair.FromPrivateKey(aliceHex)
	alice := payment.FromPublicKey(aliceKeyPair.PublicKey, &network.Regtest, nil)
	aliceAddress, _ := alice.WitnessPubKeyHash()
	//Bob
	bobKeyPair, _ := keypair.FromPrivateKey(bobHex)
	bob := payment.FromPublicKey(bobKeyPair.PublicKey, &network.Regtest, nil)
	bobAddress, _ := bob.WitnessPubKeyHash()
	println("alice: "+aliceAddress, "bob: "+bobAddress, "taxi: "+treasuryAddress)

	_, err := faucet(aliceAddress, explorerURL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = faucet(treasuryAddress, explorerURL)
	if err != nil {
		t.Fatal(err)
	}
	// Retrieve Alice utxos
	aliceUtxos, err := e.GetUnspents(aliceAddress)
	if err != nil {
		panic(err.Error())
	}
	aliceUtxoValue := aliceUtxos[0].Value()
	bobValue := uint64(5000000)

	emptyPset, _ := pset.New([]*transaction.TxInput{}, []*transaction.TxOutput{}, 2, 0)
	p := &partial.Partial{Data: emptyPset}

	p.AddInput(aliceUtxos[0].Hash(), aliceUtxos[0].Index(), &partial.WitnessUtxo{network.Regtest.AssetID, aliceUtxoValue, alice.Script}, nil)
	p.AddOutput(network.Regtest.AssetID, bobValue, bob.Script, false)
	//Alice's change
	p.AddOutput(network.Regtest.AssetID, aliceUtxoValue-bobValue, alice.Script, false)

	// Retrieve Taxi utxos
	taxiUtxos, err := e.GetUnspents(treasuryAddress)
	if err != nil {
		panic(err.Error())
	}

	feeValue := uint64(500)
	taxiUtxoValue := uint64(taxiUtxos[0].Value())
	// Add input for fee
	p.AddInput(taxiUtxos[0].Hash(), uint32(taxiUtxos[0].Index()), &partial.WitnessUtxo{network.Regtest.AssetID, taxiUtxoValue, treasury.Script}, nil)
	// Taxi's change
	p.AddOutput(network.Regtest.AssetID, taxiUtxoValue-feeValue, treasury.Script, false)
	// Taxi adds Fee output
	p.AddOutput(network.Regtest.AssetID, feeValue, []byte{}, false)

	// Taxi signs last added input with his own private key
	err = p.SignWithPrivateKey(1, treasuryKeyPair)
	if err != nil {
		panic(err.Error())
	}

	// back to Alice now signs with his privatekey
	err = p.SignWithPrivateKey(0, aliceKeyPair)
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
