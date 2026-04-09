//go:build server || full || mini
// +build server full mini

package build

import (
	_ "github.com/voidluo/trojan-go/proxy/server"
)
