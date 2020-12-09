package explorer

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ExplorerService interface {
	GetUnspents(address string) ([]Utxo, error)
	Broadcast(txHex string) (string, error)
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

// GetUnspents returns utxos for a given addresses
func (explorer explorerService) GetUnspents(address string) ([]Utxo, error) {
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

	var utxos []Utxo
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
