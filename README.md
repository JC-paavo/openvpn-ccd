# OpenVPN CCD 配置管理系统

## 项目概述
OpenVPN CCD 配置管理系统是一个用于管理 OpenVPN 客户端特定配置（CCD）的 Web 应用程序。它允许管理员创建、编辑和管理账号、模板以及相关路由配置，支持 IRoute 账号和普通账号的区分管理。

## 功能特性
- **账号管理**：支持创建、编辑 IRoute 账号和普通账号，管理账号信息、路由和关联模板。
- **模板管理**：允许创建、编辑模板，关联 IRoute 路由，方便批量配置。
- **配置生成**：根据账号信息和关联模板自动生成 CCD 配置文件。
- **日志记录**：记录用户操作和系统请求信息，便于审计和故障排查。

## 项目结构
```
openvpn-ccd/
├── .env                # 环境变量配置文件
├── .gitignore          # Git 忽略文件
├── Controller/         # 控制器层，处理业务逻辑
│   ├── account.go
│   ├── login.go
│   └── template.go
├── Middle/             # 中间件层，处理请求拦截和日志记录
│   └── initMiddle.go
├── Router/             # 路由层，定义路由规则
│   ├── Static.go
│   ├── account.go
│   ├── init.go
│   ├── login.go
│   ├── route.go
│   └── template.go
├── model/              # 数据模型层，定义数据库模型和操作
│   ├── ccdoper.go
│   └── model.go
├── static/             # 静态资源目录
│   ├── css/
│   └── js/
├── templates/          # 模板文件目录，存放 HTML 模板
│   ├── accounts.html
│   ├── add_account.html
│   ├── add_template.html
│   ├── edit_account.html
│   ├── edit_template.html
│   ├── error.html
│   ├── index.html
│   ├── login.html
│   ├── logs.html
│   └── templates.html
├── go.mod              # Go 模块依赖文件
├── go.sum              # Go 模块依赖校验文件
└── main.go             # 项目入口文件
```
## 安装步骤

### 1. 克隆项目
```bash
git clone https://github.com/your-repo/openvpn-ccd.git
cd openvpn-ccd
go mod tidy
```

### 2. 配置
编辑.env
```bash
ADMIN_USER=admin
ADMIN_PASS=admin
CCD_DIR=ccd
SESSION_SECRET=1234
```

### 3. 启动服务
```bash
go run main.go -mode web
```
oepnvpn 验证账号密码
```bash
go run main.go -mode cmd -user USERNAME -pass PASSWORD
```

### 4. 访问
访问 URL_ADDRESS访问 http://localhost:8080 ，使用管理员账号登录。



