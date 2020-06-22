package config

import (
	"log"

	"github.com/spf13/viper"
)

const (
	// ConfigPathKey defines the config path
	ConfigPathKey = "CONFIG_PATH"
	//ExplorerEndpointKey defines the explorer to fetch data
	ExplorerEndpointKey = "EXPLORER_ENDPOINT"
	// LambdaContextKey defines if running on AWS Lambda + API Gateway or standalone
	LambdaContextKey = "LAMBDA_CONTEXT"
	// ListeningPortKey defines the port to run in a standalone context
	ListeningPortKey = "LISTENING_PORT"
	// NetworkKey defines the network
	NetworkKey = "NETWORK"
	//MongoEndpoint defines the endpoint for the mongo instance
	MongoEndpoint = "MONGO_ENDPOINT"
	// PrivateKeyKey defines the private key used by the taxi
	PrivateKeyKey = "PRIVATE_KEY"
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("TAXI")
	vip.AutomaticEnv()

	vip.SetDefault(NetworkKey, "regtest")
	vip.SetDefault(ExplorerEndpointKey, "https://nigiri.network/liquid/api")
	vip.SetDefault(ConfigPathKey, "taxi.config")
	vip.SetDefault(LambdaContextKey, true)
	vip.SetDefault(ListeningPortKey, 8080)

	vip.SetConfigName(vip.GetString(ConfigPathKey))
	vip.AddConfigPath(".")
	err := vip.ReadInConfig()
	if err != nil {
		log.Println("[INFO] No config found. Skip")
	}

	if vip.GetString(MongoEndpoint) == "" {
		log.Panic("[ERROR] MONGO_ENDPOINT is required")
	}

	if vip.GetString(PrivateKeyKey) == "" {
		log.Panic("[ERROR] PRIVATE_KEY is required")
	}

}

// GetString ...
func GetString(key string) string {
	return vip.GetString(key)
}

// GetInt ...
func GetInt(key string) int {
	return vip.GetInt(key)
}

// GetBool ...
func GetBool(key string) bool {
	return vip.GetBool(key)
}
