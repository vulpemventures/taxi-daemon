package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type rpcClient struct {
	serverAddr string
	httpClient *http.Client
	timeout    int
}

type rpcRequest struct {
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	Id      int64       `json:"id"`
	JsonRpc string      `json:"jsonrpc"`
}

type rpcResponse struct {
	Id     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Err    interface{}     `json:"error"`
}

func newRpcClient(url string, timeout int) (*rpcClient, error) {
	var httpClient *http.Client

	if withTLS := strings.HasPrefix(url, "https"); withTLS {
		t := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: t}
	} else {
		httpClient = &http.Client{}
	}

	c := &rpcClient{
		serverAddr: url,
		httpClient: httpClient,
		timeout:    timeout,
	}

	if _, err := c.HandleRequest("getblockchaininfo", nil); err != nil {
		return nil, fmt.Errorf("failed to connect to node: %s", err)
	}

	return c, nil
}

func (c *rpcClient) HandleRequest(
	method string, params []interface{},
) (interface{}, error) {
	status, resp, err := c.call(method, params)
	if err != nil {
		return nil, err
	}
	var out interface{}
	err = json.Unmarshal(resp.Result, &out)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%v", resp)
	}

	return out, nil
}

func (c *rpcClient) call(
	method string, params interface{},
) (status int, rr rpcResponse, err error) {
	status = http.StatusInternalServerError
	connectTimer := time.NewTimer(time.Duration(c.timeout) * time.Second)
	rpcR := rpcRequest{method, params, time.Now().UnixNano(), "1.0"}
	payloadBuffer := &bytes.Buffer{}
	if err = json.NewEncoder(payloadBuffer).Encode(rpcR); err != nil {
		return
	}

	req, err := http.NewRequest("POST", c.serverAddr, payloadBuffer)
	if err != nil {
		return
	}
	req.Header.Add("Content-Type", "application/json;charset=utf-8")
	req.Header.Add("Accept", "application/json")

	resp, err := c.doTimeoutRequest(connectTimer, req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		out := map[string]map[string]interface{}{}
		json.Unmarshal(data, &out)
		err = fmt.Errorf("method %s failed with error: %s", method, out["error"]["message"].(string))
		status = resp.StatusCode
		return
	}

	if err = json.Unmarshal(data, &rr); err != nil {
		return
	}

	status = resp.StatusCode
	return
}

func (c *rpcClient) doTimeoutRequest(
	timer *time.Timer, req *http.Request,
) (*http.Response, error) {
	type result struct {
		resp *http.Response
		err  error
	}
	done := make(chan result, 1)
	go func() {
		resp, err := c.httpClient.Do(req)
		done <- result{resp, err}
	}()
	select {
	case r := <-done:
		return r.resp, r.err
	case <-timer.C:
		return nil, fmt.Errorf("timeout reading data from server")
	}
}
