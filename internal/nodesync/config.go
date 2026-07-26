package nodesync

import "github.com/voidluo/trojan-go/config"

const Name = "NODE_SYNC"

type Config struct {
	Node NodeConfig `json:"node" yaml:"node"`
}

type NodeConfig struct {
	Enabled   bool   `json:"enabled" yaml:"enabled"`
	MasterURL string `json:"master_url" yaml:"master_url"`
	Secret    string `json:"secret" yaml:"secret"`
	// ServerDomain is this worker's own public domain name. It is reported to
	// the master on every sync/heartbeat via the X-Node-Domain header so the
	// master stores a domain (not the request source IP) in
	// nodes.address / nodes.sni.
	//
	// This matters because subscriptions publish nodes.address as the Clash
	// `server` and nodes.sni as the TLS SNI. If those hold a bare IP, the SNI
	// will not match the node's certificate and every client connection to
	// that node fails the TLS handshake.
	ServerDomain string `json:"server_domain" yaml:"server_domain"`
	// NodeLocation is this worker's human-readable region label (e.g. 新加坡).
	// It is reported to the master via the X-Node-Location header and becomes
	// nodes.name, which subscriptions render as `tcp-<name>` / `h-<name>`.
	// Without it the master can only fall back to the domain, producing node
	// names like `tcp-xjp.liteops.top`.
	NodeLocation  string `json:"node_location" yaml:"node_location"`
	SyncInterval  int    `json:"sync_interval" yaml:"sync_interval"`
	TrafficOutbox string `json:"traffic_outbox" yaml:"traffic_outbox"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return &Config{
			Node: NodeConfig{
				SyncInterval: 30,
			},
		}
	})
}
