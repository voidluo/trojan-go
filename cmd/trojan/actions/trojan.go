package actions

import (
	"fmt"
	"os/exec"
)

const serviceName = "trojan-go"

func runSystemctl(cmd string) {
	out, err := exec.Command("systemctl", cmd, serviceName).CombinedOutput()
	if err != nil {
		fmt.Printf("\033[31m操作失败: %v\033[0m\n", err)
	}
	fmt.Println(string(out))
}

// TrojanStart 启动服务
func TrojanStart() {
	fmt.Printf("正在启动 %s...\n", serviceName)
	runSystemctl("start")
}

// TrojanStop 停止服务
func TrojanStop() {
	fmt.Printf("正在停止 %s...\n", serviceName)
	runSystemctl("stop")
}

// TrojanRestart 重启服务
func TrojanRestart() {
	fmt.Printf("正在重启 %s...\n", serviceName)
	runSystemctl("restart")
}

// TrojanStatus 查看服务状态
func TrojanStatus() {
	fmt.Printf("[ %s 服务状态 ]\n\n", serviceName)
	out, err := exec.Command("systemctl", "status", serviceName, "--no-pager", "-l").CombinedOutput()
	if err != nil {
		fmt.Printf("状态查询失败: %v\n", err)
	}
	fmt.Println(string(out))
}
