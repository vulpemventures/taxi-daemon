package explorer

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

//Explorer ...
type Explorer struct {
	BaseURL string
}

// Faucet ...
func (e *Explorer) Faucet(address string) (string, error) {
	url := e.BaseURL + "/faucet"
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

//Unspents ...
func (e *Explorer) Unspents(address string) ([]map[string]interface{}, error) {
	getUtxos := func(address string) ([]interface{}, error) {
		url := e.BaseURL + "/address/" + address + "/utxo"
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var respBody interface{}
		err = json.Unmarshal(data, &respBody)
		if err != nil {
			return nil, err
		}

		return respBody.([]interface{}), nil
	}

	utxos := []map[string]interface{}{}
	for len(utxos) <= 0 {
		time.Sleep(1 * time.Second)
		u, err := getUtxos(address)
		if err != nil {
			return nil, err
		}
		for _, unspent := range u {
			utxo := unspent.(map[string]interface{})
			utxos = append(utxos, utxo)
		}
	}

	return utxos, nil
}

//Broadcast ...
func (e *Explorer) Broadcast(txHex string) (string, error) {
	url := e.BaseURL + "/tx"
	resp, err := http.Post(url, "text/plain", strings.NewReader(txHex))
	if err != nil {
		return "", err
	}
	txid, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(txid), nil
}
