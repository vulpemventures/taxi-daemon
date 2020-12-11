package application

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/vulpemventures/go-elements/transaction"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
)

type ExplorerService interface {
	GetUnspents(address string) ([]domain.Utxo, error)
	FetchTx(txId string) (string, error)
	Broadcast(txHex string) (string, error)
	FetchPrevout(utxo domain.Utxo) (*transaction.TxOutput, error)
}

type explorerService struct {
	endpoint string
	client   explorerHttpClient
}

// NewEsplorerService returns an ExplorerService interface
func NewEsplorerService(endpoint string) ExplorerService {
	return &explorerService{
		endpoint: endpoint,
		client:   newHttpClient(),
	}
}

func (explorer explorerService) FetchTx(txId string) (string, error) {
	url := fmt.Sprintf("%s/tx/%s/hex", explorer.endpoint, txId)
	status, response, err := explorer.client.get(url)
	if err != nil {
		return "", err
	}

	if status != http.StatusOK {
		return "", errStatusIsNotOk{httpCode: status}
	}

	return response, nil
}

// GetUnspents returns utxos for a given addresses
func (explorer explorerService) GetUnspents(address string) ([]domain.Utxo, error) {
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
		return nil, errStatusIsNotOk{httpCode: status}
	}

	var utxos []domain.Utxo
	err = json.Unmarshal([]byte(response), &utxos)
	if err != nil {
		return nil, err
	}

	return utxos, nil
}

// Broadcast spreads a transaction hex
func (explorer explorerService) Broadcast(txHex string) (string, error) {
	url := fmt.Sprintf("%s/tx", explorer.endpoint)
	status, response, err := explorer.client.post(url, txHex)
	if err != nil {
		return "", err
	}

	if status != http.StatusOK {
		return "", errStatusIsNotOk{httpCode: status}
	}

	return response, nil
}

func (explorer explorerService) FetchPrevout(utxo domain.Utxo) (*transaction.TxOutput, error) {
	hex, err := explorer.FetchTx(utxo.TxId)
	if err != nil {
		return nil, err
	}

	prevTx, err := transaction.NewTxFromHex(hex)
	if err != nil {
		return nil, err
	}

	return prevTx.Outputs[utxo.Vout], nil
}

// http client designed for explorer

const httpClientTimeout = 3 * time.Second

type explorerHttpClient interface {
	get(url string) (int, string, error)
	post(url string, bodyString string) (int, string, error)
}

type httpClient struct {
	client *http.Client
}

func newHttpClient() explorerHttpClient {
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
