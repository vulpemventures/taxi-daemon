package config

import (
	"fmt"
	"net"
	"strings"

	"github.com/btcsuite/btcutil"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vulpemventures/go-elements/network"
)

const (
	// ConfigPathKey defines the config path
	ConfigPathKey = "CONFIG_PATH"
	//ExplorerEndpointKey defines the explorer to fetch data
	ExplorerEndpointKey = "EXPLORER_ENDPOINT"
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
	// MinFixedFeeAmount is the minimum value allowed for LBTC fees
	MinFixedFeeAmount = 100
	// MaxFixedFeeAmount is the maximum value allowed for LBTC fees
	MaxFixedFeeAmount = 10000000
	// DatadirKey is the path of the directory used to store the taxi's database files
	DatadirKey = "DATADIR"
	// AdminPortKey serves the port for admin server
	AdminPortKey = "ADMIN_PORT"
	// TaxiPortKey defines the port to run in a standalone context
	TaxiPortKey = "PORT"
	// TLSKeyPathKey is the path to the TLS private key for taxi|admin interface
	TLSKeyPathKey = "TLS_KEY"
	// TLSCertPathKey is the path to the TLS certificate for taxi|admin
	// interface. Mandatory if TAXI_TLS_KEY is given
	TLSCertPathKey = "TLS_CERT"
	// DomainKey is TLD domain to be used for obtaining and renewing the TLS
	// certificate for the taxi|admin interface with CertMagic
	DomainKey = "DOMAIN"
	// EmailKey is the email used for TLS certificates communications.
	// Mandatory if TAXI_DOMAIN is given
	EmailKey = "EMAIL"
	// UtxoCacheIntervalKey is the number of seconds between two getUnspents process in fragments cache
	UtxoCacheIntervalKey = "CACHE_INTERVAL"
)

var (
	vip *viper.Viper
)

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("TAXI")
	vip.AutomaticEnv()

	vip.SetDefault(NetworkKey, "liquid")
	vip.SetDefault(ExplorerEndpointKey, "https://blockstream.info/liquid/api")
	vip.SetDefault(AdminPortKey, 7000)
	vip.SetDefault(TaxiPortKey, 8000)
	vip.SetDefault(MnemonicKey, "")
	vip.SetDefault(LogLevelKey, log.InfoLevel)
	vip.SetDefault(FixedFeeAmountKey, 1000)
	vip.SetDefault(UtxoTTLKey, 180)
	vip.SetDefault(DatadirKey, btcutil.AppDataDir("taxid", false))
	vip.SetDefault(UtxoCacheIntervalKey, 180)

	if err := validate(); err != nil {
		log.WithError(err).Fatal("invalid config")
	}

	log.SetLevel(log.Level(vip.GetInt(LogLevelKey)))
}

func validate() error {
	if GetString(MnemonicKey) == "" {
		return fmt.Errorf("mnemonic is missing")
	}

	if GetNetwork() == nil {
		return fmt.Errorf("network is missing")
	}

	feeAmount := GetInt(FixedFeeAmountKey)
	if feeAmount < MinFixedFeeAmount || feeAmount > MaxFixedFeeAmount {
		return fmt.Errorf("fee amount must be in range [%d, %d]", MinFixedFeeAmount, MaxFixedFeeAmount)
	}

	adminPort := GetInt(AdminPortKey)
	taxiPort := GetInt(TaxiPortKey)

	if adminPort == taxiPort {
		return fmt.Errorf("ports can't be equal")
	}

	if err := validatePort(adminPort); err != nil {
		return fmt.Errorf(
			"port %d not available for admin server: %s", adminPort, err,
		)
	}

	if err := validatePort(taxiPort); err != nil {
		return fmt.Errorf(
			"port %d not available for taxi server: %s", adminPort, err,
		)
	}

	taxiTLSKeyPathSet := GetString(TLSKeyPathKey) != ""
	taxiTLSCertPathSet := GetString(TLSCertPathKey) != ""
	if taxiTLSKeyPathSet != taxiTLSCertPathSet {
		return fmt.Errorf(
			"TLS termination for taxi server with local files requires both "+
				"%s and %s when enabled", TLSKeyPathKey, TLSCertPathKey,
		)
	}

	taxiDomainSet := GetString(DomainKey) != ""
	taxiEmailSet := GetString(EmailKey) != ""
	if taxiDomainSet != taxiEmailSet {
		return fmt.Errorf(
			"TLS termination for taxi server with CertMagic requires both "+
				"%s and %s when enabled", DomainKey, EmailKey,
		)
	}

	if taxiTLSKeyPathSet && taxiTLSKeyPathSet == taxiDomainSet {
		return fmt.Errorf(
			"TLS termination for taxi server requires exactly one between " +
				"CertMagic (email&domain) and local files (key&cert) to be given " +
				"when enabled",
		)
	}

	return nil
}

func validatePort(port int) error {
	ln, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		return err
	}
	ln.Close()
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

// GetMnemonic returns the mnemonic as a list of words
func GetMnemonic() []string {
	mnemonic := GetString(MnemonicKey)
	return strings.Split(mnemonic, " ")
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

// Set modifies the value of the `key`, replace the existing value by `value`
func Set(key string, value interface{}) {
	vip.Set(key, value)
}
