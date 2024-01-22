# go-zero-admin 

#### 后端：[https://github.com/yh-zero/go-zero-admin](https://github.com/yh-zero/go-zero-admin)
#### 前端：[https://github.com/yh-zero/go-zero-admin-vue3](https://github.com/yh-zero/go-zero-admin-vue3)

####  求兄弟们点一下 右上角的星星 star 谢谢 ~
####  求兄弟们点一下 右上角的星星 star 谢谢 ~
####  求兄弟们点一下 右上角的星星 star 谢谢 ~

------



##### 接口请求: 欢迎一起开发一起学习冲冲冲： （接口文件：data/api/go-zero-admin.openapi.json   可以使用apifox软件导入运行）

```sh
#go__zero 在 Apifox 中邀请你加入团队 go-zero-admin 可以加入项目一起测试开发
https://app.apifox.com/invite?token=TMOproaI5ycYALoINDuOI
```

#### 项目相关：

##### 启动项目： (先运行sql文件 文件在 data/db/gozero-admin-20240116.sql 运行时间最新的文件) 

##### 启动流程：（先启动redis,etcd）

##### 1-1、启动rpc： cd .\application\applet\rpc\

##### 1-2、启动rpc：  go run .\applet.go

##### 2-1、启动api： cd .\application\applet\api\

##### 2-2、启动api：  go run .\applet.go

#### 端口定义

| 类型          | 端口号    | 描述                    |
| ------------- | --------- | ----------------------- |
| applet-api    | 7001      | api的的端口号从7001开始 |
|               |           |                         |
| -   -   -   - | -   -   - | -   -   -   -           |
| applet-rpc    | 6001      | rpc的的端口号从6001开始 |

#### 代码生成例子：

```sh
######  gorm 生成对应的结构体： 先运行安装
go install gorm.io/gen/tools/gentool@latest

# 生成命令（只生产 struct）：gentool -dsn "root:123456@tcp(127.0.0.1:3306)/go-zero-admin?charset=utf8mb4&parseTime=True&loc=Local" -tables "sys_users" -onlyModel -outPath application\applet\rpc\internal\mod

gentool -dsn "root:123456@tcp(127.0.0.1:3306)/go-zero-admin?charset=utf8mb4&parseTime=True&loc=Local" -tables "sys_users" -onlyModel

###### api生成 项目根目录 运行例子： 
.\test\sh\api.bat applet

###### rpc 生成 项目根目录 运行例子： 
 .\test\sh\rpc.bat applet applet
```

#### 代码格式化

```shell
##### 格式化单个 api： 
goctl api format --dir .\user.api

##### 格式化全部 api（根目录运行）: 
goctl api format --dir .\
```

#### gorm 开发注意 (在api传值到rpc里面数据会有deleted_at零值，所以需要去掉)

```go
// 使用 Create 创建数据的时候 // 删除时间问题 0000-00-00 00:00:00.000 两种解决方法

// 1.注意添加.Omit("deleted_at")
l.svcCtx.DB.Omit("deleted_at").Create(&baseMenu)

// 2.model.DeletedAt.Valid = false


```

#### 开发注意： 

```go
// 中间件casbin: application/applet/api/internal/middleware/authority_middleware.go 
_, _ = CasB.AddPolicy(authorityId, path, method) // 如果权限数据不小心清了 把这个开启  然后api连续请求两次就会有权限  最后重新设置权限即可
```

# 微信：qq1013055366  欢迎打扰(备注：go-zero-admin)
