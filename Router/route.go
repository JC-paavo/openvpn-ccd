package Router

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"openvpn-ccd/model"
)

func routeRouter(api *gin.RouterGroup, ccdManager *model.CCDManager) {
	// 获取所有IRoute路由
	api.GET("/iroutes", func(c *gin.Context) {
		iroutes, err := ccdManager.GetAllIRoutesAccount()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, iroutes)
	})
}
