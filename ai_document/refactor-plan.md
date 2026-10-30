# go-zero-admin 后端重构方案

> 版本：v1.0（2026-10-29）
> 状态：已确认需求，待落地
> 范围：后端全新重构；前端兼容性不作约束；数据库全面重设计

---

## 1. 背景与目标

现有后端（`go-zero-admin`）基于 go-zero + GORM + Casbin，存在以下需要重构的点：

- 数据层 GORM 与 goctl model（sqlx）并存 → **统一为 goctl model（go-zero sqlx + sqlc 缓存包）**
- `data/goctl/` 下修改了官方生成模板（handler/main 注入自定义响应信封）→ **恢复官方默认模板**
- 自定义 `{code,message,result}` 响应信封 → **改用 go-zero 官方推荐的错误处理方式**
- 业务 RPC 为单体（112 个方法）→ **拆分为多 RPC**
- 目录结构对齐 go-zero 规范，尽量沿用现有 `application/` 风格，保持简洁
- 新增多租户能力：子域名访问、每租户独立整套部署、独立数据库

## 2. 已确认决策

| 项 | 决策 | 影响 |
|---|---|---|
| 多租户部署形态 | **每租户独立整套栈**（docker compose 一套/租户，独立 MySQL/Redis） | 租户隔离在基础设施层，不占 Casbin 策略维度；`dom` 槽位留给**部门**（见 3.2） |
| 租户访问入口 | **子域名**（`<slug>.example.com`），Nginx 泛域名路由 | 宿主只暴露 Nginx 80/443；租户栈不暴露端口 |
| 租户创建入口 | **新建 tenant-rpc，在平台后台页面操作** | 平台栈 = 第 0 号租户 + tenant-rpc + docker socket / nginx 配置卷 |
| HTTP 响应格式 | **go-zero 官方默认**：成功直返数据体；错误由 `httpx.SetErrorHandlerCtx` 统一输出 | 删除自定义模板与信封；swagger 用官方原生输出 |
| 服务发现 | **直连 Endpoints，去掉 etcd** | compose 少一个容器；配置写死 RPC 地址 |
| 数据库迁移 | **goose** | `data/db/migrations/*.sql` 是结构唯一事实源；goctl model 从迁移后的开发库反射生成 |
| 数据层 | **goctl model + `--cache true`**（go-zero `sqlx`/`sqlc` 缓存包） | 弃 GORM；CRUD 零 SQL、自带 Redis 行级缓存；复杂查询手写 sqlx |
| 功能范围 | 全部保留：核心 RBAC、字典、组织（部门/岗位）、文件管理、审计日志、AI 助手、App 用户/会员、设备会话。**确认废弃**：权限版本历史/回滚、IP 归属地 | 简化落在接口与表结构重设计 |

## 3. 总体架构：1 API + 5 RPC（双层）

```
                        ┌──────────── 控制面（平台栈，仅一套）────────────┐
  platform.example.com  │  api + 5×rpc + mysql + redis + nginx(:80/:443)   │
      （平台管理台）     │  tenant-rpc 挂载 docker socket 与 nginx 配置卷  │
                        └───────────────┬────────────────────────────────┘
                                        │ 创建/启停/删除租户
        ┌───────────────────────────────┼────────────────────────────────┐
        │ Nginx（唯一对外入口 :80/:443，泛域名 server 块由 tenant-rpc 生成） │
        └───┬──────────────────┬──────────────────┬─────────────────────┘
   t1.example.com       t2.example.com        t3.example.com
   ┌─────────────┐      ┌─────────────┐        ┌─────────────┐
   │ 租户1 整套栈  │      │ 租户2 整套栈  │        │ 租户3 整套栈  │
   │ api + 4rpc  │      │ api + 4rpc  │        │ api + 4rpc  │
   │ mysql+redis │      │ mysql+redis │        │ mysql+redis │
   └─────────────┘      └─────────────┘        └─────────────┘
   （compose 项目 gozero-t-<slug>，独立卷，不对宿主暴露端口）
```

### 3.1 RPC 拆分

| RPC | 内部端口 | 职责 |
|---|---|---|
| `admin-rpc` | 6001 | 认证、用户、角色、菜单（含按钮权限）、API 资源、Casbin 鉴权、组织（部门/岗位）、字典、审计、设备会话、图片验证码 |
| `app-rpc` | 6002 | App 用户注册/审核、会员开通、App 会话（独立 JWT 体系）、邮件验证码（SMTP） |
| `file-rpc` | 6003 | 文件资源、RustFS、头像与引用 |
| `ai-rpc` | 6004 | AI 助手（默认关闭，对接 DeepSeek/通义） |
| `tenant-rpc` | 6005 | 租户 provisioning / 启停 / 删除；**仅平台栈部署** |

> 更名说明：`admin-rpc` 即原单体 `applet-rpc` 的后台主体部分（认证/用户/角色/菜单/权限/组织/字典/审计/设备会话），App、文件、AI 职责分别拆入 app-rpc、file-rpc、ai-rpc。

- 平台栈为**第 0 号租户**：部署全部 5 个 RPC + Nginx，由人工 compose 部署，tenant-rpc 不管理平台栈自身。
- API 服务（:7001，栈内）通过直连 Endpoints 调用上述 RPC。
- 租户管理路由只出现在平台栈 API 配置中；普通租户栈不部署 tenant-rpc，对应路由返回 404。
- JWT：后台与 App 双体系保留，密钥各自独立且每租户随机生成。

### 3.2 鉴权设计：Casbin（接口级）+ SQL 数据范围（行级）

两类部门需求分工：

- **B：角色按部门生效（管理边界）** → Casbin domains 模型，`dom = 部门`；
- **A：部门内资源共享（行级数据可见性）** → SQL 数据范围过滤，按角色的 `data_scope` 生成部门集合条件。

Casbin 模型：

```ini
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (g(r.sub, p.sub, r.dom) || g(r.sub, p.sub, "*")) && (r.dom == p.dom || p.dom == "*") && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

约定：

- 部门域写作 `dept:<id>`；保留域 `*` 表示全局（跨部门）角色；
- 全局角色（如超级管理员）：`g, user:1, sys_admin, *` + `p, sys_admin, *, /v1/*, *`；
- 部门角色（如部门管理员）：`g, user:42, dept_admin, dept:7` + `p, dept_admin, dept:7, /v1/sys/user/*, GET`，只在部门 7 上下文生效；
- `casbin_rule` 表：p 规则为 `v0=sub, v1=dom, v2=obj, v3=act`；g 规则为 `v0=user, v1=role, v2=dom`；
- 租户隔离仍在数据库层（每租户独立库），`dom` 只承载部门，语义单一。

请求部门上下文：

- JWT 携带用户主部门；多部门用户用请求头 `X-Dept-Id` 切换操作上下文；
- Authority 中间件先校验用户对该部门的成员关系（`sys_user_dept`，内存缓存），再以 `r.dom = dept:<id>` 调 enforce；
- 与部门无关的接口（登录、个人中心等）只过 JWT，不进 Authority 中间件。

行级数据范围（需求 A）：

- 角色含 `data_scope`：`ALL` / `DEPT_AND_CHILDREN` / `DEPT` / `SELF`；
- model 层按 scope 注入部门集合条件（生成物之外的复杂查询手写 sqlx）；部门树用**物化路径** `path`（如 `/1/7/12/`），"及以下" = `path LIKE '/1/7/%'`，无需递归查询；
- Casbin 不管行级：它决定"能不能调接口"，SQL 决定"返回哪些行"。

性能方案（不变）：enforcer 驻留各租户 API 进程内存、启动时从本租户库全量加载；策略变更经 **casbin 官方 `redis-watcher`**（旧项目已验证组件）广播自动重载——admin-rpc 只负责写 `casbin_rule`，传播由 watcher 完成，无需自定义同步包。

## 4. 目录结构规范

`application/` 下按**业务模块**分目录，模块内再分 `api/` 与 `rpc/`（沿用原项目 `application/applet/api`、`application/applet/rpc` 的约定）。`admin` 模块同时含 `api/`（后台网关）与 `rpc/`（主业务 RPC）；其余模块当前只有 `rpc/`，日后需要独立 API 时在其目录下新增 `api/` 即可（如 `application/app/api` 承载 C 端接口），模块内聚、互不影响。

goctl 生成的骨架（handler/logic/server/types/pb/client/model_gen）不做模板修改，业务代码只写 logic 与 model 扩展。

```
go-zero-admin/
├── application/                              # 业务模块根：一模块一目录
│   ├── admin/                                # 管理端模块（后台 API + 主业务 RPC）
│   │   ├── api/                              # 当前唯一 REST 网关（:7001，栈内）
│   │   │   ├── admin.go                      # main：官方生成 + SetErrorHandlerCtx 注册
│   │   │   ├── etc/
│   │   │   │   └── admin.yaml                # Host/Port、JWT、各 RPC Endpoints、Casbin、限流
│   │   │   ├── desc/                         # .api 源文件（不参与编译）
│   │   │   │   ├── admin.api                 # 入口：info + import 各分组
│   │   │   │   ├── sys/
│   │   │   │   │   ├── auth.api              # 登录/登出/验证码/刷新令牌
│   │   │   │   │   ├── profile.api           # 个人中心（资料/改密/头像）
│   │   │   │   │   ├── user.api              # 用户管理
│   │   │   │   │   ├── role.api              # 角色管理
│   │   │   │   │   ├── menu.api              # 菜单与按钮
│   │   │   │   │   ├── api_resource.api      # API 资源
│   │   │   │   │   ├── org.api               # 部门/岗位
│   │   │   │   │   ├── dict.api              # 字典
│   │   │   │   │   ├── file.api              # 文件资源
│   │   │   │   │   ├── audit.api             # 审计日志
│   │   │   │   │   └── session.api           # 设备会话
│   │   │   │   ├── app/
│   │   │   │   │   ├── app_auth.api          # App 注册/登录/令牌
│   │   │   │   │   ├── app_member.api        # App 会员
│   │   │   │   │   └── app_admin.api         # App 用户管理（后台侧）
│   │   │   │   └── tenant/
│   │   │   │       └── tenant.api            # 租户管理（仅平台栈启用）
│   │   │   └── internal/
│   │   │       ├── config/config.go          # goctl 生成 + 手写字段
│   │   │       ├── handler/
│   │   │       │   ├── routes.go             # goctl 生成
│   │   │       │   └── <group>/              # goctl 生成，每分组一目录
│   │   │       ├── logic/<group>/            # goctl 生成骨架，业务手写
│   │   │       ├── casbin/                   # enforcer 构建 + 官方 redis-watcher 热刷新（单服务自用）
│   │   │       ├── middleware/
│   │   │       │   └── authority.go          # Casbin 接口鉴权（进程内 enforcer）
│   │   │       ├── svc/service_context.go    # 聚合各 RPC client、Redis、Enforcer
│   │   │       └── types/types.go            # goctl 生成
│   │   └── rpc/                              # admin-rpc（:6001）认证/用户/角色/菜单/权限/组织/字典/审计/会话
│   │       ├── pb.go                         # main（goctl 按 proto 文件名生成，勿改名）
│   │       ├── etc/
│   │       │   └── pb.yaml                   # 唯一配置（goctl 生成，含 DB/Redis/JWT）
│   │       ├── desc/
│   │       │   └── admin.proto               # proto 源（项目约定位置）
│   │       ├── pb/                           # protoc 生成（admin.pb.go / admin_grpc.pb.go）
│   │       ├── client/                       # goctl 生成的客户端（供 api 引用）
│   │       │   └── <service>/
│   │       └── internal/
│   │           ├── config/config.go
│   │           ├── logic/<service>/          # 每服务一目录，方法级 _logic.go
│   │           ├── server/<service>/         # goctl 生成
│   │           ├── svc/service_context.go    # DB 句柄、Redis
│   │           └── model/                    # 数据层（goctl model）
│   │               ├── <table>_model_gen.go  # goctl model 生成：CRUD + Redis 缓存（勿改）
│   │               ├── <table>_model.go      # 手写扩展：复杂查询 sqlx
│   │               └── vars.go               # goctl model 生成
│   ├── app/
│   │   └── rpc/                              # app-rpc（:6002）App 用户/会员/会话，结构同 admin/rpc
│   ├── file/
│   │   └── rpc/                              # file-rpc（:6003）文件资源/RustFS，结构同 admin/rpc
│   ├── ai/
│   │   └── rpc/                              # ai-rpc（:6004）AI 助手，结构同 admin/rpc
│   └── tenant/
│       └── rpc/                              # tenant-rpc（:6005）租户管理，仅平台栈部署，结构同 admin/rpc
├── data/
│   ├── db/
│   │   ├── migrations/                       # goose：00001_init.sql（业务库 12 表）、00001_tenant_registry.sql（控制库 1 表）
│   │   └── seeds/
│   │       ├── admin.sql                     # 初始管理员/角色/菜单/casbin 策略
│   │       └── demo.sql                      # 演示数据（可选加载）
│   └── scripts/
│       └── gen.ps1                           # 生成命令入口（见第 5 节）
├── docker/                                   # 构建与部署编排（旧项目风格，统一部署物）
│   ├── Dockerfile                            # 多阶段：api/admin-rpc/app-rpc/file-rpc/ai-rpc/tenant-rpc/migrate
│   ├── nginx/
│   │   ├── nginx.conf                        # 主配置
│   │   └── conf.d/                           # tenant-rpc 生成的租户 server 块（卷挂载）
│   └── tenant-stack/
│       ├── docker-compose.tenant.yml         # 租户栈模板
│       ├── env.template                      # 租户环境变量模板（随机密钥由 tenant-rpc 填充）
│       └── tenant.conf.template              # nginx server 块模板
├── pkg/                                      # 仅跨服务通用包
│   ├── xerr/                                 # 错误码：errcode.go / errmsg.go / errors.go
│   └── jwtx/                                 # JWT 生成/解析公共逻辑
├── docker-compose.yml                        # 平台栈
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

约定：

1. **模块内聚**：`application/<模块>/api|rpc`；`admin` 同时含 `api/` 与 `rpc/`，其余模块当前只有 `rpc/`。模块日后要独立 API 就在自己目录下加 `api/`，无需移动任何现有代码。
2. **模板零修改**：不存在 `data/goctl/` 目录；生成命令一律不带 `-home`。goctl 产物中标有 `Safe to edit` 的文件（logic、svc、config）允许手写，其余（routes/types/pb/server/client）只重新生成不手改。
3. **每 RPC 自洽**：`main.go / etc / desc / pb / client / internal` 六件套齐备；数据访问收敛在 `internal/model`：`*_model_gen.go` 由 goctl model 生成（勿改），复杂查询在 `*_model.go` 手写 sqlx。
4. **desc 与实现分离**：`.api` / `.proto` 源文件统一放各自 `desc/` 目录，重新生成时整目录覆盖 `internal` 可再生产物。
5. **命名**：目录与文件名一律小写下划线（`-style go_zero`）；logic 按 `<service>/<method>_logic.go` 分文件；proto 中 `service` 按业务域拆分（如 admin.proto 内含 AuthService/UserService/RoleService…），goctl 会生成对应的 server/logic/client 子目录。
6. **pkg 最小化**：只放两个以上服务共用的代码（xerr、jwtx）；单服务自用逻辑放各自 `internal/`。
7. **配置**：每服务独立 `etc/<name>.yaml`，敏感值经 compose 环境变量注入；不再使用 `docker/config.yaml.template` 分段 envsubst。

## 5. 代码生成命令规范

统一入口 `data/scripts/gen.ps1`：

```powershell
# 1. API —— 官方默认模板（无 -home）
gen api
#   goctl api go -api application/admin/api/desc/admin.api -dir application/admin/api -style go_zero

# 2. RPC —— 每服务一条命令（以 admin 为例）
gen rpc admin
#   goctl rpc protoc application/admin/rpc/desc/admin.proto `
#     --go_out=application/admin/rpc --go-grpc_out=application/admin/rpc `
#     --zrpc_out=application/admin/rpc -m --style go_zero

# 3. Model —— goctl model（go-zero sqlx + sqlc 缓存）；需先 gen db up 应用到开发库
gen model admin sys_user
#   goctl model mysql datasource --dir application/admin/rpc/internal/model `
#     --table sys_user --cache true -style go_zero --url root:123456@tcp(127.0.0.1:3306)/gozero_admin

# 4. DB —— goose
gen db up            # goose -dir data/db/migrations mysql <dsn> up
gen db create <name> # goose -dir data/db/migrations create <name> sql

# 5. Swagger —— 官方原生输出，零修补
gen swagger
#   goctl api swagger --api application/admin/api/desc/admin.api --dir data/api/generated
```

> RPC 生成**带 `-m`**（multiple 模式）：proto 内多个 service 时 goctl 要求此参数；仍生成单一 main，文件名与 proto 同名（如 `admin.proto` → `pb.go`），一个 RPC 服务 = 一个进程一个端口。

### 5.1 goctl model 约定

- **生成顺序**：写 migration → `gen db up` 应用到本地开发库 → `gen model <模块> <表>`（goctl model 从活库反射表结构）。
- **缓存**：固定 `--cache true`，生成基于 go-zero `sqlc` 包（sqlx + Redis 行级缓存）的代码；缓存节点在各 RPC 的 `etc/*.yaml` 配置（`CacheRedis`）。
- **产物**：`<table>_model_gen.go`（生成，勿改）、`<table>_model.go`（自定义扩展，手写复杂查询 sqlx）、`vars.go`。
- **复杂查询**（部门数据范围、JOIN、统计）写在 `*_model.go` 扩展文件，不塞进 logic。
- 工具版本锁定：Go 1.27.3（go.mod 工具链）、go-zero v1.10.3 / goctl v1.10.2、MySQL 9.7、jwt/v5 v5.3.1、x/crypto v0.58.0；**go.mod 为依赖版本唯一事实源**，文档版本号仅为快照参考。

## 6. 错误处理规范（不改模板）

go-zero v1.10.3 官方机制（源码核实 `rest/httpx/responses.go`）：

- handler 保持官方生成：成功 `httpx.OkJsonCtx` 直返数据体；错误 `httpx.ErrorCtx`。**不做任何模板修改**。
- 不注册 handler 时官方默认输出为 `http.Error` 纯文本 + 400，不适合 JSON API；因此使用官方定制入口 **`httpx.SetErrorHandlerCtx`**，在 `main.go` 注册一次：

```go
// main.go（admin/api）
httpx.SetErrorHandlerCtx(xerr.Handler)                      // 统一错误翻译
server := rest.MustNewServer(c.RestConf,
    rest.WithUnauthorizedCallback(xerr.Unauthorized))       // JWT 失败同样走统一格式
```

统一错误 JSON：**`{"code": <int>, "msg": "<string>"}`**

| 场景 | HTTP | body |
|---|---|---|
| 业务错误（`xerr.CodeError`，含 RPC 经 gRPC status 传回） | 200 | `{"code": 业务码, "msg": 消息}` |
| 请求解析 / 参数校验失败 | 400 | `{"code": 400, "msg": ...}` |
| JWT 缺失 / 无效 / 过期 | 401 | `{"code": 401, "msg": "未登录或登录已过期"}` |
| Casbin 接口鉴权拒绝 | 403 | `{"code": 403, "msg": "没有接口权限"}` |
| 未识别错误 | 500 | `{"code": 500, "msg": "系统繁忙，请稍后再试"}`（细节只记日志） |

- 业务错误码集中在 `pkg/xerr`（沿用 go-zero 官方示例 looklook 的 xerr 思路）：logic 层返回 `xerr.NewErrCodeMsg(code, msg)`；RPC 边界经 gRPC status 传递业务码与消息；`xerr.Handler` 完成上表翻译。
- 成功响应**不使用** `httpx.SetOkHandler` 包装信封，保持官方直返数据体。

## 7. 多租户设计

### 7.0 控制面与业务面分离

- **控制库**（平台栈）：仅 `tenant_registry` 一张表，存租户 slug、域名、compose 项目名、随机密钥、状态。
- **业务库**（每租户独立）：完整 12 张表（见第 8 章），跑在租户栈自己的 MySQL 容器中，与平台及其他租户完全隔离。
- **平台栈不跑业务**：平台只部署 api（租户管理接口）+ tenant-rpc + 控制库 + nginx + rustfs。

### 7.1 租户创建流程（平台管理台操作）

1. 填写租户 slug（如 `t1`）→ 校验子域名可用性（查 `tenant_registry`）
2. 生成随机密钥：MySQL root、JWT_ACCESS_SECRET、APP_JWT_ACCESS_SECRET
3. `tenant_registry` 插入记录（slug、domain、compose_project、密钥、status=active）
4. 渲染 `docker/tenant-stack/env.template` → 生成租户专属 `.env`
5. 执行 `docker compose -p gozero-t-t1 -f docker/tenant-stack/docker-compose.tenant.yml --env-file <租户.env> up -d --build --wait`
6. 租户栈内 migrate job 自动执行 `goose up`（00001_init.sql）+ seeds（初始管理员）
7. 渲染 `docker/tenant-stack/tenant.conf.template` → 写入 `docker/nginx/conf.d/t1.conf`
8. `docker exec <nginx> nginx -s reload`
9. 更新 `tenant_registry.status = active`；失败则记 `error_msg`

### 7.2 租户生命周期操作

| 操作 | 实现 |
|---|---|
| 停用 | `docker compose -p gozero-t-<slug> stop` + 更新 status；Nginx 返回 503 |
| 启用 | `docker compose -p gozero-t-<slug> start` + 更新 status |
| 重置管理员密码 | 直接连租户库 UPDATE sys_user.password_hash |
| 删除 | `docker compose -p gozero-t-<slug> down -v` + 删 Nginx conf + 软删 registry 记录 |

### 7.3 默认约定

| 项 | 默认 |
|---|---|
| 平台栈 | nginx + api（仅租户管理接口）+ tenant-rpc + 控制库 MySQL + rustfs；不跑业务 RPC |
| 租户栈 | 完整 api + 4 RPC + MySQL + Redis + migrate（见 docker/tenant-stack/） |
| RustFS | 平台级共享，每租户独立 bucket |
| 本地开发 | `*.localhost`（127.0.0.1 泛解析） |
| 生产域名 | 泛解析 DNS `*.example.com` + 泛域名证书 |
| 共享网络 | `gozero-edge`；租户 api 以 `<slug>-api` 别名注册（防服务名冲突） |
| 安全 | tenant-rpc 挂 docker socket ≈ 宿主 root，仅平台内网可达；API 仅超管可访问 |

## 8. 数据库重构方向

- **控制库**（平台栈）：库名 `gozero_platform`，仅 `tenant_registry` 一张表（见 00001_tenant_registry.sql）。
- **业务库**（每租户）：库名 `gozero_admin`，12 张表（见下）；每租户独立 MySQL 9.7 容器 + 独立卷；collation `utf8mb4_0900_ai_ci`。
- 初始化 = goose migrations + seeds，由栈内 migrate job 在首次启动时执行。

### 表清单（12 张）

| # | 表 | 域 | 说明 |
|---|---|---|---|
| 1 | `sys_dept` | RBAC | 部门树，物化路径 `path`（如 `/1/7/12/`） |
| 2 | `sys_user` | RBAC | 后台用户，单部门 `dept_id`、头像 `avatar_file_id` |
| 3 | `sys_role` | RBAC | 角色，`data_scope`（ALL/DEPT_AND_CHILDREN/DEPT/SELF） |
| 4 | `sys_menu` | RBAC | 菜单+按钮（`type`：1目录/2菜单/3按钮，`perm_code`） |
| 5 | `sys_role_menu` | RBAC | 角色-菜单关联 |
| 6 | `sys_api` | RBAC | API 资源（method+path 唯一，casbin obj 来源） |
| 7 | `casbin_rule` | RBAC | p+g 同表，`v1=dom` 装部门，`*`=全局 |
| 8 | `sys_dict` | 系统 | 字典，字典项合并为 JSON 数组 `items` |
| 9 | `sys_file` | 文件 | RustFS 引用（bucket/object_key/ref_count） |
| 10 | `sys_audit_log` | 审计 | 操作审计 |
| 11 | `app_user` | App | App 用户（email/phone，bcrypt） |
| 12 | `app_membership_grant` | App | 会员授予（plan 常量枚举，request_id 幂等） |

### 简化决策

| 砍掉/合并 | 原因 | 替代 |
|---|---|---|
| ~~sys_user_dept~~ | 一个用户一个部门即可，多对多过度设计 | `sys_user.dept_id` |
| ~~sys_device_session~~ | JWT 无状态 + Redis 撤销列表，性能更好且自动过期 | Redis `revoked:<jti>` |
| ~~app_session~~ | 同上，App 侧无设备管理需求 | Redis `revoked:<jti>` |
| ~~sys_dict_item~~ | 99% 字典项 5-10 个，JSON 数组足够且免 JOIN | `sys_dict.items` JSON 列 |

## 8.5 AI 助手平台演进（三阶段）

| 阶段 | 能力 | 状态 |
|---|---|---|
| **阶段 1：工具调用 Agent** | 会话管理 + LLM 对话（DeepSeek/通义，OpenAI 兼容协议）+ 工具注册表 + function calling 多轮循环 | ✅ 已实现 |
| **阶段 2：任务编排与工作流** | 工作流 JSON DSL + 定时调度（interval 秒级）+ 人工审批节点 + 执行历史/审计 | ✅ 已实现 |
| **阶段 3：多 Agent 协作平台**（规划） | 见下 | 📋 规划中 |

### 阶段 1 设计（已实现）

- **表**：`ai_conversation`（会话）、`ai_message`（消息，含 tool_calls JSON）
- **LLM 接入**：OpenAI 兼容 chat completions 协议，stdlib `net/http` 直连（零新增依赖）；配置 `AI.Enabled` / `Provider` / `Model` / `BaseURL`，密钥经 `API_KEY_DEEPSEEK` / `API_KEY_QWEN` 环境变量注入
- **工具注册表**（`internal/tools`）：内置 `get_current_time`、`text_stats` 演示工具；新增工具实现 `Tool` 接口注册即可；工具在服务端执行，LLM 只产出调用意图
- **多轮循环**：`SendMessage` 内最多 5 轮 function calling（LLM 提议 → 服务端执行 → 结果回填），超限兜底返回文本

### 阶段 2 设计（已实现）

- **表**：`ai_workflow`（工作流定义，JSON DSL）、`ai_task`（执行记录/审批）
- **DSL**：`{"steps":[{"name":"...","tool":"...","input":{...},"approval":true}]}`——顺序执行，`approval=true` 的步骤进入 `pending_approval` 等待人工审批
- **调度**：`interval_seconds` 简单间隔调度（避免引入 cron 依赖）；后台 ticker 每 30s 扫描到期工作流
- **审批**：`ApproveTask` / `RejectTask` 接口；审批记录操作者到 `ai_task.approved_by`

### 阶段 3 规划（多 Agent 协作平台）

| 能力 | 设计 |
|---|---|
| Agent 市场 | `ai_agent` 表：角色定义（system prompt）+ 工具集 + 可见范围；创建/分享/复用 |
| 租户级隔离 | 每租户自定义 Agent 与工具；工具可访问的数据受 Casbin 数据范围约束 |
| 用量计费 | `ai_usage` 表：按租户/用户/模型统计 token 消耗；配额限流 |
| 审批流增强 | 高风险操作多级审批；审批动作接入审计日志 |
| 多 Agent 协作 | Agent 间消息传递协议；编排器（Orchestrator）拆解复杂任务分派给多个 Agent |

**阶段 3 启动条件**：阶段 1/2 在生产验证稳定后再启动；需要新增 `ai_agent`、`ai_usage` 表与计费中间件。

## 9. 落地顺序

1. ✅ 需求与架构确认（本文档）
2. 表结构设计（migrations 初稿）+ proto/api 契约设计
3. 搭骨架：api + admin-rpc，打通登录 / JWT / Casbin 最小闭环
4. 逐个实现 app-rpc、file-rpc、ai-rpc
5. 平台栈 compose（去 etcd、含 migrate job）+ tenant-rpc + Nginx 模板
6. 租户栈模板 + provisioning 联调
7. README 与运维文档

## 10. 明确废弃项

| 废弃 | 替代 |
|---|---|
| GORM（`pkg/orm`）与 goctl model 混用 | 统一 goctl model（go-zero sqlx + sqlc 缓存） |
| `data/goctl/` 自定义模板 | 官方默认模板 |
| `{code,message,result}` 信封（`pkg/result.HttpResult`） | 官方默认 + `httpx.SetErrorHandlerCtx` |
| swagger 生成后的 300 行 PowerShell 修补 | 官方 `goctl api swagger` 原生输出 |
| etcd 服务发现 | 直连 Endpoints |
| `docker/config.yaml.template` 分段 envsubst | 各服务独立 `etc/*.yaml` + compose 环境变量 |
| 权限版本历史/回滚（permission_version/change 表） | 无（确认废弃） |
| IP 归属地（ip2region） | 无（确认废弃） |
