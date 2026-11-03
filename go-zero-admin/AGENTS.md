# AGENTS.md — 后端开发规范（go-zero 规范优先）

> 先读仓库根 `../AGENTS.md`（通用纪律），再读本文件。冲突时以本文件为准；已授权范围内的常规实现、生成和验证自主完成，超出授权或涉及未明确的数据、权限边界时先确认，已批准事项不重复确认。
> 共享 API 契约以 `../ai_document/api-contract.md` 为唯一事实源；规范背后的历史事故见 `../ai_document/dev-guide.md` §5。

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
4. **目录与新增文件授权**：本目录顶层业务目录白名单 = `application/ data/ docker/ pkg/ test/`；模块白名单 = `admin/ app/ file/ ai/ tenant/`。新增白名单外目录、模块或架构层先确认；已授权接口经 goctl 生成的正常分组目录按既有授权执行。新增手写后端文件先列路径、职责和必要性，经确认再创建；已批准清单不重复确认。`docker/` 编排与模板不擅自改动，需调整时先确认。
5. **禁止自主引入依赖**（根 §6）：`go.mod` 任何新增/升级须先批准；`go mod tidy -diff` 必须为空。
6. **错误与响应**：logic 的业务错误统一使用 `pkg/xerr`，不操作 HTTP 响应；handler 保留生成模板的标准响应调用，API 错误出口统一 `httpx.SetErrorHandlerCtx(xerr.Handler)`，成功响应遵守共享契约。禁止自行另建响应信封或绕过统一出口。
7. **分层与数据访问**：handler/server → logic → model 的分层、SQL/事务/缓存纪律与历史整改验收见 §3，均为硬性约束。
8. **sqlx 扫描纪律**：自定义多列查询使用带 `db` tag 的本地 row struct，禁止直扫 pb；COUNT、单列 ID 等允许使用匹配的标量或标量切片。需要将时间转为 int64 时，在 SQL 层用 `UNIX_TIMESTAMP()` 并为表达式显式设置列别名；不强改生成 model 的字段类型。
9. **命名规范**：RPC client 字段 `XxxRpc`；缩写词小写派（`Id`/`Url`/`Rpc`/`Ip`）。
10. **proto package 唯一性**：package 与模块同名（`adminpb`/`apppb`/`filepb`/`aipb`/`tenantpb`），禁止都用 `pb`。
11. **验收纪律**：功能对接逐模块真实验证，按根规范 §8 的改动触发条件覆盖失败和边界场景；纯整理、纯文档按各自验证要求执行。
12. **pkg 定位**：≥2 服务使用或为公共基础设施；pkg 禁止 import 任何 `internal/` 包。
13. **配置**：直接写 `etc/<模块>.yaml`；加载统一带 `conf.UseEnv()`，字段值可直填，也可用 `${ENV_VAR}` 占位符从环境变量注入（compose 经 `env_file: .env.local` 提供，文件缺失不报错）。环境变量按“模块 + YAML 配置层级”转全大写下划线命名，如 `APP_SMTP_PASSWORD` 对应 App 的 `SMTP.Password`；AI 密钥为 `AI_DEEPSEEK_KEY` / `AI_QWEN_KEY` 对应 `AI.DeepSeekKey` / `AI.QwenKey`，svc 按 Provider 选用。yaml 直填的真实密钥由 verify.ps1 第 7 步扫描默认拦截；私有仓库部署可显式 `-AllowYamlSecrets` 放行。外部基础镜像要求的变量沿用其官方名称。
14. **事故索引**：动手涉及 Casbin domains、多租户栈、goctl 生成、sqlx 数据层前，先查 `../ai_document/dev-guide.md` §5 对应条目。

## 3. 数据访问分层（本仓库约定，不表述为 go-zero 的强制限制）

- **入口与依赖**：handler/server 负责协议处理并调用 logic；API logic 转换参数并调用 RPC，禁止直连业务数据库；svc 只创建与组装依赖。
- **RPC logic 处理业务**：校验身份、权限和业务条件，编排流程并转换返回结果；通过 model 方法访问数据库。禁止在 logic 拼接/执行 SQL，或直接调用数据库连接的 Query、Exec、Transact 等方法；数据范围等查询条件以明确的 model 参数传入。
- **model 处理数据**：生成 CRUD 满足更新语义、软删除和缓存要求时优先复用；不满足时，在当前服务现有 `internal/model/<表>_model.go` 增加接口方法及实现。自定义查询、分页、关联写入、软删除均放扩展文件，使用参数化 SQL；logic 负责 model 结构与 pb 的转换。禁止修改 `*_model_gen.go`，不为整理 SQL 另建 repository/DAO 等架构层；新增手写文件仍按规则 4 确认。
- **事务保证整体成功或失败**：需要原子性的多次写入封装为 model 方法，使用现有 `TransactCtx` 和同一个 `sqlx.Session`，不得用多个独立 CRUD 调用冒充事务。依赖当前数据库状态的关键校验，应通过原子条件更新，或同一事务内的锁与写入保护，不能仅依赖 logic 先查再写；单条写入不因本规则一律要求开启事务。
- **缓存与提交一起核对**：复用生成方法前确认其缓存行为；自定义写入确认数据库写入或事务提交成功后，按实际生成键规则失效 ID、旧/新唯一索引等受影响缓存。不得忽略缓存处理错误，或用清空 Redis 代替精确失效；数据库已提交而缓存处理失败时，必须区分这一结果，不能宣称事务已回滚。策略通知等后续动作仅在对应写入成功后触发。
- **历史整改与验收**：新写或修改的数据访问遵守本节。已授权的数据访问重构须连同 SQL、事务、缓存完整核对；范围外遗留记录在 `../ai_document/dev-guide.md` §5 踩坑清单，不因更新规范就标为已修复。迁移后按根规范 §8 验证查询结果、更新/清空、软删除、失败回滚及缓存等受影响行为，编译通过不代替行为验收。

## 4. AI 模块专项（application/ai）

1. **LLM 调用纪律**：
   - 一律经 `pkg/llm` 封装（eino ChatModel），禁止 logic 内裸写 HTTP 调模型接口
   - function calling 轮次上限 5（`send_message_logic`），超限必须兜底返回，禁止死循环
   - 每次对话落库 token 用量（`ai_message.tokens`），阶段 3 计费依赖此数据
2. **工具开发（pkg/agenttools）**：
   - 新工具 = 实现 `Tool` 接口 + `init()` 注册，禁止散落的工具实现
   - `Description()`/`Params()` 是给 LLM 的决策依据，必须准确完整（写清什么时候用、参数含义）
   - 工具参数视为**不可信输入**：Execute 内必须校验，禁止直接拼接 SQL/命令/路径
3. **工作流 DSL**：
   - 定义只走 `{"steps":[{name,tool,input,approval}]}` 结构，变更时同步 `run_workflow_logic` 的解析与校验
   - `approval=true` 仅中间步骤生效（最后一步不中断，无后续可续）；改动此语义必须同步前端审批交互
4. **AI 安全**：
   - 模型密钥支持 yaml 直填或 `${AI_DEEPSEEK_KEY}`/`${AI_QWEN_KEY}` 占位符（见 §2.13）；公开仓库提交前必须过 verify.ps1 密钥扫描，安全基线按根 §6
   - LLM 输出（含工具调用参数）一律视为不可信，落库前校验长度与格式；渲染走前端转义，禁止服务端拼接 HTML
   - 会话/工作流/任务严格按 `actor.UserId` 隔离，禁止跨用户读取（已有校验，新增查询时保持）
5. **降级约定**：`AI.Enabled=false` 或无 Key 时，所有 LLM 相关接口返回统一配置提示（已有模式，新增接口照做）。

## 5. 代码提交前自检（必过）

```powershell
.\data\scripts\verify.ps1
# 执行策略拦截时：powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\verify.ps1
```

gofmt / go vet / go build / tidy-diff / goctl api validate / 目录白名单 / yaml 密钥扫描。自检性质与可信要求按根 §8.2；该脚本不证明生成入口和注册关系完整，规则 3 的生成结果核对仍须执行。纯文档或规则修改按根规范 §8 检查，不要求构建。

后续自动化优先在现有脚本补齐注册完整性、遗留入口和原生命令退出码检查；这是待改进项，未实现前不得宣称脚本已经覆盖，规则修改本身不授权改脚本。
