package config

import (
	"github.com/gofiber/fiber/v2/log"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var Env *envConfigs

func init() {
	Env = loadEnvVariables()
}

type envConfigs struct {
	HmacKey      string `mapstructure:"HmacKey"`
	PrivateKey   string `mapstructure:"PrivateKey"`
	IsEncryption string `mapstructure:"IsEncryption"`
}

func loadEnvVariables() (config *envConfigs) {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	viper.SetConfigType("env")

	if err := viper.ReadInConfig(); err != nil {
		if os.Getenv("UNIT_TEST") == "1" {
			log.Warnf("Skipping config file read in test: %v", err)
			return &envConfigs{}
		}
		log.Fatal("Error reading env file", err)
	}
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatal(err)
	}
	config.PrivateKey = strings.ReplaceAll(config.PrivateKey, `\n`, "\n")
	return
}
