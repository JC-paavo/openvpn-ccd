package Router

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"openvpn-ccd/model"
)

func accountRouter(api *gin.RouterGroup, ccdManager *model.CCDManager) {
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
}
