package config

import (
	"fmt"
	"net"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vulpemventures/go-elements/network"
)

const (
	// ConfigPathKey defines the config path
	ConfigPathKey = "CONFIG_PATH"
	//ExplorerEndpointKey defines the explorer to fetch data
	ExplorerEndpointKey = "EXPLORER_ENDPOINT"
	// TaxiPortKey defines the port to run in a standalone context
	TaxiPortKey = "PORT"
	// AdminPortKey serves the port for admin server
	AdminPortKey = "ADMIN_PORT"
	// NetworkKey defines the network
	NetworkKey = "NETWORK"
	// MnemonicKey defines the private key used by the taxi
	MnemonicKey = "MNEMONIC"
	// FragmentAmountKey defines the amount provided by taxi for a topup
	FragmentAmountKey = "FRAGMENT_AMOUNT"
	// LogLevelKey is the level used as logus' logger level
	LogLevelKey = "LOG_LEVEL"
	// AssetHashKey is a temporary var for test
	AssetHashKey = "ASSET_HASH"
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("TAXI")
	vip.AutomaticEnv()

	vip.SetDefault(NetworkKey, "regtest")
	vip.SetDefault(ExplorerEndpointKey, "https://nigiri.network/liquid/api")
	// vip.SetDefault(ConfigPathKey, "./taxi.config.json")
	vip.SetDefault(TaxiPortKey, 8080)
	vip.SetDefault(AdminPortKey, 8081)
	vip.SetDefault(MnemonicKey, "")
	vip.SetDefault(LogLevelKey, 4)
	vip.SetDefault(FragmentAmountKey, 1000)

	validate()

	log.SetLevel(log.Level(vip.GetInt(LogLevelKey)))
}

func validate() {
	if GetMnemonic() == "" {
		log.Fatal("mnemonic is an empty string")
	}

	if GetNetwork() == nil {
		log.Fatal("network does not exist")
	}

	if GetFragmentAmount() < 1 {
		log.Fatal("fragment amount can't be lower than 1 satoshi")
	}

	p := GetTaxiPort()
	pAdmin := GetAdminPort()

	if p == pAdmin {
		log.Fatal("ports can't be equal")
	}

	if err := validatePort(p); err != nil {
		log.Fatal("port ", p, " is not available")
	}

	if err := validatePort(pAdmin); err != nil {
		log.Fatal("port ", p, " is not available")
	}
}

func validatePort(port int) error {
	ln, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		return err
	}

	err = ln.Close()
	if err != nil {
		log.Fatal("Can't close: ", err)
	}
	return nil
}

// TODO refacto ASSETHASH key

// GetAssetHash returns config's assetHash
func GetAssetHash() string {
	return vip.GetString(AssetHashKey)
}

// SetAssetHash sets hash as config's assetHash
func SetAssetHash(hash string) {
	vip.Set(AssetHashKey, hash)
}

// GetExplorerEndpoint returns the explorer endpoint as string
func GetExplorerEndpoint() string {
	return vip.GetString(ExplorerEndpointKey)
}

// GetMnemonic returns the Mnemonic of taxi as string
func GetMnemonic() string {
	return vip.GetString(MnemonicKey)
}

// GetFragmentAmount returns the value of fragment amount as integer
func GetFragmentAmount() int {
	return vip.GetInt(FragmentAmountKey)
}

// GetTaxiPort returns the port of the taxi server
func GetTaxiPort() int {
	return vip.GetInt(TaxiPortKey)
}

// GetAdminPort returns the port of the admin server
func GetAdminPort() int {
	return vip.GetInt(AdminPortKey)
}

// GetNetwork returns a pointer to the configurated network
func GetNetwork() *network.Network {
	switch vip.GetString(NetworkKey) {
	case "regtest":
		return &network.Regtest
	case "liquid":
		return &network.Liquid
	default:
		return nil
	}
}
