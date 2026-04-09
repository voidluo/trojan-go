//go:build api || full
// +build api full

package build

import (
	_ "github.com/voidluo/trojan-go/api/control"
	_ "github.com/voidluo/trojan-go/api/service"
)
