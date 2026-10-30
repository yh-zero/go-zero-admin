# AGENTS.md — 后端开发规范（go-zero 规范优先）

> 先读仓库根 `../AGENTS.md`（通用纪律），再读本文件。冲突时以本文件为准；未覆盖的情况**先问人再动手**。
> 前后端共享 API 契约以 `../ai_document/api-contract.md` 为唯一事实源（响应格式/字段命名/错误码/路由前缀）。

## 1. 项目速览

- 技术栈：Go 1.27.3 / go-zero v1.10.3 / goctl v1.10.2 / MySQL 9.7 / Redis 8 / Casbin
- 架构：单 API（`application/admin/api`）+ 5 RPC（`admin` / `app` / `file` / `ai` / `tenant`）
- 数据层：**goctl model `--cache true`**（go-zero `sqlx`/`sqlc` 缓存包）——**禁止 GORM、禁止 sqlx 之外的数据库库**
- 鉴权：Casbin domains 模型（`dom` = 部门）；接口级 Casbin + 行级 SQL 数据范围
- 多租户：每租户独立整套栈 + 独立库，子域名经 Nginx 路由；租户管理走 tenant-rpc

## 2. 硬性规则（违反一律返工）

1. **不改官方模板**：不存在 `data/goctl/`；所有 goctl 命令不带 `-home`。
2. **生成物不手改**（含命名约定）：
   - `gen.ps1 rpc` 已带 `--name-from-filename`：goctl 按 **proto 文件名**命名生成物（`ai.proto` → `ai.go` + `etc/ai.yaml`）；main 内默认配置路径 `etc/<模块>.yaml`
   - etc 只保留 `<模块>.yaml`；goctl 按 package 名生成的 `<pkg>.yaml` 骨架是冗余，删除
   - `*_model_gen.go`、`<pkg>/`、`client/`、`handler/routes.go`、`types/types.go`、`server/`——改 `desc/` 源文件后重新生成。
3. **生成命令唯一入口** `data/scripts/gen.ps1`：
   - 新/改 API 接口 → 改 `application/admin/api/desc/**.api` → `gen.ps1 api` → 在 `internal/logic` 实现
   - 新/改 RPC 方法 → 改 `application/<模块>/rpc/desc/<模块>.proto` → `gen.ps1 rpc <模块>` → 在 `internal/logic` 实现
   - 新/改表 → 新建 goose migration（**不改已发布迁移**）→ `gen.ps1 db up` → `gen.ps1 model <模块> <表>`
4. **禁止新增目录**：本目录顶层白名单 = `application/ data/ docker/ pkg/ test/`；`application/` 下模块白名单 = `admin/ app/ file/ ai/ tenant/`。确需新增须先经人工确认。
5. **禁止自主引入依赖**：`go.mod` 任何新增/升级 `require` 必须先经人工批准；`go mod tidy -diff` 必须为空。
6. **错误处理**：logic 一律返回 `pkg/xerr` 错误；API 出口统一为 `httpx.SetErrorHandlerCtx(xerr.Handler)`；**禁止自建响应信封、禁止在 handler/logic 直接写 HTTP 响应**（响应格式见 api-contract.md §1）。
7. **数据层纪律**：CRUD 用生成 model；复杂查询（数据范围/JOIN/统计）写在 `internal/model/<表>_model.go` 扩展文件，**禁止塞进 logic**；跨表事务在 logic 用 `sqlx.SqlConn` 事务包装。
8. **sqlx 扫描纪律**：手写 `QueryRowsCtx`/`QueryRowCtx` 必须配合**带 `db` tag 的本地 row struct** 过渡，禁止直接扫描 pb 类型（编译不拦截，运行时才报 `not matching destination to scan`）。时间字段统一 SQL 层 `UNIX_TIMESTAMP()` 转 int64，禁止 DATETIME 直扫 int64。
9. **命名规范**：
   - RPC client 字段统一 `XxxRpc`（如 `UserRpc`/`AuthRpc`），不用 `Svc`/`RPC` 后缀
   - 时间戳：DB DATETIME → SQL `UNIX_TIMESTAMP()` → Go int64
   - 缩写词遵循 Go 惯例小写派：`Id`/`Url`/`Rpc`/`Ip`（与 goctl 生成物一致）
10. **proto package 唯一性**：每个模块 proto `package` 与模块同名（`adminpb`/`apppb`/`filepb`/`aipb`/`tenantpb`），禁止都用 `pb`（protobuf 全局注册冲突，编译不报错、运行时 panic）。
11. **e2e 验证纪律**：每完成一个 `.api`/proto 模块对接，立即真实调用验证（登录→该模块接口），禁止积攒多模块后一次性测。
12. **pkg 定位**：放 `pkg/` 须满足 ≥2 服务使用，或为公共基础设施（xerr/jwtx/objectstorage/llm/agenttools/apptoken/casbinauth）。pkg 包**禁止 import 任何 `internal/` 包**——依赖显式经函数参数传入。
13. **配置**：所有配置直接写各服务 `etc/<模块>.yaml`（开源项目用户直接填值）；支持环境变量覆盖；敏感值（AI Key）留空走注入。

## 3. 禁止事项速查

- ❌ 新建白名单外目录、新模块目录、以及任何「看起来更合理」的脚手架
- ❌ 给 goctl 加 `-home`、修改官方模板
- ❌ 手改生成代码后直接重新生成（先把手写逻辑保留在 logic / model 扩展中）
- ❌ 自定义 HTTP 响应包装/信封、绕过 xerr
- ❌ 在 api 层直连数据库（必须经 RPC client）
- ❌ 修改 `docker/` 部署模板（除非任务本身就是改部署）

## 4. 提交前自检（必过）

```powershell
.\data\scripts\verify.ps1
# 若执行策略拦截：powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\verify.ps1
```

包含：gofmt、`go vet`、`go build ./...`、`go mod tidy -diff`、`goctl api validate`、顶层目录白名单检查。
