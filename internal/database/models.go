package database

import (
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User 代理用户模型
type User struct {
	gorm.Model
	Hash        string    `gorm:"uniqueIndex;not null" json:"hash"`   // Trojan 密码的 SHA224 哈希值
	Password    string    `json:"password"`                           // 明文密码（方便管理端查看和分发）
	Quota       int64     `gorm:"default:-1" json:"quota"`            // 流量限额 (字节, -1 为无限)
	Used        int64     `gorm:"default:0" json:"used"`              // 已用总流量
	Upload      int64     `gorm:"default:0" json:"upload"`            // 上传
	Download    int64     `gorm:"default:0" json:"download"`          // 下载
	ExpiryTime  time.Time `json:"expiry_time"`                        // 过期时间
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
	return db, nil
}
