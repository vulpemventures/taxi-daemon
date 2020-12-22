package application

import (
	"github.com/btcsuite/btcd/txscript"
	log "github.com/sirupsen/logrus"
	"github.com/tdex-network/tdex-daemon/pkg/explorer"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/liquid-taxi/pkg/fragmenter"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

// TaxiFragmenterService is the application service handling splitting, signing and broadcasting of taxi utxos
type TaxiFragmenterService interface {
	Start()
	Stop()
}

type taxiFragmenter struct {
	signerWallet                         wallet.Wallet
	fragmenter                           fragmenter.Service
	derivationPathOfAddressToFragmentate string
	explorer                             explorer.Service
	errorHandler                         func(err error)
}

func (f *taxiFragmenter) Start() {
	go f.fragmenter.Start()
	fragmentPSETChan := f.fragmenter.GetFragmentPSETChan()

	for fragmentPSET := range fragmentPSETChan {
		go f.handleFragmentPSET(fragmentPSET)
	}
}

func (f *taxiFragmenter) Stop() {
	f.fragmenter.Stop()
}

// handleFragmentPSET uses the wallet to sign the pset, then it broadcast it
func (f taxiFragmenter) handleFragmentPSET(fragmentPSET string) {
	signed, err := f.sign(fragmentPSET)
	if err != nil {
		f.errorHandler(err)
		return
	}

	txid, err := f.broadcastPSET(signed)
	if err != nil {
		f.errorHandler(err)
		return
	}

	log.Info("fragment tx broadcasted: ", txid)
}

func (f taxiFragmenter) sign(fragmentPSET string) (string, error) {
	p, err := pset.NewPsetFromBase64(fragmentPSET)
	if err != nil {
		return "", err
	}

	for inputIndex := range p.Inputs {
		signOpts := wallet.SignInputArgs{
			PsetBase64: fragmentPSET,
			Path:       f.derivationPathOfAddressToFragmentate,
			InputIndex: inputIndex,
			HashType:   txscript.SigHashAll,
		}

		fragmentPSET, err = f.signerWallet.SignInput(signOpts)
		if err != nil {
			return "", err
		}
	}

	p, err = pset.NewPsetFromBase64(fragmentPSET)
	if err != nil {
		return "", err
	}

	valid, err := p.ValidateAllSignatures()
	if err != nil {
		return "", err
	}

	if !valid {
		return "", wallet.ErrInvalidSignature
	}

	return fragmentPSET, nil
}

func (f *taxiFragmenter) broadcastPSET(base64 string) (string, error) {
	p, err := pset.NewPsetFromBase64(base64)
	if err != nil {
		return "", err
	}

	err = pset.FinalizeAll(p)
	if err != nil {
		return "", err
	}

	tx, err := pset.Extract(p)
	if err != nil {
		return "", err
	}

	hexString, err := tx.ToHex()
	if err != nil {
		return "", err
	}

	return f.explorer.BroadcastTransaction(hexString)
}
