package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

const (
	// OceanWalletAddrKey is the address to connect to the ocean wallet service
	OceanWalletAddrKey = "OCEAN_ADDRESS"
	// LogLevelKey is the level used as logus' logger level
	LogLevelKey = "LOG_LEVEL"
	// DatadirKey is the path of the directory used to store the taxi's database files
	DatadirKey = "DATADIR"
	// PortKey defines the port to run in a standalone context
	PortKey = "PORT"
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
	// ExternalUrlKey is used by the internal grpc gateway to connect to the
	// grpc server when using certificates released by a CA.
	// If not defined the url formed with TAXI_DOMAIN:TAXI_PORT will be used.
	ExternalUrlKey = "EXTERNAL_URL"
	// TaxiSwaggerKey is used to specify the folder containing the json swagger
	// file for taxi interface.
	TaxiSwaggerKey = "SWAGGER_PATH"
	// AdminSwaggerKey is used to specify the folder containing the json swagger
	// file for admin interface.
	AdminSwaggerKey = "ADMIN_SWAGGER_PATH"
)

var (
	vip *viper.Viper

	rxDNSName = regexp.MustCompile(
		`^([a-zA-Z]{1}[a-zA-Z0-9-]{0,62}){1}(\.[a-zA-Z0-9]{1}[a-zA-Z0-9-]{0,62})*[\.]?$`,
	)
	rxIP = regexp.MustCompile(
		`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])$`,
	)
)

func init() {
	vip = viper.New()
	vip.SetEnvPrefix("TAXI")
	vip.AutomaticEnv()

	vip.SetDefault(PortKey, 8000)
	vip.SetDefault(LogLevelKey, log.InfoLevel)
	vip.SetDefault(DatadirKey, btcutil.AppDataDir("taxid", false))

	if err := validate(); err != nil {
		log.WithError(err).Fatal("invalid config")
	}

	if err := initDatadir(); err != nil {
		log.WithError(err).Fatal("config: error while creating datadir")
	}
}

// GetString returns the value associated with the key as string
func GetString(key string) string {
	return vip.GetString(key)
}

// GetInt returns the value associated with the key as Int
func GetInt(key string) int {
	return vip.GetInt(key)
}

func validate() error {
	oceanWalletAddr := vip.GetString(OceanWalletAddrKey)
	if len(oceanWalletAddr) == 0 {
		return fmt.Errorf("missing ocean wallet address")
	}

	port := vip.GetInt(PortKey)
	if err := validatePort(port); err != nil {
		return fmt.Errorf(
			"port %d not available: %s", port, err,
		)
	}

	taxiTLSKeyPathSet := vip.GetString(TLSKeyPathKey) != ""
	taxiTLSCertPathSet := vip.GetString(TLSCertPathKey) != ""
	if taxiTLSKeyPathSet != taxiTLSCertPathSet {
		return fmt.Errorf(
			"TLS termination for taxi server with local files requires both "+
				"%s and %s when enabled", TLSKeyPathKey, TLSCertPathKey,
		)
	}

	taxiDomainSet := vip.GetString(DomainKey) != ""
	taxiEmailSet := vip.GetString(EmailKey) != ""
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

	if url := vip.GetString(ExternalUrlKey); url != "" {
		if err := validateURL(url); err != nil {
			return fmt.Errorf("invalid external url: %s", err)
		}
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

func validateURL(urlStr string) error {
	validateIp := func(str string) bool {
		return net.ParseIP(str) != nil && rxIP.MatchString(str)
	}
	validateDomain := func(str string) bool {
		if str == "" || len(strings.Replace(str, ".", "", -1)) > 255 {
			// Domain must not be longer than 255 chars.
			return false
		}
		return rxDNSName.MatchString(str)
	}
	parts := strings.Split(urlStr, ":")
	if len(parts) != 2 {
		return fmt.Errorf("url must be in the form 'host:port'")
	}
	host, port := parts[0], parts[1]
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("port must be a positive number")
	}

	if !validateDomain(host) && !validateIp(host) {
		return fmt.Errorf("host must be either a valid ip or domain")
	}
	return nil
}

func initDatadir() error {
	datadir := vip.GetString(DatadirKey)
	return makeDirectoryIfNotExists(datadir)
}

func makeDirectoryIfNotExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, os.ModeDir|0755)
	}
	return nil
}
