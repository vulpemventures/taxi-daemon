package main

import (
	"fmt"
	"log"

	"github.com/leaanthony/clir"
	"github.com/vulpemventures/go-elements/address"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/go-elements/pset"
	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/pkg/explorer"
	"github.com/vulpemventures/liquid-taxi/pkg/partial"
	"github.com/vulpemventures/liquid-taxi/pkg/wallet"
)

const aliceHex = "bfb96a215dfb07d1a193464174b9ea8e91f2a15bba79800dea838add330f6d86"
const bobHex = "1804e76aa3016013bc9969103554668913cf697c03c23aecb28136d0e0ac16f0"

func main() {
	// Create new cli
	cli := clir.NewCli("Taxi CLI", "A command line interface to work with Liquid Taxi", "v0.0.1")
	// Default Flags
	var regtest bool = true
	var explorerURL string = "http://localhost:3001"
	// Flags
	cli.BoolFlag("regtest", "Work with local regtest", &regtest)
	cli.StringFlag("name", "Electrs REST API", &explorerURL)
	// Explorer
	e := &explorer.Explorer{BaseURL: explorerURL}
	// Network
	currentNetwork := network.Liquid
	if regtest {
		currentNetwork = network.Regtest
	}
	// Create subcommand
	createCmd := cli.NewSubCommand("create", "Create an unsigned PSET without LBTC fees")
	var from string
	var to string
	var amount int
	var asset string
	createCmd.StringFlag("from", "(REQUIRED) From address used to retrive utxos and send back the change", &from)
	createCmd.StringFlag("to", "(REQUIRED) Recipient address to send coins", &to)
	createCmd.IntFlag("amount", "(REQUIRED) Amount of coins to send to recipient", &amount)
	createCmd.StringFlag("asset", "(REQUIRED) Recipient address to send coins to", &asset)
	createCmd.Action(func() error {
		if from == "" || to == "" || asset == "" || amount <= 0 {
			return fmt.Errorf("missing required flag")
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
		fmt.Println()
		fmt.Println(b64)

		return nil
	})

	// Run the application
	err := cli.Run()
	if err != nil {
		log.Fatal(err)
	}
}

/*

func old_main() {

	client := taxi.NewTaxiProtobufClient("http://localhost:8080", &http.Client{})
	//Explorer
	e := &explorer.Explorer{BaseURL: "https://nigiri.network/liquid/api"}
	// Alice
	alice, aliceAddress, alicePrivKey := wallet.FromPrivateKey(aliceHex)
	//Bob
	bob, bobAddress, _ := wallet.FromPrivateKey(bobHex)

	reply, err := client.TopupWithAsset(context.Background(), &taxi.TopupWithAssetRequest{
		Unsigned: &taxi.Unsigned{
			Pset:       "cHNldP8BALgCAAAAAAGCCvqwtfgBeYfy16FD/DKV+b5f882CQmV2EZ7F8tASBAAAAAAA/////wIBJbJRBw4pyhkEPPM8zXMk4t2rA+zErgted8T8Dlz2yVoBAAAAAABMS0AAFgAUyCigf1gGgQVLxsLJ8IRbJXvc2GYBJbJRBw4pyhkEPPM8zXMk4t2rA+zErgted8T8Dlz2yVoBAAAAAAWplcAAFgAUnv6bsuwjxJnwKyfqP8iQ9bhkNz8AAAAAAAEBQgElslEHDinKGQQ88zzNcyTi3asD7MSuC153xPwOXPbJWgEAAAAABfXhAAAWABSe/puy7CPEmfArJ+o/yJD1uGQ3PwEDBAEAAAAAAAA=",
			SatPerByte: 0.1,
		},
		AssetHash: "ce091c998b83c78bb71a632313ba3760f1763d9cfcffae02258ffa9865a37bd2",
	})
	if err != nil {
		panic(err.Error())
	}

	println(reply.Order.Partial.Pset)

}
*/
