package main

import (
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"html/template"
	"log"
	"net/http"
	"openvpn-ccd/model"
	"os"
	"time"
)

// 加载环境变量
func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("未找到.env文件，使用默认配置")
	}
}

// 创建日志
func createLog(db *gorm.DB, user, action, details, ip, userAgent string) {
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

// 中间件：记录请求日志
func loggingMiddleware(db *gorm.DB) gin.HandlerFunc {
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
		createLog(db, userStr, "请求", details, clientIP, userAgent)
	}
}

// 中间件：验证用户登录
func authMiddleware(store sessions.Store) gin.HandlerFunc {
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

func main() {
	// 加载配置
	ccdDir := os.Getenv("CCD_DIR")
	if ccdDir == "" {
		ccdDir = "/etc/openvpn/ccd"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "openvpn_ccd.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	adminUser := os.Getenv("ADMIN_USER")
	adminPass := os.Getenv("ADMIN_PASS")

	if adminUser == "" || adminPass == "" {
		log.Fatal("请设置ADMIN_USER和ADMIN_PASS环境变量")
	}

	// 初始化日志
	logger := log.New(os.Stdout, "[OpenVPN-Manager] ", log.LstdFlags)

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logger.Fatalf("连接数据库失败: %v", err)
	}

	// 自动迁移模型
	if err := db.AutoMigrate(
		&model.Account{},
		&model.IRoute{},
		&model.Template{},
		&model.Log{},
		&model.AccountIRoute{},
		&model.TemplateIRoute{},
	); err != nil {
		logger.Fatalf("数据库迁移失败: %v", err)
	}

	// 创建CCD管理器
	ccdManager := model.NewCCDManager(ccdDir, db, logger)

	add := func(a, b int) int {
		return a + b
	}
	// 创建Gin引擎
	r := gin.Default()

	// 配置session
	store := cookie.NewStore([]byte(os.Getenv("SESSION_SECRET")))
	if os.Getenv("SESSION_SECRET") == "" {
		logger.Fatal("请设置SESSION_SECRET环境变量")
	}

	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7天
		HttpOnly: true,
		Secure:   true,
	})
	r.Use(sessions.Sessions("auth-session", store))

	// 添加日志中间件
	r.Use(loggingMiddleware(db))

	// 静态文件服务
	r.Static("/static", "./static")
	r.SetFuncMap(template.FuncMap{
		"add": add,
	})
	r.LoadHTMLGlob("templates/*")

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
			createLog(
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
		createLog(
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
			createLog(
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

	// API路由（需要认证）
	api := r.Group("/api")
	api.Use(authMiddleware(store))
	{
		// 创建或更新账号
		api.POST("/accounts", func(c *gin.Context) {
			var account model.Account
			if err := c.ShouldBindJSON(&account); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			user, _ := c.Get("username")
			username := user.(string)

			if err := ccdManager.CreateOrUpdateAccount(account, username); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "账号创建/更新成功"})
		})

		// 删除账号
		api.DELETE("/accounts/:username", func(c *gin.Context) {
			username := c.Param("username")
			user, _ := c.Get("username")
			adminUser := user.(string)

			if err := ccdManager.DeleteAccount(username, adminUser); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "账号删除成功"})
		})

		// 获取单个账号
		api.GET("/accounts/:username", func(c *gin.Context) {
			username := c.Param("username")
			account, err := ccdManager.GetAccount(username)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			// 不返回密码
			account.Password = ""
			c.JSON(http.StatusOK, account)
		})

		// 获取所有账号
		api.GET("/accounts", func(c *gin.Context) {
			accounts, err := ccdManager.GetAllAccounts()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 不返回密码
			for i := range accounts {
				accounts[i].Password = ""
			}
			c.JSON(http.StatusOK, accounts)
		})

		// 获取所有IRoute账号
		api.GET("/iroute-accounts", func(c *gin.Context) {
			accounts, err := ccdManager.GetAllIRouteAccounts()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			// 不返回密码
			for i := range accounts {
				accounts[i].Password = ""
			}
			c.JSON(http.StatusOK, accounts)
		})

		// 获取所有IRoute路由
		api.GET("/iroutes", func(c *gin.Context) {
			iroutes, err := ccdManager.GetAllIRoutes()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, iroutes)
		})

		// 创建或更新模板
		api.POST("/templates", func(c *gin.Context) {
			var templateReq struct {
				Template  model.Template `json:"template"`
				IRouteIDs []uint         `json:"iroute_ids"`
			}

			if err := c.ShouldBindJSON(&templateReq); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			user, _ := c.Get("username")
			username := user.(string)

			if err := ccdManager.CreateOrUpdateTemplate(templateReq.Template, templateReq.IRouteIDs, username); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "模板创建/更新成功"})
		})

		// 删除模板
		api.DELETE("/templates/:id", func(c *gin.Context) {
			id := c.Param("id")
			var templateID uint
			if _, err := fmt.Sscanf(id, "%d", &templateID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
				return
			}

			user, _ := c.Get("username")
			username := user.(string)

			if err := ccdManager.DeleteTemplate(templateID, username); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"message": "模板删除成功"})
		})

		// 获取单个模板
		api.GET("/templates/:id", func(c *gin.Context) {
			id := c.Param("id")
			var templateID uint
			if _, err := fmt.Sscanf(id, "%d", &templateID); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
				return
			}

			template, err := ccdManager.GetTemplate(templateID)
			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, template)
		})

		// 获取所有模板
		api.GET("/templates", func(c *gin.Context) {
			templates, err := ccdManager.GetAllTemplates()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, templates)
		})
	}

	// Web界面路由（需要认证）
	web := r.Group("")
	web.Use(authMiddleware(store))
	{
		// 首页
		web.GET("/", func(c *gin.Context) {
			user, _ := c.Get("username")

			accounts, err := ccdManager.GetAllAccounts()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			templates, err := ccdManager.GetAllTemplates()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "index.html", gin.H{
				"user":      user,
				"accounts":  accounts,
				"templates": templates,
			})
		})

		// 添加账号页面
		web.GET("/account/add", func(c *gin.Context) {
			user, _ := c.Get("username")

			irouteAccounts, err := ccdManager.GetAllIRouteAccounts()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			templates, err := ccdManager.GetAllTemplates()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "add_account.html", gin.H{
				"user":           user,
				"irouteAccounts": irouteAccounts,
				"templates":      templates,
			})
		})

		// 编辑账号页面
		web.GET("/account/edit/:username", func(c *gin.Context) {
			user, _ := c.Get("username")
			username := c.Param("username")

			account, err := ccdManager.GetAccount(username)
			if err != nil {
				c.HTML(http.StatusNotFound, "error.html", gin.H{"error": err.Error()})
				return
			}

			irouteAccounts, err := ccdManager.GetAllIRouteAccounts()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			templates, err := ccdManager.GetAllTemplates()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "edit_account.html", gin.H{
				"user":           user,
				"account":        account,
				"irouteAccounts": irouteAccounts,
				"templates":      templates,
			})
		})

		// 模板列表页面
		web.GET("/templates", func(c *gin.Context) {
			user, _ := c.Get("username")

			templates, err := ccdManager.GetAllTemplates()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "templates.html", gin.H{
				"user":      user,
				"templates": templates,
			})
		})

		// 添加模板页面
		web.GET("/template/add", func(c *gin.Context) {
			user, _ := c.Get("username")

			iroutes, err := ccdManager.GetAllIRoutes()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "add_template.html", gin.H{
				"user":    user,
				"iroutes": iroutes,
			})
		})

		// 编辑模板页面
		web.GET("/template/edit/:id", func(c *gin.Context) {
			user, _ := c.Get("username")
			id := c.Param("id")
			var templateID uint
			if _, err := fmt.Sscanf(id, "%d", &templateID); err != nil {
				c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "无效的模板ID"})
				return
			}

			template, err := ccdManager.GetTemplate(templateID)
			if err != nil {
				c.HTML(http.StatusNotFound, "error.html", gin.H{"error": err.Error()})
				return
			}

			iroutes, err := ccdManager.GetAllIRoutes()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			// 提取已选择的IRoute ID
			selectedIRouteIDs := make(map[uint]bool)
			for _, ir := range template.IRoutes {
				selectedIRouteIDs[ir.ID] = true
			}

			c.HTML(http.StatusOK, "edit_template.html", gin.H{
				"user":              user,
				"template":          template,
				"iroutes":           iroutes,
				"selectedIRouteIDs": selectedIRouteIDs,
			})
		})
		web.GET("/accounts", func(c *gin.Context) {
			user, _ := c.Get("username")

			accounts, err := ccdManager.GetAllAccounts()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "accounts.html", gin.H{
				"user":     user,
				"accounts": accounts,
			})
		})

		// 日志页面
		web.GET("/logs", func(c *gin.Context) {
			user, _ := c.Get("username")

			var logs []model.Log
			if err := db.Order("created_at DESC").Limit(100).Find(&logs).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "logs.html", gin.H{
				"user": user,
				"logs": logs,
			})
		})
	}

	// 启动服务器
	logger.Printf("服务器启动在端口 %s", port)
	logger.Printf("CCD目录: %s", ccdDir)
	logger.Printf("数据库路径: %s", dbPath)
	if err := r.Run(":" + port); err != nil {
		logger.Fatalf("启动服务器失败: %v", err)
	}
}
