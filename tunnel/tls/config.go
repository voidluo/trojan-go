package tls

import (
	"github.com/voidluo/trojan-go/config"
)

type Config struct {
	RemoteHost string          `json:"remote_addr" yaml:"remote_addr"`
	RemotePort int             `json:"remote_port" yaml:"remote_port"`
	TLS        TLSConfig       `json:"ssl" yaml:"ssl"`
	Websocket  WebsocketConfig `json:"websocket" yaml:"websocket"`
	Admin      AdminConfig     `json:"admin" yaml:"admin"`
}

// AdminConfig 管理面板配置（复用 443 端口和证书，无需额外端口）
type AdminConfig struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	Password string `json:"password" yaml:"password"` // Web 登录密码，默认 trojan@123
	Path     string `json:"path" yaml:"path"`         // 管理面板挂载路径，默认 /
	DbPath   string `json:"db" yaml:"db"`             // Sqlite 数据库路径，默认 /etc/trojan-go/trojan-go.db
	Port     int    `json:"port" yaml:"port"`         // Web 管理面板监听端口，0 为复用 443
}

type WebsocketConfig struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
}

type TLSConfig struct {
	Verify               bool     `json:"verify" yaml:"verify"`
	VerifyHostName       bool     `json:"verify_hostname" yaml:"verify_hostname"`
	CertPath             string   `json:"cert" yaml:"cert"`
	KeyPath              string   `json:"key" yaml:"key"`
	KeyPassword          string   `json:"key_password" yaml:"key_password"`
	Cipher               string   `json:"cipher" yaml:"cipher"`
	PreferServerCipher   bool     `json:"prefer_server_cipher" yaml:"prefer_server_cipher"`
	SNI                  string   `json:"sni" yaml:"sni"`
	HTTPResponseFileName string   `json:"plain_http_response" yaml:"plain_http_response"`
	FallbackHost         string   `json:"fallback_addr" yaml:"fallback_addr"`
	FallbackPort         int      `json:"fallback_port" yaml:"fallback_port"`
	ReuseSession         bool     `json:"reuse_session" yaml:"reuse_session"`
	ALPN                 []string `json:"alpn" yaml:"alpn"`
	Curves               string   `json:"curves" yaml:"curves"`
	Fingerprint          string   `json:"fingerprint" yaml:"fingerprint"`
	KeyLogPath           string   `json:"key_log" yaml:"key_log"`
	CertCheckRate        int      `json:"cert_check_rate" yaml:"cert_check_rate"`
}

func init() {
	config.RegisterConfigCreator(Name, func() any {
		return &Config{
			TLS: TLSConfig{
				Verify:         true,
				VerifyHostName: true,
				Fingerprint:    "",
				ALPN:           []string{"http/1.1"},
			},
			Admin: AdminConfig{
				Password: "trojan@123",
				Path:     "/",
				DbPath:   "/etc/trojan-go/trojan-go.db",
				Port:     0,
			},
		}
	})
}
