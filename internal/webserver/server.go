package webserver

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/voidluo/trojan-go/common"
	"github.com/voidluo/trojan-go/internal/database"
	"github.com/voidluo/trojan-go/internal/webui"
	"github.com/voidluo/trojan-go/log"
	"gorm.io/gorm"
)

// AdminServer 管理面板服务器，复用 TLS 层已建立的连接
type AdminServer struct {
	handler  http.Handler
	connChan chan net.Conn
	done     chan struct{}
}

// New 创建管理面板服务器
// db: 数据库连接; password: 登录密码; mountPath: 挂载路径; port: 独立监听端口 (0 为不开启独立监听)
func New(db *gorm.DB, password, mountPath string, port int) *AdminServer {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	if mountPath == "" {
		mountPath = "/"
	}

	secretKey := []byte("trojan-go-admin-" + password)

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
		r.GET(strings.TrimSuffix(mountPath, "/"), func(c *gin.Context) {
			c.Redirect(http.StatusFound, mountPath)
		})
		r.GET("/", func(c *gin.Context) {
			c.Redirect(http.StatusFound, mountPath)
		})
	}

	// 登录（无需鉴权）
	r.POST("/api/login", func(c *gin.Context) {
		var req struct {
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil || req.Password != password {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
			return
		}
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"admin": true,
			"exp":   time.Now().Add(time.Hour * 24).Unix(),
		})
		t, _ := token.SignedString(secretKey)
		c.JSON(http.StatusOK, gin.H{"token": t})
	})

	// 需要鉴权的管理 API
	auth := r.Group("/api", func(c *gin.Context) {
		ts := strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer ")
		token, err := jwt.Parse(ts, func(t *jwt.Token) (any, error) { return secretKey, nil })
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			c.Abort()
			return
		}
		c.Next()
	})
	{
		auth.GET("/status", func(c *gin.Context) {
			var count int64
			db.Model(&database.User{}).Count(&count)
			c.JSON(http.StatusOK, gin.H{"user_count": count})
		})

		auth.GET("/users", func(c *gin.Context) {
			var users []database.User
			db.Find(&users)
			c.JSON(http.StatusOK, users)
		})

		auth.POST("/users", func(c *gin.Context) {
			var user database.User
			if err := c.ShouldBindJSON(&user); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}
			if user.Password != "" && user.Hash == "" {
				user.Hash = common.SHA224String(user.Password)
			}
			db.Create(&user)
			c.JSON(http.StatusOK, user)
		})

		auth.DELETE("/users/:id", func(c *gin.Context) {
			db.Delete(&database.User{}, c.Param("id"))
			c.JSON(http.StatusOK, gin.H{"message": "已删除"})
		})

		auth.GET("/subscribe", func(c *gin.Context) {
			var users []database.User
			db.Where("status = ?", 0).Find(&users)
			domain := c.Query("domain")
			if domain == "" {
				domain = "your-domain.com"
			}
			c.String(http.StatusOK, generateClashConfig(users, domain, 443))
		})

		// 重启服务（通过 systemctl，进程由 systemd 重新拉起）
		auth.POST("/restart", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "正在重启..."})
			// 异步执行，先把响应返回给前端
			go func() {
				time.Sleep(200 * time.Millisecond)
				exec.Command("systemctl", "restart", "trojan-go").Run()
			}()
		})
	}

	srv := &AdminServer{
		handler:  r,
		connChan: make(chan net.Conn, 64),
		done:     make(chan struct{}),
	}

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

func generateClashConfig(users []database.User, domain string, port int) string {
	var sb strings.Builder
	sb.WriteString("port: 7890\nsocks-port: 7891\nallow-lan: true\nmode: rule\nlog-level: info\n\nproxies:\n")
	for i, u := range users {
		sb.WriteString(fmt.Sprintf("  - name: \"Trojan-%d\"\n    type: trojan\n    server: %s\n    port: %d\n    password: %s\n    udp: true\n    sni: %s\n    skip-cert-verify: false\n\n", i+1, domain, port, u.Password, domain))
	}
	sb.WriteString("proxy-groups:\n  - name: \"Trojan\"\n    type: select\n    proxies:\n")
	for i := range users {
		sb.WriteString(fmt.Sprintf("      - \"Trojan-%d\"\n", i+1))
	}
	sb.WriteString("\nrules:\n  - GEOIP,CN,DIRECT\n  - MATCH,Trojan\n")
	return sb.String()
}
