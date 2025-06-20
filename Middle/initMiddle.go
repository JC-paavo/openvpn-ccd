package Middle

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"net/http"
	"openvpn-ccd/model"
	"time"
)

// 创建日志
func CreateLog(db *gorm.DB, user, action, details, ip, userAgent string) {
	logEntry := model.Log{
		User:      user,
		Action:    action,
		Details:   details,
		IPAddress: ip,
		UserAgent: userAgent,
	}

	if err := db.Create(&logEntry).Error; err != nil {
		log.Printf("记录日志失败: %v", err)
	}
}

func LoggingMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		// 记录请求信息
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()
		latency := time.Since(start)

		// 获取当前用户（如果有）
		user, _ := c.Get("username")
		userStr, ok := user.(string)
		if !ok {
			userStr = "未认证用户"
		}

		// 记录日志
		details := fmt.Sprintf("method=%s path=%s status=%d latency=%s", method, path, statusCode, latency)
		CreateLog(db, userStr, "请求", details, clientIP, userAgent)
	}
}

// 中间件：验证用户登录
func AuthMiddleware(store sessions.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查session
		session, err := store.Get(c.Request, "auth-session")
		if err != nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 检查用户是否已登录
		if _, ok := session.Values["username"]; !ok {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}

		// 设置当前用户
		c.Set("username", session.Values["username"])

		c.Next()
	}
}
