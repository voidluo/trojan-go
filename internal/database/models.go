package database

import (
	"strings"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// User 代理用户模型
type User struct {
	ID          uint      `gorm:"primaryKey" json:"id"`               // ID 必须显式标记为 id 供前端调用
	CreatedAt   time.Time `json:"created_at"`
	Username    string    `json:"username"`                           // 用户名/备注，方便管理区别人
	Hash        string    `gorm:"uniqueIndex;not null" json:"hash"`   // Trojan 密码的 SHA224 哈希值
	Password    string    `json:"password"`                           // 明文密码（方便管理端查看和分发）
	Quota       int64     `gorm:"default:-1" json:"quota"`            // 流量限额 (字节, -1 为无限)
	Used        int64     `gorm:"default:0" json:"used"`              // 已用总流量
	Upload      int64     `gorm:"default:0" json:"upload"`            // 上传
	Download    int64     `gorm:"default:0" json:"download"`          // 下载
	ExpiryTime  *time.Time `json:"expiry_time"`                       // 过期时间
	IPLimit     int       `gorm:"default:0" json:"ip_limit"`          // IP 限制数目
	Status      int       `gorm:"default:0" json:"status"`            // 0: 正常, 1: 禁用
}

// Config 全局管理配置模型
type Config struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

// InitDb 初始化数据库
func InitDb(dbPath string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// 自动迁移模型
	err = db.AutoMigrate(&User{}, &Config{})
	if err != nil {
		return nil, err
	}

	// 初始化默认配置
	defaultRules := `  - RULE-SET,applications,DIRECT
  - DOMAIN,clash.razord.top,DIRECT
  - DOMAIN,yacd.haishan.me,DIRECT
  - RULE-SET,private,DIRECT
  - RULE-SET,reject,REJECT
  - RULE-SET,icloud,DIRECT
  - RULE-SET,apple,DIRECT
  - RULE-SET,google,DIRECT
  - RULE-SET,proxy,PROXY
  - RULE-SET,direct,DIRECT
  - RULE-SET,lancidr,DIRECT
  - RULE-SET,cncidr,DIRECT
  - RULE-SET,telegramcidr,PROXY
  - GEOIP,LAN,DIRECT
  - GEOIP,CN,DIRECT
  - MATCH,PROXY`

	defaultProviders := `  reject:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/reject.txt"
    path: ./ruleset/reject.yaml
    interval: 86400

  icloud:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/icloud.txt"
    path: ./ruleset/icloud.yaml
    interval: 86400

  apple:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/apple.txt"
    path: ./ruleset/apple.yaml
    interval: 86400

  google:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/google.txt"
    path: ./ruleset/google.yaml
    interval: 86400

  proxy:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/proxy.txt"
    path: ./ruleset/proxy.yaml
    interval: 86400

  direct:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/direct.txt"
    path: ./ruleset/direct.yaml
    interval: 86400

  private:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/private.txt"
    path: ./ruleset/private.yaml
    interval: 86400

  gfw:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/gfw.txt"
    path: ./ruleset/gfw.yaml
    interval: 86400

  greatfire:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/greatfire.txt"
    path: ./ruleset/greatfire.yaml
    interval: 86400

  tld-not-cn:
    type: http
    behavior: domain
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/tld-not-cn.txt"
    path: ./ruleset/tld-not-cn.yaml
    interval: 86400

  telegramcidr:
    type: http
    behavior: ipcidr
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/telegramcidr.txt"
    path: ./ruleset/telegramcidr.yaml
    interval: 86400

  cncidr:
    type: http
    behavior: ipcidr
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/cncidr.txt"
    path: ./ruleset/cncidr.yaml
    interval: 86400

  lancidr:
    type: http
    behavior: ipcidr
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/lancidr.txt"
    path: ./ruleset/lancidr.yaml
    interval: 86400

  applications:
    type: http
    behavior: classical
    url: "https://cdn.jsdelivr.net/gh/Loyalsoldier/clash-rules@release/applications.txt"
    path: ./ruleset/applications.yaml
    interval: 86400`

	seeds := []Config{
		{Key: "site_title", Value: "Trojan-Go 管理面板"},
		{Key: "clash_rules", Value: defaultRules},
		{Key: "clash_rule_providers", Value: defaultProviders},
		{Key: "traffic_reset_day", Value: "0"},
	}
	for _, seed := range seeds {
		db.FirstOrCreate(&Config{}, seed)
	}

	// 补丁：修正历史存量数据中的错误规则组名称 (静默处理)
	var oldRule Config
	if db.Where("key = ? AND value LIKE ?", "clash_rules", "%MATCH,Trojan%").Limit(1).Find(&oldRule).RowsAffected > 0 {
		newVal := strings.ReplaceAll(oldRule.Value, "MATCH,Trojan", "MATCH,PROXY")
		db.Model(&Config{}).Where("key = ?", "clash_rules").Update("value", newVal)
	}

	return db, nil
}
