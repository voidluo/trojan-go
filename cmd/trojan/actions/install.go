package actions

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/voidluo/trojan-go/cmd/trojan/menu"
	"gopkg.in/yaml.v3"
)

// FirstTimeDeploy 首次部署一键化流程
func FirstTimeDeploy() {
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

	fmt.Println("\n\033[36m=== 步骤 1: 申请 SSL 证书 ===\033[0m")
	domain := getStdin("请输入您的域名 (如 proxy.example.com): ", "Enter your domain (e.g. proxy.example.com): ")
	email := getStdin("请输入邮箱 (用于注册账户): ", "Enter your email (for ACME registration): ")

	if domain == "" || email == "" {
		fmt.Println("域名或邮箱不能为空")
		return
	}

	fmt.Printf("\n正在为 %s 申请证书 (HTTP-01 验证)... \n", domain)
	certs, err := obtainCert(domain, email)
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
	adminPwd := getStdin("设置管理面板密码 (默认 trojan@123): ", "Set admin password (default trojan@123): ")
	if adminPwd == "" {
		adminPwd = "trojan@123"
	}
	adminPortStr := getStdin("设置管理面板端口 (默认 443 端口复用): ", "Set admin port (default 443 reuse): ")
	adminPort := 443
	if adminPortStr != "" {
		fmt.Sscanf(adminPortStr, "%d", &adminPort)
	}

	// 构造极简 YAML
	// 注意：内部逻辑中 Port=443 或 0 都代表端口复用
	conf := map[string]interface{}{
		"run_type":    "server",
		"local_addr":  "0.0.0.0",
		"local_port":  443,
		"remote_addr": "127.0.0.1",
		"remote_port": 80,
		"ssl": map[string]interface{}{
			"cert": crtPath,
			"key":  keyPath,
			"sni":  domain,
		},
		"admin": map[string]interface{}{
			"enabled":  true,
			"password": adminPwd,
			"port":     adminPort,
		},
	}

	data, _ := yaml.Marshal(conf)
	os.MkdirAll("/etc/trojan-go", 0755)
	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		fmt.Printf("\033[31m保存配置文件失败: %v\033[0m\n", err)
		return
	}

	fmt.Println("\n\033[32m===部署成功 ! ===\033[0m")
	fmt.Printf("1. 配置文件已生成: %s\n", configPath)
	fmt.Printf("2. 您现在可以运行 'systemctl start trojan-go' (如果已配置 systemd)\n")
	fmt.Printf("   或者直接运行 'trojan-go -config %s'\n", configPath)
}
