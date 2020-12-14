package wallet

import (
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/tyler-smith/go-bip39"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/payment"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
)

type DerivationPath []uint32

var (
	// DefaultBaseDerivationPath m/84'/0'
	defaultBaseDerivationPath = DerivationPath{
		hdkeychain.HardenedKeyStart + 84,
		hdkeychain.HardenedKeyStart + 0,
	}
)

type Wallet interface {
	DerivePath(path string, network *network.Network) (*payment.Payment, error)
	SignInput(args SignInputArgs) error
}

type wallet struct {
	mnemonic  []string
	masterKey []byte
}

func NewWallet(words []string) (Wallet, error) {
	seed := generateSeedFromMnemonic(words)
	masterKey, err := generateSigningMasterKey(seed)
	if err != nil {
		return nil, err
	}

	return &wallet{
		mnemonic:  words,
		masterKey: masterKey,
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

type SignInputArgs struct {
	Updater     *pset.Updater
	UnsignedTx  *transaction.Transaction
	Path        string
	InputIndex  int
	WitnessUtxo *transaction.TxOutput
	HashType    txscript.SigHashType
}

func (w *wallet) SignInput(args SignInputArgs) error {
	privKey, pubKey, err := w.deriveSigningKeyPair(args.Path)
	if err != nil {
		return err
	}

	sigHash := args.UnsignedTx.HashForWitnessV0(
		args.InputIndex,
		args.WitnessUtxo.Script,
		args.WitnessUtxo.Value,
		args.HashType,
	)

	if err := args.Updater.AddInSighashType(args.HashType, args.InputIndex); err != nil {
		return err
	}

	signature, err := privKey.Sign(sigHash[:])
	if err != nil {
		return err
	}

	sigWithHashType := append(signature.Serialize(), byte(args.HashType))

	// sign with the updater
	if _, err := args.Updater.Sign(
		args.InputIndex,
		sigWithHashType,
		pubKey.SerializeCompressed(),
		nil,
		nil,
	); err != nil {
		return err
	}

	return nil
}

func (w *wallet) DerivePath(path string, network *network.Network) (*payment.Payment, error) {
	_, pubkey, err := w.deriveSigningKeyPair(path)
	if err != nil {
		return nil, err
	}

	p2wpkh := payment.FromPublicKey(pubkey, network, nil)
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
