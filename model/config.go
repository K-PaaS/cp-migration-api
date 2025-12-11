package model

type Config struct {
	IsEncryption string `yaml:"isEncryption"`
	HmacKey      string `yaml:"hmacKey"`
	PrivateKey   string `yaml:"privateKey"`
}
