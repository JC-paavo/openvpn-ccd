package Router

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"openvpn-ccd/model"
)

func accountRouter(api *gin.RouterGroup, ccdManager *model.CCDManager) {

	api.POST("/accounts", func(c *gin.Context) {
		// 创建或更新账号
		var Account struct {
			Username    string   `json:"username"`
			Password    string   `json:"password"`
			DisplayName string   `json:"display_name"`
			Email       string   `json:"email"`
			Phone       string   `json:"phone"`
			IsIRoute    bool     `json:"is_iroute"`
			Enabled     bool     `json:"enabled"`
			Routes      []string `json:"routes"`
			TemplateIDs []string `json:"template_ids"`
			IrouteIDs   []string `json:"iroute_ids"`
		}
		if err := c.ShouldBindJSON(&Account); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var TemplateIDsUints []uint
		var IrouteIDsrUnits []uint
		for _, templateID := range Account.TemplateIDs {
			var templateIDUint uint
			if _, err := fmt.Sscanf(templateID, "%d", &templateIDUint); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
				return
			}
			TemplateIDsUints = append(TemplateIDsUints, templateIDUint)
		}
		for _, IrouteID := range Account.IrouteIDs {
			var IrouteIDsrUnit uint
			if _, err := fmt.Sscanf(IrouteID, "%d", &IrouteIDsrUnit); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
				return
			}
			IrouteIDsrUnits = append(IrouteIDsrUnits, IrouteIDsrUnit)
		}
		// 转换为模型结构
		newAccount := model.Account{

			Username:    Account.Username,
			Password:    Account.Password,
			DisplayName: Account.DisplayName,
			Email:       Account.Email,
			Phone:       Account.Phone,
			IsIRoute:    Account.IsIRoute,
			Enabled:     Account.Enabled,
			// 需要将routes转换为模型中的Route结构
		}
		newAccount.Routes = make([]model.Route, len(Account.Routes))
		for i, route := range Account.Routes {
			newAccount.Routes[i] = model.Route{Route: route}
		}
		user, _ := c.Get("username")
		username := user.(string)
		if err := ccdManager.CreateOrUpdateAccount(&newAccount, username, c, IrouteIDsrUnits, TemplateIDsUints, newAccount.Routes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "账号创建/更新成功"})
	})
	api.PUT("/accounts", func(c *gin.Context) {
		var Account struct {
			Username    string   `json:"username"`
			Password    string   `json:"password"`
			DisplayName string   `json:"display_name"`
			Email       string   `json:"email"`
			Phone       string   `json:"phone"`
			IsIRoute    bool     `json:"is_iroute"`
			Enabled     bool     `json:"enabled"`
			Routes      []string `json:"routes"`
			TemplateIDs []string `json:"template_ids"`
			IrouteIDs   []string `json:"iroute_ids"`
		}
		if err := c.ShouldBindJSON(&Account); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		var TemplateIDsUints []uint
		var IrouteIDsrUnits []uint
		for _, templateID := range Account.TemplateIDs {
			var templateIDUint uint
			if _, err := fmt.Sscanf(templateID, "%d", &templateIDUint); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
				return
			}
			TemplateIDsUints = append(TemplateIDsUints, templateIDUint)
		}
		for _, IrouteID := range Account.IrouteIDs {
			var IrouteIDsrUnit uint
			if _, err := fmt.Sscanf(IrouteID, "%d", &IrouteIDsrUnit); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的模板ID"})
				return
			}
			IrouteIDsrUnits = append(IrouteIDsrUnits, IrouteIDsrUnit)
		}
		// 转换为模型结构
		newAccount := model.Account{
			Model:       gorm.Model{},
			Username:    Account.Username,
			Password:    Account.Password,
			DisplayName: Account.DisplayName,
			Email:       Account.Email,
			Phone:       Account.Phone,
			IsIRoute:    Account.IsIRoute,
			Enabled:     Account.Enabled,
			Routes:      make([]model.Route, len(Account.Routes)),
			// 需要将routes转换为模型中的Route结构
		}
		if len(Account.Routes) > 0 {
			for i, route := range Account.Routes {
				newAccount.Routes[i] = model.Route{Route: route}
			}

		} else {
			newAccount.Routes = nil
		}
		//fmt.Println(newAccount.Routes)
		user, _ := c.Get("username")
		username := user.(string)
		if err := ccdManager.CreateOrUpdateAccount(&newAccount, username, c, IrouteIDsrUnits, TemplateIDsUints, newAccount.Routes); err != nil {
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
	// 获取账号关联的IRoute账号
	api.GET("/accounts/:username/iroutes", func(c *gin.Context) {
		username := c.Param("username")
		iroutes, err := ccdManager.GetAccountIRouteAccounts(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, iroutes)
	})
	// 获取账号关联的模板
	api.GET("/accounts/:username/templates", func(c *gin.Context) {
		username := c.Param("username")
		templates, err := ccdManager.GetAccountTemplates(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, templates)
	})
}
