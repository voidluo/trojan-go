package actions

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/voidluo/trojan-go/common"
	"github.com/voidluo/trojan-go/internal/database"
)

var dbPath = "/etc/trojan-go/trojan-go.db"

// SetDBPath 设置数据库路径
func SetDBPath(path string) {
	dbPath = path
}

// UserList 列出所有用户
func UserList() {
	db, err := database.InitDb(dbPath)
	if err != nil {
		fmt.Printf("\033[31m数据库连接失败: %v\033[0m\n", err)
		return
	}
	var users []database.User
	db.Find(&users)
	if len(users) == 0 {
		fmt.Println("暂无用户。")
		return
	}
	fmt.Printf("\033[1m%-5s %-20s %-20s %-15s %-12s\033[0m\n", "ID", "用户名/备注", "密码", "已用流量(MB)", "状态")
	fmt.Println(strings.Repeat("─", 80))
	for _, u := range users {
		status := "\033[32m正常\033[0m"
		if u.Status == 1 {
			status = "\033[31m禁用\033[0m"
		}
		if u.ExpiryTime != nil && !u.ExpiryTime.IsZero() && u.ExpiryTime.Before(time.Now()) {
			status = "\033[33m已过期\033[0m"
		}
		name := u.Username
		if name == "" {
			name = "-"
		}
		fmt.Printf("%-5d %-20s %-20s %-15.2f %s\n", u.ID, name, u.Password, float64(u.Used)/1024/1024, status)
	}
}

// UserAdd 添加用户（交互输入）
func UserAdd() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("请输入新用户的密码 (必填): ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if password == "" {
		fmt.Println("\033[31m密码不能为空！\033[0m")
		return
	}

	fmt.Print("请输入备注/用户名 (必填): ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	if username == "" {
		fmt.Println("\033[31m用户名/备注不能为空！\033[0m")
		return
	}

	db, err := database.InitDb(dbPath)
	if err != nil {
		fmt.Printf("\033[31m数据库连接失败: %v\033[0m\n", err)
		return
	}

	user := database.User{
		Username: username,
		Password: password,
		Hash:     common.SHA224String(password),
	}

	// 预检：检查哈希（密码）是否已存在
	var existing database.User
	if err := db.Where("hash = ?", user.Hash).First(&existing).Error; err == nil {
		fmt.Printf("\033[31m创建用户失败: 此密码已被用户 [%s] 使用！\033[0m\n", existing.Username)
		return
	}

	if err := db.Create(&user).Error; err != nil {
		fmt.Printf("\033[31m创建用户失败: %v\033[0m\n", err)
		return
	}
	fmt.Printf("\033[32m✓ 用户添加成功！\033[0m [备注: %s] [密码: %s]\n", username, password)
}

// UserDelete 删除用户（交互输入 ID）
func UserDelete() {
	fmt.Print("请输入要删除的用户 ID: ")
	var id uint
	fmt.Scan(&id)

	db, err := database.InitDb(dbPath)
	if err != nil {
		fmt.Printf("\033[31m数据库连接失败: %v\033[0m\n", err)
		return
	}

	if err := db.Delete(&database.User{}, id).Error; err != nil {
		fmt.Printf("\033[31m删除失败: %v\033[0m\n", err)
		return
	}
	fmt.Printf("\033[32m✓ 用户 ID=%d 已删除。\033[0m\n", id)
}
