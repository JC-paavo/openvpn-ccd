package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"openvpn-ccd/Router"
	"openvpn-ccd/model"
	"os"
)

// 加载环境变量
func init() {
	if err := godotenv.Load(); err != nil {
		log.Println("未找到.env文件，使用默认配置")
	}
}

// 中间件：记录请求日志

func main() {
	// 加载配置
	ccdDir := os.Getenv("CCD_DIR")
	if ccdDir == "" {
		ccdDir = "/etc/openvpn/ccd"
	}
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "openvpn_ccd.db"
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	// 初始化日志
	logger := log.New(os.Stdout, "[OpenVPN-Manager] ", log.LstdFlags)

	// 连接数据库
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		logger.Fatalf("连接数据库失败: %v", err)
	}

	// 自动迁移模型
	if err := db.AutoMigrate(
		&model.Account{},
		&model.Route{},
		&model.Template{},
		&model.Log{},
		&model.AccountRoute{},
		&model.TemplateRoute{},
		&model.AccountTemplate{},
	); err != nil {
		logger.Fatalf("数据库迁移失败: %v", err)
	}
	// 创建CCD管理器
	ccdManager := model.NewCCDManager(ccdDir, db, logger)
	// 创建Gin引擎
	r := gin.Default()

	Router.InitRoute(r, ccdManager, db, logger)

	// 启动服务器
	logger.Printf("服务器启动在端口 %s", port)
	logger.Printf("CCD目录: %s", ccdDir)
	logger.Printf("数据库路径: %s", dbPath)
	if err := r.Run(":" + port); err != nil {
		logger.Fatalf("启动服务器失败: %v", err)
	}
}
