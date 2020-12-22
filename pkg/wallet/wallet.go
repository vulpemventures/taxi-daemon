package wallet

import (
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/tyler-smith/go-bip39"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
)

// DerivationPath defines path to derive the master key of the wallet
type DerivationPath []uint32

var (
	// DefaultBaseDerivationPath m/84'/0'
	defaultBaseDerivationPath = DerivationPath{
		hdkeychain.HardenedKeyStart + 84,
		hdkeychain.HardenedKeyStart + 0,
	}
)

// Wallet represents the user, it is able to derive new addresses and sign an input
type Wallet interface {
	DerivePath(path string) (*payment.Payment, error)
	SignInput(args SignInputArgs) (string, error)
	CreatePSET(inputs []explorer.Utxo, outputs []PsetOutputOpts) (string, error)
}

// SignInputArgs is a wrapper for SignInput method arguments
type SignInputArgs struct {
	PsetBase64 string
	Path       string
	InputIndex int
	HashType   txscript.SigHashType
}

type wallet struct {
	mnemonic  []string
	masterKey []byte
	network   network.Network
}

// NewWallet is a factory function for Wallet interface
func NewWallet(words []string, network network.Network) (Wallet, error) {
	seed := generateSeedFromMnemonic(words)
	masterKey, err := generateSigningMasterKey(seed)
	if err != nil {
		return nil, err
	}

	return &wallet{
		mnemonic:  words,
		masterKey: masterKey,
		network:   network,
	}, nil
}

func generateSeedFromMnemonic(mnemonic []string) []byte {
	m := strings.Join(mnemonic, " ")
	return bip39.NewSeed(m, "")
}

func generateSigningMasterKey(seed []byte) ([]byte, error) {
	hdNode, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	for _, step := range defaultBaseDerivationPath {
		hdNode, err = hdNode.Child(step)
		if err != nil {
			return nil, err
		}
	}
	return base58.Decode(hdNode.String()), nil
}

// SignInput lets the user to sign a pset input
func (w *wallet) SignInput(args SignInputArgs) (string, error) {
	privKey, pubKey, err := w.deriveSigningKeyPair(args.Path)
	if err != nil {
		return "", err
	}

	script := payment.FromPublicKey(pubKey, &w.network, nil).Script

	p, err := pset.NewPsetFromBase64(args.PsetBase64)
	if err != nil {
		return "", err
	}

	updater, err := pset.NewUpdater(p)
	if err != nil {
		return "", err
	}

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

	base64, err := p.ToBase64()
	if err != nil {
		return "", err
	}

	return base64, err
}

// DerivePath gets a derivation path and return a payment.Payment pointer
func (w *wallet) DerivePath(path string) (*payment.Payment, error) {
	_, pubkey, err := w.deriveSigningKeyPair(path)
	if err != nil {
		return nil, err
	}

	p2wpkh := payment.FromPublicKey(pubkey, &w.network, nil)
	return p2wpkh, nil
}

func (w *wallet) deriveSigningKeyPair(path string) (
	*btcec.PrivateKey,
	*btcec.PublicKey,
	error,
) {
	hdNode, err := hdkeychain.NewKeyFromString(
		base58.Encode(w.masterKey),
	)
	if err != nil {
		return nil, nil, err
	}

	derivationPath, _ := parseDerivationPath(path)
	for _, step := range derivationPath {
		hdNode, err = hdNode.Child(step)
		if err != nil {
			return nil, nil, err
		}
	}

	privateKey, err := hdNode.ECPrivKey()
	if err != nil {
		return nil, nil, err
	}
	publicKey, err := hdNode.ECPubKey()
	if err != nil {
		return nil, nil, err
	}

	return privateKey, publicKey, nil
}

// PsetOutputOpts is using to create TxOutput
type PsetOutputOpts struct {
	WitnessScript []byte
	Asset         string
	Amount        uint64
}

// CreatePset is an util function used to generate taxi transaction
func (w *wallet) CreatePSET(
	inputs []explorer.Utxo,
	outputs []PsetOutputOpts,
) (string, error) {
	// create taxi tx
	partial, err := pset.New([]*transaction.TxInput{}, []*transaction.TxOutput{}, 2, 0)
	if err != nil {
		return "", err
	}

	updater, err := pset.NewUpdater(partial)
	if err != nil {
		return "", err
	}

	for inputIndex, utxo := range inputs {
		// inputs
		input, prevout, err := utxo.Parse()
		if err != nil {
			return "", err
		}

		updater.AddInput(input)
		err = updater.AddInWitnessUtxo(prevout, inputIndex)
		if err != nil {
			return "", err
		}
	}

	// outputs
	for _, outputOpts := range outputs {
		// 1. payout output
		elementsValue, err := elementsutil.SatoshiToElementsValue(outputOpts.Amount)
		if err != nil {
			return "", err
		}

		out := transaction.NewTxOutput(
			assetHashBytes(outputOpts.Asset),
			elementsValue,
			outputOpts.WitnessScript,
		)

		updater.AddOutput(out)
	}

	base64encoded, err := partial.ToBase64()
	if err != nil {
		return "", nil
	}

	return base64encoded, nil
}

// ParseDerivationPath converts a derivation path string to the
// internal binary representation
func parseDerivationPath(strPath string) (DerivationPath, error) {
	var path DerivationPath

	elems := strings.Split(strPath, "/")
	switch {
	case strPath == "":
		return nil, ErrNullDerivationPath

	case containsEmptyString(elems):
		return nil, ErrMalformedDerivationPath
	case len(elems) < 2:
		return nil, ErrMalformedDerivationPath

	case len(elems) > 1:
		if strings.TrimSpace(elems[0]) == "m" {
			elems = elems[1:]
		}

	default:
		return nil, ErrInvalidDerivationPath
	}

	// all remaining elems are relative, append one by one
	for _, elem := range elems {
		elem = strings.TrimSpace(elem)
		var value uint32

		if strings.HasSuffix(elem, "'") {
			value = hdkeychain.HardenedKeyStart
			elem = strings.TrimSpace(strings.TrimSuffix(elem, "'"))
		}

		// use big int for convertion
		bigval, ok := new(big.Int).SetString(elem, 0)
		if !ok {
			return nil, fmt.Errorf("invalid elem '%s' in path", elem)
		}

		max := math.MaxUint32 - value
		if bigval.Sign() < 0 || bigval.Cmp(big.NewInt(int64(max))) > 0 {
			if value == 0 {
				return nil, fmt.Errorf("elem %v must be in range [0, %d]", bigval, max)
			}
			return nil, fmt.Errorf("elem %v must be in hardened range [0, %d]", bigval, max)
		}
		value += uint32(bigval.Uint64())

		path = append(path, value)
	}

	return path, nil
}

func containsEmptyString(composedPath []string) bool {
	for _, s := range composedPath {
		if s == "" {
			return true
		}
	}
	return false
}

func assetHashBytes(asset string) []byte {
	buf, _ := hex.DecodeString(asset)
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf
}
