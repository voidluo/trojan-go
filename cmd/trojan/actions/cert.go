package actions

import (
	"fmt"
	"os"

	"github.com/voidluo/trojan-go/cmd/trojan/menu"
)

// ApplyCert 交互式申请 SSL 证书
func ApplyCert() {
	// 基础权限检查 (针对 Linux)
	if os.Geteuid() != 0 {
		msg := "错误：申请证书需要写入 /etc/ 目录，请使用 sudo 运行此程序！"
		if menu.CurrentLang == menu.EN {
			msg = "Error: ACME requires sudo privileges!"
		}
		fmt.Printf("\033[31m%s\033[0m\n", msg)
		return
	}

	title := "=== Let's Encrypt 证书一键申请 ==="
	if menu.CurrentLang == menu.EN {
		title = "=== ACME Certificate Request ==="
	}
	fmt.Printf("\033[36m%s\033[0m\n", title)

	domain := getStdin("请输入域名 (如: example.com): ", "Enter your domain (e.g. example.com): ")
	email := getStdin("请输入邮箱 (用于注册账户): ", "Enter your email (for ACME registration): ")

	if domain == "" || email == "" {
		fmt.Println("域名或邮箱不能为空")
		return
	}

	progress := fmt.Sprintf("\n正在为 %s 申请证书 (HTTP-01 验证)... ", domain)
	note := "[注意] 此操作将临时启动 80 端口验证，请确保 80 端口未被占用且已解析到本机。"
	if menu.CurrentLang == menu.EN {
		progress = fmt.Sprintf("\nRequesting certificate for %s (HTTP-01)... ", domain)
		note = "[Note] Port 80 must be available and mapped to this server."
	}
	fmt.Println(progress)
	fmt.Printf("\033[33m%s\033[0m\n\n", note)

	certs, err := obtainCert(domain, email)
	if err != nil {
		fmt.Printf("\033[31m申请失败: %v\033[0m\n", err)
		return
	}

	// 保存证书 (默认到标准路径，除非是首次部署，这里保留 ApplyCert 的原有简易逻辑)
	os.MkdirAll("/etc/trojan-go", 0755)
	os.WriteFile("/etc/trojan-go/cert.pem", certs.Certificate, 0644)
	os.WriteFile("/etc/trojan-go/key.pem", certs.PrivateKey, 0600)
	
	successMsg := "✓ 证书申请成功！"
	if menu.CurrentLang == menu.EN {
		successMsg = "✓ Certificate obtained successfully!"
	}
	fmt.Printf("\033[32m%s\033[0m\n  证书: /etc/trojan-go/cert.pem\n  私钥: /etc/trojan-go/key.pem\n", successMsg)
}
