# go-zero-admin

## 本地开发：启动依赖

在项目根目录执行，不需要 `.env` 或 `.env.example`：

```powershell
docker compose up -d
docker compose ps
```

根目录 `docker-compose.yml` **只包含 MySQL、Redis、etcd 和 Swagger UI，不包含 API/RPC 服务或部署 profile**。Docker Desktop 统一显示为 `go-zero-admin`。

| 服务 | 本机地址 | 开发配置 |
| --- | --- | --- |
| MySQL | `127.0.0.1:3306` | 用户 `root`，密码 `123456`，数据库 `goZero-admin` |
| Redis | `127.0.0.1:6379` | 无密码 |
| etcd | `127.0.0.1:2379` | 本地服务发现 |
| Swagger UI | [http://localhost:8080](http://localhost:8080) | 查看、调试接口 |

上述固定密码仅用于本机开发，配置中的端口仅绑定本机。MySQL 继续使用 `go-zero-admin_mysql_data` 数据卷；已有数据库密码不会因修改配置自动改变。空数据卷首次启动会导入 `data/db/gozero-admin-20240129.sql`，已有数据卷不重复导入。

```powershell
# 查看日志
docker compose logs --tail=100
# 暂停依赖，保留容器与数据
docker compose stop
# 恢复启动
docker compose up -d
```

不要用 `docker compose down -v` 作为停止命令，它会删除数据库数据。Docker只启动开发依赖；前端真实接口联调时，通过VS Code或本机分别运行RPC（6001）与API（7001）。

## 前后端接口接入

Vben前端的模块实现、开发规则、接口清单和验收状态见 [接口接入开发流程](../go-zero-admin-vben/DEVELOPMENT-WORKFLOW.zh-CN.md)。本轮保留原有示例页面，新增用户、菜单、角色、API权限、字典及工具页；新增2个角色按钮授权接口，当前共43个HTTP接口。代码已接入，完整HTTP CRUD和新登录回归尚未完成，验证码操作的单独确认仍待回复；OSS/SMTP未配置，真实上传与邮件送达暂跳过。

已有数据库先备份权限相关表及 `sys_users`，再依次执行 [权限增量初始化](data/db/migrations/20260925_business_access.sql) 和 [活跃用户名唯一约束](data/db/migrations/20260925_user_active_username.sql)：

```powershell
docker cp .\data\db\migrations\20260925_business_access.sql gozero-mysql:/tmp/20260925_business_access.sql
docker cp .\data\db\migrations\20260925_user_active_username.sql gozero-mysql:/tmp/20260925_user_active_username.sql
docker exec -it gozero-mysql mysql --default-character-set=utf8mb4 -u root -p goZero-admin
```

进入MySQL后依次执行：

```sql
SOURCE /tmp/20260925_business_access.sql;
SOURCE /tmp/20260925_user_active_username.sql;
EXIT;
```

第一份脚本补齐缺少的资源、admin当前角色权限及历史基础权限修正；第二份添加活跃用户名唯一约束，保留软删除后的用户名复用能力。检测到已有活跃重名或不兼容表结构时会停止，不改名、不删除用户数据。两份脚本均可重跑，本机均已备份后连续执行两遍验证；用户名脚本含DDL，部分执行失败时修正原因后重跑。完成后重启RPC、刷新前端权限，详细说明见开发流程第7节。

## 接口文档：访问、使用和更新

- 浏览器文档：[http://localhost:8080](http://localhost:8080)。JSON：[http://localhost:8080/swagger.json](http://localhost:8080/swagger.json)。
- 当前文档文件：[go-zero-admin.swagger.json](data/api/generated/go-zero-admin.swagger.json)，也可直接导入 Apifox。旧的 `data/api/go-zero-admin.openapi.json` 为历史文档，不作为最新联调依据。
- 查看接口不需要 API/RPC 运行。点击 **Try it out → Execute** 时，需要启动真实后端及其依赖；默认通过文档容器转发到宿主机 `7001` 端口，后端未运行时可能返回 502。
- 需要登录的接口：先调用登录接口取得 accessToken，在 **Authorize** 中填写完整的 `Bearer <accessToken>`。
- 成功响应使用 `code/message/result/returnData/success/timestamp`。业务错误和 JWT 失效可能返回 HTTP 200，需检查业务码；GET参数使用query。验证码接口返回captchaId和图片，登录必须携带captchaId及输入的验证码。

新增或修改接口后，在项目根目录重新生成：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\swagger.ps1
```

脚本兼容 Windows PowerShell 5.1，不需要安装 `pwsh`。需有 Go 和 goctl；当前验证版本可通过 `go install github.com/zeromicro/go-zero/tools/goctl@v1.10.2` 安装。脚本只生成文档，不修改业务源码。

更新流程：修改 `.api` 和接口实现 → 运行生成脚本 → 刷新 Swagger 网页 → 核对方法、路径、参数及响应 → 提交源码和生成的 JSON。不要手动修改生成 JSON；响应包装、鉴权或上传约定变化时同步调整 `test/sh/swagger.ps1`。单纯更新 JSON 无需重启容器。

如需调整文档端口或 API 目标，直接修改根目录 Compose 中 Swagger 的端口和 `SWAGGER_API_URL`，再执行：

```powershell
docker compose up -d --no-deps --force-recreate swagger-ui
```

API 目标填写容器可达的协议、主机和端口，不带路径或末尾斜杠。Linux 上仅监听宿主机 127.0.0.1 的 API 不能通过 host-gateway 访问。

## 服务器部署

部署单独使用 [docker/deploy-compose.yml](docker/deploy-compose.yml)，包含 API/RPC 和依赖，项目名为 `go-zero-admin-deploy`。与根目录开发配置分开，不要用开发密码部署服务器。具体配置、打包、启动和更新步骤见 [部署说明](docker/部署说明.md)。

#### 后端：[https://github.com/yh-zero/go-zero-admin](https://github.com/yh-zero/go-zero-admin)
#### 前端：[https://github.com/yh-zero/go-zero-admin-vue3](https://github.com/yh-zero/go-zero-admin-vue3)

####  求兄弟们点一下 右上角的星星 star 谢谢 ~
####  求兄弟们点一下 右上角的星星 star 谢谢 ~
####  求兄弟们点一下 右上角的星星 star 谢谢 ~

------



##### 历史 Apifox 项目参考（最新接口使用上方生成的 Swagger）

```sh
#go__zero 在 Apifox 中邀请你加入团队 go-zero-admin 可以加入项目一起测试开发
https://app.apifox.com/invite?token=TMOproaI5ycYALoINDuOI
```

#### 项目相关：

##### Docker 部署

部署文件和中文操作说明统一放在 `docker/` 目录。请先阅读 [Docker 部署说明](docker/部署说明.md)。

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

#### 权限维护

通过角色管理的菜单、接口和按钮授权入口维护权限。缺少初始化资源时使用上面的增量脚本；收到403时核对角色、接口路径和HTTP方法，不在鉴权中间件中按请求自动添加权限。


##### 角色管理：

<img src="./data/doc/authority.png" alt="角色管理"  />

##### 菜单管理：

<img src="./data/doc/menu.png" alt="菜单管理"  />

##### API管理：

<img src="./data/doc/api.png" alt="api管理"  />

##### 用户管理：

<img src="./data/doc/user.png" alt="用户管理"  />

##### 字典管理：

<img src="./data/doc/dictionary.png" alt="字典管理"  />

##### 字典使用例子：

<img src="./data/doc/dictionary-usage.png" alt="字典使用例子"  />


#### goctl 模板说明（test/goctl/）

当前生成脚本使用 `--home test/goctl`。以下两个版本无关模板提供自定义行为；`1.10.3/api/` 中保留同内容的版本快照。升级goctl后应先用临时API验证生成结果，再生成业务代码。

| 文件 | 自定义内容 |
| --- | --- |
| test/goctl/api/handler.tpl | 统一响应封装：`result.HttpResult(r, w, resp, err)` 替换默认的 httpx.ErrorCtx/OkJsonCtx 分支 |
| test/goctl/api/main.tpl | JWT 过期统一响应：`rest.WithUnauthorizedCallback` 返回 `xerr.TOKEN_EXPIRE_ERROR` |

```sh
###### 生成代码时指定模板（test/sh/api.bat、rpc.bat 里的 --home 同步修改即可）：
goctl api go -api xxx.api -dir . --home test/goctl
```

# 微信：qq1013055366  欢迎打扰(备注：go-zero-admin)
