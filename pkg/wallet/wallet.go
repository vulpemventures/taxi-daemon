package wallet

import (
	"errors"

	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/slip77"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
)

// Wallet represents the user, it is able to derive new addresses and sign an input
type Wallet interface {
	DerivePath(path string, confidential bool) (*PaymentInfo, error)
	DeriveBlindingKeyPair(script []byte) (*btcec.PrivateKey, *btcec.PublicKey, error)
	SignInput(args SignInputArgs) (string, error)
	CreatePSET(inputs []explorer.Utxo, outputs []Receiver) (string, error)
	FinalizeAndExtract(psetBase64 string) (string, error)
	GetNetwork() network.Network
}

// PaymentInfo is the return type of DerivePath, it describes a payment (confidential or not)
type PaymentInfo struct {
	DerivationPath     string
	Payment            *payment.Payment
	BlindingPrivateKey *btcec.PrivateKey
}

// Receiver is the abstraction of a transaction output
type Receiver struct {
	Address string
	Asset   string
	Amount  uint64
}

type wallet struct {
	masterKey         []byte
	masterBlindingKey []byte
	network           network.Network
}

// NewWallet is a factory function for Wallet interface
func NewWallet(
	words []string,
	rootPath DerivationPath,
	network network.Network,
) (Wallet, error) {
	seed := generateSeedFromMnemonic(words)
	masterKey, err := generateSigningMasterKey(seed, rootPath)
	if err != nil {
		return nil, err
	}

	masterBlindingKey, err := generateBlindingMasterKey(seed)
	if err != nil {
		return nil, err
	}

	return &wallet{
		masterKey:         masterKey,
		masterBlindingKey: masterBlindingKey,
		network:           network,
	}, nil
}

// SignInputArgs is a wrapper for SignInput method arguments
type SignInputArgs struct {
	PsetBase64 string
	Path       string
	InputIndex int
	HashType   txscript.SigHashType
}

func (a SignInputArgs) validate() error {
	if _, err := pset.NewPsetFromBase64(a.PsetBase64); err != nil {
		return errors.New("invalid partial transaction")
	}

	if _, err := ParseDerivationPath(a.Path); err != nil {
		return err
	}

	if a.InputIndex < 0 {
		return errors.New("invalid input index")
	}

	return nil
}

// FinalizeAndExtract takes a signed PSET as input, finalize it and extract the transaction hex
func (w *wallet) FinalizeAndExtract(base64PSET string) (string, error) {
	finalPSET, err := pset.NewPsetFromBase64(base64PSET)
	if err != nil {
		return "", err
	}

	valid, err := finalPSET.ValidateAllSignatures()
	if err != nil {
		return "", err
	}

	if !valid {
		return "", ErrInvalidSignature
	}

	err = pset.FinalizeAll(finalPSET)
	if err != nil {
		return "", err
	}

	tx, err := pset.Extract(finalPSET)
	if err != nil {
		return "", err
	}

	return tx.ToHex()
}

// GetNetwork is a getter for the wallet's network
func (w *wallet) GetNetwork() network.Network {
	return w.network
}

// SignInput lets the user to sign a pset input
func (w *wallet) SignInput(args SignInputArgs) (string, error) {
	if err := args.validate(); err != nil {
		return "", err
	}

	privKey, pubKey, err := deriveSigningKeyPair(w.masterKey, args.Path)
	if err != nil {
		return "", err
	}

	script := payment.FromPublicKey(pubKey, &w.network, nil).Script

	p, _ := pset.NewPsetFromBase64(args.PsetBase64)
	updater, _ := pset.NewUpdater(p)

	psetInput := p.Inputs[args.InputIndex]

	sigHash := p.UnsignedTx.HashForWitnessV0(
		args.InputIndex,
		script,
		psetInput.WitnessUtxo.Value,
		args.HashType,
	)

	if err := updater.AddInSighashType(args.HashType, args.InputIndex); err != nil {
		return "", err
	}

	signature, err := privKey.Sign(sigHash[:])
	if err != nil {
		return "", err
	}

	sigWithHashType := append(signature.Serialize(), byte(args.HashType))

	// sign with the updater
	if _, err := updater.Sign(
		args.InputIndex,
		sigWithHashType,
		pubKey.SerializeCompressed(),
		nil,
		nil,
	); err != nil {
		return "", err
	}

	// check the signature
	valid, err := p.ValidateInputSignatures(args.InputIndex)
	if err != nil {
		return "", err
	}
	if !valid {
		return "", ErrInvalidSignature
	}

	return p.ToBase64()
}

// DeriveBlindingKeyPair derives the SLIP77 blinding key pair from the provided
// output script
func (w *wallet) DeriveBlindingKeyPair(script []byte) (
	*btcec.PrivateKey,
	*btcec.PublicKey,
	error,
) {
	if len(script) <= 0 {
		return nil, nil, ErrNullOutputScript
	}

	if len(w.masterBlindingKey) <= 0 {
		return nil, nil, ErrNullBlindingMasterKey
	}

	slip77Node, err := slip77.FromMasterKey(w.masterBlindingKey)
	if err != nil {
		return nil, nil, err
	}

	return slip77Node.DeriveKey(script)
}

// DerivePath gets a derivation path and return a payment.Payment pointer
func (w *wallet) DerivePath(path string, confidential bool) (*PaymentInfo, error) {
	_, pubkey, err := deriveSigningKeyPair(w.masterKey, path)
	if err != nil {
		return nil, err
	}

	p2wpkh := payment.FromPublicKey(pubkey, &w.network, nil)

	if !confidential {
		return &PaymentInfo{
			DerivationPath:     path,
			Payment:            p2wpkh,
			BlindingPrivateKey: nil,
		}, nil
	}

	blindingPrivKey, blindingPubKey, err := w.DeriveBlindingKeyPair(p2wpkh.WitnessScript)
	if err != nil {
		return nil, err
	}

	confP2wpkh := payment.FromPublicKey(pubkey, &w.network, blindingPubKey)

	return &PaymentInfo{
		DerivationPath:     path,
		Payment:            confP2wpkh,
		BlindingPrivateKey: blindingPrivKey,
	}, nil
}

// CreatePset is an util function used to generate taxi transaction
func (w *wallet) CreatePSET(
	utxos []explorer.Utxo,
	receivers []Receiver,
) (string, error) {
	inputs := make([]*transaction.TxInput, len(utxos), len(utxos))
	prevouts := make([]*transaction.TxOutput, len(utxos), len(utxos))
	for i, u := range utxos {
		input, prevout, err := u.Parse()
		if err != nil {
			return "", err
		}
		inputs[i], prevouts[i] = input, prevout
	}

	outputs := make([]*transaction.TxOutput, len(receivers), len(receivers))
	for i, r := range receivers {
		output, err := r.toTxOutput()
		if err != nil {
			return "", err
		}
		outputs[i] = output
	}

	partial, err := pset.New(inputs, outputs, 2, 0)
	if err != nil {
		return "", err
	}

	updater, err := pset.NewUpdater(partial)
	if err != nil {
		return "", err
	}

	for i, prevout := range prevouts {
		// error can be ignored here since it's sure that both the prevout and the
		// pset are valid at this point.
		updater.AddInWitnessUtxo(prevout, i)
	}

	return partial.ToBase64()
}

func (r *Receiver) toTxOutput() (*transaction.TxOutput, error) {
	asset, err := assetHashBytes(r.Asset)
	if err != nil {
		return nil, err
	}
	value, err := elementsutil.SatoshiToElementsValue(r.Amount)
	if err != nil {
		return nil, err
	}
	script := make([]byte, 0)
	if len(r.Address) > 0 {
		script, err = address.ToOutputScript(r.Address)
		if err != nil {
			return nil, err
		}
	}

	return transaction.NewTxOutput(asset, value, script), nil
}
