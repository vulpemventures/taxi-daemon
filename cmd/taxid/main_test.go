package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/prometheus/common/log"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/tj/assert"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/liquid-taxi/config"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
	g "google.golang.org/grpc"
)

var emptyContext = context.Background()

func TestTaxiServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// In the scenario, alice will send USDT to bob
	// However, alice has only USDT and would like to pay fees using them.
	aliceKeyPair, _ := btcec.NewPrivateKey(btcec.S256())
	alice := payment.FromPublicKey(aliceKeyPair.PubKey(), &network.Regtest, nil)
	aliceAddress, _ := alice.WitnessPubKeyHash()

	// Let's simulate alice's USDTs
	_, USDT, err := mint(config.GetString(config.ExplorerEndpointKey), aliceAddress, 100000)

	// start taxi
	go main()
	// give time to server to serve
	time.Sleep(10 * time.Second)

	// SERVER SIDE --->
	// The admin connects to admin interface
	conn, err := g.Dial("localhost:7000", g.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	adminClient := admin.NewAdminClient(conn)

	// Init with USDT
	_, err = adminClient.AddAsset(emptyContext, &admin.AddAssetRequest{
		AssetHash: USDT,
		Price:     20.36,
	})
	if err != nil {
		t.Fatal(err)
	}

	faReply, err := adminClient.FundingAddress(emptyContext, &admin.FundingAddressRequest{})
	if err != nil {
		t.Fatal(err)
	}
	addr := faReply.GetAddresses()[0]

	if _, err := faucet(addr, config.GetString(config.ExplorerEndpointKey)); err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	// CLIENT SIDE --->
	// Alice connect to server
	conn, err = g.Dial("localhost:8000", g.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewTaxiClient(conn)

	// Alice topup with USDT
	reply, err := client.TopupWithAsset(emptyContext, &pb.TopupWithAssetRequest{
		AssetHash: USDT,
	})
	if err != nil {
		t.Fatal(err)
	}

	b64Tx := reply.Topup.GetPartial()
	p, err := pset.NewPsetFromBase64(b64Tx)
	if err != nil {
		t.Fatal(err)
	}

	signatureIsValid, err := p.ValidateAllSignatures()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, signatureIsValid)

	// Let's create our Bob
	bobKeyPair, _ := btcec.NewPrivateKey(btcec.S256())
	bob := payment.FromPublicKey(bobKeyPair.PubKey(), &network.Regtest, nil)
	// bobAddress, _ := bob.WitnessPubKeyHash()

	// Now, alice adds his own inputs and outputs
	aliceExplorerSvc := explorer.NewService(config.GetString(config.ExplorerEndpointKey))
	time.Sleep(3 * time.Second)
	utxos, err := aliceExplorerSvc.GetUnspents(aliceAddress, [][]byte{})

	aliceUpdater, err := pset.NewUpdater(p)
	if err != nil {
		t.Fatal(err)
	}

	bobWant := uint64(100)
	totalToSend := bobWant + reply.Topup.GetAssetAmount()
	change := uint64(0)

	var witnessUtxoValue []byte

	for _, utxo := range utxos {
		if utxo.Asset() == USDT && utxo.Value() >= totalToSend {
			aliceInput, prevout, err := utxo.Parse()
			if err != nil {
				t.Fatal(err)
			}

			witnessUtxoValue = prevout.Value
			change = utxo.Value() - totalToSend

			aliceUpdater.AddInput(aliceInput)
			err = aliceUpdater.AddInWitnessUtxo(prevout, 1)
			if err != nil {
				t.Fatal(err)
			}

			break
		}
	}

	USDTbuf, _ := hex.DecodeString(USDT)
	USDTasset := append([]byte{0x01}, elementsutil.ReverseBytes(USDTbuf)...)

	value, err := elementsutil.SatoshiToElementsValue(bobWant)
	if err != nil {
		t.Fatal(err)
	}

	sendToBobOutput := transaction.NewTxOutput(USDTasset, value, bob.WitnessScript)
	aliceUpdater.AddOutput(sendToBobOutput)

	if change > 0 {
		value, err := elementsutil.SatoshiToElementsValue(change)
		if err != nil {
			t.Fatal(err)
		}

		changeOutput := transaction.NewTxOutput(USDTasset, value, alice.WitnessScript)
		aliceUpdater.AddOutput(changeOutput)
	}

	// Now alice sign the input
	if err := aliceUpdater.AddInSighashType(txscript.SigHashAll, 1); err != nil {
		t.Fatal(err)
	}

	sigHash := p.UnsignedTx.HashForWitnessV0(
		1,
		alice.Script,
		witnessUtxoValue,
		txscript.SigHashAll,
	)

	sig, err := aliceKeyPair.Sign(sigHash[:])
	if err != nil {
		t.Fatal(err)
	}

	sigWithHashType := append(sig.Serialize(), byte(txscript.SigHashAll))

	if _, err := aliceUpdater.Sign(
		1,
		sigWithHashType,
		aliceKeyPair.PubKey().SerializeCompressed(),
		nil,
		nil,
	); err != nil {
		t.Fatal(err)
	}

	valid, err := p.ValidateAllSignatures()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, true, valid)

	err = pset.FinalizeAll(p)
	if err != nil {
		t.Fatal(err)
	}

	// Extract the final signed transaction from the Pset wrapper.
	finalTx, err := pset.Extract(p)
	if err != nil {
		t.Fatal(err)
	}

	// Serialize the transaction and try to broadcast.
	txHex, err := finalTx.ToHex()
	if err != nil {
		t.Fatal(err)
	}

	log.Info(txHex)

	// TODO uncomment when crawler is implemented
	// _, err = aliceExplorerSvc.Broadcast(txHex)
	// if err != nil {
	// 	t.Fatal(err)
	// }
}

func mint(apiURL string, address string, amount int) (string, string, error) {
	url := fmt.Sprintf("%s/mint", apiURL)
	payload := map[string]interface{}{"address": address, "quantity": amount}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", "", err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	respBody := map[string]interface{}{}
	err = json.Unmarshal(data, &respBody)
	if err != nil {
		return "", "", err
	}

	return respBody["txId"].(string), respBody["asset"].(string), nil
}

func faucet(address string, baseURL string) (string, error) {
	url := baseURL + "/faucet"
	payload := map[string]string{"address": address}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
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
