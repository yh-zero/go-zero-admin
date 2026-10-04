# go-zero-admin

基于 **go-zero + GORM + Casbin** 的管理后台，前端使用 **Vue Vben Admin / web-antdv-next**。提供用户、角色、菜单、接口权限、字典和基础工具，适合作为业务管理系统的开发起点。

- 后端仓库：[yh-zero/go-zero-admin](https://github.com/yh-zero/go-zero-admin)
- 配套前端：[yh-zero/go-zero-admin-vben](https://github.com/yh-zero/go-zero-admin-vben)
- 在线演示：[yh9527.top（HTTPS）](https://yh9527.top) · [IP入口](http://175.178.67.80)。账号 `admin / 123456`，需图片验证码；当前为只读演示，AI 未配置 Key。
- 本地接口文档：[Swagger UI](http://localhost:8080) · [Swagger JSON](data/api/generated/go-zero-admin.swagger.json)

**开发方式：Docker 启动依赖，API、业务 RPC、AI RPC 在本机运行，前端单独启动。服务器部署使用另一份 Compose 配置。**

## 目录

- [功能与页面](#功能与页面)
- [架构与代码目录](#架构与代码目录)
- [环境要求](#环境要求)
- [本地开发启动](#本地开发启动)
- [接口文档与调试](#接口文档与调试)
- [开发流程与代码生成](#开发流程与代码生成)
- [维护约定与升级](#维护约定与升级)
- [可选服务配置](#可选服务配置)
- [服务器部署](#服务器部署)
- [检查与测试](#检查与测试)
- [常见问题](#常见问题)
- [AI Agent 本机配置](#ai-agent-本机配置)
- [AI 接口与运行边界](#ai-接口与运行边界)
- [相关文档与许可证](#相关文档与许可证)

## 功能与页面

| 模块 | 当前功能 |
| --- | --- |
| 登录与权限 | 图片验证码、JWT 登录、当前用户查询、自助改密、服务端退出与会话撤销 |
| 用户管理 | 搜索与分页、新增与编辑、角色分配、启用 / 冻结、重置密码、删除 |
| 角色管理 | 角色树、默认首页、菜单 / 按钮 / API 独立授权，分组搜索、选择统计与变更预览 |
| 菜单管理 | 菜单树、页面组件与图标选择、按钮定义维护、排序、隐藏与缓存 |
| API 管理 | 路径 / 分组 / 方法筛选、资源维护、批量删除、Swagger 差异预览与选择同步 |
| 字典管理 | 字典与字典项维护、状态、排序、预览 |
| 管理工具 | 阿里云 OSS 图片上传、SMTP 邮箱验证码发送 |
| 审计日志 | 写操作与登录成功/失败、事务审计、分页和条件查询，参数白名单脱敏 |
| 组织与数据范围 | 部门树、岗位、用户归属，全部/本人/本部门/下级/指定部门，文件查询实际 SQL 隔离 |
| 文件资源 | 上传登记、归属和引用、删除保护与重试、私有图片短时签名访问 |
| 设备会话 | 本人/管理员会话列表、单设备撤销，与原全设备退出兼容 |
| AI 助手 | 独立 AI RPC、异步任务与轮询、会话历史、取消与工具调用；默认关闭 |
| Vben 原有页面 | 保留工作台、分析页、组件演示，与后端业务菜单合并显示 |

下面是当前前端连接本地后端的实际截图（2026-09-25）。页面数据为开发示例；截图随代码保存在本仓库，GitHub 可直接展示。

### 菜单管理

![菜单管理：树形列表与页面组件映射](data/doc/screenshots/menu.jpg)

<details>
<summary>查看其他模块截图：角色、API、用户、字典、管理工具</summary>

### 角色管理

![角色管理：菜单、按钮和接口分别授权](data/doc/screenshots/authority.jpg)

### API 管理

![API 管理：筛选与接口资源列表](data/doc/screenshots/api.jpg)

### 用户管理

![用户管理：按用户名搜索](data/doc/screenshots/user.jpg)

### 字典管理

![字典管理：字典与字典项入口](data/doc/screenshots/dictionary.jpg)

### 管理工具

![管理工具：图片上传和邮箱验证码](data/doc/screenshots/tools.jpg)

</details>

当前代码注册 **76 个 HTTP 接口**：70 个管理接口使用 `/v1/sys` 前缀，6 个 AI 接口使用 `/v1/ai` 前缀，均由原 API 的 `7001` 端口提供。尚未实现的旧菜单组件显示“尚未接入”。OSS 和 SMTP 的真实外部验收仍需配置。[后续开发计划](DEVELOPMENT_PLAN.md) 只记录未完成的开发与验收；运行、配置与升级方式统一维护在本文。

## 架构与代码目录

```mermaid
flowchart LR
    Browser[浏览器 / Vben 前端] --> Proxy[Vite 开发代理 / Nginx]
    Proxy --> API[applet-api :7001]
    Swagger[Swagger UI :8080] --> API
    API --> RPC[applet-rpc :6001]
    API --> AI[applet-ai-rpc :6002]
    AI --> RPC
    AI --> Provider[DeepSeek / Qwen]
    RPC --> MySQL[(MySQL)]
    AI --> MySQL
    API --> Redis[(Redis)]
    RPC --> Redis
    AI --> Redis
    API --> Etcd[(etcd)]
    RPC --> Etcd
    AI --> Etcd
```

API 层处理 HTTP 参数、JWT、当前会话校验、审计上下文和响应封装；业务 RPC 实现管理业务、数据库操作及 Casbin 权限校验。独立 AI RPC 管理持久异步任务并调用模型，通过业务 RPC 核对用户权限、查询工具数据。模型调用不占用普通 HTTP/RPC 请求等待时间，前端提交后轮询任务。权限版本与策略在 MySQL 同事务提交，实例在鉴权前核对版本，并定期恢复同步；Redis 通知加快传播，etcd 用于 RPC 注册与发现。生产两个 RPC 均要求 App/Token 认证和 StrictControl，本机未认证开发 RPC 只监听回环地址。前端只访问原 `7001` HTTP API，不直接访问 `6001` 或 `6002`。

建议将两个仓库放在同一级目录：

```text
go-zero-github/
├── go-zero-admin/                  # 后端仓库
│   ├── application/applet/
│   │   ├── api/
│   │   │   ├── desc/               # .api 接口定义
│   │   │   ├── etc/                # 本机 API 配置
│   │   │   └── internal/           # handler、logic、types、middleware
│   │   └── rpc/
│   │       ├── desc/               # .proto 协议
│   │       ├── etc/                # 本机 RPC 配置
│   │       ├── internal/           # 业务 logic、model、服务上下文
│   │       ├── client/             # 生成的 RPC 客户端
│   │       └── pb/                 # 生成的 protobuf 代码
│   ├── application/ai/rpc/         # 独立 AI RPC：协议、配置、任务与工具执行
│   ├── pkg/                        # 响应、错误码、ORM、权限等公共实现
│   ├── data/
│   │   ├── api/generated/          # 当前 Swagger JSON
│   │   ├── db/migrations/          # 增量 SQL
│   │   └── doc/screenshots/        # README 页面截图
│   ├── docker/                     # 服务器部署与 Swagger 配置
│   ├── test/goctl/                 # 自定义 goctl 模板
│   ├── test/sh/                    # 生成与联调脚本
│   └── docker-compose.yml         # 仅本地开发依赖
└── go-zero-admin-vben/             # 前端仓库
    └── apps/web-antdv-next/        # 业务前端应用
```

## 环境要求

| 工具 | 要求 / 当前项目配置 |
| --- | --- |
| Go | 推荐 1.26.x，与 Docker 构建版本一致；`go.mod` 声明 1.24.0 |
| Docker | Docker Engine / Docker Desktop，启用 Compose v2（`docker compose`） |
| Node.js | `^22.18.0` 或 `^24.12.0`，以配套前端 `package.json` 为准 |
| pnpm | `11.16.0`，与前端 `packageManager` 保持一致 |
| PowerShell | Windows PowerShell 5.1 即可运行 Swagger 和数据库管理脚本，无需安装 `pwsh` |
| goctl | 修改接口或生成文档时需要；当前脚本使用已验证的 `v1.10.2` |
| protobuf 工具 | 仅重新生成 RPC 时需要 `protoc`、`protoc-gen-go`、`protoc-gen-go-grpc` |

后端框架当前为 go-zero `v1.10.3`，使用 MySQL 8、Redis 7、etcd 3.5。直接启动已生成的代码不需要安装 goctl 或 protobuf 工具。

## 本地开发启动

以下示例以 Windows PowerShell 为主。除特别说明外，命令都在 **`go-zero-admin` 根目录**执行；已经拉取的项目可跳过克隆。

### 1. 获取前后端代码

在准备存放项目的父目录执行：

```powershell
git clone https://github.com/yh-zero/go-zero-admin.git
git clone https://github.com/yh-zero/go-zero-admin-vben.git
cd go-zero-admin
```

### 2. 启动开发依赖

```powershell
docker compose up -d
docker compose ps
```

等待 `docker compose ps` 中 MySQL、Redis、etcd 显示 `healthy` 后，再执行下面的数据库步骤。首次初始化空数据卷可能需要更长时间。

根目录 Compose 不需要 `.env` / `.env.example`，也不会启动 API、业务 RPC、AI RPC 或前端。Docker Desktop 中统一归入 `go-zero-admin` 项目。本机 AI 普通参数在 `application/ai/rpc/etc/ai.yaml`，密钥由后文的 `dev.ps1` 加载 `.env.local`。

| 服务 | 容器名称 | 本机地址 | 本地配置 |
| --- | --- | --- | --- |
| MySQL | `gozero-mysql` | `127.0.0.1:3306` | 用户 `root`，密码 `123456`，数据库 **`goZero-admin`** |
| Redis | `gozero-redis` | `127.0.0.1:6379` | 无密码 |
| etcd | `gozero-etcd` | `127.0.0.1:2379` | 本地 RPC 服务发现 |
| Swagger UI | `gozero-swagger` | [localhost:8080](http://localhost:8080) | 查看与调试接口 |

这些容器端口只绑定本机。MySQL 使用命名卷 `go-zero-admin_mysql_data`；Redis、etcd 未配置持久化卷。首次使用空 MySQL 卷时自动导入 [基线 SQL](data/db/gozero-admin-20240129.sql)，已有数据卷不会重复导入，也不会因修改 Compose 而自动修改数据库密码。

### 3. 初始化或升级数据库

**新库和已有库都需要应用增量迁移。** 使用统一脚本按文件名顺序执行，不再手工逐个 `SOURCE`。基线 SQL 只用于空数据卷的首次初始化，不能重新导入已有业务库；`Init` 也不会清空已有数据库。

Windows PowerShell 5.1，在后端根目录执行：

```powershell
# 首次准备：启动并等待开发依赖，然后备份并执行未应用的迁移
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\db.ps1 -Action Init

# 日常升级：MySQL 已运行时检查、备份或应用新增迁移
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\db.ps1 -Action Status
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\db.ps1 -Action Backup
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\db.ps1 -Action Migrate
```

Linux Shell 的对应命令（需 `sha256sum`）：

```sh
sh test/sh/db.sh init
sh test/sh/db.sh status
sh test/sh/db.sh backup
sh test/sh/db.sh migrate
```

`Init` 可代替第 2 步的启动命令；开发模式只启动 MySQL、Redis、etcd、Swagger，不启动三个后端服务。`Status` 不写迁移数据；`Backup` 单独导出当前数据库；`Migrate` 只应用未登记的脚本。脚本使用所选 Compose 中 MySQL 的数据库名及凭据，不需要宿主机安装 MySQL 客户端。已有运行项目升级时，先停止 API、AI RPC、业务 RPC，确认全部停止后再备份、迁移；AI 后台任务也会写库。

早期基础迁移如下；完整清单以 [迁移目录](data/db/migrations/) 和 `Status` 输出为准：

| 脚本 | 用途 |
| --- | --- |
| [20260925_business_access.sql](data/db/migrations/20260925_business_access.sql) | 补齐管理菜单、按钮、API 及管理员权限，修正历史菜单方法配置 |
| [20260925_user_active_username.sql](data/db/migrations/20260925_user_active_username.sql) | 活跃用户名唯一，允许复用已软删除用户名 |
| [20260926_00_method_dictionary.sql](data/db/migrations/20260926_00_method_dictionary.sql) | 仅修正完全匹配旧初始化数据的 HTTP 方法字典 POST 值；用户修改过的数据不自动改动 |
| [20260926_api_sync_access.sql](data/db/migrations/20260926_api_sync_access.sql) | 登记 API 同步接口及按钮，并补内置管理员的对应授权 |
| [20260926_business_unique.sql](data/db/migrations/20260926_business_unique.sql) | 为菜单、API、字典及字典项增加兼容软删除的唯一约束，发现冲突先停止 |
| [20260926_session_version.sql](data/db/migrations/20260926_session_version.sql) | 为用户增加会话版本；升级前签发的旧 JWT 需要重新登录 |

独立 AI RPC 复用 [20261003_zz_ai_agent.sql](data/db/migrations/20261003_zz_ai_agent.sql) 创建的 `sys_ai_conversations`、`sys_ai_runs`、`sys_ai_messages` 三表，本次服务拆分没有新增迁移，不修改已应用脚本。AI 服务当前仍使用同一 MySQL：并发限制需要锁定 `sys_users` 中的用户行，提交/取消审计与任务变更同事务写入 `sys_audit_logs`；不要只导入 AI 三表就启动服务。

迁移器在 `schema_migrations` 保存文件名、SHA-256 校验值和执行时间。已应用且校验一致的文件会跳过；修改已应用文件会报错，应新增迁移文件。执行待应用迁移前自动备份至 **`bin/db-backups/`**，同时生成 `.sha256` 文件；备份失败则不继续。备份包含选中数据库的结构和数据，不含 Redis、OSS 或源码。

迁移遇错立即停止，不把失败文件登记为已应用。MySQL DDL 会隐式提交，不能假设全部步骤自动回滚；确认错误及实际结构后处理冲突，再重跑。为避免并发升级，脚本使用 MySQL 容器内 `/tmp/gozero-db-migrate.lock`；若进程异常终止留下锁，先确认没有其他迁移仍在执行，再清理该空锁目录并重试。不要在迁移运行中删锁。

迁移完成后按业务 RPC → AI RPC → API 顺序启动，重新登录或刷新前端权限。正常通过管理接口保存权限会自动同步，不需要重启服务。

### 4. 启动业务 RPC、AI RPC 和 API

Windows 可在后端根目录使用托管脚本，按 `rpc:6001` → `ai-rpc:6002` → `api:7001` 顺序构建、启动；PID 和日志路径记录在 `bin/dev/managed/services.json`：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Start
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Status
```

有待应用迁移时，先 `-Action Stop` 停止三个托管服务，再按上文备份、迁移，最后 `-Action Start`。脚本先核对全部存活 PID 与项目内二进制的实际路径，再按 API → AI RPC → 业务 RPC 停止；任一 PID 复用或路径不符会在停止前拒绝并保留清单。操作通过 `bin/dev/managed/operation.lock` 排他锁避免并发。手动运行或调试时使用下面三个入口；`.env.local` 只由托管脚本加载，手动入口需单独为 AI RPC 提供环境。

先下载 Go 依赖：

```powershell
go mod download
```

**终端一：业务 RPC**（在后端根目录执行）：

```powershell
go run ./application/applet/rpc -f ./application/applet/rpc/etc/applet.yaml
```

**终端二：AI RPC**（另开终端，在后端根目录执行；默认关闭模型调用时也可启动）：

```powershell
go run ./application/ai/rpc -f ./application/ai/rpc/etc/ai.yaml
```

**终端三：API**（另开终端，在后端根目录执行）：

```powershell
go run ./application/applet/api -f ./application/applet/api/etc/applet-api.yaml
```

也可在 VS Code 分别调试三个入口，使用相同的 `-f` 配置路径。先确认业务 RPC 正常，再启动 AI RPC，最后启动 API。配置地址变动时同步核对：

- [API 配置](application/applet/api/etc/applet-api.yaml)：端口、Redis、etcd、JWT、OSS。
- [RPC 配置](application/applet/rpc/etc/applet.yaml)：MySQL、Redis、etcd、JWT、默认重置密码。
- [AI RPC 配置](application/ai/rpc/etc/ai.yaml)：MySQL、etcd、RPC 认证、下游业务 RPC，以及 `AI` 模型参数；API Key 从 AI RPC 进程环境读取。
- API 与业务 RPC 的 `JwtAuth.AccessSecret` 必须一致；业务服务的 etcd 键为 `applet.rpc`，AI 服务为 `ai.rpc`。API 的 `AppletRPC` / `AIRPC` 分别连接两个服务，AI 的 `AppletRPC` 连接业务服务。

检查 API 能否返回验证码数据：

```powershell
Invoke-RestMethod http://127.0.0.1:7001/v1/sys/randomImage
```

### 5. 启动前端并登录

另开终端，进入同级前端仓库根目录：

```powershell
cd ..\go-zero-admin-vben
pnpm install --frozen-lockfile
pnpm dev
```

如果从其他目录打开终端，请直接进入实际的 `go-zero-admin-vben` 路径。依赖在前端仓库根目录安装，不要只复制或单独安装 `apps/web-antdv-next`。

| 项目 | 地址 / 值 |
| --- | --- |
| 前端 | [http://127.0.0.1:5999](http://127.0.0.1:5999) |
| 本地开发账号 | `admin` |
| 本地开发密码 | `123456` |
| API | `http://127.0.0.1:7001` |
| RPC | `127.0.0.1:6001`，供后端内部调用 |
| AI RPC | `127.0.0.1:6002`，供后端内部调用 |

登录时填写当前图片验证码。验证码不是固定值；当前实现始终校验验证码，配置中的旧 `Isdev` 注释不能作为绕过依据。已有数据库若修改过账号密码，以实际数据为准。

前端开发配置位于 `apps/web-antdv-next/.env.development`，已关闭 Mock。`vite.config.ts` 将 `/api` 转发到 `http://127.0.0.1:7001`，并去掉 `/api` 前缀：

```text
浏览器：http://127.0.0.1:5999/api/v1/sys/menu/getMenu
后端：  http://127.0.0.1:7001/v1/sys/menu/getMenu
```

端口被占用时，以前端启动终端输出的地址为准。

### 6. 日常停止与重启

托管服务使用下面命令。数据库升级前使用 `Stop`，完成备份、迁移后使用 `Start`；修改 AI YAML 参数或环境中的模型 Key 后使用 `Restart`。

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Stop
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Restart
```

```powershell
# 在后端根目录查看开发依赖日志
docker compose logs --tail=100

# 停止依赖，保留容器和数据
docker compose stop

# 再次启动依赖
docker compose up -d
```

手动服务在各自终端按 `Ctrl+C` 停止：先 API，再 AI RPC，最后业务 RPC，之后再停止依赖；启动按相反顺序。前端在自己的终端停止。`docker compose down -v` 会删除数据库卷，不能当作日常停止命令。

## 接口文档与调试

### 访问 Swagger

开发依赖启动后访问 [Swagger UI](http://localhost:8080)，JSON 地址为 [swagger.json](http://localhost:8080/swagger.json)。当前文件是 [data/api/generated/go-zero-admin.swagger.json](data/api/generated/go-zero-admin.swagger.json)，格式为 Swagger 2.0，可导入 Apifox 等工具。

查看文档不要求 API / RPC 已运行；**执行接口需要完整后端和依赖**。Swagger 容器把 `/v1/sys` 请求转发到宿主机 `7001`，避免浏览器跨域。历史文件 `data/api/go-zero-admin.openapi.json` 仅供参考，不作为最新接口依据。

### 登录与鉴权步骤

1. 调用 `GET /v1/sys/randomImage`，读取 `result.captchaId` 与 `result.captchaImg`。图片字段是 Data URL，可在浏览器查看；Swagger 通常显示字符串。
2. 调用 `POST /v1/sys/login`，提交用户名、密码、验证码 ID 和图片中的字符。
3. 从 `result.accessToken` 取得令牌，在 Swagger 的 **Authorize** 中填写完整的 `Bearer <accessToken>`。
4. 调用 `GET /v1/sys/menu/getMenu` 验证登录，再调试业务接口。GET 参数放在 query 中，不发送 GET body。

登录请求示例（验证码值需替换为当前实际值）：

```json
{
  "username": "admin",
  "password": "123456",
  "captchaId": "当前返回的 captchaId",
  "captcha": "当前图片中的字符"
}
```

图片验证码为六位字符，有效期为 120 秒，提交校验后会消费；登录失败应重新获取图片。菜单是否显示、按钮是否可见、API 是否允许调用是三类权限，需要分别配置。

### 当前用户、改密码与会话撤销

| 接口 | 用途 |
| --- | --- |
| `GET /v1/sys/me` | 按当前 JWT 查询本人最新资料，前端刷新时使用 |
| `PUT /v1/sys/changePassword` | 提交 `oldPassword`、`newPassword` 自助改密，成功后重新登录 |
| `POST /v1/sys/logout` | 撤销当前账号的所有已登录设备会话，再清理前端状态 |

这 3 个接口需要有效登录态，不需要单独分配 Casbin 业务权限。默认 JWT 有效期为 **2 小时**，API 每次验证用户状态、角色、数据库会话版本及新令牌的设备记录；改密、管理员重置密码、冻结、删除、角色或部门岗位归属变更后，旧会话不再有效。`logout` 是**账号所有设备退出**。`GET /v1/sys/session/devices` 查询自己的设备，`DELETE /v1/sys/session/device` 撤销指定设备，其他设备仍可使用；管理员接口位于 `/v1/sys/session/admin`，首版限定内置管理员操作。历史无设备ID的 JWT 在原有效期内保留版本校验。普通资料修改不撤销登录；目前仍无 refreshToken 和独立切换角色接口。

新建、重置和修改密码统一要求 **8至72字节**（中文按 UTF-8 字节计算），哈希失败不会写入数据库。历史短密码仍可正常验证。开发配置的默认重置密码为 `GoZero@2026`；生产请配置自己的 `DEFAULT_USER_PASSWORD`，启动时校验。该配置不会修改已有账号的初始密码。

前端个人中心位于 `/account`，提供真实资料和自助改密。浏览器缓存仅用于会话恢复，不能替代 `/me` 与后端鉴权；旧请求晚到时不会覆盖新登录资料或清理新会话。

成功响应示例：

```json
{
  "code": 200,
  "message": "操作成功!",
  "result": {},
  "returnData": null,
  "success": true,
  "timestamp": 1780000000
}
```

`result` 随接口变化，可为对象或 `null`；错误响应通常只有 `code`、`message`。业务错误和 JWT 失效可能仍返回 HTTP 200，必须检查业务码；当前 JWT 失效码为 `100003`，权限拒绝也可能直接返回 HTTP 403。前端 `requestClient` 已处理统一响应，业务函数直接使用解包后的数据。

### 更新 Swagger

首次需要时安装生成工具：

```powershell
go install github.com/zeromicro/go-zero/tools/goctl@v1.10.2
```

在后端根目录执行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\swagger.ps1
```

脚本以 `.api` 为来源，调用 goctl 生成并补齐项目实际的响应包装、鉴权及上传约定，**只生成文档，不修改业务源码**。脚本支持自动查找 Go 的 bin 目录，也可通过 `-GoctlPath` 指定工具路径。

更新顺序：修改接口声明与实现 → 必要时重新生成 API / RPC → **生成 Swagger → 重新构建 RPC/API** → 刷新文档验证 → 一起提交源码与 JSON。不要手工修改生成的 JSON。Swagger UI 挂载文件，更新 JSON 后刷新即可；API 资源同步使用 RPC 编译时嵌入的 Swagger，必须先生成文档再构建、重启 RPC，否则同步预览仍是旧版本。

### 同步 API 资源

API 管理页的“同步后端接口”调用 `GET /v1/sys/api/previewSync`，分别展示新增、说明变更及不在当前授权清单中的资源。后一类可能是已移除路由，也可能是登录等仍存在但无需 Casbin 授权的路由，不能直接视为接口已删除。开发者勾选后调用 `POST /v1/sys/api/applySync`，提交预览 `version` 与所选 `keys`；版本过期需重新预览并选择。

同步只新增资源或更新分组和描述，保留原资源 ID 及已有角色策略；**不会自动授予新增 API 权限，也不会删除不在授权清单中的资源**。同步完成后到角色页单独授权；其他资源需核实用途后在 API 列表处理。菜单、按钮和接口权限仍分别保存，不能用接口同步替代角色授权。

调整文档端口或后端目标时，修改根 Compose 中 Swagger 的端口和 `SWAGGER_API_URL`，然后执行：

```powershell
docker compose up -d --no-deps --force-recreate swagger-ui
```

目标填写容器可达的协议、主机、端口，例如 `http://host.docker.internal:7001`，不加接口路径或末尾斜杠。Linux 下仅监听宿主机 `127.0.0.1` 的 API 无法通过 host-gateway 访问。

## 开发流程与代码生成

### 新增一个业务模块

1. **先定义接口。** 在 `application/applet/api/desc/` 明确方法、路径、参数、分页、返回值及错误；需要 RPC 时同步修改 `application/applet/rpc/desc/applet.proto`。
2. **生成骨架并实现逻辑。** API 层负责协议适配，RPC 层负责业务与数据；数据库变更放入 `data/db/migrations/`，避免覆盖现有库。
3. **新增前端文件。** 页面放 `src/views/business/<模块>/`，请求和类型放 `src/api/business/`；优先复用现有表格、表单和弹窗。
4. **建立菜单映射。** 在前端 `src/adapter/business/pages.ts` 单处登记组件，菜单下拉与动态路由共用。保留历史 `component` 值，通过映射接到新页面，减少数据库改动。`/account` 为真实个人中心保留路径。
5. **配置三类权限。** 菜单编辑器维护按钮名称和权限标识，保留未删除按钮 ID 与原菜单参数；删除按钮会清理关联授权，修改标识需同步页面使用的 `菜单name:按钮name`。角色页按菜单或 API 分组筛选、查看新增/撤销明细并分别保存；修改非当前角色仅局部更新，修改当前角色才刷新权限。API 策略使用 `/v1/sys/...` 与实际方法，不包含 `/api` 代理前缀。
6. **验证并更新文档。** 检查成功与失败、空数据、重复提交、零值 / 空值、普通角色无权限等流程，重新生成 Swagger。

前端业务代码尽量独立于 Vben 原有页面和共享包，通过配置、适配器、插槽扩展；确需改动框架文件时记录原因与回归项，便于以后升级。详细接入、AI 页面及 Vben 升级约定见配套前端 [README](https://github.com/yh-zero/go-zero-admin-vben/blob/HEAD/README.md)，未完成验收见前端根目录 FRONTEND_DEVELOPMENT_PLAN.md。

### 常用生成命令

下列命令在后端根目录执行。生成前保留当前改动，生成后检查差异，尤其是自定义 handler、鉴权、上传限制和已实现的业务逻辑。

```powershell
# 修改 .api 后生成 API 代码
.\test\sh\api.bat applet

# 修改 .proto 后生成 RPC 代码，需要 protobuf 工具在 PATH 中
.\test\sh\rpc.bat applet applet
.\test\sh\rpc.bat ai ai

# 格式化 API 定义
goctl api format --dir ./application/applet/api/desc

# 更新接口文档
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\swagger.ps1
```

API 脚本使用 `test/goctl` 下的自定义模板：

| 模板 | 作用 |
| --- | --- |
| [handler.tpl](test/goctl/api/handler.tpl) | 使用 `result.HttpResult` 统一响应 |
| [main.tpl](test/goctl/api/main.tpl) | JWT 失效回调返回统一业务错误码 |

`test/goctl/1.10.3/api/` 保留版本快照，实际 API 脚本使用版本无关的模板覆盖。RPC 脚本保留 `--name-from-filename`，避免入口名称随 proto 包名变化。升级 Go、go-zero、goctl 或 Vben 时分开处理，先验证生成差异，再做功能回归。

模型位于 `application/applet/rpc/internal/model/`。使用 GORM 生成工具时先输出到临时目录，再合并需要的结构；不要直接覆盖项目已有模型。创建包含软删除字段的记录时，确保 `DeletedAt.Valid = false` 或省略 `deleted_at`，避免写入无效的零日期。

## 维护约定与升级

### go-zero 分层与生成

`.api` / `.proto` 是接口来源；handler 负责参数解析与统一响应，RPC server 转发到各方法独立 logic。共享依赖放 `internal/svc`，配置结构放 `internal/config`，普通参数放 `etc/*.yaml`，真实密钥通过进程环境注入。API 不连接 MySQL，业务和 AI 分别由独立 RPC 执行；`pkg` 仅承载实际跨模块复用的实现。

生成的 pb、client/server、types、routes 随契约更新，不手工维护第二份契约。代码生成前保存差异，在临时目录核对新版模板，合并项目的统一响应与 JWT 失效回调，再逐项检查上传、鉴权、上下文、事务及现有业务逻辑。不要直接覆盖已有 logic 或模型。当前模板位置和命令见上节；工具版本以脚本、锁定依赖及验证结果为准，升级时不要无条件使用 `@latest`。

AI 共享模型与任务管理器在 `svc.StartAgentRuntime` 初始化，入口注入 `logic.NewAgentExecutor`；重复初始化会拒绝，`svc` 不依赖 `logic`。授权阶段可以复用同次会话检查，执行权限和工具资源权限仍分别校验；模型请求前、工具前后及保存答案前重新鉴权。持久化结果限额包含正文与保留进度；不能用精简结构移除数据范围、取消、审计或事务边界。

Go、go-zero、goctl、GORM/JWT 和前端框架升级在独立分支分开进行，固定目标版本，审阅生成差异，验证编译、vet、测试及实际鉴权与登录。升级步骤不保证跨版本 API 全部兼容；实际判断以当前源码与回归结果为准。

### 权限写入与恢复

权限变更使用 `accessutil.PolicyTransaction`：同一带 context 的事务锁定管理员保护行，更新业务和 `casbin_rule`，校验管理员恢复入口，递增 `sys_policy_versions(id=1)` 并写 `commitPolicy` 审计，任何步骤失败一起回滚。提交后通过容量 1 的唤醒队列通知单个后台工作线程，响应不等待 Redis 发布或策略重载。生产 enforcer 使用只读适配器，禁止直接 `AddPolicy`、`RemovePolicy`、`SavePolicy` 绕过该事务。

实例默认每两秒检查数据库版本，Redis 通知或重连时立即检查；每个 `Enforce` 在鉴权前再次核对版本，落后时重载。数据库不可核对或重载失败则拒绝鉴权，恢复后重试；通知只是加速，丢失不影响数据库版本。检查、加载和发布有三秒上限并遵守请求期限；重载前后比对版本，连续变化最多重试三次。`ServiceContext.Close()` 取消工作线程、订阅和发布，再关闭数据库。人工修改策略必须同事务递增版本并记录审计，否则同步器无法识别。

菜单/按钮授权使用 `AdminMenuTransaction`，同事务保护管理员菜单入口并写 `commitMenu` 审计。HTTP 失败审计与敏感变更事务审计分工，均不保存密码、JWT、验证码、完整请求或策略内容。生产 RPC 认证和密钥轮换见 [部署说明](docker/部署说明.md)。

### 通用模块与数据边界

- 部门、岗位编码忽略大小写判重；禁止组织环和有引用的部门删除。已分配停用关联可保留或移除，新分配要求关联及部门祖先存在且启用。角色原有效 `custom` 范围可保留其已关联停用部门；从其他范围切换、新增或移除后重加需启用。角色 1 固定全部，未配置普通角色默认本人；范围用于文件的真实 SQL，旧管理接口仍按自身 Casbin 权限处理。
- 文件上传保留 `fileImgUrl` 并返回 `fileId`；`visibility=private` 使用私有 ACL 与 300 秒签名地址，默认公开。owner/dept 由服务端确定，部门为上传时快照；对象 key 区分大小写，不持久化签名 URL。引用唯一且幂等，首版引用维护仅角色 1，有引用不可删除。
- 文件删除 `active → deleting → deleted`，待删状态拒绝新增引用；OSS 失败保留 `deleting` 供重试。登记结果不确定时用相同对象 key 重试，仍不确定保留对象和核查键，避免误删已成功登记的对象。资源锁内的引用检查使用锁定当前读，防止 MySQL REPEATABLE READ 快照漏判。
- 设备数据来自可信请求来源及 User-Agent，不信任客户端伪造身份或转发 IP；不返回 token。本人接口始终限定本人，管理员设备接口另受角色/API限制。历史无设备 ID 的 JWT 只保留版本校验，不能单设备撤销；组织归属变更撤销旧会话。

### 菜单路径兼容迁移

`20261003_frontend_routes_compat.sql` 从根/绝对路径解析菜单完整 URL，压缩重复斜线并按大小写规范检查冲突。只调整可确认未定制的五个模块原始种子，保留菜单 ID、按钮与授权；冲突时使用 `<原路径>-module-<菜单ID>`，再尝试 `-1` 至 `-99`。修复和审计同事务，重复执行幂等。

输出 `UNCHANGED`、`REPAIRED`、`MISSING` 或 `SKIPPED`。种子已定制、路径无法确认或候选耗尽时保留原数据，人工核对冲突；不会自动处理两个自定义菜单冲突。不得修改已经应用的前一份迁移或校验和。隔离回归入口为 `test/sh/db-regression.ps1`。

## 可选服务配置

### 图片上传

配置 API 的 `Oss.Endpoint`、`AccessKeyId`、`AccessKeySecret`、`BucketName`。本机也支持通过 `AliYunOss_go-zero-admin` 环境变量传入同名字段组成的 JSON；Docker 部署模板使用 `OSS_*` 环境变量。

接口为 `POST /v1/sys/base/uploadFileImg`，使用 `multipart/form-data`，文件字段名 **`file_img`**；支持 JPG、PNG、GIF、WebP，单文件最大 **10 MB**。未配置 OSS 时其他管理模块仍可运行，上传会返回配置或服务错误。

### 邮箱验证码

API 支持 `Mail` 配置，也支持以下环境变量：

| 环境变量 | 含义 |
| --- | --- |
| `SMTP_HOST` | SMTP 服务地址 |
| `SMTP_PORT` | 端口，默认 `465` |
| `SMTP_USER` | 登录账号 |
| `SMTP_PASSWORD` | SMTP 密码或服务商授权码 |
| `SMTP_FROM` | 发件邮箱 |

当前采用隐式 TLS 连接，需使用匹配的 SMTP 服务端口。验证码 5 分钟有效，同一邮箱至少间隔 60 秒；强制重发也不能绕过间隔限制。当前只实现发送接口，尚未形成邮箱绑定、邮件注册或找回密码的完整流程。

部署 Compose 已传递上述 `SMTP_*` 变量，填写 `docker/.env.deploy` 后重新创建 API 容器生效。本机运行时设置对应进程环境变量，或填写配置中的 `Mail`。

## 服务器部署

用于开源项目公开参考时，可使用可选的 [只读演示部署](docker/部署说明.md#公开只读演示本次云部署)：本机构建前后端运行包，服务器 `prepare → init → start`，自动配置站点 HTTPS、独立凭据和普通演示账号，AI 服务关闭且无 Key。以下保留普通完整部署流程；演示部署不替代真实业务权限设计。

演示站点损坏时参考根目录 [快速恢复](QUICK_RECOVERY.md)：校验发布包并准备独立新环境，保留原数据库和回退路径；整机丢失需要本机发布包与服务器之外的数据备份。

开发与部署配置分开使用：

| 项目 | 本地开发 | 服务器部署 |
| --- | --- | --- |
| Compose 文件 | `docker-compose.yml` | `docker/deploy-compose.yml` |
| Compose 项目名 | `go-zero-admin` | `go-zero-admin-deploy` |
| API / 业务 RPC / AI RPC | 本机或 VS Code 运行 | Docker 构建并运行，服务名 `api` / `rpc` / `ai-rpc` |
| 参数文件 | 依赖 Compose 不需要 `.env`；AI 参数在 `etc/ai.yaml`，Key 在 `.env.local` | AI 参数在 `docker/ai-rpc.yaml.template`，Key 与部署凭据在 `docker/.env.deploy` |
| 默认数据库名 | `goZero-admin` | `gozero-admin` |
| 数据卷 | 开发专用 | 部署专用，不自动共享开发数据 |

### 1. 准备完整源码与部署配置

服务器安装 Docker 与 Compose，获取完整后端源码。Docker 构建需要 `application/`、**`pkg/`**、`go.mod`、`go.sum`、`docker/` 和 `data/api/generated/`（包括嵌入文档的 `spec.go`）；数据库管理还需要 `data/db/` 与 `test/sh/`。只上传 `application/` 无法构建。推荐使用完整仓库；若打包上传，至少包含：

```sh
tar --exclude=docker/.env.deploy -czf go-zero-admin.tar.gz .dockerignore application pkg data/db data/api/generated docker test/sh go.mod go.sum
```

在服务器的后端根目录执行（Linux Shell）：

```sh
# 仅首次复制；已有服务器配置应保留
cp docker/deploy.env.template docker/.env.deploy
chmod 600 docker/.env.deploy
```

编辑 `docker/.env.deploy`，替换数据库密码、JWT 密钥、`RPC_AUTH_TOKEN` 和默认重置密码等占位值。API → 两个 RPC、AI RPC → 业务 RPC 复用同一组 RPC App/Token；两个服务分别登记认证凭据，已有不同凭据会拒绝启动，轮换步骤见部署说明。部署模板的 `DEFAULT_USER_PASSWORD` **只影响重置密码操作，不会修改初始化 SQL 中的 admin 密码**。AI 普通参数编辑 `docker/ai-rpc.yaml.template` 的 `AI` 块；`DEEPSEEK_API_KEY` / `QWEN_API_KEY` 在 `.env.deploy` 填写，只注入 `ai-rpc` 容器。AI 默认关闭，启用条件见后文。

默认 MySQL、Redis、etcd 及两个 RPC 不向宿主机暴露端口；RPC 容器分别监听内部 `6001`、`6002`。API 仍只发布 `7001`，Swagger 仅绑定本机，外部访问需配置 HTTPS 反向代理或 SSH 隧道。`REDIS_PASSWORD` 同时配置 Redis 服务、健康检查与三个后端服务，修改后应重新创建相关容器。

### 2. 初始化依赖与数据库

```sh
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml config --quiet
sh test/sh/db.sh init deploy
sh test/sh/db.sh status deploy
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml ps
```

Windows 部署的等效命令：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\db.ps1 -Action Init -Environment Deploy
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test\sh\db.ps1 -Action Status -Environment Deploy
```

部署模式 `Init` 只启动 MySQL、Redis、etcd，等待就绪后自动备份、迁移；三个后端服务在迁移成功后再启动。脚本读取部署 Compose 与 `docker/.env.deploy`，不要遗漏 `deploy` / `-Environment Deploy`，否则会操作开发环境。数据库管理行为、错误处理和锁说明与前文一致。

### 3. 构建并启动后端

```sh
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml up -d --build
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml ps
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml logs --tail=100 api ai-rpc rpc
curl -i http://127.0.0.1:7001/v1/sys/randomImage
```

Compose 中API和AI RPC分别依赖业务RPC，API启动不依赖AI；默认`up`构建并启动三服务。镜像包含 `applet-rpc`、`applet-ai-rpc`、`applet-api` 二进制。Swagger通过容器网络访问 `http://api:7001`。同一台机器若还在运行开发服务，需要调整部署端口，项目名称不同不能解决宿主机端口冲突。

### 4. 构建与发布前端

在前端仓库根目录执行：

```sh
pnpm install --frozen-lockfile
pnpm check:type:antdv-next
pnpm build
```

将 `apps/web-antdv-next/dist/` 发布到静态站点目录。当前生产配置使用 `/api` 作为接口地址，需由 Web 服务器代理到后端，**保留 `/v1/sys` 或 `/v1/ai`，去掉 `/api`**。AI 不需要新增前端代理或 HTTP 端口。例如 Nginx 与后端同机、后端使用默认发布端口时：

```nginx
location /api/ {
    proxy_pass http://127.0.0.1:7001/;
    proxy_set_header Host $host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    client_max_body_size 11m;
}
```

`proxy_pass` 的末尾斜杠用于去掉 `/api/`。若 Nginx 也在容器里，应按实际网络使用 `api:7001` 等可达地址，不能照搬宿主机地址。当前生产路由默认 hash 模式；改为 history 模式时再配置页面回退到 `index.html`。后端部署 Compose 不包含前端站点。

### 5. 更新与停止

更新时保留服务器的 `docker/.env.deploy`。先停止 HTTP 入口 `api`，再停止后台任务服务 `ai-rpc` 和业务 `rpc`，确认三个服务全部停止后备份、执行迁移，成功后才运行新版本；仅停止 API 不能阻止 AI 后台写库。应用发布不会自动执行增量 SQL，本次 AI 服务拆分也没有新增迁移，继续按 `Status` 应用尚未登记的既有脚本。先在开发或构建环境重新生成并提交 Swagger，服务器再构建，确保 RPC 内嵌文档和接口声明一致。

```sh
# 更新源码，保留 docker/.env.deploy
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml stop api ai-rpc rpc
sh test/sh/db.sh backup deploy
sh test/sh/db.sh status deploy
sh test/sh/db.sh migrate deploy
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml up -d --build
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml logs --tail=100 api ai-rpc rpc

# 日常停止，保留数据卷
docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml stop
```

Windows 使用 `db.ps1 -Action Backup|Status|Migrate -Environment Deploy` 对应操作（每次选择一个 Action）。只更新 Swagger UI 的显示文件可以直接刷新；要更新接口同步预览仍需重建 RPC。不要用 `down -v` 代替停止或升级。

## 检查与测试

后端根目录：

```powershell
go test ./...
go vet ./...
docker compose config --quiet

# 一键格式检查、静态检查、测试及构建
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/verify.ps1

# 附加隔离 MySQL 迁移回归，不升级当前业务库
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/verify.ps1 -IncludeDBRegression
```

前端根目录：

```powershell
pnpm check:type:antdv-next
pnpm test:unit
pnpm build
```

部分 MySQL 专用回归需要显式 `GO_ZERO_MYSQL_TEST_DSN`，默认全量检查会标记跳过。配置具有 CREATE/DROP DATABASE 权限的专用测试账号及服务地址后运行下面的定向命令；这些 fixture 忽略 DSN 中原库名，自动建立并清理随机隔离库，不打开或升级原数据库。凭据只在当前测试进程环境提供，结束后移除，不保存到文档或日志；首次运行前仍须核对实际测试源码及目标 MySQL。

```powershell
# 先在当前独立终端安全设置 GO_ZERO_MYSQL_TEST_DSN，提供测试 MySQL 服务与专用账号。
go test ./pkg/agentjobs ./application/applet/rpc/internal/logic/fileresourceservice -run MySQL -count=1
Remove-Item Env:GO_ZERO_MYSQL_TEST_DSN
```

单元测试通过后仍需真实联调：验证码登录、菜单刷新、列表查询和分页、增删改、角色菜单 / 按钮 / 接口授权、无权限拒绝、重复数据和字段清空，以及改密 / 退出 / 冻结后旧会话失效。`session-smoke.mjs` 提供真实会话回归；需正常输入验证码，不会绕过登录。

仓库提供 [登录辅助脚本](test/sh/login.mjs) 与 [HTTP 冒烟脚本](test/sh/smoke.mjs)。仅在本地开发库执行，脚本会创建和清理测试业务数据；原始 HTTP 验收证据以 `test/reports/` 的报告为准，剩余验收见配套前端根计划。

```powershell
# 在后端根目录取验证码，脚本会输出图片路径
node .\test\sh\login.mjs image

# 查看图片后，将 123456 替换为当前图片中的六位字符
node .\test\sh\login.mjs login 123456

# 使用临时登录会话执行冒烟测试
node .\test\sh\smoke.mjs "$env:TEMP\go-zero-admin-regression\session.json"

# 会话回归会为测试用户提示新的图片验证码，按提示输入
node .\test\sh\session-smoke.mjs "$env:TEMP\go-zero-admin-regression\session.json"
```

登录辅助文件保存在系统临时目录，测试报告输出到 `test/reports/latest-smoke.json`。OSS 上传和邮件送达需真实外部服务配置后单独验证。

原始 HTTP、会话与 API 同步验收证据以测试报告为准；新增模块及 AI 的人工登录、云服务和生产验收仍见 [后续开发计划](DEVELOPMENT_PLAN.md)。

## 常见问题

| 现象 | 排查方向 |
| --- | --- |
| `pwsh` 无法识别 | 使用本文的 `powershell.exe -NoProfile -ExecutionPolicy Bypass -File ...` 命令 |
| Docker 已启动，但登录请求失败 | 根 Compose 只启动依赖；确认业务 RPC 6001、API 7001 已启动；AI 功能还需要 AI RPC 6002 |
| MySQL 连接失败 / 找不到库 | 核对容器健康、密码及大小写；开发库是 `goZero-admin`，部署默认是 `gozero-admin` |
| 修改密码配置后仍无法连接 MySQL | 已有数据卷不会自动更新数据库密码；使用实际库密码并同步配置 |
| 新库页面缺少菜单或按钮、接口返回 403 | 先运行数据库 `Status` / `Migrate`；再检查当前角色的菜单、按钮和 API 授权，确认方法与完整路径匹配 |
| 迁移提示已应用文件被修改 | 恢复该文件的已发布内容，另新增迁移；不要手改 `schema_migrations` 绕过校验 |
| 迁移提示锁已存在 | 确认其他迁移已结束，核对失败原因后清理 MySQL 容器内残留空目录 `/tmp/gozero-db-migrate.lock` |
| API 同步预览仍是旧接口 | 先重新生成 Swagger，再重新构建并启动 RPC；仅刷新 Swagger UI 不会更新 RPC 内嵌文档 |
| HTTP 200 但操作失败 | 检查响应中的 `code` 和 `message`，不能只看 HTTP 状态 |
| 验证码无效 / 登录后立即失效 | 重新获取验证码；核对 Redis、API/RPC JWT 密钥与令牌到期时间 |
| Swagger 能打开，Execute 返回 502 | 确认 API 正在运行，`SWAGGER_API_URL` 能从容器访问 |
| 接口路径出现 `/api/v1/sys` | `/api` 是前端代理前缀，真实后端和 Casbin 配置使用 `/v1/sys/...` |
| 菜单显示“尚未接入” | 对应组件尚未映射到真实业务页面，在前端业务目录新增页面并补白名单映射 |
| 上传或邮件提示未配置 | 配置 OSS / SMTP 并重启 API；这些配置不影响其他管理模块启动 |
| 删除菜单 / 角色 / 字典失败 | 检查子节点、用户、授权或字典项引用，根据后端提示先处理依赖关系 |
| 前端端口不是 5999 | 默认端口被占用时 Vite 可能选用下一端口，以启动输出为准 |

## AI Agent 本机配置

AI 普通参数集中在 [application/ai/rpc/etc/ai.yaml](application/ai/rpc/etc/ai.yaml) 的 `AI` 块：`Enabled`、`Provider`、`MaxInputChars`、`MaxSteps`、`MaxRunSeconds`，以及 `DeepSeek` / `Qwen` 的 `Model`、`BaseURL`。默认 `Enabled: false`，独立 AI RPC 仍可启动并提供本人历史/状态查询。实际运行时设置 `Enabled: true`、选择 `Provider: deepseek` 或 `qwen`，填写所选提供商支持 function calling 的模型名和可达 endpoint；配置错误会让 AI 服务拒绝启动，业务 RPC 和 API 可单独运行。当前用户还需 AI 菜单、按钮和 HTTP API 授权，业务工具继续核对原有权限。

模型 Key 继续放在后端根目录被 Git 忽略的 `.env.local`；`.env.local.example` 只保留 `DEEPSEEK_API_KEY` 和 `QWEN_API_KEY` 两个空字段。AI RPC 根据 `AI.Provider` 读取固定的对应 Key，YAML 不保存密钥。旧 `AI_AGENT_*`、`DEEPSEEK_MODEL` / `DEEPSEEK_BASE_URL`、`QWEN_MODEL` / `QWEN_BASE_URL` 不再控制配置，升级时将非密钥值迁移到 YAML 并从环境文件移除旧字段。

```powershell
# 在后端根目录操作，只在不存在时创建，保留已有配置。
if (-not (Test-Path -LiteralPath .env.local)) {
    Copy-Item -LiteralPath .env.local.example -Destination .env.local
}
# 编辑 application/ai/rpc/etc/ai.yaml 的 AI 块和 .env.local 中所选提供商的 Key。
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Restart
```

文件支持 UTF-8、空行、整行 `#` 注释、`KEY=value` 与成对单/双引号；值按原文读取，不展开变量、不执行命令，`=` 与 `#` 可以作为值内容。格式错误在停止旧服务前拒绝，仅提示文件与行号。Start/Restart 在操作进程前完整解析此文件，已有非空进程环境 Key 优先；Stop/Status 不读取。脚本执行完恢复调用者环境，不修改系统或用户环境。Key 仅由 `ai-rpc` 子进程继承，业务 RPC、API、Go 构建及数据库/依赖子进程不继承 Key；普通模型参数以 YAML 为准。若进程环境已覆盖 Key，先移除覆盖或新开终端再验证文件修改。实际密钥文件为明文，应限制访问权限；Git 和 Docker 构建上下文已排除本地密钥文件，不自动向 Docker 或前端复制真实值。直接 `go run` 或 VS Code 启动不会自动读取 `.env.local`，手动启动时只在 AI RPC 的独立终端配置 Key，前端 `.env*` 不存放模型密钥。

Docker 参数编辑 `docker/ai-rpc.yaml.template`，Key 编辑 `docker/.env.deploy`。修改任一文件后，在后端根目录执行 `docker compose --env-file docker/.env.deploy -f docker/deploy-compose.yml up -d --build --no-deps ai-rpc`，构建并重新创建 AI 容器；单纯 `restart` 不更新容器环境。涉及源码或数据库整体升级则按前文停止全部三个服务。Qwen 的 endpoint 需与账号区域匹配；功能关闭但 AI 服务在线时仍可读取本人历史。真实云端及部署验收见 [后续开发计划](DEVELOPMENT_PLAN.md)。

## AI 接口与运行边界

独立 `application/ai/rpc` 使用 Etcd 键 `ai.rpc`，业务 RPC 使用 `applet.rpc`；API 的可选 `AIRPC` 与 `AppletRPC` 分开。AI 经业务 `CheckSession` / `Enforce` 及 `AgentTools.QueryAgentAudit`、`GetAgentFileStatus`、`ListAgentDevices` 执行只读业务查询。业务 RPC 不初始化模型或任务运行器，工具名称/schema 位于 `pkg/agenttools`。

API 的 AI 客户端使用唯一后台工作线程初始化：go-zero 的 Etcd 初始读取可能同步重试，单设 `NonBlock=true` 不足以避免阻塞。初始化失败从 1 秒退避至最多 30 秒；六个接口共用状态，请求的初始化等待与实际 RPC 共用配置 Timeout 并响应取消。Etcd 恢复后继续连接，无需重启 API。框架 subscriber 初始读取不可取消，该线程最多一个，随 API 进程存活；请求不会新建恢复线程。AI 未配置或离线时返回安全错误 `100001`，普通 API 仍可启动；AI 历史也需要 AI 服务在线。

| HTTP | 用途 | AI RPC |
| --- | --- | --- |
| GET `/v1/ai/info` | enabled/configured、模型、工具可用性和执行限制 | GetAgentInfo |
| POST `/v1/ai/runs` | conversationId 可选、requestId、message；创建异步任务 | CreateAgentRun |
| GET `/v1/ai/runs/:id` | 真实进度和最终结果 | GetAgentRun |
| POST `/v1/ai/runs/:id/cancel` | 幂等停止，随后查询实际状态 | CancelAgentRun |
| GET `/v1/ai/conversations` | 本人会话分页 | ListAgentConversations |
| GET `/v1/ai/conversations/:id/messages` | 本人消息分页 | ListAgentMessages |

共 6 个 HTTP、6 个 AI RPC 与 3 个内部业务工具 RPC，不新增对外 HTTP 端口。页面需 AI 菜单，提交/停止按钮为 `ai-agent:run` / `ai-agent:cancel`，六个 HTTP API 分别授权。Actor 来自认证 API，模型前、工具前后和保存答案前继续检查会话、执行/资源权限与数据范围；注销、冻结或撤权后不能保存答案或继续读取新数据。

内置三个只读工具：审计日志最多 31 天的汇总/最近记录，已知 fileId 的文件状态/引用数，以及本人有效设备。文件工具不枚举全部文件或返回签名地址；本人设备不查询其他人。发给云模型的内容包括用户问题、最近成功历史和最少必要授权结果；不发送原始审计正文/IP/token、对象 key/签名 URL、设备会话 ID/原始 IP。工具文字视为不可信内容，不允许据此提升权限。

任务状态为 `queued`、`running`、`succeeded`、`failed`、`cancelled`、`interrupted`。requestId 为 UUID；结果不确定的重试保留同一 requestId、问题和会话，内容不同会拒绝。每会话最多一个活跃任务、每用户最多两个；外部模型请求失败不自动重试，避免重复计费。成功历史按 sequence 稳定分页，同轮 assistant 在 user 前，前端反转当前页显示；上下文最多取最近 12 条成功消息并裁去最早完整轮次，初始历史不超过 64 KiB，每轮输入/实际 HTTP 请求正文有 256 KiB 上限。

成功答案会附上服务端生成的“查询明细”，覆盖审计汇总/最近记录、文件状态和本人设备。即使模型只给出概述，授权字段仍可通过现有任务答案和历史消息显示，无需新增 API、RPC 或迁移。明细按字段白名单生成，区分匹配总数、返回样本与空结果；每组最多 20 条，明细总量不超过 4 KiB，单元格最多 64 个字符，超限有提示。答案默认上限 8 KiB；概述与明细合计超限时保留完整明细并说明省略概述。保存前重新检查会话、AI 执行权限和实际使用的工具权限；任务失败、取消或撤权时不附加明细。

工具进度只持久化名称、状态、摘要；成功答案中的明细仅保存上述授权展示字段，不保存原始工具参数/JSON、对象 key、签名 URL、设备会话 ID 或隐藏推理。前端使用受限 Markdown 和 Vue 转义文本节点展示表格、列表、标题和代码，不执行 HTML，不生成可点击链接或加载图片。DeepSeek 请求关闭 thinking，Qwen 设置 enable_thinking=false；兼容服务返回的工具回合推理仅在当前请求内存暂存/回传，不给前端或数据库。

查询展示的自动回归覆盖任务答案、历史持久化、空结果、显示限额及失败/撤权。`TestLiveProviderPersistsAuthorizedQueryDetails` 默认跳过；在独立测试终端安全设置所选提供商的 Key 和 `GO_ZERO_AI_LIVE_TEST=1` 后，可执行 `go test ./application/ai/rpc/internal/logic/agent -run '^TestLiveProviderPersistsAuthorizedQueryDetails$' -count=1 -v`。该测试最多发起两次付费模型请求，业务 RPC 和数据库使用隔离样例，不能替代正常登录后的真实业务验收；结束后移除测试标记和 Key 环境变量。

当前三个 AI 表与业务共用 MySQL：提交限流锁定 `sys_users`，任务/取消与审计同事务写 `sys_audit_logs`，不能将 AI 数据库账号权限仅限三表。提交使用 READ COMMITTED，不改变全库隔离级别。优雅退出取消运行任务；强制结束后租约到期收敛为 interrupted，不重放已领取的付费调用。迟到结果不能覆盖取消/中断或重新写答案；已返回用量可以保留，但未返回请求的计费未知。业务 RPC 离线时在下一鉴权点拒绝结果，已发 HTTP 仍受任务预算约束。

执行组件使用 [Eino](https://github.com/cloudwego/eino) 及 [Eino 扩展](https://github.com/cloudwego/eino-ext)。扩展、真实云调用及生产多实例验收统一记录在 [后续开发计划](DEVELOPMENT_PLAN.md)。

## 相关文档与许可证

- [前端 README：启动、接口接入与 Vben 升级](https://github.com/yh-zero/go-zero-admin-vben/blob/HEAD/README.md)
- [后端后续开发与剩余验收](DEVELOPMENT_PLAN.md)
- [Docker 部署与 RPC 密钥轮换](docker/部署说明.md)
- [当前 Swagger 定义](data/api/generated/go-zero-admin.swagger.json)
- [go-zero](https://github.com/zeromicro/go-zero) · [Vue Vben Admin](https://github.com/vbenjs/vue-vben-admin)

后端采用 [Apache License 2.0](LICENSE)。配套前端基于 Vue Vben Admin，许可证及上游声明见前端仓库；二次开发时保留相应声明。

欢迎提交 Issue 和 Pull Request。提交时说明问题、修改内容和验证结果；接口变更同时提交定义、实现、必要迁移与更新后的 Swagger。

交流微信：`golang-9527`（备注：go-zero-admin）。
