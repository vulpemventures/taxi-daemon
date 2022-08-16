package main

import (
	"fmt"
	"log"
	"math"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	rpcServerKey            = "rpcserver"
	outputsKey              = "outputs"
	reqTimeoutKey           = "request-timeout"
	skipUserConfirmationKey = "y"

	defaultRpcServer  = "http://admin1:123@localhost:18884"
	defaultReqTimeout = 15
)

var supportedSchemes = schemes{"http": {}, "https": {}}

func init() {
	pflag.String(
		rpcServerKey, defaultRpcServer,
		"specify the node's full endpoint <proto>://<rpcuser>:<rpcpass>@<host>:<port>",
	)
	pflag.StringToString(
		outputsKey, nil,
		"the list of address:amount to include as outputs of the funding tx",
	)
	pflag.Int(
		reqTimeoutKey, defaultReqTimeout,
		"timeout (in seconds) for requests againtst node",
	)
	pflag.Bool(skipUserConfirmationKey, false, "skip confirmation questions")

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	if err := validate(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	outputs, _ := getOutputs()
	reqTimeout := viper.GetInt(reqTimeoutKey)
	rpcServer := viper.GetString(rpcServerKey)
	wantConfirmation := !viper.GetBool(skipUserConfirmationKey)

	totAmount := 0
	outs := make([]map[string]float64, 0, len(outputs))
	for _, m := range outputs {
		mm := make(map[string]float64)
		for addr, amount := range m {
			mm[addr] = float64(amount) / math.Pow10(8)
			totAmount += amount
		}
		outs = append(outs, mm)
	}

	fmt.Println("--- Fragmenter ---")
	fmt.Printf("Totat amount to spend: %.8f BTC \n\n", float64(totAmount)/math.Pow10(8))

	client, err := newRpcClient(rpcServer, reqTimeout)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("create funding tx...")
	resp, err := client.HandleRequest(
		"createrawtransaction", []interface{}{nil, outs},
	)
	if err != nil {
		log.Fatalf("failed to create funding tx: %s", err)
	}
	unsignedTx := resp.(string)

	resp, err = client.HandleRequest(
		"fundrawtransaction",
		[]interface{}{unsignedTx, map[string]float64{"feeRate": 0.000001}},
	)
	if err != nil {
		log.Fatalf("failed to add inputs to funding tx: %s", err)
	}

	fmt.Println("blind funding tx...")
	unsignedTx = parseTxHex(resp)
	resp, err = client.HandleRequest(
		"blindrawtransaction", []interface{}{unsignedTx},
	)
	if err != nil {
		log.Fatalf("failed to blind funding tx: %s", err)
	}

	fmt.Println("sign funding tx...")

	if wantConfirmation {
		if !askForConfirmation("Are you sure to sign and broadcast the tx?") {
			log.Fatal("Abort")
		}
	}
	blindedTx := resp.(string)
	resp, err = client.HandleRequest(
		"signrawtransactionwithwallet", []interface{}{blindedTx},
	)
	if err != nil {
		log.Fatalf("failed to sign funding tx: %s", err)
	}

	fmt.Println("broadcast funding tx...")
	signedTx := parseTxHex(resp)
	resp, err = client.HandleRequest(
		"sendrawtransaction", []interface{}{signedTx},
	)
	if err != nil {
		log.Fatalf("failed to broadcast funding tx: %s", err)
	}

	txid := resp.(string)
	fmt.Println("broadcasted funding tx:", txid)
}

func validate() error {
	rpcServer := viper.GetString(rpcServerKey)
	if rpcServer == "" {
		return fmt.Errorf("missing rpc server endpoint")
	}
	u, err := url.Parse(rpcServer)
	if err != nil {
		return fmt.Errorf("invalid rpc server endpoint: %s", err)
	}
	if _, ok := supportedSchemes[u.Scheme]; !ok {
		return fmt.Errorf("invalid rpc server endpoint: unsupported proto, must be one of %s", supportedSchemes)
	}
	if u.User.Username() == "" {
		return fmt.Errorf("invalid rpc server endpoint: missing rpc user")
	}
	if _, ok := u.User.Password(); !ok {
		return fmt.Errorf("invalid rpc server endpoint: missing rpc password")
	}
	if u.Host == "" {
		return fmt.Errorf("invalid rpc server endpoint: missing rpc host")
	}

	outputs, err := getOutputs()
	if err != nil {
		return fmt.Errorf("invalid outputs: %s", err)
	}
	if len(outputs) == 0 {
		return fmt.Errorf("missing outputs as list of address:amount")
	}
	return nil
}

func askForConfirmation(message string) bool {
	var answer string
	fmt.Printf("%s [y/n]\n", message)
	fmt.Scanln(&answer)
	answer = strings.ToLower(answer)
	return answer == "y" || answer == "yes"
}

func getOutputs() ([]map[string]int, error) {
	outs := viper.GetStringMap(outputsKey)
	outputs := make([]map[string]int, 0, len(outs))
	for addr, value := range outs {
		amount, err := strconv.Atoi(value.(string))
		if err != nil {
			return nil, fmt.Errorf("invalid amount(s), must be all numbers")
		}
		if amount <= 0 {
			return nil, fmt.Errorf("amounts must be all positive")
		}
		outputs = append(outputs, map[string]int{addr: amount})
	}
	return outputs, nil
}

func parseTxHex(r interface{}) string {
	m := r.(map[string]interface{})
	return m["hex"].(string)
}

type schemes map[string]struct{}

func (s schemes) String() string {
	list := make([]string, 0, len(s))
	for scheme := range s {
		list = append(list, scheme)
	}
	return strings.Join(list, ", ")
}
