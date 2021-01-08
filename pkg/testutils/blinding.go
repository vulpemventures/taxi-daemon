package testutils

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/vulpemventures/go-elements/pset"
)

// BlindTransaction is a blinder step for a pset
func BlindTransaction(
	p *pset.Pset,
	inBlindKeys, outBlindKeys [][]byte,
) error {
	outBlindPubKeys := make([][]byte, 0, len(outBlindKeys))
	for _, k := range outBlindKeys {
		_, pubkey := btcec.PrivKeyFromBytes(btcec.S256(), k)
		outBlindPubKeys = append(outBlindPubKeys, pubkey.SerializeCompressed())
	}

	psetBase64, err := p.ToBase64()
	if err != nil {
		return err
	}

	for {
		ptx, _ := pset.NewPsetFromBase64(psetBase64)
		blinder, err := pset.NewBlinder(
			ptx,
			inBlindKeys,
			outBlindPubKeys,
			nil,
			nil,
		)
		if err != nil {
			return err
		}

		for {
			if err := blinder.Blind(); err != nil {
				if err != pset.ErrGenerateSurjectionProof {
					return err
				}
				continue
			}
			break
		}

		if pset.VerifyBlinding(ptx, inBlindKeys, outBlindKeys, nil) {
			*p = *ptx
			break
		}
	}
	return nil
}
