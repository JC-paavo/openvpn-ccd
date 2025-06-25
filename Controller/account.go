package Controller

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"openvpn-ccd/model"
	"strconv"
)

func AccountCreate(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
		if err := ccdManager.CreateOrUpdateAccount(newAccount, username, c, IrouteIDsrUnits, TemplateIDsUints, newAccount.Routes); err != nil {
			fmt.Println(err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "账号创建/更新成功"})
		return
	}

}

func AccountUpdate(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			c.JSON(http.StatusBadRequest, gin.H{"json error": err.Error()})
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
			// 需要将routes转换为模型中的Route结构
		}
		newAccount.Routes = make([]model.Route, len(Account.Routes))
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

		if err := ccdManager.CreateOrUpdateAccount(newAccount, username, c, IrouteIDsrUnits, TemplateIDsUints, newAccount.Routes); err != nil {
			fmt.Printf("更新账号失败!%s", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "更新账号失败!"})
		}

		c.JSON(http.StatusOK, gin.H{"message": "账号创建/更新成功"})
	}
}

func AccountDelete(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		user, _ := c.Get("username")
		adminUser := user.(string)

		if err := ccdManager.DeleteAccount(username, adminUser); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "账号删除成功"})
	}
}

func AccountOneinfo(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		account, err := ccdManager.GetAccount(username)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		// 不返回密码
		account.Password = ""
		c.JSON(http.StatusOK, account)
	}
}

func AccountAllInfo(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func AccountIRouteAccounts(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func AccountIrouterOneinfo(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		iroutes, err := ccdManager.GetAccountIRouteAccounts(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, iroutes)
	}
}

func AccountTemplates(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		username := c.Param("username")
		templates, err := ccdManager.GetAccountTemplates(username)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, templates)
	}
}

func StaticAccount(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func StaticAccountAdd(ccdManager *model.CCDManager) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func StaticAccountEdit(ccdManager *model.CCDManager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}
