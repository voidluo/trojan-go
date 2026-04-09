package main

import (
	"fmt"
	"os"

	"github.com/voidluo/trojan-go/cmd/trojan/actions"
	"github.com/voidluo/trojan-go/cmd/trojan/menu"
)

func main() {
	// 允许通过环境变量指定数据库路径
	if dbPath := os.Getenv("TROJAN_DB"); dbPath != "" {
		actions.SetDBPath(dbPath)
	}

	// ─── 二级菜单：trojan 管理 ────────────────────────────
	trojanMenu := &menu.Menu{
		Title: menu.L{"trojan管理", "Trojan Management"},
		Items: []menu.Item{
			{Label: menu.L{"启动服务", "Start Service"}, Action: actions.TrojanStart},
			{Label: menu.L{"停止服务", "Stop Service"}, Action: actions.TrojanStop},
			{Label: menu.L{"重启服务", "Restart Service"}, Action: actions.TrojanRestart},
			{Label: menu.L{"查看状态", "Check Status"}, Action: actions.TrojanStatus},
		},
	}

	// ─── 二级菜单：用户管理 ────────────────────────────────
	userMenu := &menu.Menu{
		Title: menu.L{"用户管理", "User Management"},
		Items: []menu.Item{
			{Label: menu.L{"查看用户列表", "List Users"}, Action: actions.UserList},
			{Label: menu.L{"添加用户", "Add User"}, Action: actions.UserAdd},
			{Label: menu.L{"删除用户", "Delete User"}, Action: actions.UserDelete},
		},
	}

	// ─── 二级菜单：安装管理 ────────────────────────────────
	installMenu := &menu.Menu{
		Title: menu.L{"安装管理", "Installation Management"},
		Items: []menu.Item{
			{Label: menu.L{"申请SSL证书", "Apply SSL Certificate"}, Action: actions.ApplyCert},
			{Label: menu.L{"首次部署 (交互式)", "First-time Deployment (Interactive)"}, Action: actions.FirstTimeDeploy},
		},
	}

	// ─── 二级菜单：web 管理 ────────────────────────────────
	webMenu := &menu.Menu{
		Title: menu.L{"web管理", "Web Management"},
		Items: []menu.Item{
			{Label: menu.L{"管理后台说明", "Admin Panel Instructions"}, Action: func() {
				if menu.CurrentLang == menu.CN {
					fmt.Println("\033[36mWeb 管理后台已内置于 trojan-go 主程序中。\033[0m")
					fmt.Println("1. 访问端口: 443 (HTTPS)")
					fmt.Println("2. 访问地址: https://您的域名/")
					fmt.Println("3. 默认密码: trojan@123")
					fmt.Println("\n\033[33m建议：请优先通过配置文件开启 admin 块，并重启服务。\033[0m")
				} else {
					fmt.Println("\033[36mWeb admin panel is built-in to trojan-go.\033[0m")
					fmt.Println("1. Port: 443 (HTTPS)")
					fmt.Println("2. URL: https://your-domain.com/")
					fmt.Println("3. Default Pwd: trojan@123")
				}
			}},
		},
	}

	// ─── 二级菜单：查看配置 ────────────────────────────────
	configMenu := &menu.Menu{
		Title: menu.L{"查看配置", "View Configuration"},
		Items: []menu.Item{
			{Label: menu.L{"显示当前配置", "Show Current Config"}, Action: actions.ShowConfig},
		},
	}

	// ─── 二级菜单：生成json ────────────────────────────────
	jsonMenu := &menu.Menu{
		Title: menu.L{"生成json", "Generate JSON"},
		Items: []menu.Item{
			{Label: menu.L{"生成客户端配置", "Generate Client Config"}, Action: actions.GenerateClientJSON},
		},
	}

	// ─── 根菜单 ───────────────────────────────────────────
	root := &menu.Menu{
		Title: menu.L{"主菜单", "Main Menu"},
		Items: []menu.Item{
			{Label: menu.L{"trojan管理", "Trojan Management"}, Sub: trojanMenu},
			{Label: menu.L{"用户管理", "User Management"}, Sub: userMenu},
			{Label: menu.L{"安装管理", "Installation"}, Sub: installMenu},
			{Label: menu.L{"web管理", "Web Admin"}, Sub: webMenu},
			{Label: menu.L{"查看配置", "Config Info"}, Sub: configMenu},
			{Label: menu.L{"生成json", "Client JSON"}, Sub: jsonMenu},
			{Label: menu.L{"切换语言 / Toggle Language", "Toggle Language / 切换语言"}, Action: func() {
				if menu.CurrentLang == menu.CN {
					menu.CurrentLang = menu.EN
				} else {
					menu.CurrentLang = menu.CN
				}
			}},
		},
	}

	if root.Run() {
		if menu.CurrentLang == menu.CN {
			fmt.Println("\n再见！")
		} else {
			fmt.Println("\nGoodbye!")
		}
	}
}
