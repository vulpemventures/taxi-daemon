package application

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

const (
	mnemonic         = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"
	explorerEndpoint = "http://localhost:3001"
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
