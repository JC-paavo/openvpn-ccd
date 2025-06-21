package model

import "gorm.io/gorm"

// 账号模型
type Account struct {
	gorm.Model
	Username                string          `gorm:"unique;not null" json:"username"`
	Password                string          `gorm:"not null" json:"-"` // 不返回密码
	DisplayName             string          `gorm:"not null" json:"display_name"`
	Email                   string          `gorm:"unique;not null" json:"email"`
	Phone                   string          `gorm:"unique;not null" json:"phone"`
	IsIRoute                bool            `gorm:"default:false;column:is_iroute" json:"is_iroute"`
	Routes                  []Route         `gorm:"default:null;many2many:account_routes;" json:"routes"`
	Enabled                 bool            `gorm:"default:true" json:"enabled"`
	Templates               []Template      `gorm:"many2many:account_templates" json:"template,omitempty"`
	IRouteAccounts          map[uint]string `gorm:"-"` // 普通账号关联的IRoute账号列表
	ReferencedTemplateNames map[uint]string `gorm:"-"` // IRoute账号被哪些模板关联的模板名称列表
	//IRoutes     []IRoute `gorm:"many2many:account_iroutes;" json:"iroutes,omitempty"`
}

// IRoute模型
type Route struct {
	gorm.Model
	Route string `gorm:"unique;not null" json:"route"`
	//	AccountID *uint      `json:"account_id"`
	Accounts  []Account  `gorm:"many2many:account_routes" json:"accounts"`
	Templates []Template `gorm:"many2many:template_routes;" json:"templates,omitempty"`
}

// 模板模型
type Template struct {
	gorm.Model
	Name        string `gorm:"unique;not null" json:"name"`
	Description string `json:"description"`
	//Type        string    `json:"type"` // 运维管理员, 开发人员, 项目经理, 技术经理
	Routes      []Route   `gorm:"many2many:template_routes;" json:"iroutes,omitempty"`
	Accounts    []Account `gorm:"many2many:account_templates" json:"accounts,omitempty"`
	IRouteCount int       `gorm:"-"`
}

// 日志模型
type Log struct {
	gorm.Model
	User      string `json:"user"`
	Action    string `json:"action"`
	Details   string `json:"details"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent"`
}

// 关联模型
type AccountRoute struct {
	AccountID uint `gorm:"primaryKey"`
	RouteID   uint `gorm:"primaryKey"`
}

type TemplateRoute struct {
	TemplateID uint `gorm:"primaryKey"`
	RouteID    uint `gorm:"primaryKey"`
}

type AccountTemplate struct {
	AccountID  uint `gorm:"primaryKey"`
	TemplateID uint `gorm:"primaryKey"`
}
