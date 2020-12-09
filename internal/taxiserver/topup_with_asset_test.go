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

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/internal/explorer"
	"github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

const defaultExplorer = "http://localhost:3001"

func TestTopupWithAsset(t *testing.T) {
	time.Sleep(5 * time.Second)
	explorerURL, ok := os.LookupEnv("TAXI_EXPLORER_ENDPOINT")
	if !ok {
		explorerURL = defaultExplorer
	}
	//Explorer
	e := explorer.NewEsplorerService(explorerURL)

	// Network
	currentNetwork := network.Regtest

	// Alice
	fromKeyPair, _ := btcec.NewPrivateKey(btcec.S256())
	fromPayment := payment.FromPublicKey(fromKeyPair.PubKey(), &currentNetwork, nil)
	from, _ := fromPayment.WitnessPubKeyHash()
	fromScript := fromPayment.WitnessScript
	//Bob
	toKeyPair, _ := btcec.NewPrivateKey(btcec.S256())
	toPayment := payment.FromPublicKey(toKeyPair.PubKey(), &currentNetwork, nil)
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
	amount := 25000000000000
	// The asset change
	change := aliceUtxo.Value - uint64(amount)

	amountValue, err := elementsutil.SatoshiToElementsValue(uint64(amount))
	if err != nil {
		t.Error(err)
	}

	changeValue, err := elementsutil.SatoshiToElementsValue(uint64(change))
	if err != nil {
		t.Error(err)
	}

	assetH := assetHash(asset)
	input := transaction.NewTxInput(aliceUtxo.Hash(), aliceUtxo.Vout)
	toOutput := transaction.NewTxOutput(assetH, amountValue, toScript)
	fromOutput := transaction.NewTxOutput(assetH, changeValue, fromScript)

	inputs := []*transaction.TxInput{input}
	outputs := []*transaction.TxOutput{toOutput, fromOutput}
	p, err := pset.New(inputs, outputs, 2, 0)
	if err != nil {
		t.Error(err)
	}

	updater, err := pset.NewUpdater(p)
	if err != nil {
		t.Error(err)
	}

	aliceWitnessUtxo, err := aliceUtxo.FetchPrevout(e)
	if err != nil {
		t.Error(err)
	}

	if err := updater.AddInWitnessUtxo(aliceWitnessUtxo, 0); err != nil {
		t.Error(err)
	}

	// Base64 encode
	b64, err := p.ToBase64()
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
	p.Outputs = p.Outputs[:1]
	p.UnsignedTx.Outputs = p.UnsignedTx.Outputs[:1]
	// Add recalculated change after estimate with taxi
	recalculatedChange := uint64(change) - replyEstimate.AssetAmount

	recalculatedChangeValue, err := elementsutil.SatoshiToElementsValue(recalculatedChange)
	if err != nil {
		t.Error(err)
	}

	newChangeOutput := transaction.NewTxOutput(assetH, recalculatedChangeValue, fromScript)

	updater.AddOutput(newChangeOutput)

	// Base64 encode
	b64, err = p.ToBase64()
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
		OrderId:   replyEstimate.OrderId,
	})
	if err != nil {
		t.Fatalf("TopupWithAsset: %s", err)
	}

	decoded, err := pset.NewPsetFromBase64(replyTopup.GetOrder().GetPartial().GetPset())
	if err != nil {
		t.Fatalf("decode: %s", err)
	}

	signUpdater, err := pset.NewUpdater(decoded)
	if err != nil {
		t.Error(err)
	}

	if err := signUpdater.AddInSighashType(txscript.SigHashAll, 0); err != nil {
		t.Error(err)
	}

	sigHash := decoded.UnsignedTx.HashForWitnessV0(
		0,
		fromPayment.Script,
		decoded.Inputs[0].WitnessUtxo.Value,
		txscript.SigHashAll,
	)

	sig, err := fromKeyPair.Sign(sigHash[:])
	if err != nil {
		t.Error(err)
	}

	sigWithHashType := append(sig.Serialize(), byte(txscript.SigHashAll))

	_, err = signUpdater.Sign(0, sigWithHashType, fromKeyPair.PubKey().SerializeCompressed(), nil, nil)
	if err != nil {
		t.Error(err)
	}

	err = pset.FinalizeAll(decoded)
	if err != nil {
		t.Fatal(err)
	}
	// Extract the final signed transaction from the Pset wrapper.
	finalTx, err := pset.Extract(decoded)
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
