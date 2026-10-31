# 本地开发环境拓扑与联调 SOP

> 联调/验证前先读本文。环境异常时按 §3 排查，别猜。

## 1. 标准拓扑

全部后端由 `go-zero-admin/docker-compose.yml` 统一管理（Docker Desktop 归组为 `gozero-platform`）：

| 组件 | 容器 | 宿主地址 | 说明 |
|---|---|---|---|
| MySQL 9.7 | `gozero-new-mysql` | `127.0.0.1:3307` | 业务库 `gozero_admin` + 控制库 `gozero_platform`；迁移用 `goose -dir data/db/migrations/<business|platform>` |
| Redis 8 | `gozero-redis8` | `127.0.0.1:16379` | 缓存 + Casbin 策略通知频道 |
| RustFS | `gozero-integration-rustfs` | `127.0.0.1:19000`（控制台 :19001） | S3 兼容对象存储 |
| admin-rpc | `gozero-platform-admin-rpc-1` | 仅容器网络 :6001 | |
| app-rpc / file-rpc / ai-rpc | `gozero-platform-*-1` | 仅容器网络 :6002-6004 | |
| tenant-rpc | `gozero-platform-tenant-rpc-1` | 仅容器网络 :6005 | 仅平台栈 |
| admin-api | `gozero-platform-api-1` | `127.0.0.1:7001` | REST 网关，前端代理目标 |
| nginx | `gozero-platform-nginx-1` | :80 | 多租户子域名路由 |
| 前端 dev | `go-zero-admin-react` → http://localhost:5999 | `/v1` 已代理到 7001 | 宿主 pnpm dev，不进 compose |

种子账号：`admin / Admin@123456`（登录需验证码；开发环境验证码答案经 `image_base64` 直接返回，base64 解码即可）。

**配置双套制**（改配置时注意是否两套都要同步）：
- 容器内服务读 `docker/etc/*.yaml`（compose 卷挂载；地址为容器网络服务名 `mysql:3306` / `redis:6379` / `rustfs:9000`）
- 宿主 `go run` 热调读 `application/*/etc/*.yaml`（地址为 `127.0.0.1` + 宿主映射端口 3307/16379/19000）

## 2. 启动

### 方式一：compose 一键全量（日常）

```powershell
cd go-zero-admin
docker compose up -d --build   # 首次构建约 5-15 分钟，之后秒级
docker compose ps              # 确认全部 running
````

前端（单独，宿主运行）：

```powershell
cd go-zero-admin-react; pnpm dev   # http://localhost:5999
````

### 方式二：单服务 go run 热调（改代码时用）

```powershell
docker compose up -d mysql redis rustfs   # 只起依赖
go run ./application/admin/rpc -f application/admin/rpc/etc/admin.yaml   # 其余服务以此类推
````

注意：宿主 `go run` 与 compose 同时跑同一服务会抢端口（尤其 api 的 7001），二选一。

### 数据初始化（首次克隆）

`data/db/init/mysql-all.sql` 是随仓库分发的完整初始化库（admin 全量演示数据 + platform 含 3 条演示租户行），compose 已把它挂到 MySQL 的 `/docker-entrypoint-initdb.d`——**空卷首次 `docker compose up -d` 自动导入，无需手动操作**。已有数据卷不会重复执行；想重置就 `docker compose down -v` 后再 `up -d`。

初始化账号：`admin` / `Admin@123456`（仅本地开发，生产必改）；DB root 统一 `123456`。

### 数据恢复（本地快照）

`data/db/backup/` 是**本地**快照目录（已 gitignore，含口令哈希不进仓库）。恢复用**最新**目录：

```powershell
docker cp data/db/backup/<最新目录>/mysql-all.sql gozero-new-mysql:/tmp/
docker exec -e MYSQL_PWD=123456 gozero-new-mysql sh -c "mysql -uroot < /tmp/mysql-all.sql"
````

> 2026-10-31 之前的旧备份（如 20261031-112912）内含随机 root 密码，导入会把 root 锁死；之后的备份 root 统一为本地约定 `123456`，导入即用。

从零初始化（无备份、想从头跑迁移）时在宿主执行迁移与种子：

```powershell
goose -dir data/db/migrations/business mysql "root:123456@tcp(127.0.0.1:3307)/gozero_admin" up
goose -dir data/db/migrations/platform mysql "root:123456@tcp(127.0.0.1:3307)/gozero_platform" up
```

## 3. 故障排查（按命中率排序）

### ① api 启动即退出 / 反复重启
api 启动时要经 admin-rpc 加载 Casbin 策略，rpc 未就绪会 panic——compose 已配 `restart: on-failure` 自愈。
持续失败看日志：`docker compose logs api admin-rpc --tail 50`。

### ② 请求诡异失败（404/连接拒）
**先查端口被谁占了**——旧项目 docker 的 `gozero-api` 常占用 7001：
```powershell
netstat -ano | Select-String ':7001.*LISTENING'
docker stop gozero-api   # 暂停旧项目容器；联调完记得 docker start 恢复
````

### ③ 登录报"账号未激活"/数据改了不生效
goctl model 带 Redis 缓存，**直接改库后必须清缓存**（仅限本地排障；会连带清 Casbin 通知频道键，任务数据清理不适用，见根规范 §7.6）：
```powershell
docker exec gozero-redis8 redis-cli FLUSHALL
````

### ④ docker exec 里中文乱码 / 输出被吞
- 管道传 SQL 中文会乱码：改用 `docker cp` 文件进容器再执行
- mysql 客户端密码警告吞掉输出：用 `-e MYSQL_PWD=...` 代替 `-p`

### ⑤ PowerShell 脚本执行
- ps1 文件必须 **UTF-8 with BOM**（无 BOM 中文注释会解析崩溃）
- 续行符是反引号 ` 不是反斜杠 \
- 执行策略拦截：`powershell -NoProfile -ExecutionPolicy Bypass -File <脚本>`

### ⑥ goose 找不到
GOPATH 随 toolchain 切换会变。用绝对路径或重新 `go install github.com/pressly/goose/v3/cmd/goose@latest`。

### ⑦ AI 密钥
ai-rpc 的 `AI.Enabled=false` 时无需密钥；启用时目前**只能写进 yaml**（`docker/etc/ai-rpc.yaml` 或宿主 `application/ai/rpc/etc/ai.yaml`）——环境变量注入（`API_KEY_DEEPSEEK` 等）代码尚未实现，属已知缺陷。

## 4. 联调后清理（任务完成定义的一部分）

1. 停掉本次 `go run` 的进程；compose 栈按任务要求保留或 `docker compose stop`
2. 删除 `.tmp_*` 目录、调试日志
3. `docker start gozero-api` 恢复旧项目容器（如曾暂停）
4. `git status` 确认工作区干净
