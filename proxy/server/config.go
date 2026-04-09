package server

import (
	"github.com/voidluo/trojan-go/config"
	"github.com/voidluo/trojan-go/proxy/client"
)

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return new(client.Config)
	})
}
