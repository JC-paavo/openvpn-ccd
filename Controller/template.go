package Controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"openvpn-ccd/model"
	"strconv"
)

func TemplateCreate(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if err := ccdManager.CreateOrUpdateTemplate(templateReq.Template, templateReq.IRouteIDs, username, c.Request.Method); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "模板创建/更新成功"})
	}
}

func TemplateUpdate(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		if err := ccdManager.CreateOrUpdateTemplate(templateReq.Template, templateReq.IRouteIDs, username, c.Request.Method); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "模板创建/更新成功"})
	}
}

func TemplateDelete(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func TemplateGetOne(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func TemplateGetAll(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		templates, err := ccdManager.GetAllTemplates()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, templates)
	}
}

func StaticTemplate(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func StaticTemplateAdd(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func StaticTemplateEdit(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func StaticTemplateDelete(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {}
}
