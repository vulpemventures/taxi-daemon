package explorer

import (
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

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
