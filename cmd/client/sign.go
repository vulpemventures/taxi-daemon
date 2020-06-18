package main

import (
	"bytes"
	"fmt"

	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/liquid-taxi/pkg/partial"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

func signAction(psetBase64 string, privateKey string) error {
	if psetBase64 == "" || privateKey == "" {
		return fmt.Errorf("missing required flag")
	}

	decoded, err := pset.NewPsetFromBase64(psetBase64)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	pay, _, privKey := wallet.FromPrivateKey(privateKey)

	psetWithFees := &partial.Partial{Data: decoded}
	for i := 0; i < len(psetWithFees.Data.Inputs); i++ {
		currInput := psetWithFees.Data.Inputs[i]
		if bytes.Equal(currInput.WitnessUtxo.Script, pay.Script) {
			println(i)
			err := psetWithFees.SignWithPrivateKey(i, pay, privKey)
			if err != nil {
				return fmt.Errorf("sign: %w", err)
			}
		}
	}
	pFinalized := psetWithFees.Data
	err = pset.FinalizeAll(pFinalized)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	if !pFinalized.IsComplete() {
		return fmt.Errorf("pset not complete: %w", err)
	}

	err = pFinalized.SanityCheck()
	if err != nil {
		return fmt.Errorf("sanity check: %w", err)
	}

	// Extract the final signed transaction from the Pset wrapper.
	finalTx, err := pset.Extract(pFinalized)
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	// Serialize the transaction and try to broadcast.
	txHex, err := finalTx.ToHex()
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}

	fmt.Println()
	fmt.Println(txHex)

	return nil
}