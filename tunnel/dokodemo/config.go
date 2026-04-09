package dokodemo

import "github.com/voidluo/trojan-go/config"

type Config struct {
	LocalHost  string `json:"local_addr" yaml:"local_addr"`
	LocalPort  int    `json:"local_port" yaml:"local_port"`
	TargetHost string `json:"target_addr" yaml:"target_addr"`
	TargetPort int    `json:"target_port" yaml:"target_port"`
	UDPTimeout int    `json:"udp_timeout" yaml:"udp_timeout"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return &Config{
			UDPTimeout: 60,
		}
	})
}
