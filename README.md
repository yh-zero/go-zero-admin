# go-zero-admin

## 接口请求: 欢迎一起开发一起学习冲冲冲： 
### go__zero 在 Apifox 中邀请你加入团队 go-zero-admin https://app.apifox.com/invite?token=IRQ9GQqcYr88i3b0pJao-
### 微信：qq1013055366  欢迎打扰

#### 介绍

### 启动项目： (先运行sql语句 文件在 db/go-zero-admin.sql)
#### 1-1启动rpc： cd .\application\applet\rpc\
#### 1-2启动rpc：  go run .\applet.go

#### 2-1启动api： cd .\application\applet\api\
#### 2-2启动api：  go run .\applet.go


#### 后端规范：

##### 返回数据规范：result:{name: "nanLi"} - 对前端后端都友好

##### 不推荐直接返回数据如：result： "nanLi"

### 端口定义

#### api 端口 7001 开始
##### applet-api: 7001


#### rpc 端口 6001 开始
##### applet.rpc: 6001

#### 代码生成例子：

##### gorm 生成对应的结构体：

###### 安装： go install gorm.io/gen/tools/gentool@latest

###### 生成命令（只生产 struct）：gentool -dsn "root:123456@tcp(127.0.0.1:3306)/performance_gozero_system?charset=utf8mb4&parseTime=True&loc=Local" -tables "sys_users" -onlyModel -outPath application\applet\rpc\internal\mod
gentool -dsn "root:123456@tcp(127.0.0.1:3306)/mm_system?charset=utf8mb4&parseTime=True&loc=Local" -tables "sys_users" -onlyModel

##### api 生成
###### 项目根目录 运行例子 1： .\test\sh\api.bat applet

##### rpc 生成
###### 项目根目录 运行例子： .\test\sh\rpc.bat applet applet

#### 代码格式化
##### 格式化单个 api： goctl api format --dir .\user.api
##### 格式化全部 api（根目录运行）: goctl api format --dir .\

#### gorm 开发注意

##### 使用 Create 创建数据的时候 // 删除时间问题 0000-00-00 00:00:00.000 两种解决方法

##### 1.注意添加.Omit("deleted_at")

##### 2.model.DeletedAt.Valid = false

##### 例如 l.svcCtx.DB.Omit("deleted_at").Create(&baseMenu)

## 注意： 中间件 这个需要删除 _, _ = CasB.AddPolicy(authorityId, path, method) // 临时添加保存通过 方便开发
