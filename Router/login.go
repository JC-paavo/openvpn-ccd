package Router

import (
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"openvpn-ccd/Middle"
)

func loginRouter(r *gin.Engine, store cookie.Store, db *gorm.DB, adminUser, adminPass string) {
	// 登录相关路由
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	r.POST("/login", func(c *gin.Context) {
		username := c.PostForm("username")
		password := c.PostForm("password")

		if username == adminUser && password == adminPass {
			session, _ := store.Get(c.Request, "auth-session")
			session.Values["username"] = username
			session.Save(c.Request, c.Writer)

			// 记录登录日志
			Middle.CreateLog(
				db,
				username,
				"登录",
				"登录成功",
				c.ClientIP(),
				c.Request.UserAgent(),
			)

			c.Redirect(http.StatusFound, "/")
			return
		}

		// 记录登录失败日志
		Middle.CreateLog(
			db,
			username,
			"登录",
			"登录失败，密码错误",
			c.ClientIP(),
			c.Request.UserAgent(),
		)

		c.HTML(http.StatusBadRequest, "login.html", gin.H{"error": "用户名或密码错误"})
	})

	r.GET("/logout", func(c *gin.Context) {
		session, _ := store.Get(c.Request, "auth-session")
		if username, ok := session.Values["username"]; ok {
			// 记录登出日志
			Middle.CreateLog(
				db,
				username.(string),
				"登出",
				"用户主动登出",
				c.ClientIP(),
				c.Request.UserAgent(),
			)
		}

		session.Options.MaxAge = -1
		session.Save(c.Request, c.Writer)
		c.Redirect(http.StatusFound, "/login")
	})
}
