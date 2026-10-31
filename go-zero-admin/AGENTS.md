# AGENTS.md — 后端开发规范（go-zero 规范优先）

> 先读仓库根 `../AGENTS.md`（通用纪律），再读本文件。冲突时以本文件为准；已授权范围内的常规实现、生成和验证自主完成，超出授权或涉及未明确的数据、权限边界时先确认，已批准事项不重复确认。
> 共享 API 契约以 `../ai_document/api-contract.md` 为唯一事实源；规范背后的历史事故见 `../ai_document/pitfalls.md`。

## 1. 项目速览

- 技术栈：Go 1.27.3（`go 1.27.0` + `toolchain go1.27.3`）/ go-zero v1.10.3 / goctl v1.10.2 / MySQL 9.7 / Redis 8（compose 已锁 `redis:8`）/ Casbin
- 架构：单 API（`application/admin/api`）+ 5 RPC（admin/app/file/ai/tenant）
- 数据层：**goctl model `--cache true`**——禁止 GORM、禁止 sqlx 之外的数据库库
- 鉴权：Casbin domains 模型（`dom` = 部门）+ Redis 策略热更新；接口级 Casbin + 行级 SQL 数据范围
- 多租户：每租户独立栈 + 独立库，子域名经 Nginx 路由

## 2. 硬性规则（违反一律返工）

1. **不改官方模板**：所有 goctl 命令不带 `-home`。
2. **文件分类与维护边界**：
   - 严格生成物：`*_model_gen.go`、protobuf 包内生成文件、RPC client/server、`routes.go`、`types.go`；修改对应 `.api` / `.proto` / 初始化 SQL 来源后重新生成，禁止手改
   - 生成骨架：logic、config、svc、main；允许按职责补充业务、配置及依赖组装。handler 由 goctl 创建，保留标准协议处理，接口签名变化通过 `gen.ps1 api -RefreshHandlers` 刷新指定入口，不手造接口骨架
   - 手写扩展：中间件、model 扩展及必要辅助函数；允许在职责范围维护，新增文件按规则 4 确认。中间件按职责拆合，不规定固定文件数量；纯整理保持注册顺序、作用范围及鉴权/审计行为不变
   - `gen.ps1 rpc` 已带 `--name-from-filename`：goctl 按 proto 文件名命名（`ai.proto` → `ai.go` + `etc/ai.yaml`）
   - etc 只保留 `<模块>.yaml`；`<pkg>.yaml` 骨架是冗余，删除
3. **生成命令唯一入口** `data/scripts/gen.ps1`：
   - 新/改 API 接口 → 改 `desc/**.api` → `gen.ps1 api` → `internal/logic` 实现
   - 新/改 RPC 方法 → 改 `desc/<模块>.proto` → `gen.ps1 rpc <模块>` → `internal/logic` 实现
   - 新/改表 → 更新 `data/db/init/mysql-all.sql`（完整结构与初始化数据的唯一入口）→ 导入新开发库 → `gen.ps1 model <模块> <表>`；`data/db/` 只保留 `init/`，采用完整初始化，不维护增量升级脚本
   - 生成后核对定义与路由、RPC service 注册、handler/logic 分组及签名一致；改名或移动检查旧入口和旧引用，保留已有业务逻辑。goctl 会跳过部分已有文件，命令成功不能代替这些检查；新增 RPC service 按脚本提示补齐 main 注册，不绕过生成入口
4. **目录与新增文件授权**：本目录顶层业务目录白名单 = `application/ data/ docker/ pkg/ test/`；模块白名单 = `admin/ app/ file/ ai/ tenant/`。新增白名单外目录、模块或架构层先确认；已授权接口经 goctl 生成的正常分组目录按既有授权执行。新增手写后端文件先列路径、职责和必要性，经确认再创建；已批准清单不重复确认。
5. **禁止自主引入依赖**：`go.mod` 任何新增/升级须先批准；`go mod tidy -diff` 必须为空。
6. **错误与响应**：logic 的业务错误统一使用 `pkg/xerr`，不操作 HTTP 响应；handler 保留生成模板的标准响应调用，API 错误出口统一 `httpx.SetErrorHandlerCtx(xerr.Handler)`，成功响应遵守共享契约。禁止自行另建响应信封或绕过统一出口。
7. **分层与数据访问**（本项目约定）：handler/server 负责协议处理并调用 logic；API logic 做参数转换并调用 RPC，禁止直连业务数据库；RPC logic 负责业务规则和流程编排；svc 只负责依赖创建与组装，不放用户、头像等业务编排。CRUD 用生成 model，复杂 SQL、跨表事务及对应缓存一致性放 `internal/model/<表>_model.go` 扩展文件，事务用 `sqlx.SqlConn` 包装。
8. **sqlx 扫描纪律**：`QueryRowsCtx`/`QueryRowCtx` 必须配带 `db` tag 的本地 row struct，禁止直扫 pb；时间字段 SQL 层 `UNIX_TIMESTAMP()` 转 int64。
9. **命名规范**：RPC client 字段 `XxxRpc`；缩写词小写派（`Id`/`Url`/`Rpc`/`Ip`）。
10. **proto package 唯一性**：package 与模块同名（`adminpb`/`apppb`/`filepb`/`aipb`/`tenantpb`），禁止都用 `pb`。
11. **验收纪律**：功能对接逐模块真实验证，按根规范 §7 的改动触发条件覆盖失败和边界场景；纯整理、纯文档按各自验证要求执行。
12. **pkg 定位**：≥2 服务使用或为公共基础设施；pkg 禁止 import 任何 `internal/` 包。
13. **配置**：直接写 `etc/<模块>.yaml`；AI Key 留空走环境变量注入，命名约定 `API_KEY_<PROVIDER>`（现有 `API_KEY_DEEPSEEK` / `API_KEY_QWEN`）。
14. **事故索引**：动手涉及 Casbin domains、多租户栈、goctl 生成、sqlx 数据层前，先查 `../ai_document/pitfalls.md` 对应条目。

## 3. 禁止事项速查

- ❌ 白名单外目录、给 goctl 加 `-home`、手改生成物后直接重新生成
- ❌ 自定义响应信封/绕过 xerr、api 层直连数据库、擅自改 `docker/` 模板

## 4. 代码提交前自检（必过）

```powershell
.\data\scripts\verify.ps1
# 执行策略拦截时：powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\verify.ps1
```

gofmt / go vet / go build / tidy-diff / goctl api validate / 目录白名单。该脚本是静态与构建门槛，不含完整功能验收，也不证明所有生成入口和注册关系完整；规则 3 的生成结果核对仍须执行。纯文档或规则修改按根规范 §7 检查，不要求构建。

后续自动化优先在现有脚本补齐注册完整性、遗留入口和原生命令退出码检查；这是待改进项，未实现前不得宣称脚本已经覆盖，规则修改本身不授权改脚本。
