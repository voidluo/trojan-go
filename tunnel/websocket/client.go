package websocket

import (
	"context"
	"strings"

	"golang.org/x/net/websocket"

	"github.com/voidluo/trojan-go/common"
	"github.com/voidluo/trojan-go/config"
	"github.com/voidluo/trojan-go/log"
	"github.com/voidluo/trojan-go/tunnel"
)

type Client struct {
	underlay tunnel.Client
	hostname string
	path     string
}

func (c *Client) DialConn(*tunnel.Address, tunnel.Tunnel) (tunnel.Conn, error) {
	conn, err := c.underlay.DialConn(nil, &Tunnel{})
	if err != nil {
		return nil, common.NewError("websocket cannot dial with underlying client").Base(err)
	}
	url := "wss://" + c.hostname + c.path
	origin := "https://" + c.hostname
	wsConfig, err := websocket.NewConfig(url, origin)
	if err != nil {
		return nil, common.NewError("invalid websocket config").Base(err)
	}
	wsConn, err := websocket.NewClient(wsConfig, conn)
	if err != nil {
		return nil, common.NewError("websocket failed to handshake with server").Base(err)
	}
	return &OutboundConn{
		Conn:    wsConn,
		tcpConn: conn,
	}, nil
}

func (c *Client) DialPacket(tunnel.Tunnel) (tunnel.PacketConn, error) {
	return nil, common.NewError("not supported by websocket")
}

func (c *Client) Close() error {
	return c.underlay.Close()
}

func NewClient(ctx context.Context, underlay tunnel.Client) (*Client, error) {
	cfg := config.FromContext(ctx, Name).(*Config)
	if cfg.Websocket.Path != "" && !strings.HasPrefix(cfg.Websocket.Path, "/") {
		oldPath := cfg.Websocket.Path
		cfg.Websocket.Path = "/" + cfg.Websocket.Path
		log.Infof("自动修正 WebSocket 路径: %s -> %s", oldPath, cfg.Websocket.Path)
	}
	if !strings.HasPrefix(cfg.Websocket.Path, "/") {
		return nil, common.NewError("websocket path must start with \"/\"")
	}
	if cfg.Websocket.Host == "" {
		cfg.Websocket.Host = cfg.RemoteHost
		log.Warn("empty websocket hostname")
	}
	log.Debug("websocket client created")
	return &Client{
		hostname: cfg.Websocket.Host,
		path:     cfg.Websocket.Path,
		underlay: underlay,
	}, nil
}
