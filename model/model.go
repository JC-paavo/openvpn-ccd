package model

import "gorm.io/gorm"

// 账号模型
type Account struct {
	gorm.Model
	Username    string   `gorm:"unique;not null" json:"username"`
	Password    string   `gorm:"not null" json:"-"` // 不返回密码
	DisplayName string   `gorm:"not null" json:"display_name"`
	Email       string   `gorm:"unique;not null" json:"email"`
	Phone       string   `gorm:"unique;not null" json:"phone"`
	IsIRoute    bool     `gorm:"default:false;column:is_iroute" json:"is_iroute"`
	Route       []Route  `gorm:"default:null;foreignKey:AccountID" json:"route"`
	Enabled     bool     `gorm:"default:true" json:"enabled"`
	TemplateID  *uint    `json:"template_id"`
	Template    Template `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	//IRoutes     []IRoute `gorm:"many2many:account_iroutes;" json:"iroutes,omitempty"`
}

// IRoute模型
type Route struct {
	gorm.Model
	Route     string     `gorm:"unique;not null" json:"route"`
	AccountID *uint      `json:"account_id"`
	Account   Account    `gorm:"foreignKey:AccountID" json:"account,omitempty"`
	Templates []Template `gorm:"many2many:template_iroutes;" json:"templates,omitempty"`
}

// 模板模型
type Template struct {
	gorm.Model
	Name        string    `gorm:"unique;not null" json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"` // 运维管理员, 开发人员, 项目经理, 技术经理
	Routes      []Route   `gorm:"many2many:template_iroutes;" json:"iroutes,omitempty"`
	Accounts    []Account `gorm:"foreignKey:TemplateID" json:"accounts,omitempty"`
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
type AccountIRoute struct {
	AccountID uint `gorm:"primaryKey"`
	IRouteID  uint `gorm:"primaryKey"`
}

type TemplateIRoute struct {
	TemplateID uint `gorm:"primaryKey"`
	IRouteID   uint `gorm:"primaryKey"`
}
