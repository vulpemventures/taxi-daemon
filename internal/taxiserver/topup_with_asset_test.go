package taxiserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/tiero/ocean/pkg/explorer/blockstream"
	"github.com/tiero/ocean/pkg/keypair"
	"github.com/tiero/ocean/pkg/partial"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

const aliceHex = "bfb96a215dfb07d1a193464174b9ea8e91f2a15bba79800dea838add330f6d86"
const bobHex = "1804e76aa3016013bc9969103554668913cf697c03c23aecb28136d0e0ac16f0"
const defaultExplorer = "http://localhost:3001"

func TestTopupWithAsset(t *testing.T) {
	time.Sleep(2 * time.Second)
	explorerURL, ok := os.LookupEnv("TAXI_EXPLORER_ENDPOINT")
	if !ok {
		explorerURL = defaultExplorer
	}
	//Explorer
	e := blockstream.NewExplorer(explorerURL)

	// Network
	currentNetwork := network.Regtest

	// Alice
	fromKeyPair, _ := keypair.FromPrivateKey(aliceHex)
	fromPayment := payment.FromPublicKey(fromKeyPair.PublicKey, &currentNetwork, nil)
	from, _ := fromPayment.WitnessPubKeyHash()
	fromScript := fromPayment.WitnessScript
	//Bob
	toKeyPair, _ := keypair.FromPrivateKey(bobHex)
	toPayment := payment.FromPublicKey(toKeyPair.PublicKey, &currentNetwork, nil)
	toScript := toPayment.WitnessScript
	asset, err := mint(from, 500000, explorerURL)
	if err != nil {
		t.Fatalf("mint: %s", err)
	}
	time.Sleep(5 * time.Second)
	utxos, err := e.GetUnspents(from)
	if err != nil {
		t.Fatalf("unspents: %s", err)
	}
	aliceUtxo := utxos[0]
	// How much we send to Bob
	amount := 250000
	// The asset change
	change := aliceUtxo.Value() - uint64(amount)

	// Empty PSET
	psetWithoutFees := partial.NewPartial(&currentNetwork)

	psetWithoutFees.AddInput(aliceUtxo.Hash(), aliceUtxo.Index(), &partial.WitnessUtxo{Asset: asset, Value: aliceUtxo.Value(), Script: fromScript}, nil)
	psetWithoutFees.AddOutput(asset, uint64(amount), toScript, false)
	psetWithoutFees.AddOutput(asset, change, fromScript, false)

	// Base64 encode
	b64, err := psetWithoutFees.Data.ToBase64()
	if err != nil {
		t.Fatalf("base64: %s", err)
	}

	serv := Server{}
	replyEstimate, err := serv.GetAssetEstimate(context.Background(), &taxi.GetAssetEstimateRequest{
		Unsigned: &taxi.Unsigned{
			Pset:       b64,
			SatPerByte: 0.5,
		},
		AssetHash: asset,
	})
	if err != nil {
		t.Fatalf("taxi: %s", err)
	}
	if replyEstimate.AssetAmount <= 0 {
		t.Fatalf("taxi: Invalid number")
	}

	// Delete last inserted element that in our case is the change
	psetWithoutFees.Data.Outputs = psetWithoutFees.Data.Outputs[:1]
	psetWithoutFees.Data.UnsignedTx.Outputs = psetWithoutFees.Data.UnsignedTx.Outputs[:1]
	// Add recalculated change after estimate with taxi
	recalculatedChange := uint64(change) - replyEstimate.AssetAmount
	psetWithoutFees.AddOutput(asset, recalculatedChange, fromScript, false)

	// Base64 encode
	b64, err = psetWithoutFees.Data.ToBase64()
	if err != nil {
		t.Fatalf("base64: %s", err)
	}

	// Lets topup with taxi server
	replyTopup, err := serv.TopupWithAsset(context.Background(), &taxi.TopupWithAssetRequest{
		Unsigned: &taxi.Unsigned{
			Pset:       b64,
			SatPerByte: 0.5,
		},
		AssetHash: asset,
	})

	decoded, err := pset.NewPsetFromBase64(replyTopup.GetOrder().GetPartial().GetPset())
	if err != nil {
		t.Fatalf("decode: %s", err)
	}

	psetWithFees := &partial.Partial{Data: decoded, Network: &currentNetwork}
	err = psetWithFees.SignWithPrivateKey(0, fromKeyPair)
	if err != nil {
		t.Fatalf("SignWithPrivateKey: %s", err)
	}

	err = pset.FinalizeAll(psetWithFees.Data)
	if err != nil {
		t.Fatal(err)
	}
	// Extract the final signed transaction from the Pset wrapper.
	finalTx, err := pset.Extract(psetWithFees.Data)
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

	if len(res) <= 0 || strings.Contains(res, "sendrawtransaction") {
		t.Fatalf("Expected transaction to be broadcasted, failed for reason: %s", res)
	}

	println(res)
}

// Mint ...
func mint(address string, qty int, baseURL string) (string, error) {
	url := baseURL + "/mint"
	payload := map[string]interface{}{"address": address, "quantity": qty}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := http.Post(url, "appliation/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respBody := map[string]interface{}{}
	err = json.Unmarshal(data, &respBody)
	if err != nil {
		return "", err
	}

	return respBody["asset"].(string), nil
}
