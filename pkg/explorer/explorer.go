package explorer

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/vulpemventures/go-elements/transaction"
)

// Explorer provides a set of function used by the taxi service
type Explorer interface {
	GetUnspents(address string) ([]Utxo, error)
	FetchTx(txID string) (string, error)
	Broadcast(txHex string) (string, error)
	FetchPrevout(utxo Utxo) (*transaction.TxOutput, error)
}

type explorer struct {
	endpoint string
	client   explorerHTTPClient
}

// NewEsplorerService returns an explorer interface
func NewEsplorerService(endpoint string) Explorer {
	return &explorer{
		endpoint: endpoint,
		client:   newHTTPClient(),
	}
}

func (explorer explorer) FetchTx(txID string) (string, error) {
	url := fmt.Sprintf("%s/tx/%s/hex", explorer.endpoint, txID)
	status, response, err := explorer.client.get(url)
	if err != nil {
		return "", err
	}

	if status != http.StatusOK {
		return "", ErrStatusIsNotOk{httpCode: status}
	}

	return response, nil
}

// GetUnspents returns utxos for a given addresses
func (explorer explorer) GetUnspents(address string) ([]Utxo, error) {
	url := fmt.Sprintf(
		"%s/address/%s/utxo",
		explorer.endpoint,
		address,
	)

	status, response, err := explorer.client.get(url)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, ErrStatusIsNotOk{httpCode: status}
	}

	var utxos []utxo
	err = json.Unmarshal([]byte(response), &utxos)
	if err != nil {
		return nil, err
	}

	var results []Utxo
	for _, u := range utxos {
		results = append(results, &u)
	}

	return results, nil
}

// Broadcast spreads a transaction hex
func (explorer explorer) Broadcast(txHex string) (string, error) {
	url := fmt.Sprintf("%s/tx", explorer.endpoint)
	status, response, err := explorer.client.post(url, txHex)
	if err != nil {
		return "", err
	}

	if status != http.StatusOK {
		return "", ErrStatusIsNotOk{httpCode: status}
	}

	return response, nil
}

func (explorer explorer) FetchPrevout(utxo Utxo) (*transaction.TxOutput, error) {
	hex, err := explorer.FetchTx(utxo.Key().TxID)
	if err != nil {
		return nil, err
	}

	prevTx, err := transaction.NewTxFromHex(hex)
	if err != nil {
		return nil, err
	}

	return prevTx.Outputs[int(utxo.Key().Vout)], nil
}

// http client designed for explorer

const httpClientTimeout = 3 * time.Second

type explorerHTTPClient interface {
	get(url string) (int, string, error)
	post(url string, bodyString string) (int, string, error)
}

type httpClient struct {
	client *http.Client
}

func newHTTPClient() explorerHTTPClient {
	return &httpClient{
		client: &http.Client{
			Timeout: httpClientTimeout,
		},
	}
}

func (c httpClient) send(request *http.Request) (int, string, error) {
	response, err := c.client.Do(request)
	if err != nil {
		return -2, "", err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return -3, "", err
	}

	return response.StatusCode, string(body), nil
}

// get process a GET request
func (c httpClient) get(url string) (int, string, error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return -1, "", err
	}

	return c.send(request)
}

func (c httpClient) post(url string, bodyString string) (int, string, error) {
	body := strings.NewReader(bodyString)
	request, err := http.NewRequest("POST", url, body)
	if err != nil {
		return -1, "", err
	}

	request.Header.Set("Content-type", "text/plain")

	return c.send(request)
}
