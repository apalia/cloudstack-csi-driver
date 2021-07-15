package cloud

import (
	"fmt"

	"gopkg.in/gcfg.v1"
)

// Config holds CloudStack connection configuration.
type Config struct {
	APIURL    string
	APIKey    string
	SecretKey string
	VerifySSL bool
}

// csConfig wraps the config for the CloudStack cloud provider.
// It is taken from https://github.com/apache/cloudstack-kubernetes-provider
// in order to have the same config in cloudstack-kubernetes-provider
// and in this cloudstack-csi-driver
type csConfig struct {
	Global struct {
		APIURL      string `gcfg:"api-url"`
		APIKey      string `gcfg:"api-key"`
		SecretKey   string `gcfg:"secret-key"`
		SSLNoVerify bool   `gcfg:"ssl-no-verify"`
		ProjectID   string `gcfg:"project-id"`
		Zone        string `gcfg:"zone"`
	}
}

// ReadConfig reads a config file with a format defined by CloudStack
// Cloud Controller Manager, and returns a CloudStackConfig.
func ReadConfig(configFilePath string) (*Config, error) {
	cfg := &csConfig{}
	if err := gcfg.ReadFileInto(cfg, configFilePath); err != nil {
		return nil, fmt.Errorf("could not parse CloudStack config: %w", err)
	}

	return &Config{
		APIURL:    cfg.Global.APIURL,
		APIKey:    cfg.Global.APIKey,
		SecretKey: cfg.Global.SecretKey,
		VerifySSL: cfg.Global.SSLNoVerify,
	}, nil
}
