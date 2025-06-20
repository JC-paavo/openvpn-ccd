package Router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"openvpn-ccd/model"
)

func templateRouter(api *gin.RouterGroup, ccdManager *model.CCDManager) {

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

		if err := ccdManager.CreateOrUpdateTemplate(templateReq.Template, templateReq.IRouteIDs, username, c.Request.Method); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "模板创建/更新成功"})
	})
	api.PUT("/templates", func(c *gin.Context) {
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
