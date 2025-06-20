package Router

import (
	"fmt"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"html/template"
	"net/http"
	"openvpn-ccd/Middle"
	"openvpn-ccd/model"
)

func staticRoute(r *gin.Engine, store cookie.Store, ccdManager *model.CCDManager, db *gorm.DB) {
	// 静态文件服务
	r.Static("/static", "./static")
	add := func(a, b int) int {
		return a + b
	}
	r.SetFuncMap(template.FuncMap{
		"add": add,
	})
	r.LoadHTMLGlob("templates/*")
	// Web界面路由（需要认证）
	web := r.Group("")
	web.Use(Middle.AuthMiddleware(store))
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
			for _, ir := range template.Routes {
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
}
