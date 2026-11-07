# go-zero-admin 开发手册

> 本文回答“如何配置、启动、验证和排障”，按任务选读；不重复维护规范正文、接口契约或单次验收记录。
> 开发边界见[根规范](../AGENTS.md)、[后端规范](../go-zero-admin/AGENTS.md)、[前端规范](../go-zero-admin-react/AGENTS.md)；业务语义见[API 契约](api-contract.md)，已知缺口见[问题清单](README.md#后端剩余事项)。

## 1. 开发环境与配置

首次准备与工具版本见[快速开始](../README.md#快速开始)，以依赖声明、锁文件和实际工具版本为准；Docker 使用 Linux 容器模式，基础镜像须可拉取或已缓存。标准拓扑以 [Compose](../go-zero-admin/docker-compose.yml) 为准：

| 服务 | 宿主访问地址 | 容器网络 / 用途 |
|---|---|---|
| MySQL | `127.0.0.1:3307` | `mysql:3306`，业务库 `gozero_admin` |
| Redis | `127.0.0.1:16379` | `redis:6379`，缓存、会话与策略通知 |
| RustFS | `127.0.0.1:19000`，控制台 `:19001` | `rustfs:9000`，S3 兼容存储 |
| admin-api | `127.0.0.1:7001` | `platform-api:7001`，前端代理目标 |
| admin/app/file/ai/tenant RPC | 默认不映射宿主端口 | 各服务内部 `6001..6005`；tenant 仅平台栈 |
| Qdrant | `127.0.0.1:16333`（REST/面板）、`:16334`（gRPC） | `qdrant:6334`，知识库向量索引；宿主映射仅供宿主调试 ai-rpc |
| Nginx | `:80` | 按域名路由；平台为 `localhost` / `platform.localhost` |
| 前端 dev | `http://127.0.0.1:5999` | 宿主运行，业务请求代理到 API |

### 配置双套制

| 运行方式 | 配置来源 | 地址与环境变量 |
|---|---|---|
| Compose | 后端 `docker/etc/*.yaml`，卷挂载到容器 | 使用容器服务名；AI/App 的 `env_file` 注入后端 `.env.local` |
| 宿主 `go run` | 后端 `application/*/etc/*.yaml` | 使用宿主映射端口；Go 不自动读取 `.env.local` |
| YAML 直填 | 所用配置字段的字面值 | 不因同名环境变量存在而自动覆盖 |
| YAML `${ENV_VAR}` | 进程环境变量 | 仅启用 `conf.UseEnv()` 的加载入口展开；字段留空仍为空 |

六个服务主入口和租户存储二次加载均启用 `UseEnv()`；变更配置时仍须核对宿主/容器两套，以及目标进程实际获得的环境变量。Compose 目前只为 AI/App 注入 `.env.local`，其他服务需要占位符时须显式配置注入。

### AI 与邮件

在后端目录复制模板（不覆盖已有本地配置），再自行填写所需凭据；模板是变量说明，不是运行凭据文件：

```powershell
# 工作目录：go-zero-admin/
if (-not (Test-Path -LiteralPath .env.local)) { Copy-Item .env.local.example .env.local }
```

- 变量名见 [.env.local.example](../go-zero-admin/.env.local.example)，AI 启停按[契约](api-contract.md#ai-http-与执行约定)。默认占位符未取得密钥时模型降级；YAML 已直填有效密钥时，不要求存在 `.env.local`。公开主线禁止真实凭据。
- SMTP 两套配置默认使用 `smtp.163.com:465` 和隐式 TLS；更换服务商时同时核对 Host/Port/ImplicitTLS。From 留空回退 Username，账号和授权码经模板所列变量注入。配置一致和本地假 SMTP 回归不能代替真实发信验收。
- 宿主运行可直填 YAML，或在启动终端用 `$env:变量名=...` 设置进程环境；`setx` 不更新当前终端。env 变更须重新创建相关容器，单独 restart 不更新环境；租户不自动继承平台凭据。

### 数据初始化

**初始化 SQL 会 DROP 并重建业务表，不是已有库的升级脚本。** 唯一入口为 [mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql)：
- 新建开发库：完整导入；Compose 的 MySQL 空数据卷首次启动自动导入，已有卷不自动覆盖或更新结构。
- 已有库改结构：先核对目标与数据处理方案；生成 model 时使用已导入完整 SQL 的新开发库，不为生成代码清空业务库。
- 重置已有库：须明确授权、确认目标并备份；不能把重跑初始化或删除数据卷当作通用排障手段。
- 演示账号与默认凭据见[快速开始](../README.md#快速开始)，仅限开发、部署前必须替换；后台图形验证码由 admin-rpc 的 `Captcha.Enabled` 控制。

### AI 本地开发免图形验证码

AI 在本机开发、联调和功能验收启动时，默认使用忽略的本地覆盖配置关闭图形验证码，账号密码和会话/权限仍正常校验。将 `docker/etc/admin-rpc.yaml` 复制为 `docker/etc/admin-rpc.ai-dev.local.yaml`，只把副本的 `Captcha.Enabled` 改为 `false`；原文件保持默认值。已有副本先核对，不覆盖其中本地设置。

本地 `docker-compose.ai-dev.local.yaml` 内容为：

```yaml
services:
  admin-rpc:
    volumes:
      - ./docker/etc/admin-rpc.ai-dev.local.yaml:/app/etc/admin.yaml:ro
```

从后端目录使用 `docker compose --env-file .env.local -f docker-compose.yml -f docker-compose.ai-dev.local.yaml up -d --no-deps --no-build --wait admin-rpc` 应用验证码配置；其他服务按各自启动流程处理，后端代码改变仍须先编译并构建对应镜像。通过 `/v1/sys/auth/captcha` 核对 `enabled=false` 且图片、ID 为空，再用有效测试账号正常登录。两个 `*.local.yaml` 文件均被忽略，不提交；远程/生产/租户启动不使用这个本地覆盖文件。验证码专项验收改用标准配置并核对 `enabled=true`。

开源初始化的演示管理员为 `admin / Admin@123456`，属于可公开的开发默认凭据，可以写入规范和文档。AI 可直接核对本地开发库与完整初始化 SQL 后正常登录，无需重复索要这些演示凭据。若 README 的本地改动与 SQL 不一致，以实际初始化来源和运行库核对结果为准，保留用户既有文档改动；此流程不授权改密、重置账号或公开真实凭据、AI Key、会话令牌。

### 数据库备份

- 备份默认保留运行库的完整原始 SQL，不擅自删行、改值、脱敏或加密；历史备份不覆盖，初始化来源 `mysql-all.sql` 不作为运行库备份替代物。
- 先核对实际运行库和容器/连接来源，使用一致性快照导出，记录来源、导出时间与文件校验值；在独立环境恢复验证，不导回现有业务库，也不启动会调用外部服务的业务进程。
- 真实邮箱与 AI Key 不提交；不含这两类数据及根规范禁止公开信息的明文备份，按用户指定路径和分支直接提交，日期快照沿用后端 `data/db/init/YYYYMMDD/`。公开提交仍遵守根规范安全基线；受阻的完整原件保存在被忽略的本地目录，不擅自生成脱敏或加密替代文件。
- 分别说明“导出完成”“恢复验证通过”“本地提交完成”“远程推送完成”，不能用其中一项代替其余状态。

## 2. 构建、启动与就绪检查

新建或重置数据库先看[数据初始化](#数据初始化)；常规代码部署不重复导入 SQL 或删除数据卷。

### Compose（日常默认）

[Dockerfile](../go-zero-admin/docker/Dockerfile)只复制宿主预编译的 Linux 二进制，`docker compose --build` **不会编译 Go 源码**。首次运行或后端代码变更后，先执行[构建入口](../go-zero-admin/data/scripts/build-images.ps1)；构建失败立即停止，不部署旧产物。

```powershell
# 终端 A：从仓库根目录执行
cd go-zero-admin
powershell -NoProfile -ExecutionPolicy Bypass -File ./data/scripts/build-images.ps1
if ($LASTEXITCODE -ne 0) { throw "后端编译失败，停止部署" }
docker compose up -d --build --wait
if ($LASTEXITCODE -ne 0) { throw "Compose 启动失败，先排查日志" }
docker compose ps
if ($LASTEXITCODE -ne 0) { throw "无法读取容器状态" }
```

前端另开终端；首次或依赖声明/锁文件变化后安装依赖：

```powershell
# 终端 B：从仓库根目录执行
cd go-zero-admin-react
pnpm install --frozen-lockfile
if ($LASTEXITCODE -ne 0) { throw "依赖安装失败，停止启动" }
pnpm dev
```

只改 YAML：重启读取它的服务；修改 `.env.local`：在后端目录 `docker compose up -d --force-recreate ai-rpc app-rpc`，检查退出码并复验对应功能。代码变更则先重新编译、构建镜像再创建容器，不用 restart 代替部署。

### 宿主单服务调试（手动重启，不是自动热更新）

先核对目标配置与 `svc` 依赖，不对所有服务套同一启动命令。例：从后端目录只启动依赖，再调试 admin-rpc：

```powershell
# 工作目录：go-zero-admin/；先确认不会干扰现有实例和数据
docker compose up -d mysql redis rustfs
if ($LASTEXITCODE -ne 0) { throw "基础依赖启动失败" }
go run ./application/admin/rpc -f application/admin/rpc/etc/admin.yaml
```

- 上例业务如需调用 FileRpc，还须保证对应端点可达；其他服务按实际依赖准备，不以进程能启动推定功能可用。
- 宿主 API 默认连 `127.0.0.1:6001/6003`，启动需访问 AdminRpc/FileRpc，并经 admin-rpc 加载 Casbin 策略；仅启动 MySQL/Redis/RustFS 不够。默认 Compose 的 RPC 不对宿主暴露，须运行所需宿主 RPC，或使用已明确授权的端口映射/混合环境。
- 端口冲突看宿主实际监听/映射：API 的 7001 会冲突，RPC 容器内部同号端口不必然冲突。不要顺手停掉现有服务，也不要把双实例当成同一运行版本；测试优先独立端口与隔离数据。

### 就绪不等于验收

确认启动命令、配置挂载及运行产物对应本轮改动；不确定时比较容器二进制与本地产物摘要。容器 Up / Compose wait 通过不等于全部业务可用：

```powershell
# 任意终端：仅探针示例，不代替业务验证
$ready = Invoke-RestMethod -Uri "http://127.0.0.1:7001/ready" -TimeoutSec 15 -ErrorAction Stop
if ($ready.code -ne 200 -or -not $ready.result.ready) { throw "API 尚未就绪" }
```

再检查 `/ping`、前端入口及受影响接口/页面；HTML 200 只说明页面可达，AI/邮件还需各自验证。持久失败查[排障表](#4-常见故障排查)，不靠不断重启掩盖错误。

## 3. 日常开发与验证流程

规范正文以三层 AGENTS.md 为唯一事实源，本节不复制其内容，仅提示执行顺序：确认范围与授权（根 §5.7）→ 契约先行与生成入口（根 §5.2、后端 §2）→ 实现、针对性回归及必要静态检查（根 §9、后端 §3）→ 受影响功能验收（根 §9）→ 按授权提交前执行完整自检（后端 §7、前端 §8）→ 精确清理与交付（根 §9.4–9.6）。开发阶段不默认检查无关功能或重复全量回归；每个受影响模块实现后先验证再推进依赖模块。未覆盖或失败列[待办](README.md#后端剩余事项)，不伪装全部通过。

AI 发送与 model 已恢复 Go 回归：本地模型桩验证工具检查点与 SSE 终帧，真实数据库/缓存回归需显式设置 `GOZERO_AI_TEST_DSN`、`GOZERO_AI_TEST_REDIS` 并使用独立 MySQL/Redis 实例；变量缺失时这些集成用例明确跳过，不能记为业务验证通过。运行 `go test ./application/ai/rpc/internal/logic/aiservice ./application/ai/rpc/internal/model ./application/admin/api/internal/handler/ai -count=1` 可复验。其他模块仍缺少 Go 行为回归；`test/verify_script_test.ps1` 继续在独立临时夹具中验证自检脚本的失败出口，CI 默认环境不代表数据库/Redis 集成验证。

知识库创建、更新与重试均提交持久后台作业：请求只等待接收回执，使用普通请求的 30 秒等待上限，不再由页面等待 125 秒的同步摄取结果。MySQL 同事务保存文档/分块代次、批次和作业，2 个处理槽从队列领取；单次处理截止 90 秒、租约 120 秒，失败清理另有 5 秒窗口。`queued` 可跨重启继续领取；租约过期的 `running` 标记失败并提示核对，不自动重放付费调用。相同知识库内标题、正文和 Embedding 模型完全相同的创建继续复用现有文档，批次作业显示 `reused`，不重复 Embedding。模型、分块、代次保护和上下文预算以共享契约为准。

知识库从 `/ai/knowledge` 进入独立工作区 `/ai/knowledge/:id`，含“资料”“测试资料检索”和一次分析区；`?doc_id=:doc_id` 可直接打开本人资料的当前原文。资料列表展示可使用/处理中/失败计数、名称与处理阶段，可按名称和状态筛选。原文单独读取，编辑携带 `expected_version`，冲突先核对新版本；元信息轮询与全文读取协调，状态变化后再刷新所需正文，旧摄取结果不能覆盖新修订。一次分析按问题召回片段并生成带来源结果，不先要求保存流程；历史结果保留当时摘录，“打开当前原文”只用于核对最新版本，不能冒充当时原文。

“粘贴资料”和 TXT/Markdown 导入共用后台批次。正文须非空白、最多 65536 UTF-8 字节，标题最多 128 字符；文件严格按 UTF-8 解码并保留 BOM/CRLF，每批最多 20 篇，浏览器最多并发提交 2 篇。收到回执即可关闭，返回“查看导入批次”查询 `queued/running/succeeded/failed/cancelled/reused`；`pending/stage=queued` 仅表示接收成功，不能当作处理完成。停止批次只取消排队项并阻止后续入队，运行项继续执行。离开页面后停止发送尚未发出的项；页面保留批次与原请求标识，返回后明确显示未提交项和结果未知项，不自动补传。单篇编辑/重试也保留接收回执；网络中断时先核对原批次，404 不作为自动换请求标识重发的依据。浏览器只保存回执标识，不将导入正文复制到批次恢复记录。

相关回归入口包括前端 `node --test scripts/aiKnowledge.test.mjs scripts/aiKnowledgeDocuments.test.mjs scripts/aiKnowledgeImportReceipts.test.mjs scripts/aiWorkspacePolling.test.mjs scripts/aiWorkspaceTransport.test.mjs`，以及后端 `go test ./pkg/llm ./pkg/agenttools ./application/ai/rpc/internal/svc ./application/ai/rpc/internal/model ./application/ai/rpc/internal/logic/aiservice -run '^(TestEmbed|TestKb|TestCompact|TestContextBudget|TestAIStream)' -count=1`。数据库/Redis 摄取用例仍需显式隔离环境；Embedding/聊天使用本地 HTTP 桩，向量使用受控桩，不调用付费服务。命令是复验入口，实际运行结果与真实页面验收范围分别记录，不能据此推定真实供应商或生产负载已通过。仓库合成演示素材见 [ai-rag](ai-rag/README.md)，共 11 篇；PDF/Office/OCR、混合检索、重排及逐篇全文/全库审阅尚未接入。

知识库“测试资料检索”通过 `POST /v1/sys/ai/kbs/:id/search` 返回真实片段、来源 ID、相似度及服务端耗时，复用线上检索器；服务端 10 秒截止、前端等待 15 秒，不调用聊天模型。查看页面操作码和服务端 Casbin 路由授权分别生效，知识库归属仍须通过。新增路由通过“接口管理 → 同步路由”登记目录，保留原角色权限，不因页面入口可见就自动授予 API。已有实例需核对本轮新增批次/作业表与接口授权，不重导初始化 SQL 覆盖业务数据。固定题集与评测工具使用方式见 [直接召回测试](ai-rag/README.md#直接召回测试)，先用真实片段区分检索问题与回答问题，再评估分段、阈值或重排；目前无最低相似度阈值。

AI SSE 路由单独在 `desc/sys/ai.api` 的 `@server` 块声明 `sse: true`，经 `gen.ps1 api` 生成 `rest.WithSSE()`；同名 `group: ai` 不会合并不同块的路由选项，不要给包含普通 JSON 接口的整个块启用 SSE。框架接管 SSE 响应头及连接写入期限，客户端仍需 `Accept: text/event-stream` 旁路普通 HTTP 超时；业务截止时间继续由 AI RPC 控制。handler 保留代理防缓冲、命名事件、心跳及 gRPC 桥接；Flush 沿当前审计/go-zero 透传 writer 展开并优先保留 `FlushError`，再交给 `ResponseController`，避免框架无错误返回的 `Flush` 吞掉底层错误。写入或 Flush 失败即退出，不要用直接断言 `http.Flusher` 的 `httpx.Stream` 覆盖该行为。路由与错误边界可运行 `go test ./application/admin/api/internal/handler/... -count=1` 复验，无需数据库或模型凭据。

前端 AI 回归在前端目录运行 `node --test scripts/aiClient.test.mjs scripts/aiChat.test.mjs scripts/aiSend.test.mjs scripts/chatScroll.test.mjs`，验证流式鉴权、权限、原请求重查、防重复发送、权威结果回填、分页游标、迟到响应、安全文本渲染，以及滚动跟随、上翻暂停和历史锚点；不调用真实模型。AI 对话入口为 `/ai/chat`，位于一级“AI 功能”下；只读、创建、发送、删除需分别配置菜单操作码，服务端仍需对应 Casbin 路由授权。新增库使用完整 SQL 内置菜单，已有库通过正常菜单/角色接口精确添加并保留原授权，不能重导初始化 SQL。

AI 四模块入口为“知识库”`/ai/knowledge`、“常用流程”`/ai/workflows`、“运行记录”`/ai/tasks`、“工具与能力”`/ai/tools`，知识库详情 `/ai/knowledge/:id` 继承父页面权限且不增加侧栏项。页面、操作码与接口授权见共享契约。已有开发库按需通过“菜单管理”补齐“AI 功能”页面和操作权限，再通过“角色管理 → 功能授权”补勾，保留原授权；新增菜单不会自动关联超级管理员。保存前核对界面“已选 N 项”，保存后重新读取并从侧栏进入对应页面确认，不以 DOM 勾选属性或源码注册替代实际生效检查。已有业务记录、账号与接口策略无需因导航调整重置。

“常用流程”的新建和编辑进入 `/ai/workflows/new`、`/ai/workflows/:id/edit`，沿用父菜单权限，不新增侧栏项。资料分析模板默认只有一步，知识库与问题引用本次输入，运行时单独填写；JSON 模板明确为格式整理，已有多步骤与静态参数定义继续兼容。编辑器可以直接加入能力，普通参数选择固定内容、本次输入或前一步结果，复杂参数保留高级 JSON，服务端按 v2 引用规则和工具 Schema 校验。支持复制流程和步骤，审批等高级选项按需展开。

当前未保存内容可“试运行并查看结果”，知识库工作区也可直接“开始分析”；两者通过 `POST /v1/sys/ai/tasks/run` 留下正式任务，`workflow_id=0` 表示未关联常用流程，不自动创建流程。可在结果确认后“保存为常用流程”，也可直接保存；已保存流程的手动运行使用保存定义。草稿按当前会话、部门和流程保留在当前标签页，刷新可恢复，保存成功或明确丢弃后清除。运行 `request_id`、原定义和原输入保留到结果确认，结果未知时核对原运行，不生成另一任务标识。浏览器存储失败时同页内存仍保留标识；刷新前须到运行记录核对。

`kb_analyze` 的注册能力始终可见，运行仍要求检索与聊天模型均已配置；内部执行一次检索和一次有界分析，不向聊天 Agent 暴露该嵌套模型工具。模型适配沿用 Eino；生成与 Embedding 按供应商计费。结果展示分析、实际参考片段和已知聊天用量，不含 Embedding，供应商未返回时标记不完整，0 不等于免费。运行和批准只等待提交（普通请求 30 秒），工作流后台每轮 90 秒、每实例 8 并发。审批前预览本次已解析参数；取消停止后续步骤且不撤回已有模型调用或外部作用。进程中断不自动重放，详情读取会将连续 100 秒未更新的 `running` 标记未知失败；工作流持久执行队列、定时调度和跨人审批仍未实现。

“运行记录”以可读标题、流程名快照和输入摘要查找本人任务，详情优先显示结果、实际引用来源与已知用量；输入、步骤记录、耗时、原始输出和错误放在可展开的技术详情。未保存试运行与已保存流程统一留存，历史记录不按最新流程定义重算。列表和详情状态刷新协调：有父列表元信息时复用其变化，独立详情按元信息接口观察状态，变化后补读完整记录；后台标签页不轮询，终态、写操作与失败停止自动刷新，任务自动观察最多 2 分钟。这是实现行为说明，实际请求量与并发性能需另行测量。

“工具与能力”按中文用途展示资料分析、查找资料、JSON 格式整理及本地工具的输入输出示例，可从目录创建流程或在编辑器加入当前流程；参数 Schema 默认折叠。`retrieval_configured`、`analysis_configured` 表示服务端对应客户端已配置，**不是供应商实时健康探测**；未知、未配置与读取失败分别展示，不能据“已配置”推定本次调用成功。目录与工作区不回显密钥或供应商地址。

前端复验入口为 `node --test --test-concurrency=1 scripts/aiProductWorkspace.test.mjs scripts/aiWorkflowDataflow.test.mjs scripts/aiWorkflowEditor.test.mjs scripts/aiWorkflowAnalysis.test.mjs scripts/aiTasksTools.test.mjs scripts/aiToolCapabilities.test.mjs`；浏览器探针 `scripts/aiToolCapabilities.browser.js` 和 `scripts/aiWorkflowRun.browser.js` 仅在独立会话、全 API 响应桩下运行，不可直接对真实业务执行。响应桩回归不能替代真实主栈配置、权限和页面闭环验收。

对话输入区可以指定本人知识库并选择“仅依据资料回答”；实际检索片段保存为历史来源快照，后续文档修改不会改写旧消息。依据模式的问题最多2000 UTF-8字节，无可用片段时不调用聊天模型；候选片段不等于一定有答案，资料不足仍需模型按约束说明。会话支持标题/正文搜索、重命名/置顶，成功助手消息可评价并补充说明。完整导出逐页读取，支持取消与身份切换保护；已加载消息导出单独标明。受限Markdown只创建React元素，不渲染原始HTML或自动加载图片。

此前同步摄取版本的局部测量记录：相同 owner/KB/软删过滤下，11 篇演示资料的列表 CLI UTF-8 序列化结果从 13095 降到 914 字节（剔除 12181 字节正文），COUNT+分页仍为 2 次 SQL；单文档状态只查 1 次元信息。任务摘要固定夹具从 197068 降到 80 字节，仍为 2 次 SQL；后台取消观察每秒只读 status。阶段可见性当时增加一次摄取 CAS 事务。这些历史数据不作为本轮后台批次、搜索字段或轮询协调实现的性能验收；本轮未据此宣称模型时延或并发吞吐提升，需要独立数据集与隔离负载重新测量。

原有工作区回归入口保留：前端 `node --test scripts/aiWorkspace.test.mjs scripts/aiWorkflows.test.mjs scripts/aiTasksTools.test.mjs`，后端菜单源检查 `powershell -NoProfile -ExecutionPolicy Bypass -File test/ai_workspace_seed_test.ps1`。隔离工作流回归使用 `go test ./application/ai/rpc/internal/logic/aiservice -run '^TestAIWorkspace' -count=1 -v`，只有显式设置 `GOZERO_AI_TEST_DSN`、`GOZERO_AI_TEST_REDIS` 才执行数据库/缓存场景，缺环境时明确跳过。另有 `scripts/aiKnowledgeImportReceipts.test.mjs` 与全响应桩浏览器入口 `scripts/aiKnowledgeImportReceipts.browser.js`，覆盖离页、迟到回执、配额失败与部分确认不重复渲染。四模块本地主栈已验证文本导入、直接分析及来源、草稿/保存流程、审批与失败、筛选和窄屏；完整边界见[问题清单](README.md#后端剩余事项)。运行命令按本次影响范围选择，不默认重跑聊天或其他模块。

追加复查入口：`scripts/aiDraftRunAttempts.test.mjs` 验证存储异常下保留原运行标识；`scripts/aiDraftRunAccess.browser.js` 在独立6009端口、全API响应桩下验证任务快照权限与SPA离页恢复，不能直接对业务数据执行。后端 `go test ./application/ai/rpc/internal/logic/aiservice -run '^TestKbBackground' -count=1 -v` 配合上述隔离环境，覆盖更新作业过期时新旧向量补偿、跨知识库请求标识竞争及事务回滚。任务快照保存需要运行记录查看权限；存储异常的内存备份仅在当前页面运行环境内恢复，硬刷新或关闭标签页前仍需核对运行记录。

2026-11-07 工作区版本的有限性能证据：两篇合成文本的后台接收回执分别约471/573毫秒，接收耗时不包含Embedding完成；真实浏览器就绪原文和终态任务各为一次完整详情GET、零额外meta GET。普通任务详情由原先两次SELECT改为一次，超时状态恢复分支例外；流程列表不返回定义，同批三条流程详情合计1166字节定义从列表省去。KB计数会增加统计查询工作，检索的Embedding/向量/原文调用链没有减少，不能将这些局部证据解释为模型提速或生产吞吐结论。

认证可复用[登录 HTTP 回归入口](../go-zero-admin/test/login_e2e.ps1)：只用专用测试账号，正常取得一次性图形码，不读数据库/缓存绕过验证，也不退出其他设备。纯文档只需检查内容、链接与 `git diff --check`，不强制构建或全量 E2E。外部环境和故障场景边界见[问题清单](README.md#后端剩余事项)。

[变更 HTTP 回归入口](../go-zero-admin/test/mutation_e2e.ps1) 仅接受本机地址，使用具备管理权限的开发测试账号，覆盖角色/菜单/字典部分更新和本地 `text_stats` 两审批点。凭据与正常图形码通过 `MUTATION_E2E_*` 参数/环境变量传入，清理清单须指向仓库内被忽略的新文件；脚本精确软删本轮对象并仅退出本次会话，无删除入口的任务 ID 留在清单供精确清理，不调用模型或邮件。

## 4. 常见故障排查

通用 go-zero 问题（生成、配置、运行时）先查[官方排障](../.ai-context/zero-skills/troubleshooting/common-issues.md)；本表只保留本仓库历史事故的检查路径，不维护次数或重复禁令；查看日志时不回显凭据/完整请求，涉及写库或停服务先核对授权。下列 Compose 命令在后端目录执行：

| 症状 | 先检查与处理 | 如何确认 / 边界 |
|---|---|---|
| API 退出、反复重启 | `docker compose logs api admin-rpc --tail 50`；检查 admin-rpc 就绪、Casbin 策略与数据库/Redis | restart 策略只缓解启动时序；持续失败必须查因，再验证 `/ready` 与业务 |
| 7001 可达、Nginx 502 | API 重建后可能缓存旧上游地址，确认平台上游服务，再按下例校验并 reload | 用平台 Host 复验；未知 Host/IP 直访默认站点 404 是域名隔离设计 |
| 404 / 连接拒绝 / 请求到旧服务 | `netstat -ano` 查端口、PID 与容器映射，核对请求域名和代理目标 | 不停无关项目；选择独立测试端口，确认命中本轮实例 |
| `unknown service` / 新 RPC 不生效 | 核对调用方连接的模块/端口及 main 的服务注册，不能把 FileRpc 误连 admin-rpc；查 `gen.ps1 rpc` 告警 | 生成命令成功不保证已有 main 已补注册；调用新增方法复验 |
| 多 protobuf 包注册冲突 / etc 出现重复 YAML | 核对模块独立 proto package、`--name-from-filename` 和运行 `-f` 路径 | 不因重复就盲删配置；保留模块配置与手写内容，按规则消除冗余 |
| `not matching destination to scan` / 时间查询报数据库错误 | 自定义查询用带 `db` tag 的 row struct，不直扫 pb；时间转 int64 按规则用 `UNIX_TIMESTAMP()` 与显式别名 | 逐列检查查询/目标类型并真实读取，不以编译通过代替扫描验证 |
| 直接改库后仍读旧数据 | 查看实际生成 model 的 ID/旧新唯一索引缓存键，仅精确失效受影响记录；授权变更核对策略通知 | 重新读取确认；不以 FLUSH 或全库清空代替定位 |
| AI/SMTP 填了配置却不可用 | 核对[配置双套制](#配置双套制)、实际加载入口、Provider、SMTP 主机/端口/TLS 和容器是否重建 | 区分未配置降级与外部调用失败；不打印密钥或把占位符展开误认为读取 env 文件 |
| 知识库摄取 503 / 检索无结果 | 核对 `Qdrant.Address` 与 `AI_QWEN_KEY`（Embedding 固定走 qwen）双套配置、ai-rpc 是否重建、qdrant 容器日志；向量维度与 `EmbeddingDimensions` 不一致会在摄取时报维度错误 | 未配置降级是设计行为；Qdrant 数据是派生索引，灾难恢复 = 清空集合后将文档逐个 retry 重建，不修向量数据 |
| 知识库检索到已删除内容 | 属双写漂移窗口：检索回 MySQL 复核应拦截；若仍出现，查 chunk status 与文档软删标记 | 以 MySQL 为准核对并按明确文档/分块 ID 人工补偿；当前没有运行期自动对账任务，不能等待其自动收敛 |
| PowerShell 中文乱码、解析失败或 SQL 导入乱码 | 区分终端显示、文件编码与数据库存储；先 `Get-Content -Encoding UTF8`/原始字节核对，再考虑修改文件 | 兼容 Windows PowerShell 5.1 的含中文脚本保留 UTF-8 BOM；不要因显示乱码重写源文件 |

```powershell
# 后端目录；仅在本项目确需刷新上游时执行，校验失败不 reload
docker compose exec -T nginx nginx -t
if ($LASTEXITCODE -ne 0) { throw "Nginx 配置校验失败" }
docker compose exec -T nginx nginx -s reload
if ($LASTEXITCODE -ne 0) { throw "Nginx reload 失败" }
```

代理复验应带 `Host: localhost`（例如向 `http://127.0.0.1/ready` 发请求），不修改默认站点来绕过租户隔离。

SQL 传输优先复制 UTF-8 文件到容器后重定向导入，并显式 `--default-character-set=utf8mb4`，避免 PowerShell 管道转码；凭据走安全输入/进程环境而非命令明文。怀疑二次编码时先备份、核对实际字符与目标行列，再按授权精确修复，不套用无条件 UPDATE 模板。PowerShell 续行用反引号而非反斜杠，脚本执行策略仅对本次调用使用 `-ExecutionPolicy Bypass`，不修改机器全局策略。

## 5. 清理与恢复

按[根完成定义](../AGENTS.md#9-任务完成定义)处理，原则是只恢复本轮临时改动，而不是清空整个环境：
1. 记录并按准确 PID/任务 ID、容器名、路径与临时数据 ID 清理；保留用户要求运行的项目栈，不按进程名/通配符批量删除。
2. 恢复本轮临时配置与经授权调整的服务，确认测试端口释放；回归测试、用户数据及既有备份保留。
3. 用 `git status`/差异确认已有改动未被覆盖；可以保留本任务未提交内容，不为“工作区干净”执行 checkout/reset。测试 Git/hook 使用独立临时仓库，不在真实仓库造提交再撤销。

## 6. 专项维护入口

### 登录猫动画

维护细节与唯一生成命令见[素材说明](../go-zero-admin-react/public/mascots/README.md)、[生成器与测试](../go-zero-admin-react/scripts/build-login-cat.md)。来源图集为 [login-cat-atlas.png](../go-zero-admin-react/src/assets/login-cat-atlas.png)（3×3、1536×1536），生成 `.riv` 与静态备用图，不手改产物；依赖版本查前端声明/锁文件，历史授权不等于允许任意加依赖。

| 场景 | 保留的行为边界 |
|---|---|
| 闲置/指针 | 呼吸、眨眼、左右转头，进出阈值不同以防边界闪动 |
| 表单/密码聚焦 | 看向表单；密码框聚焦时捂眼、离开后放爪；动画只接收聚焦/结果状态，不读取密码内容 |
| 登录成功 | 成功反馈后约 1.1 秒保存会话并跳转原目标；资源未就绪/窄屏直接完成，不无限等待动画事件 |
| 登录失败 | 失败反馈不取代原错误提示、验证码清空与刷新流程 |
| 桌面/小屏 | 768px 及以上默认互动（当前包括减少动态效果设置，无手动开关）；以下隐藏装饰角色且不加载 Rive |
| 资源失败/卸载 | 素材和 WASM 本地托管；失败显示静态备用图，表单仍可用；清理事件、计时器和动画帧 |

这是图片姿态与 Rive 轨道组合，不是实时三维模型。涉及素材或交互时，按专项说明运行测试，再实际操作焦点、成功/失败、资源失败、快速切换与窄屏；Node 加载或模拟登录通过不等于真实后端登录验收。Rive API 兼容性按锁定版本核对，不为消除弃用提示擅自升级。效果参考：[登录页动图](screenshots/login-cat.gif)（不是当前版本的验收证据）。
