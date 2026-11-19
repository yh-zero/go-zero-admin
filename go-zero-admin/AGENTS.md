# 后端开发规范

开发前先查[根规范](../AGENTS.md)与 [go-zero 官方指令](../.ai-context/go-zero/00-instructions.md)。框架用法按需查[官方知识库](../.ai-context/zero-skills/SKILL.md)，HTTP 行为以[共享契约](../ai_document/api-contract.md)为准。

## 1. 项目边界

一个 Go 模块承载 admin API 及 admin、app、file、ai、tenant 五个 RPC。业务实现放在 `application/`，共用基础设施或多服务复用的代码放在 `pkg/`，其中不导入服务 `internal` 包。现有业务目录还有 `data/`、`docker/`、`test/`。新增模块、架构层、顶层目录或依赖，以及超出任务的部署、数据或权限变更，按根规范确认授权。技术栈与部署边界见[架构说明](../ai_document/architecture.md)。

## 2. 生成入口与文件边界

统一使用[生成脚本](data/scripts/gen.ps1)，沿用官方默认模板、脚本钉住的版本和参数。四类入口从本目录运行。数据库以 `data/db/init/` 下的完整 SQL 为来源；既有库按实际差异升级，不为生成 model 导入重建脚本。

| 来源与命令 | 生成后核对 |
|---|---|
| 改 `application/admin/api/desc/` 下的定义，运行 `./data/scripts/gen.ps1 api` | 对照路由与类型；签名变化用 `-RefreshHandlers` 显式刷新相应入口，核对逻辑分组与旧引用。 |
| 改模块 `rpc/desc/<模块>.proto`，运行 `./data/scripts/gen.ps1 rpc admin`（替换模块名） | 使用独立 proto package，如 `adminpb`。脚本按 proto 文件名命名并检查新增 service；已有 main 可能被 goctl 跳过，须核对实际注册。模块配置保留 `etc/<模块>.yaml`。 |
| 更新[完整初始化 SQL](data/db/init/mysql-all.sql)，导入新开发库，运行 `./data/scripts/gen.ps1 model admin sys_user`（替换模块与表） | goctl model 使用 `--cache true`。已有库的维护遵循[数据初始化](../ai_document/dev-guide.md#数据初始化)，生成操作不重置现有业务库。 |
| 运行 `./data/scripts/gen.ps1 swagger` | 从同一组 `.api` 更新 `data/api/generated/`。 |

严格生成物包括 `*_model_gen.go`、protobuf 文件、RPC `client/<service>/*.go` 和 `internal/server/<service>/*_server.go`、`routes.go`、`types.go` 及 Swagger 描述。改来源后重新生成。logic、config、svc、main 是可扩展骨架；handler 保留协议处理。中间件、model 扩展和测试按职责手写。引用类型与方法先核对实际签名。手写命名沿用项目的 `XxxRpc`、`Id`、`Url`、`Ip` 风格。

## 3. 请求、数据与响应

- handler/server 处理协议，API logic 转换参数并调用 RPC。API 请求的 validator 约束写在 `.api` 的 validate 标签，可选字段以 `omitempty` 开头；按数据库列宽、枚举、RPC 或契约选取约束。logic 用 svc 的 Validator 调用 `StructCtx`，失败返回 `xerr.REQUEST_PARAM_ERROR`。RPC 根据可信身份独立校验权限和业务条件。
- SQL、分页、关联写入和软删除放在现有 model 扩展。按更新语义、软删和缓存能力选择生成方法或参数化查询；logic 传入明确数据范围。列表关联和统计使用集合查询或受限批量读取，避免逐行查询形成 N+1。多列查询扫入带 `db` 标签的 row struct，单列可扫入标量。所需 Unix 时间在 SQL 中显式转换并命名。
- 需要原子性的多次写入用同一 `TransactCtx` 和 `sqlx.Session`。依赖现状的检查用条件写入或事务锁保护。提交后按实际规则失效 ID 和新旧唯一索引缓存，区分数据库写入失败与提交后缓存处理失败；策略通知在成功写入后发送。
- 业务错误使用 `pkg/xerr`。普通 JSON 端点沿用 `httpx.SetOkHandler` 和 `SetErrorHandlerCtx`，保留审计与统一信封。SSE、下载等由 handler 按契约输出；参数解析、校验及建立 RPC 流失败走统一错误出口；响应提交后走协议错误流程，写入或 Flush 失败立即结束连接。保持身份、权限和审计链。

## 4. 配置

在 `internal/config` 定义字段。对含环境占位符的 YAML 用 `conf.UseEnv()` 加载，同时核对主入口、二次加载、宿主与容器配置；租户涉及同一能力时还须核对生成模板。直接 `go run` 前在启动终端显式设置进程环境；框架不会自动读取 `.env.local`。Compose 的 `--env-file` 负责编排插值，服务用 `env_file` 或 `environment` 接收进程变量。租户使用专属配置和凭据，核对已有快照与新建模板各自的生效范围。实际构建、启动和就绪检查见[开发手册](../ai_document/dev-guide.md#2-构建启动与就绪检查)。

## 5. AI、工作流与知识库

- 外部模型统一经 `pkg/llm` 封装和用量治理。聊天发送编排限制最多五轮工具调用；每次真实供应商尝试记独立账本，消息/步骤保存该结果的已知用量，未取得的部分保留未知。密钥、提示词和原文遵循[共享契约](../ai_document/api-contract.md)的日志与响应边界。
- 工具实现 `pkg/agenttools.Tool`，准确描述用途和参数，执行前验证长度、格式及业务约束。无状态工具可在 `init` 注册；需依赖的工具在 svc 装配时 `RegisterIfAbsent`。归属从可信上下文 `WithActor` 传递。模型输出及工具参数按不可信输入处理，禁止直接拼接 SQL、命令、文件路径或 HTML。
- 会话、流程、任务和知识库按操作者隔离。审批在指定步骤执行前暂停，包含最后一步；执行快照、断点和原子领取共同防止迟到请求重复执行。DSL、降级、取消和结果未知的行为以共享契约为准。
- MySQL 提交资料与分块后才执行 Embedding 和 Qdrant upsert，随后确认分块就绪。向量操作经 `pkg/vectordb.Store`，删除对不存在的 ID 幂等。检索按 owner/kb 过滤后回 MySQL 复核归属、就绪及软删状态；Redis 缓存与 Qdrant 索引均为派生数据。
- 集合名含 Embedding 模型和维度，更换时使用新集合并核对重建范围。Qdrant 服务端与 go-client minor 版本配对，升级按根规范授权并先做数据卷快照。

## 6. 验证

开发阶段按根规范验证受影响的行为及其直接调用方。提交后端代码前运行 `./data/scripts/verify.ps1`；脚本含格式、编译、vet、依赖零差异、API 定义和配置检查。行为验收按[日常开发与验证流程](../ai_document/dev-guide.md#3-日常开发与验证流程)选择相应场景。
