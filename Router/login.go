package Router

import (
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"openvpn-ccd/Controller"
)

func loginRouter(r *gin.Engine, store cookie.Store, db *gorm.DB, adminUser, adminPass string) {
	// 登录相关路由
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	r.POST("/login", Controller.UserLogin(store, db, adminUser, adminPass))

	r.GET("/logout", Controller.UserLogout(store, db))
}
