package adapter

import "github.com/voidluo/trojan-go/config"

type Config struct {
	LocalHost string `json:"local_addr" yaml:"local_addr"`
	LocalPort int    `json:"local_port" yaml:"local_port"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return new(Config)
	})
}
