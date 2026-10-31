# AGENTS.md — 后端开发规范（go-zero 规范优先）

> 先读仓库根 `../AGENTS.md`（通用纪律），再读本文件。冲突时以本文件为准；未覆盖的情况**先问人再动手**。
> 共享 API 契约以 `../ai_document/api-contract.md` 为唯一事实源；规范背后的历史事故见 `../ai_document/pitfalls.md`。

## 1. 项目速览

- 技术栈：Go 1.27.3（`go 1.27.0` + `toolchain go1.27.3`）/ go-zero v1.10.3 / goctl v1.10.2 / MySQL 9.7 / Redis 8（compose 已锁 `redis:8`）/ Casbin
- 架构：单 API（`application/admin/api`）+ 5 RPC（admin/app/file/ai/tenant）
- 数据层：**goctl model `--cache true`**——禁止 GORM、禁止 sqlx 之外的数据库库
- 鉴权：Casbin domains 模型（`dom` = 部门）+ Redis 策略热更新；接口级 Casbin + 行级 SQL 数据范围
- 多租户：每租户独立栈 + 独立库，子域名经 Nginx 路由

## 2. 硬性规则（违反一律返工）

1. **不改官方模板**：所有 goctl 命令不带 `-home`。
2. **生成物不手改**：
   - `gen.ps1 rpc` 已带 `--name-from-filename`：goctl 按 proto 文件名命名（`ai.proto` → `ai.go` + `etc/ai.yaml`）
   - etc 只保留 `<模块>.yaml`；`<pkg>.yaml` 骨架是冗余，删除
   - `*_model_gen.go`、`<pkg>/`、`client/`、`routes.go`、`types.go`、`server/`——改 `desc/` 源文件后重新生成
3. **生成命令唯一入口** `data/scripts/gen.ps1`：
   - 新/改 API 接口 → 改 `desc/**.api` → `gen.ps1 api` → `internal/logic` 实现
   - 新/改 RPC 方法 → 改 `desc/<模块>.proto` → `gen.ps1 rpc <模块>` → `internal/logic` 实现
   - 新/改表 → 新建 goose migration（**不改已发布迁移**）→ `gen.ps1 db up` → `gen.ps1 model <模块> <表>`
4. **禁止新增目录**：本目录顶层白名单 = `application/ data/ docker/ pkg/ test/`；模块白名单 = `admin/ app/ file/ ai/ tenant/`。新增须先经人工确认。
5. **禁止自主引入依赖**：`go.mod` 任何新增/升级须先批准；`go mod tidy -diff` 必须为空。
6. **错误处理**：logic 一律返回 `pkg/xerr`；API 出口统一 `httpx.SetErrorHandlerCtx(xerr.Handler)`；禁止自建信封、禁止在 handler/logic 直接写 HTTP 响应。
7. **数据层纪律**：CRUD 用生成 model；复杂查询写 `internal/model/<表>_model.go` 扩展文件；跨表事务用 `sqlx.SqlConn` 包装。
8. **sqlx 扫描纪律**：`QueryRowsCtx`/`QueryRowCtx` 必须配带 `db` tag 的本地 row struct，禁止直扫 pb；时间字段 SQL 层 `UNIX_TIMESTAMP()` 转 int64。
9. **命名规范**：RPC client 字段 `XxxRpc`；缩写词小写派（`Id`/`Url`/`Rpc`/`Ip`）。
10. **proto package 唯一性**：package 与模块同名（`adminpb`/`apppb`/`filepb`/`aipb`/`tenantpb`），禁止都用 `pb`。
11. **e2e 纪律**：每完成一个模块对接立即真实调用验证，禁止积攒；完成定义同根规范 §7。
12. **pkg 定位**：≥2 服务使用或为公共基础设施；pkg 禁止 import 任何 `internal/` 包。
13. **配置**：直接写 `etc/<模块>.yaml`；AI Key 留空走环境变量注入，命名约定 `API_KEY_<PROVIDER>`（现有 `API_KEY_DEEPSEEK` / `API_KEY_QWEN`）。
14. **事故索引**：动手涉及 Casbin domains、多租户栈、goctl 生成、sqlx 数据层前，先查 `../ai_document/pitfalls.md` 对应条目。

## 3. 禁止事项速查

- ❌ 白名单外目录、给 goctl 加 `-home`、手改生成物后直接重新生成
- ❌ 自定义响应信封/绕过 xerr、api 层直连数据库、擅自改 `docker/` 模板

## 4. 提交前自检（必过）

```powershell
.\data\scripts\verify.ps1
# 执行策略拦截时：powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\verify.ps1
```

gofmt / go vet / go build / tidy-diff / goctl api validate / 目录白名单。
