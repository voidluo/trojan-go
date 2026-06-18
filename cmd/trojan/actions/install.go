package actions

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/voidluo/trojan-go/cmd/trojan/menu"
)

// FirstTimeDeploy 首次部署一键化流程
func FirstTimeDeploy() {
	// 动态生成 6 位随机字符组成 websocket 路径
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	randChars := make([]byte, 6)
	for i := range randChars {
		randChars[i] = letters[r.Intn(len(letters))]
	}
	wsPath := "/stream-v2-" + string(randChars)

	if os.Geteuid() != 0 {
		msg := "错误：安装操作需要 sudo 权限！"
		if menu.CurrentLang == menu.EN {
			msg = "Error: Installation requires sudo privileges!"
		}
		fmt.Printf("\033[31m%s\033[0m\n", msg)
		return
	}

	configPath := "/etc/trojan-go/config.yaml"
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("\033[33m检测到配置文件已存在，继续操作将覆盖它。\033[0m")
		ans := getStdin("是否继续？(y/n): ", "Continue? (y/n): ")
		if ans != "y" && ans != "Y" {
			return
		}
	}

	fmt.Println("\n\033[36m=== 步骤 1: 申请 SSL 证书 (ACME) ===\033[0m")
	fmt.Println("请选择证书颁发机构 (CA):")
	fmt.Println("1. Let's Encrypt (默认)")
	fmt.Println("2. BuyPass (Go SSL)")
	caChoice := getStdin("请选择 [1-2]: ", "Select [1-2]: ")
	caURL := "https://acme-v02.api.letsencrypt.org/directory"
	caName := "Let's Encrypt"
	if caChoice == "2" {
		caURL = "https://api.buypass.com/acme/directory"
		caName = "BuyPass"
	}

	fmt.Printf("\n--- 正在通过 %s 进行申请 ---\n", caName)
	domain := getStdin("请输入您的域名 (如 proxy.example.com): ", "Enter your domain (e.g. proxy.example.com): ")
	email := getStdin("请输入邮箱 (用于注册账户): ", "Enter your email (for ACME registration): ")

	if domain == "" || email == "" {
		fmt.Println("域名或邮箱不能为空")
		return
	}

	fmt.Printf("\n正在为 %s 准备证书申请 (HTTP-01 验证)... \n", domain)
	certs, err := obtainCert(domain, email, caURL)
	if err != nil {
		fmt.Printf("\033[31m申请失败: %v\033[0m\n", err)
		return
	}

	// 按照用户需求创建目录：/etc/trojan-go/tls/{domain}/
	tlsDir := filepath.Join("/etc/trojan-go/tls", domain)
	os.MkdirAll(tlsDir, 0755)

	crtPath := filepath.Join(tlsDir, domain+".crt")
	keyPath := filepath.Join(tlsDir, domain+".key")

	os.WriteFile(crtPath, certs.Certificate, 0644)
	os.WriteFile(keyPath, certs.PrivateKey, 0600)
	fmt.Println("\033[32m✓ 证书已保存至:", tlsDir, "\033[0m")

	fmt.Println("\n\033[36m=== 步骤 2: 配置管理参数 ===\033[0m")
	adminUser := getStdin("设置管理面板用户名 (默认 admin): ", "Set admin username (default admin): ")
	if adminUser == "" {
		adminUser = "admin"
	}
	adminPwd := getStdin("设置管理面板密码 (默认 trojan@123): ", "Set admin password (default trojan@123): ")
	if adminPwd == "" {
		adminPwd = "trojan@123"
	}
	
	fmt.Println("\n\033[33m[架构说明]\033[0m")
	fmt.Println("我们推荐使用 80 端口。管理后台将监听 80，而 443 端口的代理")
	fmt.Println("在接收到普通网页请求时会自动回落到 80，实现完美伪装。")
	
	adminPortStr := getStdin("设置管理面板监听端口 (推荐 80): ", "Set admin port (recommended 80): ")
	adminPort := 80 // 默认改为专业模式下的 80
	if adminPortStr != "" {
		fmt.Sscanf(adminPortStr, "%d", &adminPort)
	}

	// 构造双配置方案，解决回落冲突并提供专业注释模板
	
	// 1. 代理专用配置模板 (config.yaml)
	proxyTmpl := `run_type: server
local_addr: 0.0.0.0
local_port: 443

ssl:
  cert: {{.CertPath}}
  key: {{.KeyPath}}
  sni: {{.Domain}}         # 预设的 SNI 域名
  verify: false             # 设置为 false 提高浏览器直接访问的兼容性
  verify_hostname: false    # 设置为 false 解决 SNI 不匹配导致的协议错误

  # 回落机制：将普通网页请求转发至本地 80 端口的管理后台
  fallback_addr: 127.0.0.1
  fallback_port: {{.AdminPort}}
  # 备选首页：当回落目标不可用时展示的备选页面
  plain_http_response: /etc/trojan-go/index.html

mux:                # 开启多路复用，提高小文件传输效率
  enabled: true
websocket:
  enabled: true
  # 【警告】千万不要用 /ws、/trojan 等常见词汇。用随机生成的字符串最安全。
  path: "{{.WSPath}}"
  host: "{{.Domain}}"
admin:
  enabled: true
  username: "{{.User}}"
  password: "{{.Pass}}"
  port: 0
  db: /etc/trojan-go/trojan-go.db
  path: /admin

log:
  level: 1  # 设为 1 或 0 (TRACE)
  access: /etc/trojan-go/log/trojan-go/access.log
  error: /etc/trojan-go/log/trojan-go/error.log
`

	// 2. Web 专用配置模板 (web_config.yaml)
	webTmpl := `# =================================================================
# Trojan-Go Web 管理后台配置文件 (Web / 80)
# =================================================================
# 此进程独立运行于 80 端口，专门处理管理面板逻辑。

run_type: server

admin:
  enabled: true
  username: "{{.User}}"    # 面板登录用户名
  password: "{{.Pass}}"    # 面板登录密码
  port: {{.AdminPort}}     # 服务真正在 80 端口监听
  db: "/etc/trojan-go/trojan-go.db" # 数据库路径
  path: "/"                # 面板挂载根路径
`

	os.MkdirAll("/etc/trojan-go", 0755)
	
	// 数据填充
	dataMap := map[string]interface{}{
		"CertPath":  crtPath,
		"KeyPath":   keyPath,
		"Domain":    domain,
		"AdminPort": adminPort,
		"User":      adminUser,
		"Pass":      adminPwd,
	}

	// 填充替换逻辑（简单替换，不引入 template 包以保持代码精简）
	proxyContent := proxyTmpl
	proxyContent = strings.ReplaceAll(proxyContent, "{{.CertPath}}", crtPath)
	proxyContent = strings.ReplaceAll(proxyContent, "{{.KeyPath}}", keyPath)
	proxyContent = strings.ReplaceAll(proxyContent, "{{.Domain}}", domain)
	proxyContent = strings.ReplaceAll(proxyContent, "{{.AdminPort}}", fmt.Sprintf("%d", adminPort))
	proxyContent = strings.ReplaceAll(proxyContent, "{{.WSPath}}", wsPath)

	webContent := webTmpl
	webContent = strings.ReplaceAll(webContent, "{{.User}}", adminUser)
	webContent = strings.ReplaceAll(webContent, "{{.Pass}}", adminPwd)
	webContent = strings.ReplaceAll(webContent, "{{.AdminPort}}", fmt.Sprintf("%d", adminPort))

	os.WriteFile(configPath, []byte(proxyContent), 0644)
	os.WriteFile("/etc/trojan-go/web_config.yaml", []byte(webContent), 0644)

	_ = dataMap // 占位避错

	// 创建内置首页
	os.WriteFile("/etc/trojan-go/index.html", []byte("<h1>Welcome to Trojan-Go Modern Suite</h1>"), 0644)

	// ─── 自动安装二进制文件 ────────────────────────────
	fmt.Printf("\n正在安装二进制文件到系统路径...\n")
	// 安装 trojan-go
	installedProxy := false
	if _, err := os.Stat("./trojan-go"); err == nil {
		runCmd("cp", "-f", "./trojan-go", "/usr/bin/trojan-go")
		runCmd("chmod", "+x", "/usr/bin/trojan-go")
		installedProxy = true
	} else if _, err := os.Stat("./trojan-go-linux-amd64"); err == nil {
		runCmd("cp", "-f", "./trojan-go-linux-amd64", "/usr/bin/trojan-go")
		runCmd("chmod", "+x", "/usr/bin/trojan-go")
		installedProxy = true
	}
	if installedProxy {
		fmt.Println("✓ 已安装 trojan-go 至 /usr/bin/trojan-go")
	}

	// 安装 trojan 管理工具自身
	installedCli := false
	if _, err := os.Stat("./trojan"); err == nil {
		runCmd("cp", "-f", "./trojan", "/usr/bin/trojan")
		runCmd("chmod", "+x", "/usr/bin/trojan")
		installedCli = true
	} else if _, err := os.Stat("./trojan-linux-amd64"); err == nil {
		runCmd("cp", "-f", "./trojan-linux-amd64", "/usr/bin/trojan")
		runCmd("chmod", "+x", "/usr/bin/trojan")
		installedCli = true
	}
	if installedCli {
		fmt.Println("✓ 已安装 trojan 至 /usr/bin/trojan")
	}

	// 修正配置目录权限，确保证书可读
	runCmd("chmod", "-R", "0755", "/etc/trojan-go")

	// ─── 创建 Systemd 服务文件 ────────────────────────────
	// 1. 代理核心服务 (依赖于 Web 服务提供的回落支持)
	proxySvc := `[Unit]
Description=Trojan-Go Proxy Service
After=network.target trojan-web.service

[Service]
Type=simple
LimitNOFILE=65536
ExecStart=/usr/bin/trojan-go -config /etc/trojan-go/config.yaml
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
`
	// 2. Web 管理服务
	webSvc := `[Unit]
Description=Trojan-Go Web Management Service
After=network.target

[Service]
Type=simple
LimitNOFILE=65536
ExecStart=/usr/bin/trojan-go web -config /etc/trojan-go/web_config.yaml
Restart=on-failure
RestartSec=10s

[Install]
WantedBy=multi-user.target
`
	
	os.WriteFile("/etc/systemd/system/trojan-go.service", []byte(proxySvc), 0644)
	os.WriteFile("/etc/systemd/system/trojan-web.service", []byte(webSvc), 0644)

	fmt.Println("\n\033[36m=== 步骤 3: 启动双服务 (Web & Proxy) ===\033[0m")
	if err := runCmd("systemctl", "daemon-reload"); err == nil {
		// 先启动 Web 服务 (80) 以便 Proxy (443) 验证回落地址
		svcs := []string{"trojan-web", "trojan-go"}
		for _, s := range svcs {
			fmt.Printf(" [!] 正在激活 %s...\n", s)
			runCmd("systemctl", "enable", s)
			if err := runCmd("systemctl", "start", s); err != nil {
				fmt.Printf("\033[31m(!) 警告: %s 启动可能失败，请检查日志。\033[0m\n", s)
			}
		}
		
		fmt.Println("\n正在验证服务存活状态...")
		time.Sleep(2 * time.Second) // 等待服务就绪
		for _, s := range svcs {
			active, _ := exec.Command("systemctl", "is-active", s).Output()
			if string(active) != "active\n" {
				fmt.Printf("\033[31m❌ %s 启动失败! 错误日志如下:\033[0m\n", s)
				out, _ := exec.Command("journalctl", "-u", s, "-n", "10", "--no-pager").CombinedOutput()
				fmt.Println(string(out))
			} else {
				fmt.Printf("\033[32m✅ %s 运行正常\033[0m\n", s)
			}
		}
	}

	fmt.Println("\n\033[32m=== 部署成功 ! ===\033[0m")
	fmt.Printf("1. 双服务已就绪: trojan-go (443) 和 trojan-web (80)\n")
	fmt.Printf("2. 访问方式:\n")
	fmt.Printf("   - https://%s/ (通过 443 回落至后台)\n", domain)
	fmt.Printf("   - http://%s/  (直连 80 端口后台)\n", domain)
	fmt.Printf("3. 检查状态: trojan status\n")
}

// 简单的命令执行工具
func runCmd(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
