package memory

import (
	"github.com/voidluo/trojan-go/config"
)

type Config struct {
	Passwords []string `json:"password" yaml:"password"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return &Config{}
	})
}
