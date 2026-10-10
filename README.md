# go-zero-admin

基于 **go-zero + GORM + Casbin** 的管理后台，配套前端为 **Vue Vben Admin**。包含用户、角色、菜单、API 权限、字典、组织、文件、审计和 AI 助手。

- [后端仓库](https://github.com/yh-zero/go-zero-admin) : https://github.com/yh-zero/go-zero-admin
- [前端仓库](https://github.com/yh-zero/go-zero-admin-vben) : https://github.com/yh-zero/go-zero-admin-vben
- [在线演示](https://yh9527.top) : https://yh9527.top（admin / 123456，只读）
- [Swagger UI](http://localhost:8080) : http://localhost:8080
- [接口 JSON](data/api/generated/go-zero-admin.swagger.json)



## 接口文档

- [Swagger UI](http://localhost:8080) : http://localhost:8080
- [接口 JSON](data/api/generated/go-zero-admin.swagger.json) : data/api/generated/go-zero-admin.swagger.json



## 环境要求

- Go ≥ 1.24，Docker 与 Docker Compose v2；前端：Node.js ^22.18.0 或 ^24.12.0，pnpm 11.16.0
- 本机脚本使用 Windows PowerShell 5.1

## 快速开始

```powershell
# 1. 启动依赖（MySQL、Redis、etcd）
docker compose up -d

# 2. 首次向空库导入 data/db/gozero-admin.sql
docker cp .\data\db\gozero-admin.sql gozero-mysql:/tmp/gozero-admin.sql
docker exec gozero-mysql sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql --user=root --database="$MYSQL_DATABASE" < /tmp/gozero-admin.sql'

# 3. 启动后端（业务 RPC 6001、AI RPC 6002、API 7001）
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Start
```

前端：在同级 `go-zero-admin-vben` 目录执行 `pnpm install --frozen-lockfile && pnpm dev`，用 `admin / 123456` 和图片验证码登录。

停止与重启：`dev.ps1 -Action Stop` / `-Action Restart`。`docker compose down -v` 会删除数据库卷。

## 配置

- [API](application/applet/api/etc/applet-api.yaml)：端口、Redis、etcd、JWT、OSS、SMTP
- [业务 RPC](application/applet/rpc/etc/applet.yaml)：MySQL、Redis、etcd、JWT、默认重置密码
- [AI RPC](application/ai/rpc/etc/ai.yaml)：数据库、RPC 与模型参数

API 与业务 RPC 的 `JwtAuth.AccessSecret` 必须一致；业务 RPC 与 AI RPC 使用同一业务库。

## AI 助手

默认关闭。启用：在 ai.yaml 设 `AI.Enabled: true` 并选择 `Provider`（deepseek 或 qwen）。密钥优先读环境变量 `API_KEY_DEEPSEEK` / `API_KEY_QWEN`，未设置时用 yaml 的 `APIKey` 字段兜底。改后 `dev.ps1 -Action Restart` 生效。

## 数据库升级

已有库升级（先停服务再迁移，不重放全量 SQL）：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Stop
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/db.ps1 -Action Backup
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/db.ps1 -Action Migrate
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/dev.ps1 -Action Start
```

备份在 `bin/db-backups/`；`db.ps1 -Action Status` 可查看待迁移项。

## 代码生成

需要 goctl v1.10.2（RPC 另需 protobuf）：

```powershell
go install github.com/zeromicro/go-zero/tools/goctl@v1.10.2
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/gen.ps1 api applet      # API
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/gen.ps1 rpc applet applet  # RPC
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/gen.ps1 swagger          # Swagger 文档
```

API 生成使用项目自定义模板 [test/goctl/api/](test/goctl/api/)（统一响应格式与 JWT 失效处理）。

## 检查与测试

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/verify.ps1                       # 格式、静态检查、测试与构建
powershell -NoProfile -ExecutionPolicy Bypass -File test/sh/verify.ps1 -IncludeDBRegression  # 附加数据库迁移/导入回归（需本机 MySQL）
```

## 脚本一览（test/sh/）

| 脚本 | 用途 |
|------|------|
| `dev.ps1` | 启动/停止/重启后端（Start/Stop/Restart/Status） |
| `db.ps1` / `db.sh` | 数据库迁移、备份、状态（Windows / Linux） |
| `gen.ps1` | 代码生成：`api` / `rpc` / `model` / `swagger` |
| `verify.ps1` | 一键检查与测试 |
| `regression.ps1` | 回归测试：`-Scope env` 环境隔离 / `-Scope db` 数据库 |
| `smoke.mjs` / `session-smoke.mjs` | API 冒烟测试 / 会话专项测试 |
| `package-demo.ps1` | 打包部署文件 |
| `env.ps1` / `mysql-client.sh` | 内部工具，被上述脚本引用 |

## 服务器部署

使用 `docker/deploy-compose.yml`，完整步骤见 [部署说明](docker/部署说明.md)；故障恢复见 [快速恢复](QUICK_RECOVERY.md)。

## 相关文档

- [配套前端](https://github.com/yh-zero/go-zero-admin-vben) · [部署记录](CLOUD_DEPLOYMENT_NOTES.md) · [Apache 2.0 许可证](LICENSE)
