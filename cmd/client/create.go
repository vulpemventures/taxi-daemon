package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/pkg/partial"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
	"github.com/vulpemventures/liquid-taxi/rpc/taxi"
)

func createAction(from string, to string, asset string, amount int, regtest bool, explorerURL string, taxiURL string) error {
	if from == "" || to == "" || asset == "" || amount <= 0 {
		return fmt.Errorf("missing required flag")
	}

	// Explorer
	e := &explorer.Explorer{BaseURL: explorerURL}
	// Network
	currentNetwork := network.Liquid
	if regtest {
		currentNetwork = network.Regtest
	}

	fromScript, err := address.ToOutputScript(from, currentNetwork)
	if err != nil {
		return fmt.Errorf("from: %w", err)
	}
	toScript, err := address.ToOutputScript(to, currentNetwork)
	if err != nil {
		return fmt.Errorf("to: %w", err)
	}

	assetBytes := wallet.AssetHashToBytes(asset, false)

	utxos, err := e.Unspents(from)
	if err != nil {
		return fmt.Errorf("unspents: %w", err)
	}

	selectedUtxos, change, err := wallet.CoinSelect(utxos, uint64(amount), asset)
	if err != nil {
		return fmt.Errorf("coin selection: %w", err)
	}

	emptyPset, _ := pset.New([]*transaction.TxInput{}, []*transaction.TxOutput{}, 2, 0)
	psetWithoutFees := &partial.Partial{Data: emptyPset}
	for _, utxo := range selectedUtxos {
		psetWithoutFees.AddInput(utxo.Hash, utxo.Index, &partial.WitnessUtxo{Asset: assetBytes, Value: utxo.Value, Script: fromScript}, nil)
	}
	psetWithoutFees.AddOutput(assetBytes, uint64(amount), toScript)
	psetWithoutFees.AddOutput(assetBytes, uint64(change), fromScript)

	b64, err := psetWithoutFees.Data.ToBase64()
	if err != nil {
		return fmt.Errorf("base64: %w", err)
	}

	client := taxi.NewTaxiProtobufClient(taxiURL, &http.Client{})
	replyEstimate, err := client.GetAssetEstimate(context.Background(), &taxi.GetAssetEstimateRequest{
		Unsigned: &taxi.Unsigned{
			Pset:       b64,
			SatPerByte: 0.1,
		},
		AssetHash: asset,
	})
	if err != nil {
		return fmt.Errorf("taxi: %w", err)
	}
	if replyEstimate.AssetAmount <= 0 {
		return fmt.Errorf("taxi: Invalid number")
	}

	// Delete last inserted element that in our case is the change
	psetWithoutFees.Data.Outputs = psetWithoutFees.Data.Outputs[:1]
	psetWithoutFees.Data.UnsignedTx.Outputs = psetWithoutFees.Data.UnsignedTx.Outputs[:1]

	// Add recalculated change after estimate with taxi
	psetWithoutFees.AddOutput(assetBytes, uint64(change)-replyEstimate.AssetAmount, fromScript)

	b64, err = psetWithoutFees.Data.ToBase64()
	if err != nil {
		return fmt.Errorf("base64: %w", err)
	}

	fmt.Println()
	fmt.Println(b64)

	return nil
}
