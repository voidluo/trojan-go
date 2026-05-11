package webserver

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/voidluo/trojan-go/common"
	"github.com/voidluo/trojan-go/internal/database"
	"github.com/voidluo/trojan-go/internal/webui"
	"github.com/voidluo/trojan-go/log"
	"github.com/voidluo/trojan-go/statistic"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

// AdminServer 管理面板服务器，复用 TLS 层已建立的连接
type AdminServer struct {
	db       *gorm.DB
	handler  http.Handler
	connChan chan net.Conn
	done     chan struct{}

	lastActive time.Time // 后端会话活动时间，用于超时注销

	// 初始配置（当数据库未设置时作为回退）
	configUser string
	configPass string

	// 代理核心认证器引用（支持多条代理链路聚合），用于同步面板用户到代理层
	auths []statistic.Authenticator

	// 传输层特性，用于订阅配置生成
	wsEnabled  bool
	wsPath     string
	muxEnabled bool
}

// SetAuth 绑定代理核心认证器，并将数据库中已有的用户同步到认证器中。
// 这是连接 Web 面板（SQLite）与代理核心（内存认证）的关键桥梁。
func (s *AdminServer) SetAuth(auth statistic.Authenticator) {
	s.auths = append(s.auths, auth)
	// 从数据库加载所有用户，注入认证器
	var users []database.User
	// 只加载状态为正常 (Status=0) 且未过期的用户
	now := time.Now()
	s.db.Where("status = ?", 0).Find(&users)

	count := 0
	for _, u := range users {
		// 检查是否过期
		if u.ExpiryTime != nil && !u.ExpiryTime.IsZero() && u.ExpiryTime.Before(now) {
			continue
		}
		if u.Hash != "" {
			if err := auth.AddUser(u.Hash); err == nil {
				count++
			} else {
				log.Debugf("sync user %s to auth: %v (may already exist)", u.Username, err)
			}
		}
	}
	log.Infof("synced %d active users from database to proxy authenticator", count)
}

// New 创建管理面板服务器
func New(db *gorm.DB, username, password, mountPath string, port int, wsEnabled bool, wsPath string, muxEnabled bool) *AdminServer {
	srv := &AdminServer{
		db:         db,
		connChan:   make(chan net.Conn, 64),
		done:       make(chan struct{}),
		configUser: username,
		configPass: password,
		lastActive: time.Now(), // 初始化即激活一次
		wsEnabled:  wsEnabled,
		wsPath:     wsPath,
		muxEnabled: muxEnabled,
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	if mountPath == "" {
		mountPath = "/"
	}



	// 提供 index.html 的处理器
	serveIndex := func(c *gin.Context) {
		data, _ := webui.ReadIndex()
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	}

	// 挂载首页（SPA 模式下，首页直接返回 HTML，路由由前端 #/ 处理）
	if mountPath == "/" {
		r.GET("/", serveIndex)
	} else {
		r.GET(mountPath, serveIndex)
		trimmedPath := strings.TrimSuffix(mountPath, "/")
		if trimmedPath != mountPath && trimmedPath != "" {
			r.GET(trimmedPath, func(c *gin.Context) {
				c.Redirect(http.StatusFound, mountPath)
			})
		}
		// 引导根目录到挂载点
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, mountPath)
		})
	}

	// ─── 统一 API 路由组 ───
	apiGroup := r.Group("/api")

	// ─── 登录（无需鉴权） ──────────────────────────────
	apiGroup.POST("/login", srv.handleLogin)

	// ─── 需要鉴权的管理 API ──────────────────────────────
	auth := apiGroup.Group("/", func(c *gin.Context) {
		// 跳过登录接口
		if c.Request.URL.Path == "/api/login" {
			c.Next()
			return
		}

		// 检查 Token
		ts := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		// 如果 Header 里没有，尝试从 Query 里拿 (例如备份下载)
		if ts == "" {
			ts = c.Query("token")
		}

		token, _ := jwt.Parse(ts, func(t *jwt.Token) (any, error) {
			effPass := password
			var cfgP database.Config
			if srv.db.Where("key = ?", "admin_password").First(&cfgP).Error == nil {
				effPass = cfgP.Value
			}
			return []byte(effPass), nil
		})

		if token == nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "身份凭证无效"})
			c.Abort()
			return
		}

		// 只有在成功解析 Token 后，才检查活动时间超时
		if !srv.lastActive.IsZero() && time.Since(srv.lastActive) > 30*time.Minute {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "会话已过期，请重新登录"})
			c.Abort()
			return
		}

		// 刷新活动时间
		srv.lastActive = time.Now()
		c.Next()
	})
	
	// ─── 系统设置 API ──────────────────────────────
	auth.GET("/settings", srv.handleGetSettings)
	auth.POST("/settings", srv.handleUpdateSettings)
	auth.POST("/settings/admin", srv.handleUpdateAdmin)

	auth.GET("/settings/backup", srv.handleBackup)
	auth.POST("/settings/restore", srv.handleRestore)
	{
		// ─── 系统运维与状态 ──────────────────────────────
		auth.GET("/status", srv.handleGetStatus)
		auth.GET("/server-info", srv.handleGetServerInfo)

		// ─── 公开订阅接口 (无需 JWT) ─────────────────────
		r.GET("/sub", srv.handleSub)

		// ─── 用户管理 ──────────────────────────────
		auth.GET("/users", srv.handleListUsers)
		auth.POST("/users", srv.handleAddUser)
		auth.PUT("/users/:id", srv.handleUpdateUser)
		auth.DELETE("/users/:id", srv.handleDeleteUser)

		// ─── 流量限额 ──────────────────────────────
		auth.POST("/users/:id/quota", srv.handleUpdateQuota)
		auth.DELETE("/users/:id/data", srv.handleClearTraffic)

		// ─── 过期管理 ──────────────────────────────
		auth.POST("/users/:id/expire", srv.handleSetExpire)
		auth.DELETE("/users/:id/expire", srv.handleCancelExpire)

		// ─── 分享链接 ──────────────────────────────
		auth.GET("/users/:id/share", srv.handleShare)

		// ─── 系统运维 ──────────────────────────────
		auth.GET("/logs", srv.handleGetLogs)
		auth.POST("/service", srv.handleServiceControl)
		auth.POST("/restart", srv.handleRestart)
	}

	srv.handler = r
	
	// 流量自动重置任务
	go srv.resetTrafficWorker()
	// 流量实时同步任务 (内存 -> 数据库)
	go srv.trafficSyncWorker()

	// 启动一个标准 HTTP 服务器消费由 TLS 层转入的已解密连接 (443 复用)
	httpSrv := &http.Server{Handler: r}
	go httpSrv.Serve(srv) //nolint:errcheck

	// 如果配置了独立端口，额外启动监听
	if port > 0 {
		go func() {
			log.Infof("admin panel: also listening on http://0.0.0.0:%d", port)
			if err := http.ListenAndServe(fmt.Sprintf(":%d", port), r); err != nil {
				log.Error("admin panel: standalone port listener failed:", err)
			}
		}()
	}

	return srv
}

// ─── API 处理方法 ────────────────────────────────

func (s *AdminServer) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}

	effUser, effPass := s.configUser, s.configPass
	var cfgU, cfgP database.Config
	if s.db.Where("key = ?", "admin_username").First(&cfgU).Error == nil {
		effUser = cfgU.Value
	}
	if s.db.Where("key = ?", "admin_password").First(&cfgP).Error == nil {
		effPass = cfgP.Value
	}

	if req.Username == effUser && req.Password == effPass {
		s.lastActive = time.Now()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"user": req.Username,
			"exp":  time.Now().Add(time.Hour * 24).Unix(),
		})
		t, _ := token.SignedString([]byte(effPass))
		c.JSON(http.StatusOK, gin.H{"token": t})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
	}
}

func (s *AdminServer) handleGetSettings(c *gin.Context) {
	var cfgs []database.Config
	s.db.Find(&cfgs)
	res := make(map[string]string)
	for _, v := range cfgs {
		res[v.Key] = v.Value
	}
	if _, ok := res["admin_username"]; !ok {
		res["admin_username"] = s.configUser
	}
	c.JSON(http.StatusOK, res)
}

func (s *AdminServer) handleUpdateSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for k, v := range req {
		s.db.Save(&database.Config{Key: k, Value: v})
	}
	c.JSON(http.StatusOK, gin.H{"message": "设置已更新"})
}

func (s *AdminServer) handleUpdateAdmin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
		return
	}
	if req.Username != "" {
		s.db.Save(&database.Config{Key: "admin_username", Value: req.Username})
	}
	if req.Password != "" {
		s.db.Save(&database.Config{Key: "admin_password", Value: req.Password})
	}
	c.JSON(http.StatusOK, gin.H{"message": "管理员凭据已更新，请重新登录"})
}

func (s *AdminServer) handleBackup(c *gin.Context) {
	tStr := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
	if tStr == "" {
		tStr = c.Query("token")
	}
	token, _ := jwt.Parse(tStr, func(t *jwt.Token) (any, error) {
		effPass := s.configPass
		var cfgP database.Config
		if s.db.Where("key = ?", "admin_password").First(&cfgP).Error == nil {
			effPass = cfgP.Value
		}
		return []byte(effPass), nil
	})
	if token == nil || !token.Valid {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	var users []database.User
	s.db.Find(&users)
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=users_backup.json")
	c.JSON(http.StatusOK, users)
}

func (s *AdminServer) handleRestore(c *gin.Context) {
	var users []database.User
	if err := c.ShouldBindJSON(&users); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的备份文件"})
		return
	}
	count := 0
	for _, u := range users {
		var exist database.User
		if s.db.Where("hash = ?", u.Hash).First(&exist).Error != nil {
			u.ID = 0
			s.db.Create(&u)
			if len(s.auths) > 0 && u.Status == 0 {
				for _, a := range s.auths {
					a.AddUser(u.Hash)
				}
			}
			count++
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("恢复成功: %d 个用户", count)})
}

func (s *AdminServer) handleGetStatus(c *gin.Context) {
	var count int64
	s.db.Model(&database.User{}).Count(&count)
	c.JSON(http.StatusOK, gin.H{"user_count": count})
}

func (s *AdminServer) handleGetServerInfo(c *gin.Context) {
	cpuPercent, _ := cpu.Percent(0, false)
	vmInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")
	loadInfo, _ := load.Avg()
	hostInfo, _ := host.Info()
	tcpConns, _ := psnet.Connections("tcp")
	udpConns, _ := psnet.Connections("udp")
	c.JSON(http.StatusOK, gin.H{
		"cpu":    cpuPercent,
		"memory": vmInfo,
		"disk":   diskInfo,
		"load":   loadInfo,
		"host":   hostInfo,
		"connections": gin.H{
			"tcp": len(tcpConns),
			"udp": len(udpConns),
		},
	})
}

func (s *AdminServer) handleSub(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.String(http.StatusBadRequest, "Missing token")
		return
	}
	var user database.User
	if err := s.db.Where("hash = ? AND status = 0", token).First(&user).Error; err != nil {
		c.String(http.StatusNotFound, "Invalid token or user disabled")
		return
	}
	if user.ExpiryTime != nil && !user.ExpiryTime.IsZero() && user.ExpiryTime.Before(time.Now()) {
		c.String(http.StatusForbidden, "User expired")
		return
	}
	domain, _, _ := net.SplitHostPort(c.Request.Host)
	if domain == "" {
		domain = c.Request.Host
	}
	c.Header("Content-Type", "text/yaml; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=clash-%s.yaml", user.Username))
	c.String(http.StatusOK, generateClashConfig(s.db, []database.User{user}, domain, 443, s.wsEnabled, s.wsPath, s.muxEnabled))
}

func (s *AdminServer) handleListUsers(c *gin.Context) {
	var users []database.User
	s.db.Find(&users)
	c.JSON(http.StatusOK, users)
}

func (s *AdminServer) handleAddUser(c *gin.Context) {
	var user database.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的参数"})
		return
	}
	if user.Username == "" || user.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}
	user.Hash = common.SHA224String(user.Password)
	var existing database.User
	if err := s.db.Where("hash = ?", user.Hash).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("此密码已被用户 [%s] 使用", existing.Username)})
		return
	}
	if err := s.db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if len(s.auths) > 0 && user.Status == 0 {
		for _, a := range s.auths {
			a.AddUser(user.Hash)
		}
	}
	c.JSON(http.StatusOK, user)
}

func (s *AdminServer) handleUpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		Status     *int   `json:"status"`
		ExpiryDays *int   `json:"expiry_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var user database.User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	oldHash := user.Hash
	updates := map[string]interface{}{}
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Password != "" {
		newHash := common.SHA224String(req.Password)
		var existing database.User
		if err := s.db.Where("hash = ? AND id != ?", newHash, id).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"error": "此密码已被其他用户使用"})
			return
		}
		updates["password"] = req.Password
		updates["hash"] = newHash
		user.Hash = newHash // 用于后续同步
	}
	if req.ExpiryDays != nil {
		if *req.ExpiryDays > 0 {
			expiry := time.Now().AddDate(0, 0, *req.ExpiryDays)
			updates["expiry_time"] = &expiry
		} else {
			updates["expiry_time"] = nil
		}
	}
	s.db.Model(&user).Updates(updates)

	// 同步所有存在的核心
	if len(s.auths) > 0 {
		for _, a := range s.auths {
			a.DelUser(oldHash)
		}
		s.db.First(&user, id) // 获取更新后的状态
		if user.Status == 0 {
			for _, a := range s.auths {
				a.AddUser(user.Hash)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

func (s *AdminServer) handleDeleteUser(c *gin.Context) {
	var user database.User
	if s.db.First(&user, c.Param("id")).Error == nil {
		if len(s.auths) > 0 && user.Hash != "" {
			for _, a := range s.auths {
				a.DelUser(user.Hash)
			}
		}
		s.db.Delete(&user)
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (s *AdminServer) handleUpdateQuota(c *gin.Context) {
	id := c.Param("id")
	var req struct { Quota int64 `json:"quota"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.db.Model(&database.User{}).Where("id = ?", id).Update("quota", req.Quota)
	c.JSON(http.StatusOK, gin.H{"message": "限额设置成功"})
}

func (s *AdminServer) handleClearTraffic(c *gin.Context) {
	s.db.Model(&database.User{}).Where("id = ?", c.Param("id")).Updates(map[string]interface{}{
		"used": 0, "upload": 0, "download": 0,
	})
	c.JSON(http.StatusOK, gin.H{"message": "流量已清空"})
}

func (s *AdminServer) handleSetExpire(c *gin.Context) {
	id := c.Param("id")
	var req struct { Days int `json:"days"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	expiry := time.Now().AddDate(0, 0, req.Days)
	s.db.Model(&database.User{}).Where("id = ?", id).Update("expiry_time", &expiry)
	c.JSON(http.StatusOK, gin.H{"message": "过期时间设置成功", "expiry": expiry})
}

func (s *AdminServer) handleCancelExpire(c *gin.Context) {
	s.db.Model(&database.User{}).Where("id = ?", c.Param("id")).Update("expiry_time", nil)
	c.JSON(http.StatusOK, gin.H{"message": "已取消限期"})
}

func (s *AdminServer) handleShare(c *gin.Context) {
	var user database.User
	if err := s.db.First(&user, c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}
	domain := c.DefaultQuery("domain", c.Request.Host)
	if strings.Contains(domain, ":") {
		domain, _, _ = net.SplitHostPort(domain)
	}
	remark := url.QueryEscape(fmt.Sprintf("%s:%d", domain, 443))
	link := fmt.Sprintf("trojan://%s@%s:%d#%s", user.Password, domain, 443, remark)
	subLink := fmt.Sprintf("https://%s/sub?token=%s", c.Request.Host, user.Hash)
	c.JSON(http.StatusOK, gin.H{"link": link, "sub_link": subLink, "username": user.Username, "password": user.Password})
}

func (s *AdminServer) handleGetLogs(c *gin.Context) {
	lines := c.DefaultQuery("lines", "300")
	level := c.Query("level")
	args := []string{"-u", "trojan-go", "-n", lines, "--no-pager", "-o", "cat"}
	if level != "" && level != "all" {
		p := ""
		switch level {
		case "info": p = "6"
		case "warn": p = "4"
		case "error": p = "3"
		}
		if p != "" { args = append(args, "-p", p) }
	}
	out, err := exec.Command("journalctl", args...).CombinedOutput()
	if err != nil {
		c.String(http.StatusInternalServerError, "获取日志失败: "+err.Error()+"\n"+string(out))
		return
	}
	c.String(http.StatusOK, string(out))
}

func (s *AdminServer) handleServiceControl(c *gin.Context) {
	var req struct { Action string `json:"action"` }
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效动作"})
		return
	}
	validActions := map[string]bool{"start": true, "stop": true, "restart": true, "status": true}
	if !validActions[req.Action] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持该操作"})
		return
	}
	if req.Action == "status" {
		out, _ := exec.Command("systemctl", "is-active", "trojan-go").Output()
		c.JSON(http.StatusOK, gin.H{"status": strings.TrimSpace(string(out))})
		return
	}
	go exec.Command("systemctl", req.Action, "trojan-go").Run()
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("正在尝试 %s 服务...", req.Action)})
}

func (s *AdminServer) handleRestart(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "正在重启..."})
	go func() {
		time.Sleep(200 * time.Millisecond)
		exec.Command("systemctl", "restart", "trojan-go").Run()
	}()
}

// ServeConn 将一条已完成 TLS 握手的连接交给管理面板处理
// 由 TLS acceptLoop 调用
func (s *AdminServer) ServeConn(conn net.Conn) {
	select {
	case s.connChan <- conn:
	case <-s.done:
		conn.Close()
	}
}

// Handler 返回 http.Handler（备用，供测试使用）
func (s *AdminServer) Handler() http.Handler {
	return s.handler
}

// --- 实现 net.Listener 接口，让 http.Server.Serve 能消费 chanListener ---

func (s *AdminServer) Accept() (net.Conn, error) {
	select {
	case conn := <-s.connChan:
		return conn, nil
	case <-s.done:
		return nil, fmt.Errorf("admin server closed")
	}
}

func (s *AdminServer) Close() error {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	return nil
}

func (s *AdminServer) Addr() net.Addr {
	return &net.TCPAddr{} // 不真实监听任何地址
}

func (s *AdminServer) resetTrafficWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			var cfg database.Config
			if s.db.Where("key = ?", "traffic_reset_day").First(&cfg).Error == nil {
				resetDay := 0
				fmt.Sscanf(cfg.Value, "%d", &resetDay)
				if resetDay > 0 && time.Now().Day() == resetDay {
					// 检查本月是否已重置过，防止一天内重置多次
					lastResetMonth := ""
					var cfgLast database.Config
					if s.db.Where("key = ?", "last_reset_month").First(&cfgLast).Error == nil {
						lastResetMonth = cfgLast.Value
					}
					currentMonth := time.Now().Format("2006-01")
					if lastResetMonth != currentMonth {
						log.Infof("自动重置日已达(%d号)，正在清空所有用户流量...", resetDay)
						s.db.Model(&database.User{}).Updates(map[string]interface{}{"used": 0, "upload": 0, "download": 0})
						s.db.Save(&database.Config{Key: "last_reset_month", Value: currentMonth})
					}
				}
			}
		case <-s.done:
			return
		}
	}
}

// trafficSyncWorker 定期将内存中的实时流量统计同步到数据库中
func (s *AdminServer) trafficSyncWorker() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if len(s.auths) == 0 {
				continue
			}
			// 从所有认证器链路收集并合并流量
			trafficMap := make(map[string]struct{ up, down uint64 })
			for _, a := range s.auths {
				for _, st := range a.ListUsers() {
					hash := st.Hash()
					up, down := st.GetTraffic()
					v := trafficMap[hash]
					v.up += up
					v.down += down
					trafficMap[hash] = v
				}
			}

			// 开启事务，合并所有流量更新操作，防止碎片化 I/O 导致 SQLite 锁定
			s.db.Transaction(func(tx *gorm.DB) error {
				for hash, t := range trafficMap {
					tx.Model(&database.User{}).Where("hash = ?", hash).Updates(map[string]interface{}{
						"upload":   int64(t.up),
						"download": int64(t.down),
						"used":     int64(t.up + t.down),
					})
				}
				return nil
			})
		case <-s.done:
			return
		}
	}
}

// ─── Clash 配置生成 ──────────────────────────────
func generateClashConfig(db *gorm.DB, users []database.User, domain string, port int, ws bool, wsPath string, mux bool) string {
	var sb strings.Builder
	// 从数据库获取自定义规则
	var cfgRules, cfgProviders database.Config
	rulesStr := ""
	if db.Where("key = ?", "clash_rules").First(&cfgRules).Error == nil {
		rulesStr = cfgRules.Value
	}
	providersStr := ""
	if db.Where("key = ?", "clash_rule_providers").First(&cfgProviders).Error == nil {
		providersStr = cfgProviders.Value
	}

	// 基础配置与 TUN/DNS
	sb.WriteString("port: 7890\nsocks-port: 7891\nallow-lan: true\nmode: rule\nlog-level: info\n\n")
	
	// DNS 配置 (TUN 模式必备，解决规则集下载域名死循环)
	sb.WriteString("dns:\n  enable: true\n  ipv6: false\n  listen: 0.0.0.0:53\n  enhanced-mode: fake-ip\n  fake-ip-range: 198.18.0.1/16\n  nameserver:\n    - 223.5.5.5\n    - 119.29.29.29\n  fallback:\n    - 8.8.8.8\n    - 1.1.1.1\n    - https://dns.google/dns-query\n  nameserver-policy:\n    'geosite:cn': 223.5.5.5\n    'github.com': 8.8.8.8\n    'cdn.jsdelivr.net': 119.29.29.29\n\n")

	// TUN 配置
	sb.WriteString("tun:\n  enable: true\n  stack: gvisor\n  auto-route: true\n  auto-detect-interface: true\n\n")

	if providersStr != "" {
		if !strings.Contains(providersStr, "rule-providers:") {
			sb.WriteString("rule-providers:\n")
		}
		sb.WriteString(providersStr)
		sb.WriteString("\n\n")
	}

	sb.WriteString("proxies:\n")
	for i, u := range users {
		name := u.Username
		if name == "" {
			name = fmt.Sprintf("Trojan-%d", i+1)
		}
		sb.WriteString(fmt.Sprintf("  - name: \"%s\"\n    type: trojan\n    server: %s\n    port: %d\n    password: %s\n    udp: %t\n    sni: %s\n    skip-cert-verify: true\n    alpn:\n      - http/1.1\n", name, domain, port, u.Password, mux, domain))
		if ws {
			sb.WriteString(fmt.Sprintf("    network: ws\n    ws-opts:\n      path: \"%s\"\n", wsPath))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("proxy-groups:\n  - name: \"PROXY\"\n    type: select\n    proxies:\n")
	for i, u := range users {
		name := u.Username
		if name == "" {
			name = fmt.Sprintf("Trojan-%d", i+1)
		}
		sb.WriteString(fmt.Sprintf("      - \"%s\"\n", name))
	}
	
	if rulesStr != "" {
		sb.WriteString("\n")
		if !strings.Contains(rulesStr, "rules:") {
			sb.WriteString("rules:\n")
		}
		sb.WriteString(rulesStr)
	} else {
		sb.WriteString("\nrules:\n  - GEOIP,CN,DIRECT\n  - MATCH,PROXY")
	}
	
	sb.WriteString("\n")
	return sb.String()
}


// ─── 流量格式化工具函数 ──────────────────────────────
func FormatBytes(bytes uint64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)
	switch {
	case bytes >= TB:
		return strconv.FormatFloat(float64(bytes)/float64(TB), 'f', 2, 64) + " TB"
	case bytes >= GB:
		return strconv.FormatFloat(float64(bytes)/float64(GB), 'f', 2, 64) + " GB"
	case bytes >= MB:
		return strconv.FormatFloat(float64(bytes)/float64(MB), 'f', 2, 64) + " MB"
	case bytes >= KB:
		return strconv.FormatFloat(float64(bytes)/float64(KB), 'f', 2, 64) + " KB"
	default:
		return strconv.FormatUint(bytes, 10) + " B"
	}
}
// RunStandalone 以外挂模式启动 Web 管理后台
func RunStandalone(configPath string) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	var cfg struct {
		Admin struct {
			Enabled  bool   `yaml:"enabled"`
			Port     int    `yaml:"port"`
			Username string `yaml:"username"`
			Password string `yaml:"password"`
			DBPath   string `yaml:"db"`
			Path     string `yaml:"path"`
		} `yaml:"admin"`
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("解析 YAML 失败: %v", err)
	}

	if !cfg.Admin.Enabled {
		return fmt.Errorf("配置文件中未启用 admin 模块")
	}

	// 初始化数据库
	db, err := database.InitDb(cfg.Admin.DBPath)
	if err != nil {
		return fmt.Errorf("初始化数据库失败: %v", err)
	}

	log.Infof("启动独立 Web 管理后台, 监听端口: %d", cfg.Admin.Port)
	srv := New(db, cfg.Admin.Username, cfg.Admin.Password, cfg.Admin.Path, cfg.Admin.Port, false, "", false)
	
	// 这里我们需要一个不会自动退出的方式运行
	// New 内部已经启动了 http.Server (如果 port > 0)
	// 我们只需要阻塞主协程
	select {
	case <-srv.done:
	}
	return nil
}
