package Router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"openvpn-ccd/Controller"
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
		web.GET("/accounts", Controller.StaticAccount(ccdManager))
		// 添加账号页面
		web.GET("/account/add", Controller.StaticAccountAdd(ccdManager))

		// 编辑账号页面
		web.GET("/account/edit/:username", Controller.StaticAccountEdit(ccdManager, db))

		// 模板列表页面
		web.GET("/templates", Controller.StaticTemplate(ccdManager))

		// 添加模板页面
		web.GET("/template/add", Controller.StaticTemplateAdd(ccdManager))

		// 编辑模板页面
		web.GET("/template/edit/:id", Controller.StaticTemplateEdit(ccdManager))

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
