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
	// FixedFeeAmountKey defines the amount provided by taxi for a topup
	FixedFeeAmountKey = "FIXED_FEE_AMOUNT"
	// LogLevelKey is the level used as logus' logger level
	LogLevelKey = "LOG_LEVEL"
	// UtxoTTLKey is the time to live for utxos lockers (in seconds)
	UtxoTTLKey = "UTXO_LOCK_IN_SECONDS"
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("TAXI")
	vip.AutomaticEnv()

	vip.SetDefault(NetworkKey, "regtest")
	vip.SetDefault(ExplorerEndpointKey, "https://nigiri.network/liquid/api")
	// vip.SetDefault(ConfigPathKey, "./taxi.config.json")
	vip.SetDefault(TaxiPortKey, 8000)
	vip.SetDefault(AdminPortKey, 7000)
	vip.SetDefault(MnemonicKey, "")
	vip.SetDefault(LogLevelKey, 4)
	vip.SetDefault(FixedFeeAmountKey, 1000)
	vip.SetDefault(UtxoTTLKey, 180)

	validate()

	log.SetLevel(log.Level(vip.GetInt(LogLevelKey)))
}

func validate() {
	if GetString(MnemonicKey) == "" {
		log.Fatal("mnemonic is an empty string")
	}

	if GetNetwork() == nil {
		log.Fatal("network does not exist")
	}

	if GetInt(FixedFeeAmountKey) < 1 {
		log.Fatal("fragment amount can't be lower than 1 satoshi")
	}

	p := GetInt(TaxiPortKey)
	pAdmin := GetInt(AdminPortKey)

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

// GetString returns the value associated with the key as string
func GetString(key string) string {
	return vip.GetString(key)
}

// GetInt returns the value associated with the key as Int
func GetInt(key string) int {
	return vip.GetInt(key)
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
