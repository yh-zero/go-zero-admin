# go-zero-admin

基于 **go-zero + GORM + Casbin** 的管理后台，配套前端为 **Vue Vben Admin / web-antdv-next**。包含用户、角色、菜单、API 权限、字典、组织、文件、审计和 AI 助手。

- [后端仓库](https://github.com/yh-zero/go-zero-admin) · [前端仓库](https://github.com/yh-zero/go-zero-admin-vben)
- [在线演示](https://yh9527.top)：`admin / 123456`，需图片验证码；只读演示，AI 默认关闭。
- [Swagger UI](http://localhost:8080) · [接口 JSON](data/api/generated/go-zero-admin.swagger.json)

## 环境要求

- Go ≥ 1.24，推荐 1.26.x；Docker 与 Docker Compose v2。
- 前端：Node.js `^22.18.0` 或 `^24.12.0`，pnpm `11.16.0`。
- 下列本机脚本使用 Windows PowerShell 5.1；命令均在对应仓库根目录执行。

## 本地开发启动

### 1. 获取项目并启动依赖

```powershell
git clone https://github.com/yh-zero/go-zero-admin.git
git clone https://github.com/yh-zero/go-zero-admin-vben.git
cd go-zero-admin
docker compose up -d
docker compose ps
```

根目录 Compose 启动 MySQL、Redis、etcd 和 Swagger。等待依赖健康后继续；后端 Go 服务在本机运行。

### 2. 导入数据库

首次安装向**空数据库**导入 [data/db/gozero-admin.sql](data/db/gozero-admin.sql)。本机默认 MySQL 为 `127.0.0.1:3306`，账号 `root / 123456`，数据库名为 `goZero-admin`。

可用数据库工具选中目标库导入，或执行：

```powershell
docker cp .\data\db\gozero-admin.sql gozero-mysql:/tmp/gozero-admin.sql
docker exec gozero-mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --database="$MYSQL_DATABASE" < /tmp/gozero-admin.sql'
```

Compose 不自动导入。SQL 已包含权限版本、变更历史、稳定按钮键、管理员新编辑入口和迁移记录，首次导入后无需重复执行权限结构迁移；已有业务库按下方升级流程处理。

本次权限升级要求前后端配套更新，旧客户端没有 `expectedRevision` 时不能保存授权。已有数据库先停 API、业务 RPC 和 AI RPC，备份后由正常迁移工具执行 [权限结构迁移](data/db/migrations/20261008_permission_revision_history.sql)。服务保持停止，再审核并单独执行 [管理员恢复脚本](data/db/operator/20261008_permission_admin_recovery_bootstrap.sql)，检查角色 1 的两条恢复授权后重启。该脚本只在角色 1 已有旧恢复能力时补新编辑入口与菜单移动预览，不由通用迁移自动收集；历史、回滚接口及普通角色由管理员通过 API 同步和角色授权明确配置。已有业务库不要重新导入初始化 SQL。

### 3. 启动后端

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Start
```

脚本依次启动业务 RPC（6001）、AI RPC（6002）和 API（7001），日志位于 `bin/dev/managed/`。

### 4. 启动前端并登录

另开终端，从后端仓库目录切换到同级前端仓库：

```powershell
cd ..\go-zero-admin-vben
pnpm install --frozen-lockfile
pnpm dev
```

访问终端显示的地址，默认 [http://127.0.0.1:5999](http://127.0.0.1:5999)。首次导入 SQL 后使用 `admin / 123456` 和当前图片验证码登录。正式使用前修改初始密码及开发密钥。

### 停止与重启

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Stop
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Restart
```

依赖单独使用 `docker compose stop` 停止、`docker compose up -d` 恢复。`down -v` 会删除数据库卷。

## 配置

- [API 配置](application/applet/api/etc/applet-api.yaml)：端口、Redis、etcd、JWT、OSS 与 SMTP。
- [业务 RPC 配置](application/applet/rpc/etc/applet.yaml)：MySQL、Redis、etcd、JWT 与默认重置密码。
- [AI RPC 配置](application/ai/rpc/etc/ai.yaml)：数据库、RPC 与模型参数。

API 与业务 RPC 的 `JwtAuth.AccessSecret` 必须一致；业务 RPC 与 AI RPC 使用同一业务库。OSS、SMTP 和 AI 按需配置。

## AI Agent 本机配置

AI 默认关闭，不影响管理功能。启用时：

1. 在 [ai.yaml](application/ai/rpc/etc/ai.yaml) 中设置 `AI.Enabled: true`、`Provider`（`deepseek` 或 `qwen`）以及对应的 `Model`、`BaseURL`。
2. 密钥配置两种方式（优先级从高到低）：
   - **环境变量**：设置 `API_KEY_DEEPSEEK` 或 `API_KEY_QWEN`（推荐生产环境）
   - **YAML 兜底**：在 `ai.yaml` 的对应提供商块中填写 `APIKey` 字段
3. 使用 `dev.ps1 -Action Restart` 重启生效。

**注意**：`ai.yaml` 会提交到 Git，填写真实密钥后请勿推送；生产环境务必使用环境变量。

## 数据库升级

已有业务库使用 [data/db/migrations](data/db/migrations/) 中的增量迁移，先停止三个后端服务，再备份、迁移和启动：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Stop
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/db.ps1 -Action Backup
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/db.ps1 -Action Migrate
# 本次权限升级首次执行时：按上方说明单独运行管理员恢复脚本，核对两条授权后再启动。
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Start
```

备份位于 `bin/db-backups/`。可用 `db.ps1 -Action Status` 检查待迁移项；不要向已有业务库重放全量 SQL，也不要修改已发布的迁移文件。

## 权限与用户资源维护

四类授权读取一致快照、保存携带版本，过期编辑必须重新比对；菜单移动先确认影响预览。按钮使用稳定 `permissionKey`。授权历史和回滚接口需要明确 API 授权，首版回滚会拒绝事件之后已有任何配置修改的情况。

用户删除前可预览归属资源，默认软删后保留文件、AI 历史和审计；资源移交为独立操作。AI 预览/移交需在业务 RPC 配置可选 `UserResources.AIDataSource`，填 AI 服务实际生效 DSN，且 network/address/schema 与业务库一致。未配置、跨库或缺表时显示 AI 不可用原因，文件仍可独立移交。DSN 通过环境替换传入，不复制密码到文档或日志。

设备会话维护要求显式保留天数和批大小，默认只预演，例如：

```powershell
go run ./application/applet/rpc/cmd/sessioncleanup -f application/applet/rpc/etc/applet.yaml -retention-days 90 -batch-size 200
```

核对结果后明确添加 `-apply` 才逐批清理旧到期/撤销会话；有效会话、近期记录和审计保留，不自动设定保留期或创建定时任务。代码和验证边界见 [后端计划](DEVELOPMENT_PLAN.md#本轮权限优化开发2026-10-09)。

## 代码生成

仅修改接口契约时需要生成工具：`goctl v1.10.2`；RPC 生成另需 protobuf 工具。

```powershell
go install github.com/zeromicro/go-zero/tools/goctl@v1.10.2
.\test\sh\api.bat applet
.\test\sh\rpc.bat applet applet
.\test\sh\rpc.bat ai ai
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/swagger.ps1
```

按需执行对应命令，生成后检查差异，保留自定义逻辑。接口变更后先更新 Swagger，再构建并重启 RPC/API。

### go-zero 自定义模板

`api.bat` 通过 `-home test/goctl/` 使用项目修改过的 API 模板：

| 模板 | 改动 |
| --- | --- |
| [handler.tpl](test/goctl/api/handler.tpl) | 使用 `result.HttpResult` 包装业务返回值与错误，统一响应格式 |
| [main.tpl](test/goctl/api/main.tpl) | 设置 JWT 未授权回调，返回统一的令牌失效业务错误码 |

本文固定的 `goctl v1.10.2` 使用 `test/goctl/api/`；[test/goctl/1.10.3/api/](test/goctl/1.10.3/api/) 保留版本快照。goctl 优先使用匹配自身版本的模板目录，调整模板或升级时需核对实际目录并保留上述改动，生成后检查统一响应与 JWT 失效处理。

## 检查与测试

```powershell
# 后端：格式、静态检查、测试与构建
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/verify.ps1

# 可选：需本机 MySQL，在随机临时库验收 SQL 导入和迁移
powershell.exe -NoProfile -ExecutionPolicy Bypass -File test/sh/verify.ps1 -IncludeDBRegression
```

部分 MySQL 专用用例需设置 `GO_ZERO_MYSQL_TEST_DSN`，未配置时跳过。

前端在自己的仓库运行 `pnpm check:type:antdv-next`、`pnpm test:unit` 和 `pnpm build`。

## 服务器部署

使用 `docker/deploy-compose.yml`；完整配置、首次导入、前端发布和升级步骤见 [部署说明](docker/部署说明.md)。该文档也包含公开只读演示部署。故障恢复见 [快速恢复](QUICK_RECOVERY.md)。

## 相关文档

- [后端开发计划](DEVELOPMENT_PLAN.md) · [配套前端说明](https://github.com/yh-zero/go-zero-admin-vben/blob/HEAD/README.md)
- [部署记录](CLOUD_DEPLOYMENT_NOTES.md) · [Apache 2.0 许可证](LICENSE)
