package proxy

import "github.com/voidluo/trojan-go/config"

type Config struct {
	RunType  string `json:"run_type" yaml:"run_type"`
	LogLevel int    `json:"log_level" yaml:"log_level"`
	LogFile  string `json:"log_file" yaml:"log_file"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return &Config{
			LogLevel: 1,
		}
	})
}
