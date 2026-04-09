package menu

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

const (
	keyESC = 0x1b
)

// Language 语言类型
type Language int

const (
	CN Language = iota
	EN
)

var CurrentLang Language = CN

// L 多语言字符串 [CN, EN]
type L []string

func (l L) Get() string {
	if int(CurrentLang) < len(l) {
		return l[CurrentLang]
	}
	if len(l) > 0 {
		return l[0]
	}
	return ""
}

// Item 菜单项
type Item struct {
	Label  L
	Action func() // nil 表示该项为子菜单入口，由 Sub 字段指定
	Sub    *Menu
}

// Menu 菜单结构
type Menu struct {
	Title  L
	Items  []Item
	parent *Menu
}

// Run 进入并运行菜单（阻塞，直到 ESC 退出）
// 返回 true 表示需要退出整个程序
func (m *Menu) Run() bool {
	for {
		clearScreen()
		m.render()

		ch := readKey()

		if ch == keyESC {
			// 若已是根菜单，退出程序
			return m.parent == nil
		}

		idx := int(ch-'1')
		if idx >= 0 && idx < len(m.Items) {
			item := &m.Items[idx]
			if item.Sub != nil {
				item.Sub.parent = m
				if item.Sub.Run() {
					return true
				}
			} else if item.Action != nil {
				clearScreen()
				item.Action()
				backMsg := "按任意键返回..."
				if CurrentLang == EN {
					backMsg = "Press any key to return..."
				}
				fmt.Printf("\n%s", backMsg)
				readKey()
			}
		}
	}
}

// render 渲染菜单显示
func (m *Menu) render() {
	header := "欢迎使用trojan管理程序"
	selectMsg := "请选择: "
	if CurrentLang == EN {
		header = "Welcome to Trojan Management Suite"
		selectMsg = "Please select: "
	}
	fmt.Printf("\033[1;36m%s\033[0m\n\n", header)

	cols := 2
	for i := 0; i < len(m.Items); i += cols {
		line := ""
		for j := 0; j < cols && i+j < len(m.Items); j++ {
			label := fmt.Sprintf("%d.%s", i+j+1, m.Items[i+j].Label.Get())
			if j == 0 {
				line += fmt.Sprintf("%-28s", label)
			} else {
				line += label
			}
		}
		fmt.Println(line)
		fmt.Println()
	}
	fmt.Print(selectMsg)
}

// clearScreen 清空终端屏幕
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// readKey 切换到 raw 模式读取单个按键
func readKey() byte {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		var b [1]byte
		os.Stdin.Read(b[:])
		return b[0]
	}
	defer term.Restore(fd, oldState)

	var b [1]byte
	os.Stdin.Read(b[:])
	return b[0]
}
