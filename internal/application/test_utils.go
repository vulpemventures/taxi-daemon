package application

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/vulpemventures/go-elements/network"
)

const (
	mnemonic         = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"
	explorerEndpoint = "http://localhost:3001"
	superToken       = "2e4c430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c4798bc4f"
	hyperToken       = "6666430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c47986666"
)

var (
	regtest            = network.Regtest
	words              = strings.Split(mnemonic, " ")
	defaultSatsPerByte = decimal.NewFromFloat(0.1)
)

// Faucet ...
func faucet(address string, baseURL string) (string, error) {
	url := baseURL + "/faucet"
	payload := map[string]string{"address": address}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "appliation/json", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	respBody := map[string]string{}
	err = json.Unmarshal(data, &respBody)
	if err != nil {
		return "", err
	}

	return respBody["txId"], nil
}
