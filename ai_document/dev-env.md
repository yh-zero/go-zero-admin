# 本地开发环境拓扑与联调 SOP

> 联调/验证前先读本文。环境异常时按 §3 排查，别猜。

## 1. 标准拓扑

全部后端由 `go-zero-admin/docker-compose.yml` 统一管理（Docker Desktop 归组为 `gozero-platform`）：

| 组件 | 容器 | 宿主地址 | 说明 |
|---|---|---|---|
| MySQL 9.7 | `gozero-new-mysql` | `127.0.0.1:3307` | 业务库 `gozero_admin` + 控制库 `gozero_platform`；统一导入 [初始化 SQL](../go-zero-admin/data/db/init/mysql-all.sql) |
| Redis 8 | `gozero-redis8` | `127.0.0.1:16379` | 缓存 + Casbin 策略通知频道 |
| RustFS | `gozero-integration-rustfs` | `127.0.0.1:19000`（控制台 :19001） | S3 兼容对象存储 |
| admin-rpc | `gozero-platform-admin-rpc-1` | 仅容器网络 :6001 | |
| app-rpc / file-rpc / ai-rpc | `gozero-platform-*-1` | 仅容器网络 :6002-6004 | |
| tenant-rpc | `gozero-platform-tenant-rpc-1` | 仅容器网络 :6005 | 仅平台栈 |
| admin-api | `gozero-platform-api-1` | `127.0.0.1:7001` | REST 网关，前端代理目标 |
| nginx | `gozero-platform-nginx-1` | :80 | 多租户子域名路由 |
| 前端 dev | `go-zero-admin-react` → http://localhost:5999 | `/v1` 已代理到 7001 | 宿主 pnpm dev，不进 compose |

种子账号：`admin / Admin@123456`（登录需填写页面上的一次性图片验证码；`image_base64` 编码的是 PNG 图片，解码后不会直接得到答案文本）。

**配置双套制**（改配置时注意是否两套都要同步）：
- 容器内服务读 `docker/etc/*.yaml`（compose 卷挂载；地址为容器网络服务名 `mysql:3306` / `redis:6379` / `rustfs:9000`）
- 宿主 `go run` 热调读 `application/*/etc/*.yaml`（地址为 `127.0.0.1` + 宿主映射端口 3307/16379/19000）

## 2. 启动

### 方式一：compose 一键全量（日常）

Dockerfile 打包宿主机预编译的 Linux 二进制；首次启动或修改后端代码后，先执行构建脚本，再启动 Compose。

```powershell
cd go-zero-admin
powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\build-images.ps1
docker compose up -d --build --wait
docker compose ps              # 确认服务状态
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

[data/db/init/mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql) 是数据库初始化唯一入口，包含业务库和控制库的完整结构与开发数据。每次新建库均完整导入该文件，不维护增量升级流程。Compose 已将其目录挂到 MySQL 的 `/docker-entrypoint-initdb.d`，**空数据卷首次 `docker compose up -d` 自动导入，无需手动操作**；已有数据卷不会重复执行，也不会自动更新结构。

初始化账号：`admin` / `Admin@123456`（仅本地开发，生产必改）；DB root 统一 `123456`。

从零初始化统一导入上述完整 SQL。需要手工导入时，在新建库环境中使用 MySQL 客户端执行同一文件。

生成数据模型前，先将完整 SQL 导入开发库，再执行 `data/scripts/gen.ps1 model <模块> <表>`；goctl model 仍从已导入的开发库反射表结构。

## 3. 故障排查（按命中率排序）

### ① api 启动即退出 / 反复重启
api 启动时要经 admin-rpc 加载 Casbin 策略，rpc 未就绪会 panic——compose 已配 `restart: on-failure` 自愈。
持续失败看日志：`docker compose logs api admin-rpc --tail 50`。

API 容器重建后，若 7001 直连正常而 Nginx 返回 502，可能是 Nginx 仍保留旧上游地址。先执行 `docker compose exec -T nginx nginx -t`，确认成功后执行 `docker compose exec -T nginx nginx -s reload`，再以 `Host: localhost` 复测代理的 `/ping` 和 `/ready`；IP 直访默认站点返回 404 是域名隔离规则。

### ② 请求诡异失败（404/连接拒）
**先查端口被谁占了**——旧项目 docker 的 `gozero-api` 常占用 7001：
```powershell
netstat -ano | Select-String ':7001.*LISTENING'
docker stop gozero-api   # 暂停旧项目容器；联调完记得 docker start 恢复
````

### ③ 登录报"账号未激活"/数据改了不生效
goctl model 带 Redis 缓存，直接修改开发库后只失效本次记录对应的缓存键。先查看对应 `*_model_gen.go` 的 ID/唯一索引键前缀，再按确切 ID 和旧、新唯一值删除；授权变化通过现有策略通知触发刷新。不要清空共享 Redis，参见根规范 §7.6：
```powershell
docker exec gozero-redis8 redis-cli DEL '<本次记录的精确缓存键>'
````

### ④ docker exec 里中文乱码 / 输出被吞
- 管道传 SQL 中文会乱码：改用 `docker cp` 文件进容器再执行
- mysql 客户端密码警告吞掉输出：用 `-e MYSQL_PWD=...` 代替 `-p`

### ⑤ PowerShell 脚本执行
- ps1 文件必须 **UTF-8 with BOM**（无 BOM 中文注释会解析崩溃）
- 续行符是反引号 ` 不是反斜杠 \
- 执行策略拦截：`powershell -NoProfile -ExecutionPolicy Bypass -File <脚本>`

### ⑥ AI 密钥
ai-rpc 的 `AI.Enabled=false` 时无需密钥；启用时在 YAML（`docker/etc/ai-rpc.yaml` 或宿主 `application/ai/rpc/etc/ai.yaml`）设置 `Enabled/Provider/Model/BaseURL` 等非敏感配置，密钥只从 `API_KEY_<PROVIDER>` 环境变量读取，供应商名称转为大写，不回退读取 YAML 密钥。平台 Compose 已传递 `API_KEY_DEEPSEEK` 和 `API_KEY_QWEN`；其他供应商需显式配置对应环境变量，宿主运行则直接设置进程环境变量。真实密钥不要写入版本库。

## 4. 联调后清理（任务完成定义的一部分）

1. 停掉本次 `go run` 的进程；compose 栈按任务要求保留或 `docker compose stop`
2. 删除 `.tmp_*` 目录、调试日志
3. `docker start gozero-api` 恢复旧项目容器（如曾暂停）
4. `git status` 确认工作区干净
