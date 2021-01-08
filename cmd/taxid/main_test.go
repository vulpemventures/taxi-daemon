package main

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/prometheus/common/log"
	"github.com/stretchr/testify/assert"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/liquid-taxi/pkg/testutils"
	pb "github.com/vulpemventures/taxi-protobuf/generated/go/taxi"
	g "google.golang.org/grpc"
)

const explorerEndpoint = "http://localhost:3001"

var emptyContext = context.Background()

func TestTaxiServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	// In the scenario, alice will send USDT to bob
	// However, alice has only USDT and would like to pay fees using them.

	explorerSvc := explorer.NewService(explorerEndpoint)

	aliceKeyPair, _ := btcec.NewPrivateKey(btcec.S256())
	aliceBlindingPrivKey, _ := btcec.NewPrivateKey(btcec.S256())
	alice := payment.FromPublicKey(aliceKeyPair.PubKey(), &network.Regtest, aliceBlindingPrivKey.PubKey())
	aliceAddress, _ := alice.ConfidentialWitnessPubKeyHash()

	// Let's simulate alice's USDTs
	_, USDT, err := explorerSvc.Mint(aliceAddress, 100000)

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

	if err := testutils.Fragment(faReply.GetAddresses()); err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

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

	blindingPrivKeyTaxi, err := hex.DecodeString(reply.PrivateBlindingKey)
	if err != nil {
		t.Fatal(err)
	}

	blindingPubKeyTaxi, err := hex.DecodeString(reply.PublicBlindingKey)
	if err != nil {
		t.Fatal(err)
	}

	privateBlindingKeys := make([][]byte, 0)
	privateBlindingKeys = append(privateBlindingKeys, blindingPrivKeyTaxi)

	publicBlindingKeys := make([][]byte, 0)
	publicBlindingKeys = append(publicBlindingKeys, blindingPubKeyTaxi)

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
	bobBlindingPrivKey, _ := btcec.NewPrivateKey(btcec.S256())
	bob := payment.FromPublicKey(bobKeyPair.PubKey(), &network.Regtest, bobBlindingPrivKey.PubKey())
	// bobAddress, _ := bob.WitnessPubKeyHash()

	// Now, alice adds his own inputs and outputs
	aliceExplorerSvc := explorer.NewService(explorerEndpoint)
	time.Sleep(3 * time.Second)
	utxos, err := aliceExplorerSvc.GetUnspents(aliceAddress, [][]byte{aliceBlindingPrivKey.Serialize()})

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

			privateBlindingKeys = append(privateBlindingKeys, aliceBlindingPrivKey.Serialize())

			break
		}
	}

	USDTbuf, _ := hex.DecodeString(USDT)
	USDTasset := append([]byte{0x01}, elementsutil.ReverseBytes(USDTbuf)...)

	LBTCbuf, _ := hex.DecodeString(network.Regtest.AssetID)
	LBTCasset := append([]byte{0x01}, elementsutil.ReverseBytes(LBTCbuf)...)

	value, err := elementsutil.SatoshiToElementsValue(bobWant)
	if err != nil {
		t.Fatal(err)
	}

	sendToBobOutput := transaction.NewTxOutput(USDTasset, value, bob.WitnessScript)
	aliceUpdater.AddOutput(sendToBobOutput)

	publicBlindingKeys = append(publicBlindingKeys, bobBlindingPrivKey.PubKey().SerializeCompressed())

	if change > 0 {
		value, err := elementsutil.SatoshiToElementsValue(change)
		if err != nil {
			t.Fatal(err)
		}

		changeOutput := transaction.NewTxOutput(USDTasset, value, alice.WitnessScript)
		aliceUpdater.AddOutput(changeOutput)
		publicBlindingKeys = append(publicBlindingKeys, aliceBlindingPrivKey.PubKey().SerializeCompressed())
	}

	// add the fee output
	feeOutput := transaction.NewTxOutput(LBTCasset, p.Inputs[0].WitnessUtxo.Value, []byte{})
	aliceUpdater.AddOutput(feeOutput)

	err = testutils.BlindTransaction(p, privateBlindingKeys, publicBlindingKeys)
	if err != nil {
		t.Fatal(err)
	}

	aliceUpdater, err = pset.NewUpdater(p)
	if err != nil {
		t.Fatal(err)
	}

	// Now alice signs the input
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

	txid, err := explorerSvc.BroadcastTransaction(txHex)
	if err != nil {
		t.Fatal(err)
	}

	log.Info("transaction broadcasted: ", txid)
}
