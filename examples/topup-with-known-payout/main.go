package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/txscript"
	"github.com/vulpemventures/go-elements/confidential"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/psetv2"
	"github.com/vulpemventures/go-elements/transaction"
	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ctx              = context.Background()
	regtest          = &network.Regtest
	taxiURL          = "localhost:8000"
	explorerURL      = "http://localhost:3001"
	fundingAmount    = 10 * uint64(math.Pow10(8))
	spendingAmount   = 1 * uint64(math.Pow10(8))
	receivingAddress = "el1qqtcxcrfyf8cd68g2hd69gpx3gj7cgd7qa26ff9tfpsnw5mkzd32swt54sqdyfee8gw3vghwp5r7aspqkhnsh29f2yuyen9x37"
)

func main() {
	fmt.Println("Creating ephemeral single key wallet...")
	privkey, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatalf("failed to create signing private key: %s", err)
	}
	blindPrivkey, err := btcec.NewPrivateKey()
	if err != nil {
		log.Fatalf("failed to create blinding private key: %s", err)
	}

	p2wpkh := payment.FromPublicKey(privkey.PubKey(), regtest, blindPrivkey.PubKey())
	addr, err := p2wpkh.ConfidentialWitnessPubKeyHash()
	if err != nil {
		log.Fatalf("failed to create confidential address: %s", err)
	}

	fmt.Println("Ephemeral single key wallet:")
	fmt.Println("Key pair:", hex.EncodeToString(privkey.Serialize()), hex.EncodeToString(privkey.PubKey().SerializeCompressed()))
	fmt.Println("Blinding key pair:", hex.EncodeToString(blindPrivkey.Serialize()), hex.EncodeToString(blindPrivkey.PubKey().SerializeCompressed()))
	fmt.Println("Confidential address:", addr)

	fmt.Printf("\n\n")
	fmt.Printf("Connecting to taxi service at %s...\n", taxiURL)
	conn, err := grpc.Dial(
		taxiURL, grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to connect to taxi service: %s", err)
	}
	client := pb.NewTaxiServiceClient(conn)

	fmt.Printf("Done.\n\n")
	res, err := client.ListAssets(ctx, &pb.ListAssetsRequest{})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Funding ephemeral single key wallet with one of taxi's supported assets...")
	asset := res.GetAssets()[0].GetAssetHash()
	amount := float64(fundingAmount) / math.Pow10(8)
	if err := faucet(addr, asset, amount); err != nil {
		log.Fatalf("failed to fund confidential address: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Retrieving ephemeral wallet's funds...")
	utxos, err := getUtxos(addr)
	if err != nil {
		log.Fatalf("failed to fetch ephemeral wallet funds: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Asking to taxi service for topup w/ asset...")
	resp, err := client.TopupWithAsset(ctx, &pb.TopupWithAssetRequest{
		AssetHash:       asset,
		EstimatedTxSize: 3000,
		MillisatPerByte: 100,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Topup:", resp.GetTopup().GetTopupId())
	topup := resp.GetTopup()
	inBlindingData := resp.GetInBlindingData()

	fmt.Printf("\n\n")
	fmt.Println("Adding ins and outs to topup transaction...")
	ptx, err := psetv2.NewPsetFromBase64(topup.GetPartial())
	if err != nil {
		log.Fatalf("failed to parse topup transaction: %s", err)
	}
	originalInputCount := int(ptx.Global.InputCount)
	updater, err := psetv2.NewUpdater(ptx)
	if err != nil {
		log.Fatalf("failed to init updater role: %s", err)
	}

	ins := make([]psetv2.InputArgs, 0, len(utxos))
	for _, u := range utxos {
		ins = append(ins, psetv2.InputArgs{
			Txid:    u["txid"].(string),
			TxIndex: uint32(u["vout"].(float64)),
		})
	}
	if err := updater.AddInputs(ins); err != nil {
		log.Fatalf("failed to add inputs to tx: %s", err)
	}

	ownedInputs := make(map[uint32]psetv2.OwnedInput)
	for i, data := range inBlindingData {
		ownedInputs[uint32(i)] = psetv2.OwnedInput{
			Index:        uint32(i),
			Value:        data.Value,
			Asset:        data.Asset,
			ValueBlinder: data.ValueBlinder,
			AssetBlinder: data.AssetBlinder,
		}
	}
	for i, in := range ins {
		prevTxHex, err := fetchTx(in.Txid)
		if err != nil {
			log.Fatalf("failed to fetch raw tx for input %d: %s", i, err)
		}

		prevTx, _ := transaction.NewTxFromHex(prevTxHex)
		assetCommitment := h2b(utxos[0]["assetcommitment"].(string))
		valueCommitment := h2b(utxos[0]["valuecommitment"].(string))
		witnessUtxo := &transaction.TxOutput{
			Asset:           assetCommitment,
			Value:           valueCommitment,
			Script:          p2wpkh.WitnessScript,
			Nonce:           prevTx.Outputs[in.TxIndex].Nonce,
			RangeProof:      prevTx.Outputs[in.TxIndex].RangeProof,
			SurjectionProof: prevTx.Outputs[in.TxIndex].SurjectionProof,
		}
		updater.AddInWitnessUtxo(witnessUtxo, originalInputCount+i)

		revelead, err := confidential.UnblindOutputWithKey(witnessUtxo, blindPrivkey.Serialize())
		if err != nil {
			log.Fatalf("failed to unblind prevout of input %d: %s", originalInputCount+i, err)
		}
		ownedInputs[uint32(originalInputCount+i)] = psetv2.OwnedInput{
			Index:        uint32(originalInputCount + i),
			Value:        revelead.Value,
			Asset:        hex.EncodeToString(elementsutil.ReverseBytes(revelead.Asset)),
			ValueBlinder: revelead.ValueBlindingFactor,
			AssetBlinder: revelead.AssetBlindingFactor,
		}
	}

	changeAmount := fundingAmount - spendingAmount - resp.GetAssetAmount()
	outs := []psetv2.OutputArgs{
		{
			Asset:        asset,
			Amount:       spendingAmount,
			Address:      receivingAddress,
			BlinderIndex: 1,
		},
		{
			Asset:        asset,
			Amount:       changeAmount,
			Address:      addr,
			BlinderIndex: 1,
		},
	}
	if err := updater.AddOutputs(outs); err != nil {
		log.Fatalf("failed to add outputs to tx: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Blinding transaction...")
	blindValidator := confidential.NewZKPValidator()
	blindGenerator, err := confidential.NewZKPGeneratorFromOwnedInputs(
		ownedInputs, nil,
	)
	if err != nil {
		log.Fatalf("failed to initialize blinder handler: %s", err)
	}
	ownedInputsSlice, err := blindGenerator.UnblindInputs(updater.Pset, nil)
	if err != nil {
		log.Fatalf("failed to unblind ephemeral wallet's inputs: %s", err)
	}

	outIndexesToBlind := []uint32{
		uint32(updater.Pset.Global.OutputCount) - 2, // second last
		uint32(updater.Pset.Global.OutputCount) - 1, // last
	}
	blindedOuts, err := blindGenerator.BlindOutputs(
		updater.Pset, outIndexesToBlind, nil,
	)
	if err != nil {
		log.Fatalf("failed to generate blinding commitments and proofs: %s", err)
	}
	blinder, err := psetv2.NewBlinder(
		updater.Pset, ownedInputsSlice, blindValidator, blindGenerator,
	)
	if err != nil {
		log.Fatalf("failed to init blinder role: %s", err)
	}
	if err := blinder.BlindLast(nil, blindedOuts); err != nil {
		log.Fatalf("failed to blind transaction: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Signing transaction...")
	inIndexesToSign := []int{
		int(blinder.Pset.Global.InputCount) - 1,
	}
	signedPtx, err := signTransactionP2WPKH(
		blinder.Pset, inIndexesToSign, privkey, p2wpkh.Script,
	)
	if err != nil {
		log.Fatalf("failed to sign transaction: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Finalizing transaction...")
	if err := psetv2.FinalizeAll(signedPtx); err != nil {
		log.Fatalf("failed to finalize transaction: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Extracting raw transaction...")
	rawTx, err := psetv2.Extract(signedPtx)
	if err != nil {
		log.Fatalf("failed to extract raw transaction: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Broadcasting transaction...")
	txhex, _ := rawTx.ToHex()
	txid, err := broadcastTx(txhex)
	if err != nil {
		log.Fatalf("failed to broadcast raw transaction: %s", err)
	}

	fmt.Println("Broadcasted transaction with fees paid by taxi:", txid)
}

func faucet(addr, asset string, amount float64) error {
	url := fmt.Sprintf("%s/faucet", explorerURL)
	payload := map[string]interface{}{
		"address": addr,
		"asset":   asset,
		"amount":  amount,
	}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if res := string(data); len(res) <= 0 || strings.Contains(res, "sendtoaddress") {
		return fmt.Errorf(res)
	}

	return nil
}

func getUtxos(addr string) ([]map[string]interface{}, error) {
	getUtxos := func(address string) ([]interface{}, error) {
		url := fmt.Sprintf("%s/address/%s/utxo", explorerURL, address)
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var respBody interface{}
		if err := json.Unmarshal(data, &respBody); err != nil {
			return nil, err
		}
		return respBody.([]interface{}), nil
	}

	utxos := []map[string]interface{}{}
	for len(utxos) <= 0 {
		time.Sleep(1 * time.Second)
		u, err := getUtxos(addr)
		if err != nil {
			return nil, err
		}
		for _, unspent := range u {
			utxo := unspent.(map[string]interface{})
			utxos = append(utxos, utxo)
		}
	}

	return utxos, nil
}

func fetchTx(txId string) (string, error) {
	url := fmt.Sprintf("%s/tx/%s/hex", explorerURL, txId)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func broadcastTx(txHex string) (string, error) {
	url := fmt.Sprintf("%s/tx", explorerURL)
	resp, err := http.Post(url, "text/plain", strings.NewReader(txHex))
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	res := string(data)
	if len(res) <= 0 || strings.Contains(res, "sendrawtransaction") {
		return "", fmt.Errorf(res)
	}
	return res, nil
}

func signTransactionP2WPKH(
	p *psetv2.Pset, inIndexesToSign []int,
	privkey *btcec.PrivateKey, script []byte,
) (*psetv2.Pset, error) {
	signer, err := psetv2.NewSigner(p)
	if err != nil {
		return nil, fmt.Errorf("failed to init signer role: %s", err)
	}
	sighashType := txscript.SigHashAll
	pubkey := privkey.PubKey()

	for _, i := range inIndexesToSign {
		signer.AddInSighashType(sighashType, i)

		in := p.Inputs[i]
		prevout := in.GetUtxo()
		tx, _ := p.UnsignedTx()
		sigHash := tx.HashForWitnessV0(i, script, prevout.Value, sighashType)

		sig := ecdsa.Sign(privkey, sigHash[:])
		sigWithHashType := append(sig.Serialize(), byte(sighashType))

		if err := signer.SignInput(
			i, sigWithHashType, pubkey.SerializeCompressed(), nil, nil,
		); err != nil {
			return nil, err
		}
	}

	valid, err := p.ValidateAllSignatures()
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, fmt.Errorf("invalid signatures")
	}

	return p, nil
}

func h2b(str string) []byte {
	buf, _ := hex.DecodeString(str)
	return buf
}
