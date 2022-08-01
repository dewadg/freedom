package config

import "github.com/spf13/viper"

type Config struct {
	ProxyPass struct {
		Target             string
		ExposedAddress     string
		ExposedAddressSSL  string
		AllowedHTTPHeaders []string
	}
	SSL struct {
		Enabled    bool
		Cert       string
		PrivateKey string
	}
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/opt/freedom")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
