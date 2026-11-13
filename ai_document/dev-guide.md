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

六个服务主入口和租户存储二次加载均启用 `UseEnv()`；变更配置时仍须核对宿主/容器两套，以及目标进程实际获得的环境变量。Compose 的整份 `.env.local` 通过 `env_file` 注入 AI/App；API、tenant-rpc 另有显式代理地址注入。Compose 插值和服务进程环境是两层，不能以服务的 `env_file` 代替 Compose 的 `--env-file` 或启动终端环境。

### 可信代理与文件上传预算

API 的 `TrustedProxies` 是显式 IP/CIDR 列表，宿主配置默认为 `[]`。只有 TCP 对端在列表内才读取来源头；`X-Forwarded-For` 从右向左跳过可信代理，坏链回退 TCP 地址，没有该头时才读取合法单 IP 的 `X-Real-IP`。直接请求即使伪造这些头，也不能改变审计和 IP 配额使用的地址。额外反向代理需按实际来源显式配置，不使用全来源信任。

新 Compose 部署为 Nginx 保留静态地址，动态池避开该地址：

| 网络 | 默认子网 | 默认动态池 | 默认 Nginx 地址与信任范围 |
|---|---|---|---|
| 平台 `default` | `172.20.0.0/16` | `172.20.128.0/17` | `172.20.0.3`，平台 API 仅信任该 `/32` |
| 租户边缘 `gozero-edge` | `172.21.0.0/16` | `172.21.128.0/17` | `172.21.0.2`，新租户 API 仅信任该 `/32` |

切换环境时整体设置 `GOZERO_PLATFORM_SUBNET`、`GOZERO_PLATFORM_IP_RANGE`、`GOZERO_PLATFORM_PROXY_IP`，以及对应的 `GOZERO_EDGE_*` 三项；代理 IP 同时用于 Nginx 地址和 API 的可信 `/32` 注入。代理必须位于对应子网内、动态池外，不能占用显式网关或默认首个可用地址，子网不得与本机其他网络冲突。[verify-network.ps1](../go-zero-admin/data/scripts/verify-network.ps1) 只渲染 Compose 并检查上述关系；存在后端 `.env.local` 时由脚本自行解析并写入进程环境后再渲染，不依赖 Compose 版本对 `--env-file` 的插值优先级，不要求 Docker 引擎在线、不打印完整配置；构建入口先执行该检查。多文件配置可用其 `-ComposeFiles` 参数核对。外部网络不重复要求 IPAM，须另行只读核对实际子网、动态池与代理地址。此处仅说明新部署；已有网络和已生成的租户配置不自动迁移，不用 `down` 或重建网络代替迁移核对。

已有栈迁移网络前记录容器ID、端点和卷，确认无在途任务，保留基础服务的数据及登录状态。本机 Redis 容器可能没有数据卷，不能通过删容器重建处理网络问题。本轮在保留原容器重连后出现“容器内正常、宿主映射端口超时”，正常重启原基础容器才刷新端口转发；Redis先确认`SAVE`成功，再逐个核对原ID、挂载、握手及健康，不使用`down -v`、prune或批量清缓存。其他项目端点和既有租户栈不在平台迁移范围内。

文件上传预算按对象存储 30 秒 → file-rpc 服务端 35 秒 → API 的 FileRpc 客户端 40 秒 → 上传 HTTP 45 秒 → 前端上传 50 秒逐层留余量；宿主、Compose 与新租户配置同步。只有上传 HTTP 路由获得 45 秒，其他文件 HTTP 路由保留原预算。修改任一层时核对调用链与生成路由，超时/取消不证明对象写入回滚，仍按文件引用与补偿机制核对结果。

### 对象存储测试素材

后端 `data/test-files/` 用于保存配合 RustFS 对象存储使用的演示图片与测试文件，供 file-rpc 上传、下载和签名 URL 的演示与联调。建议使用公开的 PNG/JPEG 演示图片，每张小于 1MB；这些素材仅供本地开发，不打包进任何镜像。

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

[Dockerfile](../go-zero-admin/docker/Dockerfile)只复制宿主预编译的 Linux 二进制，`docker compose --build` **不会编译 Go 源码**。以下示例沿用快速开始中已创建的 `.env.local`，保证启动与网络预检查使用同一变量文件。首次运行或后端代码变更后，先执行[构建入口](../go-zero-admin/data/scripts/build-images.ps1)；构建失败立即停止，不部署旧产物。

```powershell
# 终端 A：从仓库根目录执行
cd go-zero-admin
powershell -NoProfile -ExecutionPolicy Bypass -File ./data/scripts/build-images.ps1
if ($LASTEXITCODE -ne 0) { throw "后端编译失败，停止部署" }
docker compose --env-file .env.local up -d --build --wait
if ($LASTEXITCODE -ne 0) { throw "Compose 启动失败，先排查日志" }
docker compose --env-file .env.local ps
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

只改 YAML：重启读取它的服务；修改 `.env.local` 中的 AI/SMTP 变量：在后端目录 `docker compose --env-file .env.local up -d --force-recreate ai-rpc app-rpc`，检查退出码并复验对应功能。其他变量按实际归属处理；既有网络变化先单独核对迁移方案，不套用普通重建。代码变更则先重新编译、构建镜像再创建容器，不用 restart 代替部署。

### 宿主单服务调试（手动重启，不是自动热更新）

先核对目标配置与 `svc` 依赖，不对所有服务套同一启动命令。例：从后端目录只启动依赖，再调试 admin-rpc：

```powershell
# 工作目录：go-zero-admin/；先确认不会干扰现有实例和数据
docker compose --env-file .env.local up -d mysql redis rustfs
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

相关回归入口包括前端 `node --test scripts/aiKnowledge.test.mjs scripts/aiKnowledgeDocuments.test.mjs scripts/aiKnowledgeImportReceipts.test.mjs scripts/aiWorkspacePolling.test.mjs scripts/aiWorkspace.test.mjs`，以及后端 `go test ./pkg/llm ./pkg/agenttools ./application/ai/rpc/internal/svc ./application/ai/rpc/internal/model ./application/ai/rpc/internal/logic/aiservice -run '^(TestEmbed|TestKb|TestCompact|TestContextBudget|TestAIStream)' -count=1`。数据库/Redis 摄取用例仍需显式隔离环境；Embedding/聊天使用本地 HTTP 桩，向量使用受控桩，不调用付费服务。命令是复验入口，实际运行结果与真实页面验收范围分别记录，不能据此推定真实供应商或生产负载已通过。仓库合成演示素材见 [ai-rag](ai-rag/README.md)，共 11 篇；PDF/Office/OCR、混合检索、重排及逐篇全文/全库审阅尚未接入。

知识库“测试资料检索”通过 `POST /v1/sys/ai/kbs/:id/search` 返回真实片段、来源 ID、相似度及服务端耗时，复用线上检索器；服务端 10 秒截止、前端等待 15 秒，不调用聊天模型。查看页面操作码和服务端 Casbin 路由授权分别生效，知识库归属仍须通过。新增路由通过“接口管理 → 同步路由”登记目录，保留原角色权限，不因页面入口可见就自动授予 API。已有实例需核对本轮新增批次/作业表与接口授权，不重导初始化 SQL 覆盖业务数据。固定题集与评测工具使用方式见 [直接召回测试](ai-rag/README.md#直接召回测试)，先用真实片段区分检索问题与回答问题，再评估分段、阈值或重排；目前无最低相似度阈值。

AI SSE 路由单独在 `desc/sys/ai.api` 的 `@server` 块声明 `sse: true`，经 `gen.ps1 api` 生成 `rest.WithSSE()`；同名 `group: ai` 不会合并不同块的路由选项，不要给包含普通 JSON 接口的整个块启用 SSE。框架接管 SSE 响应头及连接写入期限，客户端仍需 `Accept: text/event-stream` 旁路普通 HTTP 超时；业务截止时间继续由 AI RPC 控制。handler 保留代理防缓冲、命名事件、心跳及 gRPC 桥接；Flush 沿当前审计/go-zero 透传 writer 展开并优先保留 `FlushError`，再交给 `ResponseController`，避免框架无错误返回的 `Flush` 吞掉底层错误。写入或 Flush 失败即退出，不要用直接断言 `http.Flusher` 的 `httpx.Stream` 覆盖该行为。路由与错误边界可运行 `go test ./application/admin/api/internal/handler/... -count=1` 复验，无需数据库或模型凭据。

前端 AI 回归在前端目录运行 `node --test scripts/aiClient.test.mjs scripts/aiChat.test.mjs scripts/aiSend.test.mjs scripts/chatScroll.test.mjs`，验证流式鉴权、权限、原请求重查、防重复发送、权威结果回填、分页游标、迟到响应、安全文本渲染，以及滚动跟随、上翻暂停和历史锚点；不调用真实模型。AI 对话入口为 `/ai/chat`，位于一级“AI 功能”下；只读、创建、发送、删除需分别配置菜单操作码，服务端仍需对应 Casbin 路由授权。新增库使用完整 SQL 内置菜单，已有库通过正常菜单/角色接口精确添加并保留原授权，不能重导初始化 SQL。

AI 四模块入口为“知识库”`/ai/knowledge`、“常用流程”`/ai/workflows`、“运行记录”`/ai/tasks`、“工具与能力”`/ai/tools`，知识库详情 `/ai/knowledge/:id` 继承父页面权限且不增加侧栏项。页面、操作码与接口授权见共享契约。已有开发库按需通过“菜单管理”补齐“AI 功能”页面和操作权限，再通过“角色管理 → 功能授权”补勾，保留原授权；新增菜单不会自动关联超级管理员。保存前核对界面“已选 N 项”，保存后重新读取并从侧栏进入对应页面确认，不以 DOM 勾选属性或源码注册替代实际生效检查。已有业务记录、账号与接口策略无需因导航调整重置。

“常用流程”的新建和编辑进入 `/ai/workflows/new`、`/ai/workflows/:id/edit`，沿用父菜单权限，不新增侧栏项。资料分析模板默认只有一步，知识库与问题引用本次输入，运行时单独填写；JSON 模板明确为格式整理，已有多步骤与静态参数定义继续兼容。编辑器可以直接加入能力，普通参数选择固定内容、本次输入或前一步结果，复杂参数保留高级 JSON，服务端按 v2 引用规则和工具 Schema 校验。支持复制流程和步骤，审批等高级选项按需展开。

流程保存携带读到的 `expected_revision`；真正变更在同一事务归档旧完整版本并递增 `revision`，空 patch/同值不增版本。编辑器的“版本历史”分页读取摘要、按需读取完整定义；恢复携带当前条件版本，创建更高的新版本，不执行任务、审批或工具，也不改写旧任务快照。409 先核对当前版本和草稿，网络未知先核对原目标历史版本与原条件版本，不自动重复恢复或替草稿提升基线。历史读取沿用流程查看权限，恢复需要编辑权限，服务端另核对本人归属与路由授权。

当前未保存内容可“试运行并查看结果”，知识库工作区也可直接“开始分析”；两者通过 `POST /v1/sys/ai/tasks/run` 留下正式任务，`workflow_id=0` 表示未关联常用流程，不自动创建流程。可在结果确认后“保存为常用流程”，也可直接保存；已保存流程的手动运行使用保存定义。草稿按当前会话、部门和流程保留在当前标签页，刷新可恢复，保存成功或明确丢弃后清除。运行 `request_id`、原定义和原输入保留到结果确认，结果未知时核对原运行，不生成另一任务标识。浏览器存储失败时同页内存仍保留标识；刷新前须到运行记录核对。

`kb_analyze` 仅在知识库检索与聊天模型依赖齐全时注册，运行仍须核对本人知识库归属；内部执行一次检索和一次有界分析，不向聊天 Agent 暴露该嵌套模型工具。模型适配沿用 Eino；生成与 Embedding 按供应商计费。结果展示分析、实际参考片段和已知聊天用量，不含 Embedding，供应商未返回时标记不完整，0 不等于免费。运行和批准只等待提交（普通请求 30 秒），工作流后台每轮 90 秒、每实例 8 并发。审批前预览本次已解析参数；取消停止后续步骤且不撤回已有模型调用或外部作用。进程中断不自动重放，详情读取会将连续 100 秒未更新的 `running` 标记未知失败；工作流持久执行队列、定时调度和跨人审批仍未实现。

任务执行槽与供应商并发是不同边界：任务数不能当作模型请求次数，内部检索、分析及重定向可能产生多个实际 HTTP 尝试，另受下述全栈供应商额度约束。

“运行记录”以可读标题、流程名快照和输入摘要查找本人任务，详情优先显示结果、实际引用来源与已知用量；输入、步骤记录、耗时、原始输出和错误放在可展开的技术详情。未保存试运行与已保存流程统一留存，历史记录不按最新流程定义重算。列表和详情状态刷新协调：有父列表元信息时复用其变化，独立详情按元信息接口观察状态，变化后补读完整记录；后台标签页不轮询，终态、写操作与失败停止自动刷新，任务自动观察最多 2 分钟。这是实现行为说明，实际请求量与并发性能需另行测量。

终态运行可选择另一条本人终态记录作对照，使用服务端搜索、状态和分页，按实际步骤标识配对输入、输出、状态、来源与已记录用量，不运行新任务或重算历史。缺失用量明确显示未记录；创建至结束耗时含等待，文字或耗时差异不能单独证明质量和性能提升。

“工具与能力”按中文用途展示资料分析、查找资料、JSON 格式整理及本地工具的输入输出示例，可从目录创建流程或在编辑器加入当前流程；参数 Schema 默认折叠。`retrieval_configured`、`analysis_configured` 表示服务端对应客户端已配置，**不是供应商实时健康探测**；未知、未配置与读取失败分别展示，不能据“已配置”推定本次调用成功。目录与工作区不回显密钥或供应商地址。

前端复验入口为 `node --test --test-concurrency=1 scripts/aiKnowledgeImportReceipts.test.mjs scripts/aiKnowledgeDocuments.test.mjs scripts/aiDraftRunAttempts.test.mjs scripts/aiWorkflowDataflow.test.mjs scripts/aiWorkflowEditor.test.mjs scripts/aiWorkflowAnalysis.test.mjs scripts/aiTasksTools.test.mjs scripts/aiToolCapabilities.test.mjs`；浏览器探针 `scripts/aiToolCapabilities.browser.js` 和 `scripts/aiWorkflowRun.browser.js` 仅在独立会话、全 API 响应桩下运行，不可直接对真实业务执行。响应桩回归不能替代真实主栈配置、权限和页面闭环验收。

对话输入区可以指定本人知识库并选择“仅依据资料回答”；实际检索片段保存为历史来源快照，后续文档修改不会改写旧消息。依据模式的问题最多2000 UTF-8字节，无可用片段时不调用聊天模型；候选片段不等于一定有答案，资料不足仍需模型按约束说明。会话支持标题/正文搜索、重命名/置顶，成功助手消息可评价并补充说明。完整导出逐页读取，支持取消与身份切换保护；已加载消息导出单独标明。受限Markdown只创建React元素，不渲染原始HTML或自动加载图片。

历史主栈与隔离验证范围见[验证证据](validation-evidence.md)，不同版本的投影字节数、请求量和 SQL 次数统一保留在[性能基线](ai-rag/performance-baseline.md#历史工作区投影与请求量)，不作为当前构建的验收结果。

原有工作区回归入口保留：前端 `node --test scripts/aiWorkspace.test.mjs scripts/aiWorkflows.test.mjs scripts/aiTasksTools.test.mjs`，后端菜单源检查 `powershell -NoProfile -ExecutionPolicy Bypass -File test/ai_workspace_seed_test.ps1`。隔离工作流回归使用 `go test ./application/ai/rpc/internal/logic/aiservice -run '^TestAIWorkspace' -count=1 -v`，只有显式设置 `GOZERO_AI_TEST_DSN`、`GOZERO_AI_TEST_REDIS` 才执行数据库/缓存场景，缺环境时明确跳过。另有 `scripts/aiKnowledgeImportReceipts.test.mjs` 与全响应桩浏览器入口 `scripts/aiKnowledgeImportReceipts.browser.js`，覆盖离页、迟到回执、配额失败与部分确认不重复渲染。历史验证范围见[验证证据](validation-evidence.md)，未覆盖项见[问题清单](README.md#后端剩余事项)。运行命令按本次影响范围选择，不默认重跑聊天或其他模块。

追加复查入口：`scripts/aiDraftRunAttempts.test.mjs` 验证存储异常下保留原运行标识；`scripts/aiDraftRunAccess.browser.js` 在独立6009端口、全API响应桩下验证任务快照权限与SPA离页恢复，不能直接对业务数据执行。后端 `go test ./application/ai/rpc/internal/logic/aiservice -run '^TestKbBackground' -count=1 -v` 配合上述隔离环境，覆盖更新作业过期时新旧向量补偿、跨知识库请求标识竞争及事务回滚。任务快照保存需要运行记录查看权限；存储异常的内存备份仅在当前页面运行环境内恢复，硬刷新或关闭标签页前仍需核对运行记录。


认证可复用[登录 HTTP 回归入口](../go-zero-admin/test/login_e2e.ps1)：只用专用测试账号，正常取得一次性图形码，不读数据库/缓存绕过验证，也不退出其他设备。纯文档只需检查内容、链接与 `git diff --check`，不强制构建或全量 E2E。外部环境和故障场景边界见[问题清单](README.md#后端剩余事项)。

[变更 HTTP 回归入口](../go-zero-admin/test/mutation_e2e.ps1) 仅接受本机地址，使用具备管理权限的开发测试账号，覆盖角色/菜单/字典部分更新和本地 `text_stats` 两审批点。凭据与正常图形码通过 `MUTATION_E2E_*` 参数/环境变量传入，清理清单须指向仓库内被忽略的新文件；脚本精确软删本轮对象并仅退出本次会话，无删除入口的任务 ID 留在清单供精确清理，不调用模型或邮件。

### 用量治理与持久结构维护

“用量与额度”位于 `/ai/usage`：`sys:ai:usage:read` 读取本人账本，`sys:ai:usage:admin` 查看全局，`sys:ai:usage:manage` 管理策略；管理端还实时核对超级管理员身份，前端隐藏不能代替后端授权。默认每用户每日 1000 次实际供应商 HTTP 尝试、全栈并发 8、已知 token 提醒关闭；日期按上海时区。Embedding 按真实批次、重定向按各 HTTP 尝试记账，聊天、分析、独立检索、后台摄取和质量验收统一记录来源，不拿任务数冒充调用数，也不重复累加原消息和步骤的历史 token。

MySQL 按全局策略行→用户策略行顺序锁定，统一检查全栈未过期活跃预留和用户当天额度；Redis 只作快速租约挡板，丢失活跃 key 不能放宽最终准入。活跃显示来自同口径的 MySQL 状态/租约，无法核对时显示未确认。供应商调用有 90 秒上限、活跃预留租约 120 秒；过期预留保守转 `unknown`，不自动重放。并发范围是同一独立栈及共享其 MySQL/Redis 的实例，不代表跨独立租户栈统一计额。

账本区分预留、明确未发出的 `not_sent`、确认发送与不确定结果。`not_sent` 退回预留，已进入传输但尚未确认发送就取消的尝试仍保守占额，并纳入未知提醒；失败、取消、流中断和崩溃不能推定免费。已收到的有效数字即使 SDK/业务校验拒绝仍保留，字段缺失不补零，部分已知值标记不完整；明细不保存 prompt、正文、Key 或 headers。token 阈值只提醒已知值，没有未知费用的 token/金额硬封顶，也不发送邮件。

额度或容量拒绝为业务码 429，DB/Redis/账本不可用时 503 且不启动新的供应商请求；策略 409 保留草稿，先显式读取新版本再决定保存。普通 HTTP 仍使用现有响应信封，SSE 开流后的错误走事件，不能把这些拒绝变成自动重试。

已有库升级先备份并按完整初始化来源精确核对 `ai_workflow.revision_seq`（API字段为`revision`）、`ai_workflow_revision`、`ai_usage_call`、`ai_usage_policy` 及新增索引，补齐全局策略 `owner_id=0` 和菜单/接口授权；包括 `ai_usage_call` 的 `idx_status_lease`，不能仅建表后漏索引。先在隔离库验证增量 DDL 和 model 生成来源，再对授权目标执行，不重导会 DROP 的初始化 SQL、不重置原授权。已有调用不回填成已计费账本，账本启用时间用于解释历史范围；缺表或默认策略时不得放行未计量调用。

复验按影响范围选择：前端 `scripts/aiWorkflowRevisions.test.mjs`、`scripts/aiTaskComparison.test.mjs`、`scripts/aiUsage.test.mjs`；后端 `TestUsage` 系列及 `pkg/llm` 重定向/取消用例。数据库回归仅使用显式隔离连接，各次唯一 ID/key 精确清理，禁止清库或 `FLUSHDB/FLUSHALL`。历史主栈范围和供应商请求观测见[验证证据](validation-evidence.md)与[性能基线](ai-rag/performance-baseline.md#历史工作区投影与请求量)。

### Qdrant 启动恢复与孤儿索引对账

Qdrant 配置有效但初始集合暂不可用时，ai-rpc 保留客户端，后续就绪检查按 1 秒起、最高 30 秒退避重探测，同实例并发只做一次探测。摄取 worker 先核对就绪再领取，检索/摄取先核对再调用付费 Embedding；不能仅因启动先后暂态故障就判定必须重启。配置缺失、无效地址或持续故障仍须修正配置和服务，不承诺已有失败调用自动重试。

ai-rpc 启动约 5 秒后执行首次孤儿对账，此后每轮结束等待 5 分钟；同地址/集合由 Redis 150 秒租约协调，单轮最多 120 秒。先完整读取至多 100 万索引 ID，再按每批 500 个回查 MySQL：有效知识库、文档和 `pending/ready` 分块继续保留，已失效/孤立 ID 才删除。索引快照不完整不驱动删除，批次事实源查询失败不驱动该批删除；失败/部分删除在后续周期按实际快照再核对，超扫描上限明确要求人工处理，不能据一条完成日志声称跨库事务或全部恢复。

对账只删除孤儿，不补缺失向量、不重放 Embedding，也不自动清空集合。集合丢失或模型/维度变更仍需先核对 MySQL 原文、当前集合和配置，按明确文档范围人工重试重建并评估费用；清空集合、跨库批量补偿或重建须有对应授权。检索回 MySQL 的归属、软删与 ready 校验继续生效。小集合删除与原有点保持的历史观测见[性能基线](ai-rag/performance-baseline.md#历史工作区投影与请求量)。恢复、代次竞争及失败按影响范围选用本地桩/隔离回归，持续真实故障、大集合和生产恢复仍单独验收。

### 知识库效果评测

知识库工作区增加“反馈与复核”，实际知识库分析步骤可评价；评价写入需要 `sys:ai:task:feedback`，复核需要 `sys:ai:kb:review`，读取继续使用任务/知识库查看权限。服务端Casbin及本人归属独立校验，首版无跨人审核。已有库从完整初始化来源精确添加 `ai_kb_feedback` 新表，不重导初始化SQL；通过正常菜单/角色接口补齐两个操作码并保留原授权，再使用“接口管理→同步路由”新增5个路由目录项。

复核采纳需预期答案、事实关键词、逐字引用与说明；每个事实须同时在答案与所选摘录出现，资料不足题则无事实/摘录。并发版本冲突先刷新核对，失败保留草稿，不能自动重放。采纳样本下载后按 [业务题集流程](ai-rag/business-cases.md#采纳反馈成为可运行回归)离线转为新题集，原资料/历史题集不覆盖。来源SHA和文档版本在反馈采集时核对，不冒充模型调用时的全库快照；不匹配资料须重新复核。页面采纳仅表示操作结论，AI辅助验收不等于真人签署或语义准确率。

页面已支持在采纳反馈上“生成验收题集”：读取前后两次核对反馈revision，按历史资料ID/版本及正文SHA重读当前原文，来源变化先重新复核。预览可以下载完整冻结JSON或带入质量验收页，导入也使用同一格式；生成/预览/下载不触发模型，明确开始才执行。无原文绑定的资料不足题需额外确认已核对当前资料，不能由原来的no_matches推定全库无答案。自定义快照最多256KiB、100篇资料/100题、单批最多30题；每篇正文最多64KiB。前端和CLI共用严格解析器，拒绝BOM、坏UTF-8、重复字段和不匹配证据。原字节SHA/精确ID/版本在服务端再次校验，未知结果保留原UUID与快照，不自动换标识重跑。

本轮反馈回归入口：前端 `node --test scripts/aiKnowledgeFeedback.test.mjs scripts/aiFeedbackDataset.test.mjs scripts/aiWorkflowAnalysis.test.mjs`；后端 `go test ./application/ai/rpc/internal/logic/aiservice -run 'TestKbFeedback|TestCustomEvaluation' -count=1 -v`（数据库场景必须传显式隔离DSN/Redis），API边界 `go test ./application/admin/api/internal/logic/ai -run TestKbFeedback -count=1`。新增业务题、离线转换及并发工具可运行 `node --test go-zero-admin/data/scripts/business-cases.test.mjs go-zero-admin/data/scripts/merge-kb-feedback.test.mjs go-zero-admin/data/scripts/bench-kb-quality.test.mjs go-zero-admin/data/scripts/kb-custom-dataset.test.mjs`，均为本地桩或离线验证。真实测量另见 [性能基线](ai-rag/performance-baseline.md)，模型调用需在已授权供应商和数据范围内。

[quality-cases.json](ai-rag/quality-cases.json)含40题，保留首轮development24/validation16分组；当前两组均已用于检查或失败修复，应作为历史回归，新的独立检查需另留业务样本。[标注说明](ai-rag/quality-cases.md)列出正负例、证据与规则边界。题集沿用11篇合成演示资料，不是真实用户数据。以下命令均从仓库根目录执行：

```powershell
# 默认模式等同--plan，仅校验本地题集与素材，0 HTTP请求
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --split development

# 本地HTTP桩回归，不调用真实供应商
node --test go-zero-admin/data/scripts/eval-kb-quality.test.mjs

# 显式实跑：先准备正常登录result、KB映射、脱敏运行上下文和已存在的tmp目录
node go-zero-admin/data/scripts/eval-kb-quality.mjs --run --phase baseline --split all --auth-file tmp/auth-private.json --kb-map tmp/kb-map.json --context-file tmp/quality-context.json --output tmp/quality-baseline.jsonl

# 从既有JSONL生成报告，或比较baseline与candidate；不会请求服务
node go-zero-admin/data/scripts/eval-kb-quality.mjs --report tmp/quality-baseline.jsonl --output tmp/quality-baseline.md
node go-zero-admin/data/scripts/eval-kb-quality.mjs --compare tmp/quality-baseline.jsonl tmp/quality-candidate.jsonl --output tmp/quality-compare.md
```

`--run`每题串行执行正式search和草稿`kb_analyze`，会产生真实查询Embedding和聊天费用；完整40题一轮最多提交40次分析、调用80次查询Embedding，文档核对与任务轮询的HTTP请求另计。默认API为`http://127.0.0.1:7001`，`--base`只接受本机HTTP根地址。`--auth-file`保存正常登录的result，`--kb-map`填写实际知识库ID；`--context-file`必须包含`binary_sha`、`model`、`embedding`、`config`，记录实际构建摘要、模型与预算/TopK等配置标签，不能放令牌、密钥或邮箱。每份输出必须使用仓库忽略目录`tmp/`下的新路径，不覆盖旧报告。

复验使用相同实跑命令，将`--phase`改为`candidate`并指定新的JSONL；可用`--split development|validation|all`或`--case-ids`显式选题。工具先读取当前全文核对SHA256、版本与ready状态，结束再次核对；比较要求同题集、同题目、同语料库存/版本/哈希及同聊天/Embedding模型、预算和检索配置，构建摘要变化会保留在报告。失败或不完整运行不计算提升；请求结果未知时保留原UUID、任务ID和记录，先核对原任务，不自动重试或换UUID重发。

先用开发集失败样本确定优化方案，再检查验证集，不按验证集反馈调参或改题。报告把检索命中/多文档召回/证据覆盖、回答规则标记与异常分开，异常不能计为空召回或错误回答。规则未标记也需人工核对语义，引用后否定禁答短语可能误报；这些结果不是语义准确率。P50/P95仅是注明样本量的串行经验分位数，11篇合成资料不能代表真实业务质量、生产并发或容量；已知聊天tokens不包含Embedding费用。

回答一致性修复使用独立的 [12题开发回归](ai-rag/answer-regression-cases.md) 和实测坏答夹具，原40题保持冻结。仅分析专用 `ChatBounded` 固定temperature=0，普通Chat/Stream不变；记录运行上下文时补充 `config.analysis_temperature`，旧默认采样也须明确标注，不能靠省略标签绕过同配置比较。模型内部逐问项回复由服务端校验和渲染，外部结果结构及旧任务兼容；非法回复安全失败，不自动再次付费调用。

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan --cases ai_document/ai-rag/answer-regression-cases.json
# 规则变化时离线生成重算副本；不发HTTP，也不修改原始结果
node go-zero-admin/data/scripts/eval-kb-quality.mjs --rescore tmp/answer-original.jsonl --cases ai_document/ai-rag/answer-regression-cases.json --output tmp/answer-rescored.jsonl
```

rescore仅重算派生指标并保留源SHA、任务、答案、来源、计时和原调用数；不完整、异题集、非法接受状态或事件链不一致仍拒绝写出。副本不是新执行；评分修正规则不等于业务已修复。最新构建、复验范围及残留限制见[后续基线记录](ai-rag/quality-baseline.md#回答矛盾与问项范围的后续修复)。

评测器要求完整的构建/模型/Embedding/预算标签，成功接受状态只允许pending/running/succeeded，完成记录须严格逐题串行；空召回不得夹带聊天正用量。终态耗时在GET确认时捕获。相关失败边界通过本地桩回归，无需为检查工具追加真实模型调用。前端问项转义回归在`go-zero-admin-react/`目录执行 `node --test scripts/safeMarkdown.test.mjs scripts/aiWorkflowAnalysis.test.mjs`，覆盖真实展示组件与现有聊天格式。

演示资料及题集的换行由[根属性文件](../.gitattributes)限定：两素材目录CRLF、题集元数据JSON为LF，以保持原始SHA。跨平台检出的历史证据见[验证证据](validation-evidence.md)；既有工作树不保证自动重写全部文件，校验失败时先保存并核对本地差异、用隔离新检出确认，不能用批量还原覆盖用户改动。

## 4. 常见故障排查

通用 go-zero 问题（生成、配置、运行时）先查[官方排障](../.ai-context/zero-skills/troubleshooting/common-issues.md)；本表只保留本仓库历史事故的检查路径，不维护次数或重复禁令；查看日志时不回显凭据/完整请求，涉及写库或停服务先核对授权。下列 Compose 命令在后端目录执行：

| 症状 | 先检查与处理 | 如何确认 / 边界 |
|---|---|---|
| API 退出、反复重启 | `docker compose --env-file .env.local logs api admin-rpc --tail 50`；检查 admin-rpc 就绪、Casbin 策略与数据库/Redis | restart 策略只缓解启动时序；持续失败必须查因，再验证 `/ready` 与业务 |
| 7001 可达、Nginx 502 | API 重建后可能缓存旧上游地址，确认平台上游服务，再按下例校验并 reload | 用平台 Host 复验；未知 Host/IP 直访默认站点 404 是域名隔离设计 |
| 404 / 连接拒绝 / 请求到旧服务 | `netstat -ano` 查端口、PID 与容器映射，核对请求域名和代理目标 | 不停无关项目；选择独立测试端口，确认命中本轮实例 |
| `unknown service` / 新 RPC 不生效 | 核对调用方连接的模块/端口及 main 的服务注册，不能把 FileRpc 误连 admin-rpc；查 `gen.ps1 rpc` 告警 | 生成命令成功不保证已有 main 已补注册；调用新增方法复验 |
| 多 protobuf 包注册冲突 / etc 出现重复 YAML | 核对模块独立 proto package、`--name-from-filename` 和运行 `-f` 路径 | 不因重复就盲删配置；保留模块配置与手写内容，按规则消除冗余 |
| `not matching destination to scan` / 时间查询报数据库错误 | 自定义查询用带 `db` tag 的 row struct，不直扫 pb；时间转 int64 按规则用 `UNIX_TIMESTAMP()` 与显式别名 | 逐列检查查询/目标类型并真实读取，不以编译通过代替扫描验证 |
| 直接改库后仍读旧数据 | 查看实际生成 model 的 ID/旧新唯一索引缓存键，仅精确失效受影响记录；授权变更核对策略通知 | 重新读取确认；不以 FLUSH 或全库清空代替定位 |
| AI/SMTP 填了配置却不可用 | 核对[配置双套制](#配置双套制)、实际加载入口、Provider、SMTP 主机/端口/TLS 和容器是否重建 | 区分未配置降级与外部调用失败；不打印密钥或把占位符展开误认为读取 env 文件 |
| 知识库摄取 503 / 检索无结果 | 核对 `Qdrant.Address` 与 `AI_QWEN_KEY`（Embedding 固定走 qwen）双套配置、实际产物与 qdrant 日志；再区分启动探测、配置/维度错误及供应商故障 | 初始暂态不可用会退避重探测；缺失向量不自动重建，按[恢复边界](#qdrant-启动恢复与孤儿索引对账)核对范围与费用，不先清空集合 |
| 知识库检索到已删除内容 | 属双写漂移窗口：检索回 MySQL 复核应拦截；若仍出现，查 chunk/doc/KB 状态、归属与软删标记 | 自动对账仅按完整索引快照及 MySQL 事实删除孤儿；失败/超限需核对日志和实际 ID，不能以等待一轮代替确认或自动补向量 |
| PowerShell 中文乱码、解析失败或 SQL 导入乱码 | 区分终端显示、文件编码与数据库存储；先 `Get-Content -Encoding UTF8`/原始字节核对，再考虑修改文件 | 兼容 Windows PowerShell 5.1 的含中文脚本保留 UTF-8 BOM；不要因显示乱码重写源文件 |

```powershell
# 后端目录；仅在本项目确需刷新上游时执行，校验失败不 reload
docker compose --env-file .env.local exec -T nginx nginx -t
if ($LASTEXITCODE -ne 0) { throw "Nginx 配置校验失败" }
docker compose --env-file .env.local exec -T nginx nginx -s reload
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
