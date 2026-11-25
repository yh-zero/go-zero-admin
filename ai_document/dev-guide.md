# go-zero-admin 开发手册

首次运行基础后台请从[快速开始](../README.md#快速开始)进入，仓库根目录的 Compose 在容器内构建前后端。本手册还提供[完整平台开发栈](#完整平台开发栈)的 Windows PowerShell 命令，供开发 AI、App、租户功能或在宿主上调试服务时使用。

## 1. 开发环境与配置

### 配置双套制

| 运行方式 | 后端配置 | 如何指定服务地址 |
|---|---|---|
| 基础后台演示（仓库根目录 Compose） | API 与 file-rpc 的 YAML 来自 [docker/quickstart](../go-zero-admin/docker/quickstart)，admin-rpc 使用 [docker/etc/admin-rpc.yaml](../go-zero-admin/docker/etc/admin-rpc.yaml) | 容器间使用服务名和容器端口，浏览器使用[演示端口](#基础后台演示) |
| 完整平台开发栈（后端目录 Compose） | [docker/etc](../go-zero-admin/docker/etc) 下的 YAML 挂载到容器 | 使用 Compose 服务名和容器端口 |
| 宿主 `go run` | [application](../go-zero-admin/application) 中各模块的 `api/etc` 或 `rpc/etc` | 使用宿主地址和映射端口，`-f` 指定实际加载的 YAML |

YAML 的 `${变量名}` 由使用 `UseEnv()` 的加载入口从**进程环境**展开。完整平台的 `--env-file` 供应编排文件插值；AI/App 通过服务的 `env_file` 注入进程，API 和 tenant-rpc 的代理地址由 `environment` 注入。宿主运行需在启动终端设置相应环境变量。下表为完整平台开发栈的端口。[平台 Compose 配置](../go-zero-admin/docker-compose.yml)

| 服务 | 默认宿主地址 | 容器内 |
|---|---|---|
| 前端、平台 API | `127.0.0.1:5999`、`127.0.0.1:7001` | 前端运行在宿主，API 为 `platform-api:7001` |
| MySQL、Redis | `127.0.0.1:3307`、`127.0.0.1:16379` | `mysql:3306`、`redis:6379` |
| RustFS 数据、控制台 | `127.0.0.1:19000`、`:19001` | `rustfs:9000`、`:9001` |
| Qdrant REST、gRPC | `127.0.0.1:16333`、`:16334` | `qdrant:6333`、`:6334` |
| Nginx | `localhost:80` 或 `platform.localhost:80` | 按 Host 路由到平台或租户 |
| admin/app/file/ai/tenant RPC | 默认仅在容器网络开放 | 各服务内部依次监听 `6001` 至 `6005` |

### AI 与邮件

从仓库根目录进入后端目录，只在本地配置尚不存在时复制模板。以下后端命令均以此目录为工作目录。

```powershell
cd go-zero-admin
if (-not (Test-Path -LiteralPath .env.local)) { Copy-Item .env.local.example .env.local }
```

- **对话和知识库**。默认对话模型使用 `AI_DEEPSEEK_KEY`，知识库的 Qwen Embedding 使用 `AI_QWEN_KEY`，并需要可达的 Qdrant。调整 Provider、模型或向量地址时，修改当前服务实际读取的 AI YAML。真实模型和向量化请求会产生供应商用量。
- **邮件**。App 的 SMTP 账号、授权码与发件地址从[本地环境模板](../go-zero-admin/.env.local.example)注入。默认 `smtp.163.com:465` 使用隐式 TLS，From 留空时沿用 Username；切换服务商时同时核对 Host、Port 和 ImplicitTLS。
- **使变更生效**。修改挂载的 YAML 内容后重启读取该文件的服务。修改环境变量后重新创建相应的 `ai-rpc` 或 `app-rpc` 容器，并按功能验证。租户使用自身配置。本地配置含真实凭据时仅保存在被忽略的本地文件或进程环境中。

### 数据初始化

**完整初始化 SQL 会删除并重建业务表。** 新平台栈使用空 MySQL 卷时自动导入 [mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql)，业务库为 `gozero_admin`。已有卷会继续使用原数据；功能升级时先备份，比较目标库与初始化来源，只应用本次需要的结构、索引、种子和授权变动，保留原有数据与角色授权。已有库的整库重置需要单独授权。生成 model 使用已导入完整 SQL 的新开发库。

升级文件清理重试前，核对已有库的 `sys_file` 表。缺少清理回执列时执行 `ALTER TABLE sys_file ADD COLUMN object_cleaned_at DATETIME NULL DEFAULT NULL COMMENT '对象存储确认删除时间'`。历史软删除记录没有回执，由上传者在文件列表重试确认；不会重新开放访问链接。

启用用户权限查看、批量导入和组织通讯录时，从[完整初始化 SQL](../go-zero-admin/data/db/init/mysql-all.sql)核对“后台用户工具和独立通讯录”区段。只增量补齐菜单、按钮和接口资源，按现有库实际 ID 关联角色。全局超级管理员可使用权限查看和导入。给其他角色开放导入时，同时核对新增用户和导入两个接口的授权。给普通成员开放通讯录时，同时授予通讯录菜单及其 GET 接口，并在该用户切换到自己的部门后验证。更新 Casbin 策略后按当前部署方式刷新策略。

升级审计与文件增强前，先备份并对照完整初始化 SQL 增量执行：`sys_file ADD COLUMN sha256 char(64) NOT NULL DEFAULT '' COMMENT '文件内容 SHA-256，历史文件为空'`，为 `sys_audit_log` 增加 `idx_username/idx_method/idx_ip`，为 `sys_file` 增加 `idx_original_name/idx_mime/idx_created_at`。不要重复添加已有列或索引。部署后通过“接口管理 → 同步路由”登记审计导出、文件详情、存储统计和 multipart 上传路由；普通角色沿用现有审计列表、文件列表/上传操作码，按需单独授权并验证。

审计自动保留默认关闭。确需启用时在 admin-rpc YAML 设置 `Audit.RetentionDays` 为正整数，并可调整 `CleanupIntervalSeconds`（默认 3600）及 `CleanupBatchSize`（默认 1000）。该功能按 `created_at` 删除到期审计，启用前先备份并验证保留天数；已有租户配置需分别迁移。

权限模型升级先备份，再创建 `sys_permission(menu_id,api_id)` 和 `sys_policy_state`（id=1 初始 revision=1）。按当前完整初始化 SQL 核对菜单与 API 的映射，旧库只增量补齐，不重复插入已有映射。角色授权现在按映射重建 API 策略，非超级管理员不能修改自己绑定的角色或授予超出自身的权限/数据范围；已有自定义角色应先核对映射和绑定关系。权限诊断为 `POST /v1/sys/permission/check`，可查看 allowed、命中角色、策略 revision、生成和加载时间。

### 数据库版本化迁移

新库继续使用 `data/db/init/mysql-all.sql` 完整初始化；已有库使用 `data/db/migrations/*.up.sql` / `*.down.sql` 和 `application/migrate` 补齐结构。迁移命令不修改业务代码，默认读取 `application/migrate/etc/migrate.yaml` 的 MySQL DSN，执行前必须备份并核对实际目标库。

```powershell
# 查看当前版本、dirty 状态和迁移清单
go run ./application/migrate -action status

# 执行全部待迁移版本
go run ./application/migrate -action up

# 仅在已核对目标库后回退一步
go run ./application/migrate -action down -yes

# 仅在确认当前 schema 与版本一致后修复版本状态
go run ./application/migrate -action force -version 2 -yes
```

`status` 输出 `current_version` 和 `dirty`，迁移失败时 dirty=true，先修复失败原因再重试，不要用 force 掩盖未完成的结构变更。当前迁移 001 为审计/文件增强，002 为权限映射和诊断资源；001/002 均按 information_schema 做存在性检查，可在包含这些结构的开发库上重复执行。down 仅用于开发恢复，生产回滚优先使用迁移前备份。

开源演示账号见[快速开始](../README.md#快速开始)。首次安装已有基础演示数据，历史前端样例导入见[前端说明](../go-zero-admin-react/README.md)。

### 数据库备份

1. 核对 MySQL 连接、容器和库名，用一致性快照导出**完整运行库原始 SQL**。记录来源、时间和文件校验值，使用新文件名保留已有备份。完整初始化 SQL 与运行库快照分别保存。
2. 在独立、空的环境恢复，检查关键表、数据和关联；验证环境保持业务进程停止，避免触发真实邮件或模型调用。
3. 按授权目录与分支交付，公开仓库遵守[安全基线](../AGENTS.md#7-安全基线)。受敏感信息限制时保留本地完整原件。交付记录分别说明导出、恢复验证以及实际提交和推送的结果。

### AI 本地开发免图形验证码

AI 在本机使用完整平台开发栈联调和验收时，默认使用本地覆盖配置。第一次配置时，将后端 `docker/etc/admin-rpc.yaml` 复制为 `docker/etc/admin-rpc.ai-dev.local.yaml`，把副本中的 `Captcha.Enabled` 改为 `false`。已有副本先核对，保留本地设置。在后端目录创建 `docker-compose.ai-dev.local.yaml`：

```yaml
services:
  admin-rpc:
    volumes:
      - ./docker/etc/admin-rpc.ai-dev.local.yaml:/app/etc/admin.yaml:ro
```

两个文件都在 Git 忽略范围内，仅作用于本机开发栈。继续使用有效演示凭据正常登录，保留会话、权限和服务端校验。专项测试验证码时选择标准挂载，核对接口返回的 `enabled` 为 `true`。

## 2. 构建、启动与就绪检查

### 基础后台演示

从仓库根目录运行 `docker compose up -d --build --wait`。Compose 项目名为 `gozero-quickstart`，包括 Web、API、admin-rpc、file-rpc、MySQL、Redis 和 RustFS。首次启动在独立的空 MySQL 卷中导入[初始化 SQL](../go-zero-admin/data/db/init/mysql-all.sql)，再次启动会沿用原数据。

浏览器访问[就绪接口](http://127.0.0.1:18080/ready)，确认 `code` 为 `200` 且 `result.ready` 为 `true`，再按[快速开始](../README.md#快速开始)登录并打开用户列表。启动失败时，从同一目录运行 `docker compose ps -a` 查看服务状态，运行 `docker compose logs --tail 50` 查看启动日志。`storage-init` 创建文件服务使用的公开桶和私有桶，完成后显示 `Exited (0)`。

演示 Web 和对象存储只绑定本机回环地址。Web 默认端口为 `18080`，对象存储默认端口为 `19010`。如果端口已占用，在当前 PowerShell 终端设置对应变量后启动，例如：

```powershell
$env:GOZERO_DEMO_PORT = '18081'
$env:GOZERO_DEMO_STORAGE_PORT = '19011'
docker compose up -d --build --wait
```

此例从 `http://127.0.0.1:18081` 访问后台。在仓库根目录运行 `docker compose stop` 可停用演示并保留数据卷。下次启动沿用所选端口；更换终端时先重新设置自定义端口变量。

### 完整平台开发栈

准备支持 `linux/amd64` 容器的 Docker、Docker Compose 2.24 或更新版本，以及 Go 1.27.3、Node.js 24 和 pnpm 11.28.5。版本来源见 [go.mod](../go-zero-admin/go.mod) 与 [package.json](../go-zero-admin-react/package.json)。首次构建在宿主机编译一个 API 和五个 RPC，再单独启动 Vite 开发服务器。

在后端目录准备好上文的 `.env.local`。标准开发栈使用下面一行。在同一个 PowerShell 终端执行后续命令。

```powershell
$composeArgs = @('--env-file', '.env.local', '-f', 'docker-compose.yml')
```

仅在 AI 本地开发并已创建验证码覆盖文件时，追加以下参数。

```powershell
$composeArgs += @('-f', 'docker-compose.ai-dev.local.yaml')
```

运行 `docker compose @composeArgs ps -a` 查看该配置下的容器，再选择恢复或首次构建。

**恢复原开发栈**。先核对容器属于本项目、已有镜像包含所需版本，且 admin-rpc 的挂载与所选配置一致。服务都在运行时直接检查就绪；已有容器停止时运行：

```powershell
docker compose @composeArgs up -d --no-recreate --no-build --pull never --wait
```

切换 admin-rpc 的挂载时，等其依赖就绪后单独运行 `docker compose @composeArgs up -d --no-deps --no-build --wait admin-rpc`，再从前端代理核对验证码模式。环境变量变更可使用 `docker compose @composeArgs up -d --no-deps --force-recreate --wait ai-rpc`；邮件配置改为针对 `app-rpc`。

**首次启动或后端源码已更新**。[Dockerfile](../go-zero-admin/docker/Dockerfile) 打包宿主预编译的二进制。确保基础镜像可拉取或已缓存，再从后端目录执行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File ./data/scripts/build-images.ps1
if ($LASTEXITCODE -ne 0) { throw "后端编译失败，停止启动" }
docker compose @composeArgs up -d --build --wait
if ($LASTEXITCODE -ne 0) { throw "Compose 启动失败，检查服务日志" }
```

**启动前端**。另开终端，从仓库根进入 `go-zero-admin-react`，首次执行 `pnpm install --frozen-lockfile`，成功后执行 `pnpm dev`。声明及锁文件未变化且依赖已安装时可直接启动。已有本项目的 Vite 在 `127.0.0.1:5999` 监听时复用。代理配置见[前端说明](../go-zero-admin-react/README.md#本地开发)。

### 就绪检查

在任意 PowerShell 终端执行：

```powershell
$ready = Invoke-RestMethod -Uri 'http://127.0.0.1:7001/ready' -TimeoutSec 15 -ErrorAction Stop
if ($ready.code -ne 200 -or -not $ready.result.ready) { throw "平台 API 尚未就绪" }
```

检查 API 的 `/ping` 和 [前端登录页](http://127.0.0.1:5999/login)。AI 本地模式下，请求 `http://127.0.0.1:5999/v1/sys/auth/captcha` 应返回 `code=200`、`result.enabled=false` 和空的 `captcha_id`、`image_base64`。使用正常凭据登录并打开一个获授权页面，退出本次测试会话。核对本轮的配置挂载和实际运行版本，二进制来源存疑时比较容器内和本地构建的摘要。

### 宿主单服务调试

从后端目录调试 admin-rpc 的示例：

```powershell
docker compose --env-file .env.local up -d mysql redis rustfs
if ($LASTEXITCODE -ne 0) { throw "基础依赖启动失败" }
go run ./application/admin/rpc -f application/admin/rpc/etc/admin.yaml
```

核对目标服务的依赖，并在修改后手动重启。宿主 API 默认连接宿主 `6001/6003` 的 AdminRpc/FileRpc；按其 YAML 启动所需 RPC，确保 admin-rpc 能加载策略，文件操作能访问 FileRpc。默认 Compose 只在栈内开放 RPC。宿主 API 的 `7001` 与现有 Compose 映射可能冲突，优先选择独立端口。

## 3. 日常开发与验证流程

先确认改动影响的页面、接口和直接调用方。跨端行为变更先更新[共享 API 契约](api-contract.md)，再依照[后端规范](../go-zero-admin/AGENTS.md)或[前端规范](../go-zero-admin-react/AGENTS.md)实现。开发中运行对应回归并验证受影响功能；提交前执行该端完整自检。交付与临时资源清理见[根规范](../AGENTS.md#9-任务完成定义)。

已有实例接入新页面时，在菜单和角色管理补齐对应页面与操作权限；新增 HTTP 路由通过“接口管理 → 同步路由”登记，再核对该角色的接口授权。使用目标角色访问页面并调用接口，分别确认菜单可见性和服务端权限。

后端 Go 回归从 `go-zero-admin/` 运行，前端 Node 回归从 `go-zero-admin-react/` 运行。测试 AI 的数据库与缓存分支时，`GOZERO_AI_TEST_DSN` 和 `GOZERO_AI_TEST_REDIS` 只指向独立测试实例。变量缺失时检查测试输出中的跳过项。浏览器桩、真实服务、付费模型各自记录本轮实际覆盖范围。

### 用量治理与持久结构维护

升级已有库的工作流版本或用量功能时，对照[完整初始化 SQL](../go-zero-admin/data/db/init/mysql-all.sql)核对 `ai_workflow.revision_seq`、版本表、用量账本和策略表、`idx_status_lease`、全局策略 `owner_id=0` 及菜单授权。先在隔离库验证增量变更。历史调用未建立账本的区间应在查询结果解释中注明。配额与策略冲突的处理以[共享契约](api-contract.md)为准。

### Qdrant 启动恢复与孤儿索引对账

摄取返回 503 或检索异常时，先检查当前 AI YAML 的 `Qdrant.Address`、集合、Embedding 配置和实际 gRPC 地址。配置有效时 ai-rpc 会对启动暂态故障退避重探测。之后用已有就绪文档复核；需要真实 Embedding 的操作按授权范围执行。

自动对账先取得完整索引 ID 快照，按批回查 MySQL 后只删除已确认的孤儿。快照不完整、超上限或一批查询失败时，核对日志和实际 ID。集合丢失、向量缺失、模型或维度变更，需要核对原文、配置和影响范围，评估费用后按授权逐批重建。[对账实现与限制](../go-zero-admin/application/ai/rpc/internal/svc/kb_index_reconcile.go)

### 知识库效果评测

从[知识库评测入口](ai-rag/README.md)选择冻结题集、离线校验、实跑或历史基线。只检查格式与来源时使用离线模式。执行供应商调用前确认资料范围、模型、预算和输出位置；保留题集原始字节与来源校验值，按相同口径比较结果。

## 4. 常见故障排查

以下 Compose 命令在后端目录使用启动时的 `$composeArgs`。

| 症状 | 定位与复验 |
|---|---|
| API 循环退出 | 运行 `docker compose @composeArgs logs --tail 50 api admin-rpc`，核对 RPC、数据库、Redis 和策略加载；恢复后访问 `/ready` 并正常登录。 |
| API 直连正常，Nginx 返回 502 | 核对平台 Host 与上游地址，运行 `docker compose @composeArgs exec -T nginx nginx -t`。仅配置校验成功后运行 `docker compose @composeArgs exec -T nginx nginx -s reload`，再以 `Host: localhost` 访问代理的 `/ready`。未知 Host 返回 404 属于域名隔离。 |
| 开发端口已占用，或结果来自旧版本 | 用 `netstat -ano` 和 `docker compose @composeArgs ps` 核对 PID、映射及 Vite 代理目标。只操作已确认属于本项目的实例。 |
| 新 RPC 报 `unknown service` | 核对调用方端点、对应模块 main 的服务注册和生成脚本告警，重新构建运行产物后复验该方法。 |
| 改库后仍读到旧记录或权限 | 核对受影响记录的 ID、旧新唯一索引缓存键和授权通知，精确失效后重新读取。 |
| 修改 AI 或 SMTP 配置后无效 | 核对服务使用宿主还是容器配置、实际注入的变量名及容器是否已重新创建。SMTP 同时核对主机、端口和 TLS，诊断输出避免打印密钥。 |
| SQL 导入中文异常 | 先核对 UTF-8 原始文件和目标列。将原始 SQL 复制到容器后用 `--default-character-set=utf8mb4` 从文件导入，避开 Windows PowerShell 管道转码。兼容 PowerShell 5.1 的含中文脚本保留 UTF-8 BOM。 |

**代理和网络迁移**。平台与租户 API 各自只信任实际代理的 IP/CIDR。更换网络时成组调整子网、动态地址池和固定代理地址，运行[网络校验脚本](../go-zero-admin/data/scripts/verify-network.ps1)。已有栈先记录容器、卷和端点，特别核对 Redis 持久化，再按授权迁移。容器内连通而宿主超时时，检查端口转发。额外反代应按实际地址配置可信范围。

**文件与其他专项维护**。上传超时时依次核对对象存储、file-rpc 服务端、API 的 FileRpc 客户端、上传 HTTP 路由和浏览器超时预算，按对象及引用核查处理结果。登录页素材、生成和验证见[猫咪生成器说明](../go-zero-admin-react/scripts/build-login-cat.md)。其他 go-zero 生成与运行问题见[官方排障](../.ai-context/zero-skills/troubleshooting/common-issues.md)。
