package model

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"log"
	"openvpn-ccd/utils"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CCD配置管理器
type CCDManager struct {
	ccdDir string
	db     *gorm.DB
	logger *log.Logger
}

// 创建新的CCD管理器
func NewCCDManager(ccdDir string, db *gorm.DB, logger *log.Logger) *CCDManager {
	// 确保CCD目录存在
	if _, err := os.Stat(ccdDir); os.IsNotExist(err) {
		if err := os.MkdirAll(ccdDir, 0755); err != nil {
			logger.Fatalf("无法创建CCD目录: %v", err)
		}
	}

	return &CCDManager{
		ccdDir: ccdDir,
		db:     db,
		logger: logger,
	}
}

// 创建或更新账号配置
func (m *CCDManager) CreateOrUpdateAccount(newAccount Account, user string, c *gin.Context, IrouteIDs, TemplateIDs []uint, newRoutes []Route) error {
	// 验证邮箱格式
	if !validateEmail(newAccount.Email) {
		return fmt.Errorf("邮箱格式不正确")
	}

	// 验证手机号格式
	if !validatePhone(newAccount.Phone) {
		return fmt.Errorf("手机号格式不正确")
	}

	// 验证IRoute格式（如果是IRoute账号）
	if newAccount.IsIRoute {
		for _, route := range newRoutes {
			if !validateCIDR(route.Route) {
				return fmt.Errorf("CreateOrUpdateAccount:IRoute路由格式不正确，应为 'IP地址 子网掩码'")
			}
		}
	}

	// 加密密码
	if newAccount.Password != "" {
		hashedPassword, err := utils.HashPassword(newAccount.Password)
		if err != nil {
			return fmt.Errorf("密码加密失败: %v", err)
		}
		newAccount.Password = hashedPassword
	}

	var existing Account
	var templates []Template
	if len(TemplateIDs) != 0 {
		if err := m.db.Preload("Accounts").Preload("Routes").Where("ID in ?", TemplateIDs).Find(&templates).Error; err != nil {
			return fmt.Errorf("CreateOrUpdateAccount:查询模板失败: %v", err)
		}
	}

	result := m.db.Where("username = ?", newAccount.Username).First(&existing)

	switch c.Request.Method {
	case "POST": // 创建新账号
		if result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return fmt.Errorf("CreateOrUpdateAccount:查询账号失败: %v", result.Error)
			}
		}
		if result.RowsAffected != 0 {
			return fmt.Errorf("账号已存在")
		}
		if newAccount.IsIRoute {
			var irouteAccounts []Account
			m.db.Preload("Routes").Where("is_iroute = ?", true).Find(&irouteAccounts)
			//判断iroute路由是否冲突
			if len(newRoutes) > 0 && len(irouteAccounts) > 0 {
				for _, route := range newRoutes {
					newip := strings.Split(route.Route, " ")
					for _, ic := range irouteAccounts {
						for _, irouteRoute := range ic.Routes {
							exitIp := strings.Split(irouteRoute.Route, " ")
							if res, err := utils.IsNetworkConflict(newip[0], newip[1], exitIp[0], exitIp[1]); err != nil || res {
								return fmt.Errorf("CreateOrUpdateAccount:和%s路由冲突", ic.Username)
							}
						}

					}
				}
			}
			if err := m.db.Create(&newAccount).Error; err != nil {
				return fmt.Errorf("CreateOrUpdateAccount:创建iroute账号失败: %v", err)
			}
			m.logger.Printf("用户 %s 创建了iroute账号: %s", user, newAccount.Username)
		} else {
			//查询出普通账号关联的iroute账号路由
			var iroutesRoutes []Route
			if len(IrouteIDs) != 0 {
				if err := m.db.Joins("JOIN account_routes ON account_routes.route_id = routes.id").
					Where("account_routes.account_id IN ?", IrouteIDs).
					Find(&iroutesRoutes).Error; err != nil {
					return fmt.Errorf("CreateOrUpdateAccount:查询IRoute账号路由失败: %v", err)
				}
			}
			newAccount.Templates = templates
			//account.Routes = append(account.Routes, iroutesRoutes...)
			if err := m.db.Create(&newAccount).Error; err != nil {
				return fmt.Errorf("CreateOrUpdateAccount:创建普通账号失败: %v", err)
			}
			if len(iroutesRoutes) > 0 {
				m.db.Model(&newAccount).Association("Routes").Append(iroutesRoutes)
			}
			m.logger.Printf("用户 %s 创建了普通账号: %s", user, newAccount.Username)
		}
		// 生成并写入CCD配置文件

	case "PUT":
		var iroutesRoutes []Route
		if result.Error != nil {
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查询账号不存在，请创建新账号: %v", result.Error)
			} else {
				return fmt.Errorf("查询账号失败: %v", result.Error)
			}
		}
		// 更新现有账号
		newAccount.ID = existing.ID
		if newAccount.Password == "" {
			newAccount.Password = existing.Password // 保持原有密码
		}
		if newAccount.IsIRoute {
			var irouteAccounts []Account
			m.db.Preload("Routes").Where("is_iroute = ? and id != ?", true, newAccount.ID).Find(&irouteAccounts)
			//判断iroute路由是否冲突
			if len(newRoutes) > 0 && len(irouteAccounts) > 0 {
				for _, route := range newRoutes {
					newip := strings.Split(route.Route, " ")
					for _, ic := range irouteAccounts {
						for _, irouteRoute := range ic.Routes {
							exitIp := strings.Split(irouteRoute.Route, " ")
							if res, err := utils.IsNetworkConflict(newip[0], newip[1], exitIp[0], exitIp[1]); err != nil || res {
								return fmt.Errorf("CreateOrUpdateAccount:和%s路由冲突", ic.Username)
							}
						}

					}
				}
			}
			//先拿到和这个iroute user关联的account
			var tmpaccount Account

			if err := m.db.Preload("Routes").Where("id = ?", newAccount.ID).First(&tmpaccount).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return fmt.Errorf("查询irouter账号失败: %v", err)
				}
			}

			// 1. 获取关联该IRoute账号的所有普通账号
			var normalAccounts []Account
			if err := m.db.Distinct("accounts.*").
				Joins("JOIN account_routes ON account_routes.account_id = accounts.id").
				Joins("JOIN routes ON account_routes.route_id = routes.id").
				Joins("JOIN account_routes AS iroute_ar ON iroute_ar.route_id = routes.id").
				Where("accounts.is_iroute = ? AND iroute_ar.account_id = ?", false, newAccount.ID).
				Preload("Templates").
				Preload("Routes").
				Find(&normalAccounts).Error; err != nil {
				return fmt.Errorf("查询关联普通账号失败: %v", err)
			}
			//TODO: 获取关联该IRoute账号的所有模板
			var templates []Template
			if err := m.db.Joins("JOIN template_routes ON template_routes.template_id = templates.id").
				Joins("JOIN routes ON template_routes.route_id = routes.id").
				Joins("JOIN account_routes ON account_routes.route_id = routes.id").
				Where("account_routes.account_id = ?", newAccount.ID).
				Preload("Accounts").
				Find(&templates).Error; err != nil {
				return fmt.Errorf("查询关联模板失败: %v", err)
			}

			err := m.db.Transaction(func(tx *gorm.DB) error {
				// 2. 清空当前iroute账号的iroute路由
				if err := tx.Model(&newAccount).Association("Routes").Clear(); err != nil {
					return fmt.Errorf("清除模板Irouter useer路由关联失败: %v", err)
				}
				oldrouteIDs := make([]uint, len(tmpaccount.Routes))
				for i, route := range tmpaccount.Routes {
					oldrouteIDs[i] = route.ID
				}
				// /清空所有普通账号关联的当前irouter账号的路由
				if len(oldrouteIDs) > 0 {
					if err := tx.Exec("DELETE FROM account_routes WHERE route_id IN ? AND account_id IN (SELECT id FROM accounts WHERE is_iroute = false)", oldrouteIDs).Error; err != nil {
						return fmt.Errorf("删除普通账号关联路由失败: %v", err)
					}
				}
				//if len(normalAccounts) > 0 {
				//	for _, account := range normalAccounts {
				//		// 先获取当前账号的所有路由
				//		//for _, route := range tmpaccount.Routes {
				//		//	fmt.Printf("删除普通账号路由: %s", route.Route)
				//		//	err := m.db.Model(&account).Association("Routes").Delete(route)
				//		//	if err != nil {
				//		//		fmt.Printf("删除普通账号路由失败: %v", err)
				//		//	}
				//		//}
				//		if err := m.db.Model(&AccountRoute{}).Exec("delete from account_routes where account_id = ? and route_id in ?", account.ID, oldrouteIDs).Error; err != nil {
				//			return fmt.Errorf("删除普通账号路由失败: %v", err)
				//		}
				//	}
				//}

				//删除所有模版关联的更新当前iroute路由
				if len(templates) > 0 {
					for _, template := range templates {
						// 先获取当前账号的所有路由
						for _, route := range tmpaccount.Routes {
							fmt.Printf("删除模板路由: %s", route.Route)
							err := tx.Model(&template).Association("Routes").Delete(route)
							if err != nil {
								fmt.Printf("删除模板路由失败: %v", err)
							}
						}
					}
				}

				//删除原来route表中iroter账号路由
				if len(tmpaccount.Routes) > 0 {
					//fmt.Println(oldrouteIDs)
					if err := tx.Unscoped().Where("id IN ?", oldrouteIDs).Delete(&Route{}).Error; err != nil {
						return fmt.Errorf("删除irouter账号原路由失败: %v", err)
					}
				}

				//更新账号
				if err := tx.Save(&newAccount).Error; err != nil {
					return fmt.Errorf("更新账号失败: %v", err)
				}

				//创建新的irouter账号路由
				if len(newRoutes) > 0 {
					if err := tx.Create(&newRoutes).Error; err != nil {
						return fmt.Errorf("创建新的irouter路由失败: %v", err)
					}
					if err := tx.Model(&newAccount).Association("Routes").Append(newRoutes); err != nil {
						return fmt.Errorf("更新账号关联路由失败: %v", err)
					}
					//更新iroute文件
					err := m.generateCCDConfig(newAccount.Username)
					if err != nil {
						return fmt.Errorf("更新irouter账号配置失败: %v", err)
					}
				}

				//更新普通账号关联的irouter路由
				for _, account := range normalAccounts {
					var newNoraccount Account
					tx.Where("id =?", account.ID).First(&newNoraccount)
					if err := tx.Model(&newNoraccount).Association("Routes").Append(newRoutes); err != nil {
						return fmt.Errorf("更新普通账号关联路由失败: %v", err)
					}
				}

				//更新模板关联的irouter路由
				for _, template := range templates {
					var newTemplate Template
					tx.Where("id =?", template.ID).First(&newTemplate)
					if err := tx.Model(&newTemplate).Association("Routes").Append(newRoutes); err != nil {
						return fmt.Errorf("更新模板关联路由失败: %v", err)
					}
				}
				return nil
			})

			if err != nil {
				return fmt.Errorf("事务失败: %v", err)
			}

			// 3. 更新关联模板的普通账号CCD配置
			for _, template := range templates {
				for _, account := range template.Accounts {
					if !account.IsIRoute && account.Enabled {
						if err := m.generateCCDConfig(account.Username); err != nil {
							return fmt.Errorf("更新账号 %s 配置失败: %v", account.Username, err)
						}
					}
				}
			}
			//TODO：这里要去刷新iroute账号关联的账号ccd信息
			for _, account := range normalAccounts {
				if account.Enabled {
					if err := m.generateCCDConfig(account.Username); err != nil {
						return fmt.Errorf("更新账号 %s 配置失败: %v", account.Username, err)
					}
				}
			}
			return nil
		} else {
			//查询出普通账号更新时关联的iroute账号的路由
			if len(IrouteIDs) != 0 {
				if err := m.db.Joins("JOIN account_routes ON account_routes.route_id = routes.id").
					Where("account_routes.account_id IN ?", IrouteIDs).
					Find(&iroutesRoutes).Error; err != nil {
					return fmt.Errorf("CreateOrUpdateAccount:查询IRoute账号路由失败: %v", err)
				}
			}
		}
		fmt.Println("更新操作...")
		return m.db.Transaction(func(tx *gorm.DB) error {
			//删除自定义路由
			if err := m.db.Exec("DELETE FROM account_routes WHERE account_id = ? AND route_id NOT IN (SELECT route_id FROM template_routes) AND route_id NOT IN (SELECT route_id FROM account_routes WHERE account_id IN (SELECT id FROM accounts WHERE is_iroute = true))", existing.ID).Error; err != nil {
				return fmt.Errorf("删除自定义路由失败: %v", err)
			}
			// 删除不再被引用的路由记录
			if err := m.db.Exec("DELETE FROM routes WHERE id IN (SELECT r.id FROM routes r LEFT JOIN account_routes ar ON r.id = ar.route_id WHERE ar.route_id IS NULL)").Error; err != nil {
				return fmt.Errorf("删除无用路由记录失败: %v", err)
			}

			// 先清空现有的关联
			if err := m.db.Model(&newAccount).Association("Routes").Clear(); err != nil {
				return fmt.Errorf("清除账号路由关联失败: %v", err)
			}

			if err := m.db.Model(&newAccount).Association("Templates").Clear(); err != nil {
				return fmt.Errorf("清除模板Iroutel路由关联失败: %v", err)
			}

			if err := m.db.Save(&newAccount).Error; err != nil {
				return fmt.Errorf("更新账号失败: %v", err)
			}

			if len(newRoutes) > 0 {

				if err := m.db.Create(&newRoutes).Error; err != nil {
					return fmt.Errorf("创建新的路由失败: %v", err)
				}
				if err := m.db.Model(&newAccount).Association("Routes").Append(&newRoutes); err != nil {
					return fmt.Errorf("更新账号失败: %v", err)
				}
			}
			//更新关联
			m.db.Model(&newAccount).Association("Routes").Append(iroutesRoutes)
			m.db.Model(&newAccount).Association("Templates").Append(templates)
			return nil
		})
	}
	m.generateCCDConfig(newAccount.Username)
	return nil
}

// 删除账号
func (m *CCDManager) DeleteAccount(username string, user string) error {
	var account Account
	if err := m.db.Preload("Routes").Where("username = ?", username).First(&account).Error; err != nil {
		return fmt.Errorf("查询账号失败: %v", err)
	}

	// 删除账号
	if account.IsIRoute {
		// 1. 获取关联该IRoute账号的所有普通账号
		var normalAccounts []Account
		if err := m.db.Joins("JOIN account_routes ON account_routes.account_id = accounts.id").
			Joins("JOIN routes ON account_routes.route_id = routes.id").
			Joins("JOIN account_routes AS iroute_ar ON iroute_ar.route_id = routes.id").
			Where("accounts.is_iroute =? AND iroute_ar.account_id =?", false, account.ID).
			Preload("Templates").
			Preload("Routes").
			Find(&normalAccounts).Error; err != nil {
			return fmt.Errorf("查询关联普通账号失败: %v", err)
		}
		if len(normalAccounts) > 0 {
			if err := m.db.Model(&normalAccounts).Association("Templates").Clear(); err != nil {
			}
		}

		//TODO: 获取关联该IRoute账号的所有模板
		var templates []Template
		if err := m.db.Joins("JOIN template_routes ON template_routes.template_id = templates.id").
			Joins("JOIN routes ON template_routes.route_id = routes.id").
			Joins("JOIN account_routes ON account_routes.route_id = routes.id").
			Where("account_routes.account_id =?", account.ID).
			Preload("Accounts").
			Find(&templates).Error; err != nil {
			return fmt.Errorf("查询关联模板失败: %v", err)
		}
		if len(templates) > 0 {
			return fmt.Errorf("删除账号失败: 该账号被模板关联，无法删除")
		}

	}
	//检查普通账号irouter关联
	accounts, err := m.GetAccountIRouteAccounts(username)
	if err != nil {
		return fmt.Errorf("查询普通账号关联IRoute账号失败: %v", err)
	}
	templates, err := m.GetAccountTemplates(username)
	if err != nil {
		return fmt.Errorf("查询普通账号关联模板失败: %v", err)
	}
	if len(accounts) > 0 || len(templates) > 0 {
		return fmt.Errorf("删除账号失败: 该账号有数据关联，无法删除")
	}
	//删除引用关系
	if err := m.db.Model(&account).Association("Routes").Clear(); err != nil {
		return fmt.Errorf("清除模板Irouter useer路由关联失败: %v", err)
	}

	if err := m.db.Model(&account).Association("Templates").Clear(); err != nil {
		return fmt.Errorf("清除模板Irouter useer路由关联失败: %v", err)
	}

	// 删除不再被引用的路由记录
	if err := m.db.Exec("DELETE FROM routes WHERE id IN (SELECT r.id FROM routes r LEFT JOIN account_routes ar ON r.id = ar.route_id WHERE ar.route_id IS NULL)").Error; err != nil {
		return fmt.Errorf("删除无用路由记录失败: %v", err)
	}
	if err := m.db.Unscoped().Delete(&account).Error; err != nil {
		return fmt.Errorf("删除账号失败: %v", err)
	}

	// 删除CCD配置文件
	if err := m.deleteCCDConfig(username); err != nil {
		return fmt.Errorf("删除配置文件失败: %v", err)
	}

	m.logger.Printf("用户 %s 删除了账号: %s", user, username)
	return nil
}

// 获取账号
func (m *CCDManager) GetAccount(username string) (Account, error) {
	var account Account
	if err := m.db.Preload("Routes").Preload("Routes.Accounts").Preload("Templates").Where("username = ?", username).First(&account).Error; err != nil {
		return account, err
	}
	return account, nil
}

// 获取所有账号
func (m *CCDManager) GetAllAccounts() ([]Account, error) {
	var accounts []Account
	if err := m.db.Preload("Routes.Accounts").Preload("Templates").Find(&accounts).Error; err != nil {
		return nil, err
	}
	// 为每个账号添加关联信息
	for i := range accounts {
		// 1. 普通账号的iroute关联账号列表
		if !accounts[i].IsIRoute {
			irouteAccounts := make(map[uint]string)
			for _, route := range accounts[i].Routes {
				for _, acc := range route.Accounts {
					if acc.IsIRoute {
						irouteAccounts[acc.ID] = acc.DisplayName
					}
				}
			}
			accounts[i].IRouteAccounts = irouteAccounts
		}

		// 2. IRoute账号被哪些模板关联的模板名称列表
		if accounts[i].IsIRoute {
			referencedTemplates := make(map[uint]string)
			var allTemplates []Template
			if err := m.db.Preload("Routes.Accounts").Find(&allTemplates).Error; err == nil {
				for _, tpl := range allTemplates {
					for _, route := range tpl.Routes {
						for _, acc := range route.Accounts {
							if acc.ID == accounts[i].ID {
								referencedTemplates[tpl.ID] = tpl.Name
								//referencedTemplates = append(referencedTemplates, tpl.Name)
								break
							}
						}
					}
				}
			}
			accounts[i].ReferencedTemplateNames = referencedTemplates
		}
	}

	return accounts, nil
}

// 获取所有IRoute账号
func (m *CCDManager) GetAllIRouteAccounts() ([]Account, error) {
	var accounts []Account
	if err := m.db.Where("is_iroute = ? AND enabled = ?", true, true).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// 获取所有IRoute路由
func (m *CCDManager) GetAllIRoutesAccount() ([]Account, error) {
	var Accounts []Account
	if err := m.db.Preload("Routes").Where("is_iroute=? and enabled=?", true, true).Find(&Accounts).Error; err != nil {
		return nil, err
	}
	return Accounts, nil
}

// 获取普通账号关联的IRoute账号
func (m *CCDManager) GetAccountIRouteAccounts(username string) (map[uint]string, error) {
	var account Account
	if err := m.db.Preload("Routes.Accounts", "is_iroute = ?", true).
		Where("username = ? AND is_iroute = ?", username, false).
		First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}

	irouteAccounts := make(map[uint]string)
	for _, route := range account.Routes {
		for _, acc := range route.Accounts {
			if acc.IsIRoute {
				irouteAccounts[acc.ID] = acc.DisplayName
			}
		}
	}

	return irouteAccounts, nil
}

// 获取普通账号关联的模板
func (m *CCDManager) GetAccountTemplates(username string) ([]Template, error) {
	var account Account
	if err := m.db.Preload("Templates").
		Where("username = ? AND is_iroute = ?", username, false).
		First(&account).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("查询账号失败: %v", err)
	}

	return account.Templates, nil
}

// 分页获取账号
func (m *CCDManager) GetAccountsWithPagination(offset, limit int, search string) ([]Account, error) {
	var accounts []Account
	if search != "" {
		if err := m.db.Preload("Routes.Accounts").
			Preload("Templates").
			Where("username LIKE? or display_name LIKE ?", "%"+search+"%", "%"+search+"%").
			Offset(offset).
			Limit(limit).
			Find(&accounts).Error; err != nil {
			return nil, err
		}
	} else {
		if err := m.db.Preload("Routes.Accounts").
			Preload("Templates").
			Offset(offset).
			Limit(limit).
			Find(&accounts).Error; err != nil {
			return nil, err
		}
	}
	// 为每个账号添加关联信息
	for i := range accounts {
		// 1. 普通账号的iroute关联账号列表
		if !accounts[i].IsIRoute {
			irouteAccounts := make(map[uint]string)
			for _, route := range accounts[i].Routes {
				for _, acc := range route.Accounts {
					if acc.IsIRoute {
						irouteAccounts[acc.ID] = acc.DisplayName
					}
				}
			}
			accounts[i].IRouteAccounts = irouteAccounts
		}

		// 2. IRoute账号被哪些模板关联的模板名称列表
		if accounts[i].IsIRoute {
			referencedTemplates := make(map[uint]string)
			var allTemplates []Template
			if err := m.db.Preload("Routes.Accounts").Find(&allTemplates).Error; err == nil {
				for _, tpl := range allTemplates {
					for _, route := range tpl.Routes {
						for _, acc := range route.Accounts {
							if acc.ID == accounts[i].ID {
								referencedTemplates[tpl.ID] = tpl.Name
								//referencedTemplates = append(referencedTemplates, tpl.Name)
								break
							}
						}
					}
				}
			}
			accounts[i].ReferencedTemplateNames = referencedTemplates
		}
	}

	return accounts, nil
	//var accounts []Account
	//if err := m.db.Preload("Routes.Accounts").Preload("Templates").
	//	Offset(offset).Limit(limit).
	//	Find(&accounts).Error; err != nil {
	//	return nil, err
	//}
	//return accounts, nil
}

// 分页获取模板
func (m *CCDManager) GetTemplatesWithPagination(offset, limit int, search string) ([]Template, error) {
	var templates []Template
	if search != "" {
		if err := m.db.Preload("Accounts").Preload("Routes.Accounts").Where("name LIKE?", "%"+search+"%").
			Offset(offset).Limit(limit).
			Find(&templates).Error; err != nil {
			return nil, err
		}
		return templates, nil
	}

	if err := m.db.Preload("Accounts").Preload("Routes.Accounts").
		Offset(offset).Limit(limit).
		Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

func (m *CCDManager) GetAllTemplatesCount(search string) (int64, error) {
	var templatesTotal int64

	if search != "" {
		if err := m.db.Model(&Template{}).Where("name LIKE ?", "%"+search+"%").Count(&templatesTotal).Error; err != nil {
			return 0, err
		}
		return templatesTotal, nil
	}

	if err := m.db.Model(&Template{}).Count(&templatesTotal).Error; err != nil {
		return 0, err
	}

	return templatesTotal, nil
}
func (m *CCDManager) GetAllAccountCount(search string) (int64, error) {

	var accountsTotal int64
	if search != "" {
		if err := m.db.Model(&Account{}).Where("username LIKE? or display_name LIKE ?", "%"+search+"%", "%"+search+"%").Count(&accountsTotal).Error; err != nil {
			return 0, err
		}
		return accountsTotal, nil
	}

	if err := m.db.Model(&Account{}).Count(&accountsTotal).Error; err != nil {
		return 0, err
	}

	return accountsTotal, nil
}

// 生成CCD配置文件
func (m *CCDManager) generateCCDConfig(username string) error {

	//获取更新账号的最新信息
	var tmpAccount Account
	if err := m.db.Preload("Routes").Preload("Templates.Routes").Where("username =?", username).First(&tmpAccount).Error; err != nil {
		return fmt.Errorf("查询账号失败: %v", err)
	}
	if !tmpAccount.Enabled {
		return m.deleteCCDConfig(username)
	}

	//创建配置文件
	configPath := m.getConfigPath(tmpAccount.Username)
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer file.Close()
	// IRoute账号配置
	if tmpAccount.IsIRoute {
		file.WriteString(fmt.Sprintf("#Iroute账号:%s\n", tmpAccount.DisplayName))
		for _, route := range tmpAccount.Routes {
			if _, err := file.WriteString(fmt.Sprintf("iroute %s\n", route.Route)); err != nil {
				return fmt.Errorf("写入配置文件失败: %v", err)
			}
		}

		return nil
	}

	// 普通账号配置 - 添加路由和关联iroute
	if len(tmpAccount.Routes) != 0 {
		if err := writeRoute(tmpAccount.Routes, file); err != nil {
			return err
		}
	}
	if len(tmpAccount.Templates) != 0 {
		if err := writeTemplateRoute(tmpAccount.Templates, file); err != nil {
			return err
		}
	}

	return nil
}

// 删除CCD配置文件
func (m *CCDManager) deleteCCDConfig(username string) error {
	configPath := m.getConfigPath(username)
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		if err := os.Remove(configPath); err != nil {
			return fmt.Errorf("删除配置文件失败: %v", err)
		}
	}
	return nil
}

// 获取配置文件路径
func (m *CCDManager) getConfigPath(username string) string {
	return filepath.Join(m.ccdDir, username)
}

// 批量写push route
func writeRoute(routes []Route, file *os.File) error {
	file.WriteString(fmt.Sprintf("#自定义路由和irouter关联路由:\n"))
	for _, route := range routes {
		if _, err := file.WriteString(fmt.Sprintf("push \"route %s\"\n", route.Route)); err != nil {
			return fmt.Errorf("写入配置文件失败: %v", err)
		}
	}
	return nil
}
func writeTemplateRoute(tpls []Template, file *os.File) error {
	for _, tpl := range tpls {
		for _, route := range tpl.Routes {
			if _, err := file.WriteString(fmt.Sprintf("#模板类型路由 模版名称:%s\n", tpl.Name)); err != nil {
				return fmt.Errorf("写入配置文件失败: %v", err)
			}
			if _, err := file.WriteString(fmt.Sprintf("push \"route %s\"\n", route.Route)); err != nil {
				return fmt.Errorf("写入配置文件失败: %v", err)
			}
		}
	}
	return nil
}

// 验证邮箱格式
func validateEmail(email string) bool {
	re := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return re.MatchString(email)
}

// 验证手机号格式
func validatePhone(phone string) bool {
	re := regexp.MustCompile(`^1[3-9]\d{9}$`)
	return re.MatchString(phone)
}

// 验证CIDR格式
func validateCIDR(cidr string) bool {
	re := regexp.MustCompile(`^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3} \d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`)
	return re.MatchString(cidr)
}

// 创建或更新模板
func (m *CCDManager) CreateOrUpdateTemplate(template Template, irouteIDs []uint, user, method string) error {

	var existing Template
	var iroutes []Account
	result := m.db.Where("name = ?", template.Name).First(&existing)
	m.db.Preload("Routes").Where("id in ?", irouteIDs).Find(&iroutes)

	switch method {
	case "POST":
		if result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {

				return fmt.Errorf("查询模板失败: %v", result.Error)
			}
			// 创建新模板
			for _, iroute := range iroutes {
				template.Routes = append(template.Routes, iroute.Routes...)
			}
			if err := m.db.Create(&template).Error; err != nil {
				return fmt.Errorf("创建模板失败: %v", err)
			}
			m.logger.Printf("用户 %s 创建了模板: %s", user, template.Name)
		}

	case "PUT":
		if result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查询模板失败: %v", result.Error)
			}
		} else {
			// 更新现有模板
			template.ID = existing.ID
			if err := m.db.Save(&template).Error; err != nil {
				return fmt.Errorf("更新模板失败: %v", err)
			}
			// 先清空现有的关联
			if err := m.db.Model(&template).Association("Routes").Clear(); err != nil {
				return fmt.Errorf("清除模板IRoute关联失败: %v", err)
			}

			// 准备要关联的 Route 实例
			var routes []Route
			if err := m.db.Joins("JOIN account_routes ON account_routes.route_id = routes.id").
				Where("account_routes.account_id IN ?", irouteIDs).
				Preload("Accounts", "id IN ?", irouteIDs).
				Find(&routes).Error; err != nil {
				return fmt.Errorf("查询IRoute失败: %v", err)
			}
			// 建立新的关联
			if err := m.db.Model(&template).Association("Routes").Append(routes); err != nil {
				return fmt.Errorf("添加模板IRoute关联失败: %v", err)
			}
			// 更新所有使用此模板的账号配置
			m.logger.Printf("用户 %s 更新了模板: %s", user, template.Name)

			if err := m.updateAccountsByTemplate(template.ID); err != nil {
				return fmt.Errorf("更新模板关联的账号配置失败: %v", err)
			}
		}
	}

	return nil
}

// 删除模板
func (m *CCDManager) DeleteTemplate(id uint, user string) error {
	// 检查是否有账号使用此模板
	var count int64
	if err := m.db.Model(&Account{}).
		Joins("JOIN account_templates ON account_templates.account_id = accounts.id").
		Where("account_templates.template_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查模板使用情况失败: %v", err)
	}

	if count > 0 {
		//fmt.Println(count)
		return fmt.Errorf("模板正在被用户使用，无法删除")
	}

	var count2 int64
	if err := m.db.Model(&Route{}).
		Joins("JOIN template_routes ON template_routes.route_id = routes.id").
		Where("template_routes.template_id = ?", id).Count(&count2).Error; err != nil {
		return fmt.Errorf("检查模板使用情况失败: %v", err)
	}
	if count2 > 0 {
		return fmt.Errorf("模板正在被iroute用户关联使用，无法删除")
	}

	// 删除模板
	var template Template
	if err := m.db.Preload("Accounts").Preload("Routes").First(&template, id).Error; err != nil {
		return fmt.Errorf("查询模板失败: %v", err)
	}

	//// 清除与普通账号的关联
	//if err := m.db.Model(&template).Association("Assounts").Clear().Error; err != nil {
	//	return fmt.Errorf("清除模板普通账号关联失败: %v", err)
	//}
	//
	////清除与路由的关联
	//if err := m.db.Model(&template).Association("Routes").Clear().Error; err != nil {
	//	return fmt.Errorf("清除模板route关联失败: %v", err)
	//}

	if err := m.db.Unscoped().Delete(&template).Error; err != nil {
		return fmt.Errorf("删除模板失败: %v", err)
	}

	m.logger.Printf("用户 %s 删除了模板: %s", user, template.Name)
	return nil
}

// 获取模板
func (m *CCDManager) GetTemplate(id uint) (Template, error) {
	var template Template
	if err := m.db.Preload("Routes.Accounts", "is_iroute=? and enabled=?", true, true).First(&template, id).Error; err != nil {
		return template, err
	}
	return template, nil
}

// 获取所有模板
func (m *CCDManager) GetAllTemplates() ([]Template, error) {
	var templates []Template
	if err := m.db.Preload("Accounts").Preload("Routes.Accounts").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// 更新模板关联的所有账号配置
func (m *CCDManager) updateAccountsByTemplate(templateID uint) error {
	var accounts []Account
	if err := m.db.Preload("Templates", "id = ? ", templateID).Preload("Templates.Routes").
		Preload("Routes").Where("is_iroute=? and enabled = ?", false, true).Find(&accounts).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("无关联账号: %v", err)
		}
		return fmt.Errorf("获取模板关联的账号失败: %v", err)
	}

	for _, account := range accounts {
		if err := m.generateCCDConfig(account.Username); err != nil {
			return fmt.Errorf("更新账号 %s 配置失败: %v", account.Username, err)
		}
	}
	return nil
}

// 获取CCD文件内容
func (m *CCDManager) GetCCDConfigContent(username string) (string, error) {
	filePath := filepath.Join(m.ccdDir, username)

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取CCD文件失败: %v", err)
	}

	return string(content), nil
}
