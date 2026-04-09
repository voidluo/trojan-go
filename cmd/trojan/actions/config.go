package actions

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ShowConfig 展示当前配置文件内容
func ShowConfig() {
	paths := []string{"config.yaml", "config.yml", "config.json", "/etc/trojan-go/config.yaml", "/etc/trojan-go/config.json"}
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err == nil {
			fmt.Printf("[ 当前配置文件: %s ]\n\n", p)
			fmt.Println(string(data))
			return
		}
	}
	fmt.Println("\033[33m未找到配置文件，请在运行目录或 /etc/trojan-go/ 中放置 config.yaml 或 config.json。\033[0m")
}

// GenerateClientJSON 交互式生成客户端配置 JSON
func GenerateClientJSON() {
	reader := bufio.NewReader(os.Stdin)
	get := func(prompt string) string {
		fmt.Print(prompt)
		s, _ := reader.ReadString('\n')
		return strings.TrimSpace(s)
	}

	serverAddr := get("服务器地址 (域名或IP): ")
	port := get("服务器端口 (默认443): ")
	if port == "" {
		port = "443"
	}
	password := get("密码: ")
	sni := get("SNI/域名 (留空则与服务器地址相同): ")
	if sni == "" {
		sni = serverAddr
	}

	cfg := map[string]any{
		"run_type":    "client",
		"local_addr":  "127.0.0.1",
		"local_port":  1080,
		"remote_addr": serverAddr,
		"remote_port": port,
		"password":    []string{password},
		"ssl": map[string]any{
			"sni":    sni,
			"verify": true,
		},
		"mux": map[string]any{"enabled": true},
		"router": map[string]any{
			"enabled":        true,
			"default_policy": "proxy",
			"bypass": []string{
				"geoip:cn",
				"geoip:private",
				"geosite:cn",
			},
			"block": []string{"geosite:category-ads"},
		},
	}

	data, _ := json.MarshalIndent(cfg, "", "    ")
	outFile := "client.json"
	if err := os.WriteFile(outFile, data, 0644); err != nil {
		fmt.Printf("\033[31m保存失败: %v\033[0m\n", err)
		return
	}
	fmt.Printf("\033[32m✓ 客户端配置已生成: %s\033[0m\n\n", outFile)
	fmt.Println(string(data))
}
