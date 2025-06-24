package Router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"openvpn-ccd/model"
	"strconv"
)

func staticRoute(web *gin.RouterGroup, ccdManager *model.CCDManager, db *gorm.DB) {

	{
		// 首页
		web.GET("/", func(c *gin.Context) {
			user, _ := c.Get("username")

			// 获取分页参数，默认为第1页，每页10条
			page := c.DefaultQuery("page", "1")
			pageSize := c.DefaultQuery("page_size", "10")
			pageInt, _ := strconv.Atoi(page)
			pageSizeInt, _ := strconv.Atoi(pageSize)
			offset := (pageInt - 1) * pageSizeInt

			search := ""
			// 获取账号总数
			totalAccounts, _ := ccdManager.GetAllAccountCount(search)
			// 分页查询账号
			accounts, err := ccdManager.GetAccountsWithPagination(offset, pageSizeInt, search)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			// 获取模板总数
			totalTemplates, _ := ccdManager.GetAllTemplatesCount(search)
			// 分页查询模板
			templates, err := ccdManager.GetTemplatesWithPagination(offset, pageSizeInt, search)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "index.html", gin.H{
				"user":           user,
				"accounts":       accounts,
				"templates":      templates,
				"currentPage":    pageInt,
				"pageSize":       pageSizeInt,
				"totalAccounts":  int(totalAccounts),
				"totalTemplates": int(totalTemplates),
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
			//提取非关联路由
			var customRoutes []model.Route
			if err := db.
				Joins("JOIN account_routes ON routes.id = account_routes.route_id").
				Where("account_routes.account_id = ?", account.ID).
				Where("routes.id NOT IN (?)", db.Table("template_routes").Select("route_id")).
				Where("routes.id NOT IN (?)", db.Table("account_routes").
					Select("route_id").
					Joins("JOIN accounts ON account_routes.account_id = accounts.id").
					Where("accounts.is_iroute = ?", true)).
				Find(&customRoutes).Error; err != nil {
				// 处理错误
			}
			//fmt.Println(customRoutes)
			//提取template select
			allTemplates, err := ccdManager.GetAllTemplates()
			selectedTemplateIDs := make(map[uint]bool)
			for _, template := range account.Templates {
				selectedTemplateIDs[template.ID] = true
			}
			for _, template := range allTemplates {
				if _, ok := selectedTemplateIDs[template.ID]; !ok {
					selectedTemplateIDs[template.ID] = false
				}
			}
			//提取iroute select
			allIRouteAccounts, err := ccdManager.GetAllIRouteAccounts()
			selectedIRouteIDs := make(map[uint]bool)
			for _, irouteAccount := range allIRouteAccounts {
				selectedIRouteIDs[irouteAccount.ID] = false
			}
			for _, route := range account.Routes {
				for _, account := range route.Accounts {
					if account.IsIRoute {
						selectedIRouteIDs[account.ID] = true
					}
				}
			}

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
				"user":                user,
				"account":             account,
				"irouteAccounts":      allIRouteAccounts,
				"templates":           templates,
				"selectedTemplateIDs": selectedTemplateIDs,
				"selectedIRouteIDs":   selectedIRouteIDs,
				"customRoutes":        customRoutes,
			})
		})

		// 模板列表页面
		web.GET("/templates", func(c *gin.Context) {
			user, _ := c.Get("username")
			// 获取搜索参数
			searchQuery := c.Query("search")
			//templates, err := ccdManager.GetAllTemplates()
			// 获取分页参数，默认为第1页，每页10条
			page := c.DefaultQuery("page", "1")
			pageSize := c.DefaultQuery("page_size", "10")
			pageInt, _ := strconv.Atoi(page)
			pageSizeInt, _ := strconv.Atoi(pageSize)
			offset := (pageInt - 1) * pageSizeInt
			// 获取模板总数
			totalTemplates, _ := ccdManager.GetAllTemplatesCount(searchQuery)
			// 分页查询模板
			templates, err := ccdManager.GetTemplatesWithPagination(offset, pageSizeInt, searchQuery)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			// 计算每个模板的IRoute账号数量
			for i := range templates {
				irouteAccounts := make(map[uint]bool)
				for _, route := range templates[i].Routes {
					for _, account := range route.Accounts {
						if account.IsIRoute {
							irouteAccounts[account.ID] = true
						}
					}
				}
				templates[i].IRouteCount = len(irouteAccounts)
			}

			c.HTML(http.StatusOK, "templates.html", gin.H{
				"user":           user,
				"templates":      templates,
				"currentPage":    pageInt,
				"pageSize":       pageSizeInt,
				"totalTemplates": int(totalTemplates),
				"searchQuery":    searchQuery,
			})
		})

		// 添加模板页面
		web.GET("/template/add", func(c *gin.Context) {
			user, _ := c.Get("username")

			accounts, err := ccdManager.GetAllIRoutesAccount()
			templates, err := ccdManager.GetAllTemplates()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}
			c.HTML(http.StatusOK, "add_template.html", gin.H{
				"user":      user,
				"iroutes":   accounts,
				"templates": templates,
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

			Accounts, err := ccdManager.GetAllIRoutesAccount()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			// 提取已选择的IRoute ID
			selectedIRouteIDs := make(map[uint]bool)
			templateAccountIDs := make(map[uint]bool)
			for _, route := range template.Routes {
				for _, account := range route.Accounts {
					if account.IsIRoute {
						templateAccountIDs[account.ID] = true
					}
				}
			}
			for _, account := range Accounts {
				selectedIRouteIDs[account.ID] = templateAccountIDs[account.ID]
			}
			c.HTML(http.StatusOK, "edit_template.html", gin.H{
				"user":              user,
				"template":          template,
				"irouterUsers":      Accounts,
				"selectedIRouteIDs": selectedIRouteIDs,
			})
		})
		web.GET("/accounts", func(c *gin.Context) {
			user, _ := c.Get("username")
			searchQuery := c.Query("search")
			// 获取分页参数，默认为第1页，每页10条
			page := c.DefaultQuery("page", "1")
			pageSize := c.DefaultQuery("page_size", "10")
			pageInt, _ := strconv.Atoi(page)
			pageSizeInt, _ := strconv.Atoi(pageSize)
			offset := (pageInt - 1) * pageSizeInt

			// 获取账号总数
			totalAccounts, _ := ccdManager.GetAllAccountCount(searchQuery)
			// 分页查询账号
			accounts, err := ccdManager.GetAccountsWithPagination(offset, pageSizeInt, searchQuery)
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}
			//accounts, err := ccdManager.GetAllAccounts()
			if err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			c.HTML(http.StatusOK, "accounts.html", gin.H{
				"user":          user,
				"accounts":      accounts,
				"currentPage":   pageInt,
				"pageSize":      pageSizeInt,
				"totalAccounts": int(totalAccounts),
				"searchQuery":   searchQuery,
			})
		})

		// 日志页面
		web.GET("/logs", func(c *gin.Context) {
			user, _ := c.Get("username")
			// 获取分页参数，默认为第1页，每页10条
			page := c.DefaultQuery("page", "1")
			pageSize := c.DefaultQuery("page_size", "10")
			pageInt, _ := strconv.Atoi(page)
			pageSizeInt, _ := strconv.Atoi(pageSize)
			offset := (pageInt - 1) * pageSizeInt
			var totalLogscounts int64
			if err := db.Model(&model.Log{}).Count(&totalLogscounts).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}

			var logs []model.Log
			if err := db.Order("created_at DESC").Offset(offset).Limit(pageSizeInt).Find(&logs).Error; err != nil {
				c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
				return
			}
			c.HTML(http.StatusOK, "logs.html", gin.H{
				"user":            user,
				"logs":            logs,
				"currentPage":     pageInt,
				"pageSize":        pageSizeInt,
				"totalLogscounts": int(totalLogscounts),
			})
		})
	}
}
