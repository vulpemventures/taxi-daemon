package main

import (
	"log"

	"github.com/leaanthony/clir"
)

func main() {
	// Create new cli
	cli := clir.NewCli("Taxi CLI", "A command line interface to work with Liquid Taxi", "v0.0.1")
	// Create subcommand
	createCmd := cli.NewSubCommand("create", "Create an unsigned PSET without LBTC fees")
	// Declaration of flag variables
	var from, to, asset string
	var amount int
	var regtest bool = false
	var explorerURL string = "http://localhost:3001"
	// Required flags
	createCmd.StringFlag("from", "(REQUIRED) From address used to retrive utxos and send back the change", &from)
	createCmd.StringFlag("to", "(REQUIRED) Recipient address to send coins", &to)
	createCmd.StringFlag("asset", "(REQUIRED) Recipient address to send coins to", &asset)
	createCmd.IntFlag("amount", "(REQUIRED) Amount of coins to send to recipient", &amount)
	// Optional Flags
	createCmd.BoolFlag("regtest", "Work with local regtest", &regtest)
	createCmd.StringFlag("explorer", "Electrs REST API", &explorerURL)
	// Action
	createCmd.Action(func() error {
		return createAction(from, to, asset, amount, explorerURL, regtest)
	})

	// Topup subcommand
	topupCommand := cli.NewSubCommand("topup", "Topup given PSET with LBTC fees")
	// Declaration of flag variables
	var psetBase64 string
	var taxiURL string = "http://localhost:8080"
	// Required flags
	topupCommand.StringFlag("pset", "(REQUIRED) Partial Signed Elements Transaction base64 encoded", &psetBase64)
	// Optional Flags
	topupCommand.StringFlag("taxi", "Taxi endpoint API", &taxiURL)
	// Action
	topupCommand.Action(func() error {
		return topupAction(psetBase64, 0.1, taxiURL)
	})

	// Run the application
	err := cli.Run()
	if err != nil {
		log.Fatal(err)
	}
}
