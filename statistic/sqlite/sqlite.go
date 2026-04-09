package sqlite

import (
	"context"
	"time"

	"github.com/voidluo/trojan-go/common"
	"github.com/voidluo/trojan-go/config"
	"github.com/voidluo/trojan-go/internal/database"
	"github.com/voidluo/trojan-go/log"
	"github.com/voidluo/trojan-go/statistic"
	"github.com/voidluo/trojan-go/statistic/memory"
	"gorm.io/gorm"
)

const Name = "SQLITE"

type Authenticator struct {
	*memory.Authenticator
	db             *gorm.DB
	updateDuration time.Duration
	ctx            context.Context
}

func (a *Authenticator) updater() {
	ticker := time.NewTicker(a.updateDuration)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			log.Info("sqlite authenticator updater exiting...")
			return
		case <-ticker.C:
			// 1. 同步流量到数据库
			for _, user := range a.ListUsers() {
				sent, recv := user.ResetTraffic()
				if sent == 0 && recv == 0 {
					continue
				}
				err := a.db.Model(&database.User{}).
					Where("hash = ?", user.Hash()).
					Updates(map[string]any{
						"used":     gorm.Expr("used + ?", sent+recv),
						"upload":   gorm.Expr("upload + ?", sent),
						"download": gorm.Expr("download + ?", recv),
					}).Error
				if err != nil {
					log.Error(common.NewError("failed to update traffic to sqlite").Base(err))
				}
			}

			// 2. 从数据库同步用户状态到内存
			var dbUsers []database.User
			// 只加载处于启用状态且未过期的用户
			now := time.Now()
			err := a.db.Where("status = ? AND (expiry_time > ? OR expiry_time IS NULL)", 0, now).Find(&dbUsers).Error
			if err != nil {
				log.Error(common.NewError("failed to fetch users from sqlite").Base(err))
				continue
			}

			// 获取内存中当前所有用户
			currentUsers := make(map[string]bool)
			for _, u := range a.ListUsers() {
				currentUsers[u.Hash()] = true
			}

			// 添加或更新
			for _, du := range dbUsers {
				if !currentUsers[du.Hash] {
					a.AddUser(du.Hash)
					log.Info("new user added from sqlite:", du.Hash)
				}
				delete(currentUsers, du.Hash)
			}

			// 删除已经失效的用户
			for hash := range currentUsers {
				a.DelUser(hash)
				log.Info("user removed from memory (disabled/expired in sqlite):", hash)
			}
		}
	}
}

func NewAuthenticator(ctx context.Context) (statistic.Authenticator, error) {
	cfg := config.FromContext(ctx, Name).(*Config)
	db, err := database.InitDb(cfg.Sqlite.DbPath)
	if err != nil {
		return nil, common.NewError("failed to init sqlite db").Base(err)
	}

	memoryAuth, err := memory.NewAuthenticator(ctx)
	if err != nil {
		return nil, err
	}

	a := &Authenticator{
		Authenticator:  memoryAuth.(*memory.Authenticator),
		db:             db,
		updateDuration: time.Duration(cfg.Sqlite.CheckRate) * time.Second,
		ctx:            ctx,
	}

	go a.updater()
	log.Info("sqlite authenticator initialized")
	return a, nil
}

func init() {
	statistic.RegisterAuthenticatorCreator(Name, NewAuthenticator)
}
