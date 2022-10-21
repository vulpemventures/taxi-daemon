package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"flag"
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
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/confidential"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/psetv2"
	"github.com/vulpemventures/go-elements/transaction"
	pb "github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/go/taxi/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	taxiAddr                = flag.String("taxi-addr", "localhost:8300", "<host>:<port> of Taxi daemon to interact with")
	ctx                     = context.Background()
	regtest                 = &network.Regtest
	explorerURL             = "http://localhost:3001"
	receivingAddress        = "el1qqtcxcrfyf8cd68g2hd69gpx3gj7cgd7qa26ff9tfpsnw5mkzd32swt54sqdyfee8gw3vghwp5r7aspqkhnsh29f2yuyen9x37"
	receivingAddressInfo, _ = address.FromConfidential(receivingAddress)
)

func main() {
	flag.Parse()

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
	fmt.Printf("Connecting to taxi service at %s...\n", *taxiAddr)
	conn, err := grpc.Dial(
		*taxiAddr, grpc.WithTransportCredentials(insecure.NewCredentials()),
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
	amount := float64(1)
	if err := faucet(addr, asset, amount); err != nil {
		log.Fatalf("failed to fund confidential address: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Retrieving ephemeral wallet's funds...")
	utxos, err := getUtxos(addr)
	if err != nil {
		log.Fatalf("failed to fetch ephemeral wallet funds: %s", err)
	}
	txHex, err := fetchTx(utxos[0]["txid"].(string))
	if err != nil {
		log.Fatalf("failed to fetch ephemeral wallet funds transaction: %s", err)
	}
	prevoutTx, _ := transaction.NewTxFromHex(txHex)
	prevout := prevoutTx.Outputs[int(utxos[0]["vout"].(float64))]

	fmt.Printf("Done.\n\n")
	fmt.Println("Crafting transaction w/ known payout...")
	inputs := make([]psetv2.InputArgs, 0, len(utxos))
	for _, u := range utxos {
		inputs = append(inputs, psetv2.InputArgs{
			Txid:    u["txid"].(string),
			TxIndex: uint32(u["vout"].(float64)),
		})
	}

	outputs := []psetv2.OutputArgs{
		{
			Asset:        asset,
			Amount:       7 * uint64(math.Pow10(7)),
			Script:       receivingAddressInfo.Script,
			BlindingKey:  receivingAddressInfo.BlindingKey,
			BlinderIndex: 0,
		},
		{
			Asset:        asset,
			Amount:       3 * uint64(math.Pow10(7)),
			Script:       p2wpkh.WitnessScript,
			BlindingKey:  p2wpkh.BlindingKey.SerializeCompressed(),
			BlinderIndex: 0,
		},
	}

	pset, _ := psetv2.New(inputs, outputs, nil)
	updater, _ := psetv2.NewUpdater(pset)
	updater.AddInWitnessUtxo(0, prevout)
	updater.AddInUtxoRangeProof(0, prevout.RangeProof)
	updater.AddInSighashType(0, txscript.SigHashAll)

	fmt.Printf("Done.\n\n")
	fmt.Println("Blinding transaction...")
	blindValidator := confidential.NewZKPValidator()
	blindGenerator := confidential.NewZKPGeneratorFromBlindingKeys([][]byte{
		blindPrivkey.Serialize(),
	}, nil)

	unblindedInputs, err := blindGenerator.UnblindInputs(pset, nil)
	if err != nil {
		log.Fatalf("failed to unblind tx inputs: %s", err)
	}
	outBlindingArgs, err := blindGenerator.BlindOutputs(pset, nil, nil)
	if err != nil {
		log.Fatalf("failed to generate blinding data for tx outputs: %s", err)
	}

	blinder, err := psetv2.NewBlinder(
		pset, unblindedInputs, blindValidator, blindGenerator,
	)
	if err != nil {
		log.Fatalf("failed to initialize blinder: %s", err)
	}

	if err := blinder.BlindNonLast(nil, outBlindingArgs); err != nil {
		log.Fatalf("failed to add blind tx: %s", err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Println("Asking to taxi service for topup w/ known payout...")

	psetBase64, _ := pset.ToBase64()
	resp, err := client.TopupWithKnownPayout(ctx, &pb.TopupWithKnownPayoutRequest{
		Partial:         psetBase64,
		MillisatPerByte: 100,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Done.\n\n")
	fmt.Printf("Topup: %s\n\n", resp.GetTopup().GetTopupId())
	topup := resp.GetTopup()

	pset, _ = psetv2.NewPsetFromBase64(topup.GetPartial())

	fmt.Println("Signing transaction...")
	inIndexesToSign := []int{0}
	signedPtx, err := signTransactionP2WPKH(
		pset, inIndexesToSign, privkey, p2wpkh.Script,
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
		signer.AddInSighashType(i, sighashType)

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
