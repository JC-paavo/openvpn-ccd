package main

import (
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"log"
	"openvpn-ccd/Router"
	"openvpn-ccd/model"
	"openvpn-ccd/utils"
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

	mode := flag.String("mode", "web", "web|cmd")
	username := flag.String("username", "", "username")
	password := flag.String("password", "", "password")
	help := flag.Bool("help", false, "help")

	flag.Usage = func() {
		flag.PrintDefaults()
	}
	flag.Parse()
	if help != nil && *help {
		flag.Usage()
		return
	}

	switch *mode {
	case "web":
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
	case "cmd":
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "openvpn_ccd.db"
		}
		// 连接数据库
		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			fmt.Println("连接数据库失败:", err)
			os.Exit(1)
		}
		if *username == "" || *password == "" {
			log.Println("username and password must be set")
			os.Exit(1)
		}
		account := &model.Account{}
		if err := db.Where("username =?", *username).First(account).Error; err != nil {
			log.Println("account not found:", *username)
			os.Exit(1)
		}
		if res := utils.VerifyPassword(account.Password, *password); !res {
			log.Println("密码错误:", *username)
			os.Exit(1)
		}
		log.Println("账号验证成功:", *username)
		os.Exit(0)
	default:
		fmt.Println("mode must be web or cmd")
		return
	}

}
