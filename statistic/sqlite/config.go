package sqlite

import (
	"github.com/voidluo/trojan-go/config"
)

type SqliteConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	DbPath    string `json:"db_path" yaml:"db_path"`
	CheckRate int    `json:"check_rate" yaml:"check_rate"`
}

type Config struct {
	Sqlite SqliteConfig `json:"sqlite" yaml:"sqlite"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return &Config{
			Sqlite: SqliteConfig{
				DbPath:    "trojan-go.db",
				CheckRate: 60,
			},
		}
	})
}
