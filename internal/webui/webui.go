package webui

import (
	"embed"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed index.html
var staticFiles embed.FS

// RegisterHandlers 将 WebUI 静态资源挂载到 Gin 路由
func RegisterHandlers(r *gin.Engine) {
	r.GET("/", func(c *gin.Context) {
		data, err := staticFiles.ReadFile("index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "Internal Server Error")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}

// ReadIndex 返回嵌入的 index.html 内容
func ReadIndex() ([]byte, error) {
	return staticFiles.ReadFile("index.html")
}
