package service

import "github.com/voidluo/trojan-go/config"

const Name = "API_SERVICE"

type SSLConfig struct {
	Enabled        bool     `json:"enabled" yaml:"enabled"`
	CertPath       string   `json:"cert" yaml:"cert"`
	KeyPath        string   `json:"key" yaml:"key"`
	VerifyClient   bool     `json:"verify_client" yaml:"verify_client"`
	ClientCertPath []string `json:"client_cert" yaml:"client_cert"`
}

type APIConfig struct {
	Enabled bool      `json:"enabled" yaml:"enabled"`
	APIHost string    `json:"api_addr" yaml:"api_addr"`
	APIPort int       `json:"api_port" yaml:"api_port"`
	SSL     SSLConfig `json:"ssl" yaml:"ssl"`
}

type Config struct {
	API APIConfig `json:"api" yaml:"api"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return new(Config)
	})
}
