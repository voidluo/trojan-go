package actions

import (
	"fmt"
	"os/exec"
)

var services = []string{"trojan-go", "trojan-web"}

func runSystemctl(cmd string, svc string) {
	out, err := exec.Command("systemctl", cmd, svc).CombinedOutput()
	if err != nil {
		fmt.Printf("\033[31m%s 操作失败 (%s): %v\033[0m\n", svc, cmd, err)
	}
	fmt.Println(string(out))
}

// TrojanStart 启动服务
func TrojanStart() {
	for _, svc := range services {
		fmt.Printf("正在启动 %s...\n", svc)
		runSystemctl("start", svc)
	}
	fmt.Println()
	TrojanStatus()
}

// TrojanStop 停止服务
func TrojanStop() {
	for _, svc := range services {
		fmt.Printf("正在停止 %s...\n", svc)
		runSystemctl("stop", svc)
	}
	fmt.Println()
	TrojanStatus()
}

// TrojanRestart 重启服务
func TrojanRestart() {
	for _, svc := range services {
		fmt.Printf("正在重启 %s...\n", svc)
		runSystemctl("restart", svc)
	}
	fmt.Println()
	TrojanStatus()
}

// TrojanStatus 查看服务状态
func TrojanStatus() {
	for _, svc := range services {
		fmt.Printf("[ %s 服务状态 ]\n\n", svc)
		out, _ := exec.Command("systemctl", "status", svc, "--no-pager", "-l").CombinedOutput()
		fmt.Println(string(out))
		fmt.Println("--------------------------------------------------")
	}
}
