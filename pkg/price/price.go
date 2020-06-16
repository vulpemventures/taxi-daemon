package price

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

//SATOSHIS ...
const SATOSHIS = 100000000

// BitcoinToUsdt converts bitcoin in satoshi to USDt (in satoshi)
func BitcoinToUsdt(satoshis uint64) (uint64, error) {
	prices, err := getPriceWithPair("XBTUSDT")
	if err != nil {
		return 0, err
	}

	usdtInSatoshis := prices[0] * SATOSHIS
	OneSatoshiInUsdt := usdtInSatoshis / SATOSHIS
	return satoshis * OneSatoshiInUsdt, nil
}

// UsdtToBitcoin converts USDt in satoshi to Bitcoin (in satoshi)
func UsdtToBitcoin(satoshis uint64) (uint64, error) {
	prices, err := getPriceWithPair("XBTUSDT")
	if err != nil {
		return 0, err
	}
	btcInSatoshis := prices[1] * SATOSHIS
	OneSatoshiInBtc := btcInSatoshis / SATOSHIS
	return satoshis * OneSatoshiInBtc, nil
}

func getPriceWithPair(pair string) ([]uint64, error) {
	url := "https://api.kraken.com/0/public/Ticker?pair=" + pair
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var respBody map[string]interface{}
	err = json.Unmarshal(data, &respBody)
	if err != nil {
		return nil, err
	}

	if len(respBody["error"].([]interface{})) > 0 {
		return nil, err
	}

	result := respBody["result"].(map[string]interface{})
	XBTUSDT := result["XBTUSDT"].(map[string]interface{})
	prices := XBTUSDT["c"].([]interface{})

	var pricesAsNumber []uint64
	for _, price := range prices {
		number, err := strconv.ParseFloat(price.(string), 64)
		if err != nil {
			return nil, err
		}
		pricesAsNumber = append(pricesAsNumber, uint64(number))
	}

	return pricesAsNumber, nil
}
