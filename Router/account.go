package Router

import (
	"github.com/gin-gonic/gin"
	"openvpn-ccd/Controller"
	"openvpn-ccd/model"
)

func accountRouter(api *gin.RouterGroup, ccdManager *model.CCDManager) {
	//创建新账号
	api.POST("/accounts", Controller.AccountCreate(ccdManager))
	// 更新账号
	api.PUT("/accounts", Controller.AccountUpdate(ccdManager))
	// 删除账号
	api.DELETE("/accounts/:username", Controller.AccountDelete(ccdManager))

	// 获取单个账号
	api.GET("/accounts/:username", Controller.AccountOneinfo(ccdManager))

	// 获取所有账号
	api.GET("/accounts", Controller.AccountAllInfo(ccdManager))

	// 获取所有IRoute账号
	api.GET("/iroute-accounts", Controller.AccountIRouteAccounts(ccdManager))
	// 获取账号关联的IRoute账号
	api.GET("/accounts/:username/iroutes", Controller.AccountIrouterOneinfo(ccdManager))
	// 获取账号关联的模板
	api.GET("/accounts/:username/templates", Controller.AccountTemplates(ccdManager))
	// 在路由配置中添加
	api.GET("/accounts/:username/ccd", Controller.GetCCDConfig(ccdManager))
}
