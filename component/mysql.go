//go:build mysql || full || mini
// +build mysql full mini

package build

import (
	_ "github.com/voidluo/trojan-go/statistic/mysql"
)
