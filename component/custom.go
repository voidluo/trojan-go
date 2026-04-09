//go:build custom || full
// +build custom full

package build

import (
	_ "github.com/voidluo/trojan-go/proxy/custom"
	_ "github.com/voidluo/trojan-go/tunnel/adapter"
	_ "github.com/voidluo/trojan-go/tunnel/dokodemo"
	_ "github.com/voidluo/trojan-go/tunnel/freedom"
	_ "github.com/voidluo/trojan-go/tunnel/http"
	_ "github.com/voidluo/trojan-go/tunnel/mux"
	_ "github.com/voidluo/trojan-go/tunnel/router"
	_ "github.com/voidluo/trojan-go/tunnel/shadowsocks"
	_ "github.com/voidluo/trojan-go/tunnel/simplesocks"
	_ "github.com/voidluo/trojan-go/tunnel/socks"
	_ "github.com/voidluo/trojan-go/tunnel/tls"
	_ "github.com/voidluo/trojan-go/tunnel/tproxy"
	_ "github.com/voidluo/trojan-go/tunnel/transport"
	_ "github.com/voidluo/trojan-go/tunnel/trojan"
	_ "github.com/voidluo/trojan-go/tunnel/websocket"
)
