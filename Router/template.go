package Router

import (
	"github.com/gin-gonic/gin"
	"openvpn-ccd/Controller"
	"openvpn-ccd/model"
)

func templateRouter(api *gin.RouterGroup, ccdManager *model.CCDManager) {

	// 创建或更新模板
	api.POST("/templates", Controller.TemplateCreate(ccdManager))
	api.PUT("/templates", Controller.TemplateUpdate(ccdManager))
	// 删除模板
	api.DELETE("/templates/:id", Controller.TemplateDelete(ccdManager))

	// 获取单个模板
	api.GET("/templates/:id", Controller.TemplateGetOne(ccdManager))

	// 获取所有模板
	api.GET("/templates", Controller.TemplateGetAll(ccdManager))

}
