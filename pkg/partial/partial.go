package partial

import (
	"encoding/hex"
	"errors"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/vulpemventures/go-elements/confidential"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
)

// Partial defines a Partial Signed Elements Transaction
type Partial struct {
	Data *pset.Pset
}

// WitnessUtxo defines a witness utxo
type WitnessUtxo struct {
	Asset  []byte
	Value  uint64
	Script []byte
}

//AddInput adds an utxo to a Partial Signed Elements Transaction
func (p *Partial) AddInput(hash string, index uint32, witnessUtxo *WitnessUtxo, nonWitnessUtxo []byte) error {
	updater, err := pset.NewUpdater(p.Data)
	if err != nil {
		return err
	}

	inputHash, _ := hex.DecodeString(hash)
	inputHash = ReverseBytes(inputHash)
	inputIndex := index
	input := transaction.NewTxInput(inputHash, inputIndex)

	updater.AddInput(input)
	lastAdded := len(updater.Upsbt.Inputs) - 1

	err = updater.AddInSighashType(txscript.SigHashAll, lastAdded)
	if err != nil {
		return err
	}

	if witnessUtxo != nil {
		elementsValue, err := confidential.SatoshiToElementsValue(witnessUtxo.Value)
		if err != nil {
			return err
		}
		witnessUtxo := transaction.NewTxOutput(witnessUtxo.Asset, elementsValue[:], witnessUtxo.Script)
		updater.AddInWitnessUtxo(witnessUtxo, lastAdded)
		p.Data = updater.Upsbt
		return nil
	}

	if nonWitnessUtxo != nil {
		return errors.New("Not yet implemented. Only segwit inputs supported")
	}

	return errors.New("Either witnessUtxo or nonWitnessUtxo is missing")
}

// AddOutput adds an output to a Partial Signed Elements Transaction
func (p *Partial) AddOutput(asset []byte, value uint64, script []byte) error {
	updater, err := pset.NewUpdater(p.Data)
	if err != nil {
		return err
	}

	elementsValue, _ := confidential.SatoshiToElementsValue(value)
	output := transaction.NewTxOutput(asset, elementsValue[:], script)

	updater.AddOutput(output)
	p.Data = updater.Upsbt
	return nil
}

//SignWithPrivateKey signs a witness input with a provided EC private key
func (p *Partial) SignWithPrivateKey(index int, pay *payment.Payment, privKey *btcec.PrivateKey) error {
	updater, err := pset.NewUpdater(p.Data)
	if err != nil {
		return err
	}

	if updater.Upsbt.Inputs[index].NonWitnessUtxo != nil {
		return errors.New("Only segwit input supported")
	}
	// legacy Script
	legacyScript := append(append([]byte{0x76, 0xa9, 0x14}, pay.Hash...), []byte{0x88, 0xac}...)
	witValue := updater.Upsbt.Inputs[index].WitnessUtxo.Value
	witHash := updater.Upsbt.UnsignedTx.HashForWitnessV0(index, legacyScript, witValue[:], txscript.SigHashAll)
	sig, err := privKey.Sign(witHash[:])
	if err != nil {
		return err
	}

	sigWithHashType := append(sig.Serialize(), byte(txscript.SigHashAll))
	if err != nil {
		return err
	}

	// Update the pset adding the input signature script and the pubkey.
	_, err = updater.Sign(index, sigWithHashType, pay.PublicKey.SerializeCompressed(), nil, nil)
	if err != nil {
		return err
	}

	p.Data = updater.Upsbt
	return nil
}

// ReverseBytes returns the given byte slice with elems in reverse order.
func ReverseBytes(buf []byte) []byte {
	if len(buf) < 1 {
		return buf
	}
	for i := len(buf)/2 - 1; i >= 0; i-- {
		j := len(buf) - 1 - i
		buf[i], buf[j] = buf[j], buf[i]
	}
	return buf
}
