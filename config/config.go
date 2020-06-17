package config

import (
	"log"

	"github.com/spf13/viper"
)

const (
	// NetworkKey defines the network
	NetworkKey = "NETWORK"
	// ConfigPathKey defines the config path
	ConfigPathKey = "CONFIG_PATH"
	// LambdaContextKey defines if running on AWS Lambda + API Gateway or standalone
	LambdaContextKey = "LAMBDA_CONTEXT"
	// ListeningPortKey defines the port to run in a standalone context
	ListeningPortKey = "LISTENING_PORT"
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("TAXI")
	vip.AutomaticEnv()

	vip.SetDefault(NetworkKey, "regtest")
	vip.SetDefault(ConfigPathKey, "taxi.config")
	vip.SetDefault(LambdaContextKey, true)
	vip.SetDefault(ListeningPortKey, 8080)

	vip.SetConfigName(vip.GetString(ConfigPathKey))
	vip.AddConfigPath(".")
	err := vip.ReadInConfig()
	if err != nil {
		log.Println("No config found")
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
