package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/btcsuite/btcutil"
	"github.com/vulpemventures/taxi-daemon/api-spec/protobuf/gen/admin"
	"github.com/vulpemventures/taxi-daemon/pkg/taxiclient"
)

var (
	dataDir   = btcutil.AppDataDir("taxi-admin-cli", false)
	statePath = path.Join(dataDir, "state.json")
)

func getState() (map[string]string, error) {
	data := map[string]string{}

	file, err := ioutil.ReadFile(statePath)
	if err != nil {
		return nil, errors.New("get config state error: try 'config init'")
	}
	json.Unmarshal(file, &data)

	return data, nil
}

func setState(data map[string]string) error {
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		os.Mkdir(dataDir, os.ModeDir|0755)
	}

	file, err := os.OpenFile(statePath, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return err
	}

	currentData, err := getState()
	if err != nil {
		fmt.Println(err)
		return err
	}

	mergedData := merge(currentData, data)

	jsonString, err := json.Marshal(mergedData)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(statePath, jsonString, 0755)
	if err != nil {
		return fmt.Errorf("writing to file: %w", err)
	}

	return nil
}

func merge(maps ...map[string]string) map[string]string {
	merge := make(map[string]string, 0)
	for _, m := range maps {
		for k, v := range m {
			merge[k] = v
		}
	}
	return merge
}

func getAdminClient() (admin.AdminClient, func() error, error) {
	state, err := getState()
	if err != nil {
		return nil, nil, err
	}

	address, ok := state["rpcserver"]
	if !ok {
		return nil, nil, errors.New("set rpcserver with `config set rpcserver`")
	}

	return taxiclient.NewAdminClient(address)
}
