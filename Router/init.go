package Router

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"html/template"
	"log"
	"openvpn-ccd/Middle"
	"openvpn-ccd/model"
	"os"
)

func InitRoute(r *gin.Engine, ccdManager *model.CCDManager, db *gorm.DB, logger *log.Logger) {
	adminUser := os.Getenv("ADMIN_USER")
	adminPass := os.Getenv("ADMIN_PASS")

	if adminUser == "" || adminPass == "" {
		log.Fatal("请设置ADMIN_USER和ADMIN_PASS环境变量")
	}

	// 配置session
	store := cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
	if os.Getenv("SESSION_SECRET") == "" {
		logger.Fatal("请设置SESSION_SECRET环境变量")
	}
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7天
		HttpOnly: true,
		//Secure:   true,
	})

	r.Use(sessions.Sessions("auth-session", store))

	// 添加日志中间件
	//r.Use(Middle.LoggingMiddleware(db))

	//初始化login路由
	loginRouter(r, store, db, adminUser, adminPass)

	//初始化静态路由
	// 静态文件服务
	r.Static("/static", "./static")
	add := func(a, b int) int {
		return a + b
	}
	r.SetFuncMap(template.FuncMap{
		"add": add,
	})
	r.LoadHTMLGlob("templates/*")
	web := r.Group("")
	web.Use(Middle.AuthMiddleware(store))

	// Web界面路由（需要认证）
	staticRoute(web, ccdManager, db)

	api := r.Group("/api")
	api.Use(Middle.AuthMiddleware(store))

	//初始化account
	accountRouter(api, ccdManager)
	//初始化route
	routeRouter(api, ccdManager)
	//初始化template
	templateRouter(api, ccdManager)
}
