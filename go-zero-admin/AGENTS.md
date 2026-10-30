# AGENTS.md — AI 代理开发规范（本仓库最高约束）

> 本文件是所有人员与 AI 代理（Codex / DeepSeek Harness / Claude Code / 其他）在本仓库开发时的**唯一权威约束**。
> 架构与决策细节见 `../ai_document/refactor-plan.md`；两者冲突时以本文件为准；本文件未覆盖的情况，**先问人再动手**。

## 1. 项目速览

- 技术栈：Go 1.27.3 / go-zero v1.10.3 / goctl v1.10.2 / MySQL 9.7 / Redis 7 / Casbin
- 关键依赖：golang-jwt/jwt/v5 v5.3.1、golang.org/x/crypto v0.58.0（go.mod 为版本唯一事实源）
- 架构：单 API（`application/admin/api`）+ 5 RPC（`admin` / `app` / `file` / `ai` / `tenant`，各在 `application/<模块>/rpc`）
- 数据层：**goctl model `--cache true`**（go-zero `sqlx`/`sqlc` 缓存包）——**禁止 GORM、禁止 sqlx 之外的数据库库**
- 鉴权：Casbin domains 模型（`dom` = 部门，非租户）；接口级 Casbin + 行级 SQL 数据范围
- 多租户：每租户独立整套栈 + 独立库，子域名经 Nginx 路由；租户管理走 tenant-rpc

## 2. 硬性规则（违反一律返工）

1. **不改官方模板**：不存在 `data/goctl/`；所有 goctl 命令不带 `-home`。
2. **生成物不手改**（含命名约定）：
   - `gen.ps1 rpc` 已带 `--name-from-filename`：goctl 按 **proto 文件名**命名生成物（`ai.proto` → `ai.go` + `etc/ai.yaml`），而非 package 名；main 内默认配置路径 `etc/<模块>.yaml`
   - etc 只保留 `<模块>.yaml`；goctl 按 package 名生成的 `<pkg>.yaml` 骨架是冗余，删除
   - `*_model_gen.go`、`pb/`、`client/`、`handler/routes.go`、`types/types.go`、`server/`——改 `desc/` 源文件后重新生成。
3. **生成命令唯一入口** `data/scripts/gen.ps1`：
   - 新/改 API 接口 → 改 `application/admin/api/desc/**.api` → `gen.ps1 api` → 在 `internal/logic` 实现
   - 新/改 RPC 方法 → 改 `application/<模块>/rpc/desc/<模块>.proto` → `gen.ps1 rpc <模块>` → 在 `internal/logic` 实现
   - 新/改表 → 新建 goose migration（**不改已发布迁移**）→ `gen.ps1 db up` → `gen.ps1 model <模块> <表>`
4. **禁止新增目录**：顶层目录白名单 = `application/ data/ docker/ pkg/ test/`；`application/` 下模块白名单 = `admin/ app/ file/ ai/ tenant/`。确需新增须先经人工确认并同步 refactor-plan.md。
5. **禁止自主引入依赖**：`go.mod` 任何新增/升级 `require` 必须先经人工批准；禁止顺手升级 go-zero / goctl / MySQL 版本；`go mod tidy -diff` 必须为空。
6. **错误处理**：logic 一律返回 `pkg/xerr` 错误；API 出口统一为 `httpx.SetErrorHandlerCtx(xerr.Handler)`；**禁止自建响应信封、禁止在 handler/logic 直接写 HTTP 响应**。
7. **数据层纪律**：CRUD 用生成 model；复杂查询（数据范围/JOIN/统计）写在 `internal/model/<表>_model.go` 扩展文件，**禁止塞进 logic**；跨表事务在 logic 用 `sqlx.SqlConn` 事务包装。
8. **sqlx 扫描纪律**：手写 `QueryRowsCtx`/`QueryRowCtx` 必须配合**带 `db` tag 的本地 row struct** 过渡，禁止直接扫描 pb 类型（pb 无 `db` tag，运行时才报 `not matching destination to scan`，编译不拦截）。时间字段统一在 SQL 层用 `UNIX_TIMESTAMP()` 转 int64，或在 Go 层用 `.Unix()`，禁止 DATETIME 直扫 int64。
9. **命名规范**：
   - RPC client 字段统一 `XxxRpc`（如 `UserRpc`/`AuthRpc`），不用 `Svc`/`RPC` 后缀——与 pb 生成物风格（`UserId`）一致
   - 时间戳：DB DATETIME → SQL `UNIX_TIMESTAMP()` → Go int64（pb/JSON）
   - 缩写词遵循 Go 惯例小写派：`Id`/`Url`/`Rpc`/`Ip`（与 goctl 生成物一致）
10. **proto package 唯一性（防全局注册冲突）**：
    - 每个模块的 proto `package` 必须**与模块同名**（如 `package adminpb;` / `package apppb;` / `package filepb;`），禁止都用 `package pb;`
    - `option go_package = "./<pkg>"` 与 package 同名，生成物在 `rpc/<pkg>/` 目录
    - **原因**：protobuf 全局注册表按 proto package 去重，多个 `package pb` 的 message（Actor/PageRequest 等）同名会运行时 panic（编译不报错）
    - **检测**：新增/修改 proto 后，若 admin-api 需引用多个模块的 pb，必须确认各 package 不同

11. **e2e 验证纪律**：每完成一个 `.api`/`.proto` 模块的 REST/RPC 对接，立即跑真实调用验证（登录→该模块接口），禁止积攒多个模块后一次性测——运行时错误（扫描失败/未注册）编译期不可见，越早暴露定位成本越低。

12. **pkg 定位**：放 `pkg/` 的包须满足——(a) 当前或近期有 ≥2 个服务使用，或 (b) 是明确的**公共基础设施**（鉴权/限流/存储/LLM 协议客户端等，任何模块都可能用到）。模块私有的业务辅助逻辑放 `internal/`。pkg 包**禁止 import 任何 `internal/` 包**（Go 规则，也是解耦保障）——依赖显式经函数参数传入。
13. **命名与风格**：`-style go_zero`（小写下划线文件名）；注释、文档、提交信息用中文；标识符用英文。
14. **安全**：
    - **所有配置直接写各服务 yaml**（开源项目，用户直接填值即可）：AI 模型密钥填 `DeepSeekKey:`/`QwenKey:`，MySQL/JWT/存储凭据同理；也支持环境变量注入（compose `environment:`）供生产覆盖
    - 本地开发默认值可直接使用；生产部署按实际值填写 yaml 或环境变量覆盖
    - **禁止在宿主生成/运行可执行文件**（公司安全策略）：不用 `go run` / `go build` 产生临时 exe；需要运行代码时用 **Docker 容器** 或 **脚本语言**（Python/PowerShell）代替
    - 本地开发服务（`go run ./application/...`）属开发调试需要，允许；但**一次性工具脚本**（如生成哈希、数据迁移）禁止用 `go run` 在宿主执行
    - 必须用 Go 跑一次性任务时：写正式工具放 `test/` 目录，提前声明用途，Docker 容器内执行，用完即删

## 3. 禁止事项速查

- ❌ 新建任何顶层目录、新模块目录、以及任何「看起来更合理」的脚手架
- ❌ `go get` 任何未批准依赖（包括号称很有用的库）
- ❌ 修改官方生成模板、给 goctl 加 `-home`
- ❌ 手改生成代码后直接重新生成（先把手写逻辑保留在 logic / model 扩展中）
- ❌ 自定义 HTTP 响应包装/信封、绕过 xerr
- ❌ 在 api 层直连数据库（必须经 RPC client）
- ❌ 修改 `docker/` 部署模板（除非任务本身就是改部署）

## 4. 提交前自检（必过）

```powershell
.\data\scripts\verify.ps1
# 若系统执行策略拦截：powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\verify.ps1
```

包含：gofmt、`go vet`、`go build ./...`、`go mod tidy -diff`、`goctl api validate`、顶层目录白名单检查。

## 5. test/ 目录

手动端到端测试脚本（PowerShell），如 `login_e2e.ps1`。这些脚本不参与构建，仅供本地开发联调。

`data/test-files/` 存放演示图片/文件，配合 file-rpc 的 RustFS 对象存储联调（上传/签名 URL 等），不入镜像。

## 6. AI 工具接入

- **mcp-zero**（官方 MCP，已注册于 `.mcp.json`）：创建服务、生成 model、校验 api/proto 时优先调用其工具，代替手写命令。
- 官方规范知识：有疑问先查 go-zero 官方文档 <https://go-zero.dev>，不要凭记忆编造 goctl 参数或配置字段。
