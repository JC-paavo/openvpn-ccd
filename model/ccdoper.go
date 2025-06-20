package model

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"log"
	"os"
	"path/filepath"
	"regexp"
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
func (m *CCDManager) CreateOrUpdateAccount(account Account, user string) error {
	// 验证邮箱格式
	if !validateEmail(account.Email) {
		return fmt.Errorf("邮箱格式不正确")
	}

	// 验证手机号格式
	if !validatePhone(account.Phone) {
		return fmt.Errorf("手机号格式不正确")
	}

	// 验证IRoute格式（如果是IRoute账号）
	if account.IsIRoute {
		for _, route := range account.Routes {
			if !validateCIDR(route.Route) {
				return fmt.Errorf("IRoute路由格式不正确，应为 'IP地址 子网掩码'")
			}
		}
	}

	// 加密密码
	if account.Password != "" {
		hashedPassword, err := hashPassword(account.Password)
		if err != nil {
			return fmt.Errorf("密码加密失败: %v", err)
		}
		account.Password = hashedPassword
	}

	var existing Account
	result := m.db.Where("username = ?", account.Username).First(&existing)

	if result.Error != nil {
		if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询账号失败: %v", result.Error)
		}
		// 创建新账号
		if err := m.db.Create(&account).Error; err != nil {
			return fmt.Errorf("创建账号失败: %v", err)
		}
		m.logger.Printf("用户 %s 创建了账号: %s", user, account.Username)
	} else {
		// 更新现有账号
		account.ID = existing.ID
		if account.Password == "" {
			account.Password = existing.Password // 保持原有密码
		}
		if err := m.db.Save(&account).Error; err != nil {
			return fmt.Errorf("更新账号失败: %v", err)
		}
		m.logger.Printf("用户 %s 更新了账号: %s", user, account.Username)
	}

	// 管理IRoute关联
	// 清除旧的关联
	if err := m.db.Where("account_id = ?", account.ID).Delete(&AccountRoute{}).Error; err != nil {
		return fmt.Errorf("清除账号IRoute关联失败: %v", err)
	}

	// 添加新的关联
	if len(account.Templates) != 0 {
		// 从模板获取IRoutes
		var template Template
		for _, tpl := range account.Templates {
			if err := m.db.Preload("Routes").First(&template, tpl.ID).Error; err != nil {
				return fmt.Errorf("获取模板失败: %v", err)
			}
		}

		for _, ir := range template.Routes {
			accountIRoute := AccountRoute{
				AccountID: account.ID,
				RouteID:   ir.ID,
			}
			if err := m.db.Create(&accountIRoute).Error; err != nil {
				return fmt.Errorf("添加账号IRoute关联失败: %v", err)
			}
		}
	}

	// 生成并写入CCD配置文件
	if account.Enabled {
		return m.generateCCDConfig(account)
	} else {
		return m.deleteCCDConfig(account.Username)
	}
}

// 删除账号
func (m *CCDManager) DeleteAccount(username string, user string) error {
	var account Account
	if err := m.db.Preload("Routes").Where("username = ?", username).First(&account).Error; err != nil {
		return fmt.Errorf("查询账号失败: %v", err)
	}

	// 删除账号
	if err := m.db.Delete(&account).Error; err != nil {
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
	if err := m.db.Preload("Routes").Preload("Templates").Where("username = ?", username).First(&account).Error; err != nil {
		return account, err
	}
	return account, nil
}

// 获取所有账号
func (m *CCDManager) GetAllAccounts() ([]Account, error) {
	var accounts []Account
	if err := m.db.Preload("Routes").Preload("Templates").Find(&accounts).Error; err != nil {
		return nil, err
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
func (m *CCDManager) GetAllIRoutes() ([]Account, error) {
	var iroutes []Account
	if err := m.db.Preload("Routes").Where("is_iroute=? and enabled=?", true, true).Find(&iroutes).Error; err != nil {
		return nil, err
	}
	return iroutes, nil
}

// 生成CCD配置文件
func (m *CCDManager) generateCCDConfig(account Account) error {
	configPath := m.getConfigPath(account.Username)
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("创建配置文件失败: %v", err)
	}
	defer file.Close()

	// IRoute账号配置
	if account.IsIRoute {
		for _, route := range account.Routes {
			if _, err := file.WriteString(fmt.Sprintf("iroute %s\n", route)); err != nil {
				return fmt.Errorf("写入配置文件失败: %v", err)
			}
		}

		return nil
	}

	// 普通账号配置 - 添加路由和关联iroute
	if len(account.Routes) != 0 {
		for _, route := range account.Routes {
			if _, err := file.WriteString(fmt.Sprintf("push \"route %s\"\n", route)); err != nil {
				return fmt.Errorf("写入配置文件失败: %v", err)
			}
		}
	}

	// 添加账号关联的所有IRoute路由
	var iroutes []Route
	if err := m.db.Model(&account).Association("Routes").Find(&iroutes); err != nil {
		return fmt.Errorf("获取账号关联的IRoute路由失败: %v", err)
	}

	for _, iroute := range iroutes {
		if _, err := file.WriteString(fmt.Sprintf("push \"route %s\"\n", iroute.Route)); err != nil {
			return fmt.Errorf("写入配置文件失败: %v", err)
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

// 密码加密
func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

// 验证密码
func verifyPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
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
	result := m.db.Where("name = ?", template.Name).First(&existing)

	switch method {
	case "POST":
		if result.Error != nil {
			if !errors.Is(result.Error, gorm.ErrRecordNotFound) {
				return fmt.Errorf("查询模板失败: %v", result.Error)
			}
			// 创建新模板
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
			m.logger.Printf("用户 %s 更新了模板: %s", user, template.Name)
		}
	}

	// 清除旧的IRoute关联
	if err := m.db.Where("template_id = ?", template.ID).Delete(&TemplateRoute{}).Error; err != nil {
		return fmt.Errorf("清除模板IRoute关联失败: %v", err)
	}

	// 添加新的IRoute关联
	for _, irouteID := range irouteIDs {
		templateIRoute := TemplateRoute{
			TemplateID: template.ID,
			RouteID:    irouteID,
		}
		if err := m.db.Create(&templateIRoute).Error; err != nil {
			return fmt.Errorf("添加模板IRoute关联失败: %v", err)
		}
	}

	// 更新所有使用此模板的账号配置
	if err := m.updateAccountsByTemplate(template.ID); err != nil {
		return fmt.Errorf("更新模板关联的账号配置失败: %v", err)
	}

	return nil
}

// 删除模板
func (m *CCDManager) DeleteTemplate(id uint, user string) error {
	// 检查是否有账号使用此模板
	var count int64
	if err := m.db.Model(&Account{}).Where("template_id = ?", id).Count(&count).Error; err != nil {
		return fmt.Errorf("检查模板使用情况失败: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("模板正在被使用，无法删除")
	}

	// 删除模板
	var template Template
	if err := m.db.First(&template, id).Error; err != nil {
		return fmt.Errorf("查询模板失败: %v", err)
	}

	if err := m.db.Delete(&template).Error; err != nil {
		return fmt.Errorf("删除模板失败: %v", err)
	}

	// 清除模板IRoute关联
	if err := m.db.Where("template_id = ?", id).Delete(&TemplateRoute{}).Error; err != nil {
		return fmt.Errorf("清除模板IRoute关联失败: %v", err)
	}

	m.logger.Printf("用户 %s 删除了模板: %s", user, template.Name)
	return nil
}

// 获取模板
func (m *CCDManager) GetTemplate(id uint) (Template, error) {
	var template Template
	if err := m.db.Preload("Routes").First(&template, id).Error; err != nil {
		return template, err
	}
	return template, nil
}

// 获取所有模板
func (m *CCDManager) GetAllTemplates() ([]Template, error) {
	var templates []Template
	if err := m.db.Preload("Routes").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// 更新模板关联的所有账号配置
func (m *CCDManager) updateAccountsByTemplate(templateID uint) error {
	var accounts []Account
	if err := m.db.Where("template_id = ? AND enabled = ?", templateID, true).Find(&accounts).Error; err != nil {
		return fmt.Errorf("获取模板关联的账号失败: %v", err)
	}

	for _, account := range accounts {
		if err := m.generateCCDConfig(account); err != nil {
			return fmt.Errorf("更新账号 %s 配置失败: %v", account.Username, err)
		}
	}

	return nil
}
