# Trojan-Go Server (服务端专用版)

使用 Go 实现的高性能 Trojan 服务端，兼容原版 Trojan 协议及配置文件格式。本版本专注于服务端的高效运行与便捷管理，内置了交互式终端菜单和 Web 管理后台，提供开箱即用的运维管理能力。

## 核心特性

- **交互式终端菜单**：直接运行程序即可进入终端快捷菜单，支持 Trojan 服务的启动/停止/重启/状态查看、用户管理（查看、添加、删除）、SSL 证书申请以及首次部署的交互式指引。
- **内置 Web 管理后台**：内置基于 HTTPS 的 Web 管理后台（由 Trojan-Go 内置的 Web 服务器托管），支持：
  - 管理员账号安全登录与管理（支持终端修改密码）
  - 用户管理：可对用户进行增删改查，支持配置**流量配额 (Quota)**、**到期时间 (Expiry Time)**、**IP 连接限制 (IP Limit)** 等字段。
  - 流量数据统计：实时记录和同步用户的上传/下载流量（正确换算客户端上传与下载方向）。
  - 动态配置：支持在线更新和切换 WebSocket 伪装配置。
- **高性能传输**：
  - 兼容原版 Trojan 协议，支持 UDP 代理。
  - 支持多路复用（smux）降低延迟，提升并发性能。
  - 支持 WebSocket over TLS 传输，以便于 CDN 流量中转。
- **安全保障**：
  - 支持 Shadowsocks AEAD 二次加密，防止流量特征被 CDN 或中间人识别。
  - 支持 TLS 指纹伪造，对抗 GFW 针对 TLS Client Hello 的特征识别。
- **配置友好**：
  - 兼容原版 JSON 配置文件格式，同时支持更简洁易读的 YAML 配置。
  - 内置配置兼容性支持，自动适配新旧属性命名差异（兼容 kebab-case 和 snake_case 命名格式）。

## 使用方法

### 1. 终端交互式管理菜单
直接执行可执行文件，即可进入终端交互式管理菜单，根据提示进行服务管理、证书申请或用户管理：
```shell
./trojan-go
```

### 2. 正常启动服务端
使用配置文件启动 Trojan-Go 服务端：
```shell
./trojan-go -config config.json
# 或者
./trojan-go -config config.yaml
```

### 3. 使用 Docker 部署
```shell
docker run \
    --name trojan-go \
    -d \
    -v /etc/trojan-go/:/etc/trojan-go \
    --network host \
    voidluo/trojan-go
```

## 配置文件示例

### JSON 格式配置 (`server.json`)
```json
{
  "run_type": "server",
  "local_addr": "0.0.0.0",
  "local_port": 443,
  "remote_addr": "127.0.0.1",
  "remote_port": 80,
  "password": ["your_awesome_password"],
  "ssl": {
    "cert": "your_cert.crt",
    "key": "your_key.key",
    "sni": "www.your-awesome-domain-name.com"
  },
  "websocket": {
    "enabled": false,
    "path": "/trojan-go",
    "hostname": "www.your-awesome-domain-name.com"
  }
}
```

### YAML 格式配置 (`server.yaml`)
```yaml
run_type: server
local_addr: 0.0.0.0
local_port: 443
remote_addr: 127.0.0.1
remote_port: 80
password:
  - your_awesome_password
ssl:
  cert: your_cert.crt
  key: your_key.key
  sni: www.your-awesome-domain-name.com
websocket:
  enabled: false
  path: /trojan-go
  hostname: www.your-awesome-domain-name.com
```

## 特性说明

### Web 管理后台
默认集成了 Web 管理面板，可以通过 HTTPS 协议（如 `https://您的域名/`）进行访问。
- 支持管理员凭据的存储与校验。
- 支持实时的用户流量配额、限速、IP 限制及到期时间校验。

### 多路复用 (Mux)
服务端默认支持多路复用。如果客户端启用了多路复用（smux），服务端会自动识别并进行 smux 数据分发，无需额外配置。

### Shadowsocks AEAD 二次加密
可以在配置中启用 Shadowsocks 对流量进行二次混淆加密以实现安全中转：
```yaml
shadowsocks:
  enabled: true
  password: your_shadowsocks_password
```

## 构建说明

请确保本地的 Go 版本 >= 1.20。

使用 Makefile 进行编译：
```shell
make
```

或者使用 Go 自行编译服务端：
```shell
CGO_ENABLED=0 go build -tags "full" -trimpath -ldflags "-s -w"
```

通过设置环境变量可以进行交叉编译，例如：
```shell
# 编译 Windows x64 可执行文件
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -tags "full" -trimpath -ldflags "-s -w"

# 编译 Linux x64 可执行文件
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "full" -trimpath -ldflags "-s -w"
```

## 致谢
- [Trojan](https://github.com/trojan-gfw/trojan)
- [V2Fly](https://github.com/v2fly)
- [utls](https://github.com/refraction-networking/utls)
- [smux](https://github.com/xtaci/smux)
- [gorm](https://github.com/go-gorm/gorm)
