# 本地开发环境拓扑与联调 SOP

> 联调/验证前先读本文。环境异常时按 §3 排查，别猜。

## 1. 标准拓扑

| 组件 | 位置 | 说明 |
|---|---|---|
| MySQL 9.7 | docker 容器 `gozero-new-mysql`，宿主 `127.0.0.1:3307` | 业务库 `gozero_admin` + 控制库 `gozero_platform`；迁移用 `goose -dir data/db/migrations/<business|platform>` |
| Redis 8 | docker 容器 `gozero-redis`，`127.0.0.1:6379` | 缓存 + Casbin 策略通知频道 |
| RustFS | docker 容器 `gozero-rustfs`，`127.0.0.1:9000`（控制台 :9001） | S3 兼容对象存储 |
| admin-rpc | `go run ./application/admin/rpc` → :6001 | |
| app-rpc | → :6002 | |
| file-rpc | → :6003 | |
| ai-rpc | → :6004 | |
| tenant-rpc | → :6005 | 仅平台栈 |
| admin-api | → :7001 | REST 网关 |
| 前端 dev | `go-zero-admin-react` → http://localhost:5999 | `/v1` 已代理到 7001 |

种子账号：`admin / Admin@123456`（登录需验证码）。

## 2. 标准启动顺序

```powershell
# 1. 确认依赖容器在跑
docker ps   # gozero-new-mysql / gozero-redis / gozero-rustfs

# 2. 后端（每个一个终端/后台任务）
go run ./application/admin/rpc -f application/admin/rpc/etc/admin.yaml
go run ./application/admin/api -f application/admin/api/etc/admin.yaml

# 3. 前端
cd go-zero-admin-react; pnpm dev
```

## 3. 故障排查（按命中率排序）

### ① 服务"起来了"但请求诡异失败（404/连接拒）
**先查端口被谁占了**——旧项目 docker 的 `gozero-api` 常占用 7001：
```powershell
netstat -ano | Select-String ':7001.*LISTENING'
docker stop gozero-api   # 暂停旧项目容器；联调完记得 docker start 恢复
```

### ② 登录报"账号未激活"/数据改了不生效
goctl model 带 Redis 缓存，**直接改库后必须清缓存**：
```powershell
docker exec gozero-redis redis-cli FLUSHALL
```

### ③ docker exec 里中文乱码 / 输出被吞
- 管道传 SQL 中文会乱码：改用 `docker cp` 文件进容器再执行
- mysql 客户端密码警告吞掉输出：命令尾部加 `2>$null`

### ④ PowerShell 脚本执行
- ps1 文件必须 **UTF-8 with BOM**（无 BOM 中文注释会解析崩溃）
- 续行符是反引号 ` 不是反斜杠 \\
- 执行策略拦截：`powershell -NoProfile -ExecutionPolicy Bypass -File <脚本>`

### ⑤ goose 找不到
GOPATH 随 toolchain 切换会变。用绝对路径或重新 `go install github.com/pressly/goose/v3/cmd/goose@latest`。

## 4. 联调后清理（任务完成定义的一部分）

1. 停掉本次 `go run` 的进程
2. 删除 `.tmp_*` 目录、调试日志
3. `docker start gozero-api` 恢复旧项目容器
4. `git status` 确认工作区干净
