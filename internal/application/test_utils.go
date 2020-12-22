package application

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/vulpemventures/go-elements/network"
	"github.com/vulpemventures/liquid-taxi/internal/domain"
	"github.com/vulpemventures/liquid-taxi/internal/infrastructure/storage/inmemory"
)

const (
	mnemonic         = "clown skin ramp arctic diet fun crowd expect pride gift public twice alpha crazy help swamp soccer prepare clown act joy drive year mask"
	explorerEndpoint = "http://localhost:3001"
	superToken       = "2e4c430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c4798bc4f"
	hyperToken       = "6666430667f861d010b6c78ce6a3943d35d84ca77ca1898031b8799c47986666"
)

var (
	regtest             = network.Regtest
	words               = strings.Split(mnemonic, " ")
	defaultSatsPerByte  = decimal.NewFromFloat(0.1)
	doesnotmatterAmount = decimal.NewFromFloat(66.6)
)

// Faucet ...
func faucet(address string, baseURL string) (string, error) {
	url := baseURL + "/faucet"
	payload := map[string]string{"address": address}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
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

	time.Sleep(5 * time.Second)
	return respBody["txId"], nil
}

func newMockedPriceRepository(t *testing.T, assets []string) domain.PriceRepository {
	repo := inmemory.NewInmemoryPriceRepository()
	initialPrices := []domain.Price{}

	for _, a := range assets {
		initialPrices = append(initialPrices, domain.Price{
			AssetHash: a,
			Amount:    doesnotmatterAmount,
		})
	}

	for _, price := range initialPrices {
		err := repo.UpdateOrCreatePrice(context.Background(), price)
		if err != nil {
			t.Fatal(err)
		}
	}

	return repo
}
