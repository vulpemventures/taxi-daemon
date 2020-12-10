package explorer

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	explorerEndpoint      = "http://localhost:3001"
	unconfidentialAddress = "ert1qj2cs63xx2e60qst6ddpxarq43yl5udrs676zv7"
)

func TestGetUnspents(t *testing.T) {
	// faucet the address to create unspent
	_, err := faucet(unconfidentialAddress, explorerEndpoint)
	if err != nil {
		t.Error(err)
	}
	time.Sleep(3 * time.Second)

	// test :
	explorerSvc := NewEsplorerService(explorerEndpoint)
	unspents, err := explorerSvc.GetUnspents(unconfidentialAddress)

	assert.Nil(t, err)
	assert.Equal(t, true, len(unspents) > 0)
}

func TestIsConfidential(t *testing.T) {
	tests := []struct {
		utxo     Utxo
		expected bool
	}{
		{
			utxo: Utxo{
				ValueCommitment: "",
				NonceCommitment: "",
				AssetCommitment: "",
				Asset:           "",
				Value:           0,
			},
			expected: false,
		},
		{
			utxo: Utxo{
				ValueCommitment: "notnull",
				NonceCommitment: "notnull",
				AssetCommitment: "notnull",
				Asset:           "",
				Value:           0,
			},
			expected: true,
		},
		{
			utxo: Utxo{
				ValueCommitment: "notnull",
				NonceCommitment: "notnull",
				AssetCommitment: "notnull",
				Asset:           "notnull",
				Value:           10,
			},
			expected: false,
		},
		{
			utxo: Utxo{
				ValueCommitment: "",
				NonceCommitment: "",
				AssetCommitment: "",
				Asset:           "notnull",
				Value:           10000,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.utxo.IsConfidential())
	}
}

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
