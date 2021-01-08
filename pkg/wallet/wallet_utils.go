package wallet

import (
	"encoding/hex"
	"strings"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcutil/base58"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/tyler-smith/go-bip39"
	"github.com/vulpemventures/go-elements/elementsutil"
	"github.com/vulpemventures/go-elements/slip77"
)

func generateBlindingMasterKey(seed []byte) ([]byte, error) {
	slip77Node, err := slip77.FromSeed(seed)
	if err != nil {
		return nil, err
	}
	return slip77Node.MasterKey, nil
}

func generateSeedFromMnemonic(mnemonic []string) []byte {
	m := strings.Join(mnemonic, " ")
	return bip39.NewSeed(m, "")
}

func generateSigningMasterKey(seed []byte, rootPath DerivationPath) ([]byte, error) {
	hdNode, err := hdkeychain.NewMaster(seed, &chaincfg.MainNetParams)
	if err != nil {
		return nil, err
	}
	for _, step := range rootPath {
		hdNode, err = hdNode.Child(step)
		if err != nil {
			return nil, err
		}
	}
	return base58.Decode(hdNode.String()), nil
}

func assetHashBytes(asset string) ([]byte, error) {
	buf, err := hex.DecodeString(asset)
	if err != nil {
		return nil, err
	}
	assetBuf := append([]byte{0x01}, elementsutil.ReverseBytes(buf)...)
	return assetBuf, nil
}

func deriveSigningKeyPair(masterKey []byte, path string) (
	*btcec.PrivateKey,
	*btcec.PublicKey,
	error,
) {
	hdNode, err := hdkeychain.NewKeyFromString(
		base58.Encode(masterKey),
	)
	if err != nil {
		return nil, nil, err
	}

	derivationPath, _ := ParseDerivationPath(path)
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
