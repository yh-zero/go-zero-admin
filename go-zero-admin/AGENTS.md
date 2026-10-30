# AGENTS.md — 后端开发规范（go-zero 规范优先）

> 先读仓库根 `../AGENTS.md`（通用纪律），再读本文件。冲突时以本文件为准；授权与确认边界遵循根规范，不重复确认已授权事项。
> 共享 API 契约以 `../ai_document/api-contract.md` 为唯一事实源；规范背后的历史事故见 `../ai_document/pitfalls.md`。

## 1. 项目速览

- 技术栈：Go 1.27.3 / go-zero v1.10.3 / goctl v1.10.2 / MySQL 9.7 / Redis 8 / Casbin
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
7. **数据层纪律**：常规 CRUD 必须用生成 model；复杂查询写 `internal/model/<表>_model.go` 扩展文件，禁止在 logic 中直接写 SQL；跨表事务用 `sqlx.SqlConn` 包装。
   - 先读生成方法的实际签名和 SQL。只有生成 CRUD 无法表达有效行限定、字段保护或事务/并发约束时，才可在 model 扩展中封装必要的受保护写入，并记录生成方法的具体限制与选择理由。
   - 扩展写入必须使用参数化 SQL，维护事务、并发及受影响记录/关联的缓存失效语义；不能为复用生成方法而回写旧 `deleted_at`、复活软删除记录或丢掉行锁保护。
   - 该边界不是任意手写 CRUD 的许可。已有实现需按条件复核，不能仅因规则更新就把历史偏离标为已修复。
8. **sqlx 扫描纪律**：`QueryRowsCtx`/`QueryRowCtx` 必须配带 `db` tag 的本地 row struct，禁止直扫 pb；时间字段 SQL 层 `UNIX_TIMESTAMP()` 转 int64。
9. **命名规范**：RPC client 字段 `XxxRpc`；缩写词小写派（`Id`/`Url`/`Rpc`/`Ip`）。
10. **proto package 唯一性**：package 与模块同名（`adminpb`/`apppb`/`filepb`/`aipb`/`tenantpb`），禁止都用 `pb`。
11. **e2e 纪律**：按根规范逐模块真实验证；数据库查询须覆盖非空数据、时间与 NULL 字段，写操作核对持久化结果及缓存读取，鉴权变更核对允许与拒绝场景。未验证项如实记录。
12. **pkg 定位**：≥2 服务使用或为公共基础设施；pkg 禁止 import 任何 `internal/` 包。
13. **配置**：正式配置维护 `etc/<模块>.yaml`；AI Key 留空走环境变量注入。本地验证的临时连接与凭据可用环境变量或仓库外临时配置，不改正式配置迁就测试环境，结束后清理。
14. **测试文件**：后端新增测试代码和测试脚本统一放在 `test/`，不散放在业务源码或 `data/scripts/`；临时测试用完删除，保留验证说明，保留原有测试和正式数据导入器。
   - 遵守 Go 同包私有成员及 `internal` 导入边界，优先通过 HTTP/RPC 或可访问的公共包验证；不能直接搬移后留下不可编译测试，也不为集中存放而暴露业务私有实现或修改生成物。
15. **权限策略刷新**：构建完成的策略统一发布，不能修改正在鉴权的实例；首次快照与订阅建立、断线重连和刷新失败须有补偿/重试或定期对账机制，不能只等待下一条 Pub/Sub 消息。验证应覆盖漏通知、刷新失败及恢复后的撤权效果；正常热更新成功不能替代这些验证。

## 3. 禁止事项速查

- ❌ 白名单外目录、给 goctl 加 `-home`、手改生成物后直接重新生成
- ❌ 自定义响应信封/绕过 xerr、api 层直连数据库、擅自改 `docker/` 模板

## 4. 提交前自检（必过）

```powershell
.\data\scripts\verify.ps1
# 执行策略拦截时：powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\verify.ps1
```

gofmt / go vet / go build / tidy-diff / goctl api validate / 目录白名单。

- 每条原生命令执行后立即检查退出码，保留可诊断且脱敏的错误；`go list ./...` 失败必须终止，不能因 stdout 为空误判“骨架阶段”而跳过 vet/build/tidy。只有命令成功且确实无包时才可明确记录跳过。
- Windows 行尾兼容只允许在只读比较或临时副本中归一 CRLF；真实格式、依赖、校验和及顺序差异仍须失败，不改写源文件或锁文件来掩盖检查失败。
- 自检通过只证明相应检查通过。修改本文件不会自动修复现有自检脚本或策略刷新实现，遗留问题继续在 `../ai_document/` 中跟踪。
