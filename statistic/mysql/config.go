package mysql

import (
	"github.com/voidluo/trojan-go/config"
)

type MySQLConfig struct {
	Enabled    bool   `json:"enabled" yaml:"enabled"`
	ServerHost string `json:"server_addr" yaml:"server_addr"`
	ServerPort int    `json:"server_port" yaml:"server_port"`
	Database   string `json:"database" yaml:"database"`
	Username   string `json:"username" yaml:"username"`
	Password   string `json:"password" yaml:"password"`
	CheckRate  int    `json:"check_rate" yaml:"check_rate"`
}

type Config struct {
	MySQL MySQLConfig `json:"mysql" yaml:"mysql"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return &Config{
			MySQL: MySQLConfig{
				ServerPort: 3306,
				CheckRate:  30,
			},
		}
	})
}
