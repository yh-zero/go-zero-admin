##### 右上角求点个star 谢谢大佬们
#### 右上角求点个star 谢谢大佬们
### 右上角求点个star 谢谢大佬们

## 📺 视频教程：[![视频教程](https://img.shields.io/badge/Video_Tutorial-Bilibili-00A1D6?logo=bilibili&logoColor=white)](https://www.bilibili.com/video/BV15mW76qE3J/)
[Bilibili - 项目演示与快速上手:https://www.bilibili.com/video/BV15mW76qE3J/](https://www.bilibili.com/video/BV15mW76qE3J/)


# go-zero-admin

[![CI](https://github.com/yh-zero/go-zero-admin/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/yh-zero/go-zero-admin/actions/workflows/ci.yml)

基于 **go-zero + React** 的管理后台，包含权限、部门、文件、审计、APP 用户管理，以及 AI 对话、知识库检索与审批工作流，适合作为业务系统的开发起点。

后端采用 **1 个 HTTP API + 5 个 RPC 服务**（admin / app / file / ai / tenant），前端采用 React 19、TypeScript 和 Ant Design 6。

CI 自动执行后端 `verify.ps1`、前端类型检查与构建，结果见上方状态标识。

## 功能

| 模块 | 功能 |
|---|---|
| 工作台 | 当前部门与授权范围内的统计、最近操作和常用入口 |
| 系统管理 | 用户、角色、菜单、部门、字典、API 资源、文件、审计和个人中心 |
| 权限控制 | JWT、Casbin、部门身份和数据范围；页面入口与操作权限分开授权 |
| APP管理 | 用户资料与审核、头像、登录设备、会员开通/撤销和反馈回复 |
| AI 对话 | SSE 流式回复、会话搜索/重命名/置顶、消息评价、Markdown 导出与知识库来源 |
| 知识库 / RAG | 资料工作区、TXT/Markdown 后台导入、原文版本、带来源分析，以及评价、复核和回归样本导出 |
| 常用流程 | 一步分析模板、未保存试运行、多步骤与审批，支持版本冲突保护、历史查看及恢复为新版本 |
| 运行记录 | 一次分析与流程运行统一留存，展示实际输入、结果和来源，支持审批、取消及两次终态运行对照 |
| 用量与额度 | 汇总聊天、分析与 Embedding 的已知/未知用量，管理用户日请求额度、全栈供应商并发和已知 token 提醒 |
| 工具与能力 | 按中文用途选择能力、查看输入输出示例与配置状态，在流程中使用 |
| 租户管理 | 列表、状态筛选、只读详情及租户生命周期 HTTP/RPC 接口 |

APP 端另有独立的认证、资料、会员和反馈接口。租户运行栈需另行创建和配置；初始化库不附带虚构的运行栈或租户凭据。

## 快速开始

准备 **Go 1.27.3、PowerShell、Docker（Linux 容器）及 Compose、Node.js 24、pnpm 11.28.5**。

在仓库根目录打开 PowerShell，启动后端：

```powershell
cd go-zero-admin
if (-not (Test-Path -LiteralPath .env.local)) { Copy-Item .env.local.example .env.local }
# 按需编辑 .env.local，填写 AI Key 或 SMTP 邮箱及授权码
powershell -NoProfile -ExecutionPolicy Bypass -File .\data\scripts\build-images.ps1
if ($LASTEXITCODE -ne 0) { throw "后端编译失败，停止启动" }
docker compose --env-file .env.local up -d --build --wait
if ($LASTEXITCODE -ne 0) { throw "Compose 启动失败，请检查日志" }
```

Dockerfile 使用宿主机预编译的 Linux 二进制；首次运行或修改后端代码后，先执行构建脚本。基础镜像需可拉取或已缓存。

另开终端，从仓库根目录启动前端：

```powershell
cd go-zero-admin-react
pnpm install --frozen-lockfile
pnpm dev
```

访问 [http://127.0.0.1:5999](http://127.0.0.1:5999)，演示管理员：`admin / Admin@123456`。默认开启图片验证码，前端请求代理到 API 的 `7001` 端口。

权限演示账号 `demo_all`、`demo_tree`、`demo_dept`、`demo_self` 分别查看全部、本部门及下级、本部门、本人数据；`demo_tree` 可切换部门比较范围。APP 示例账号为 `alice@example.test`，初始化密码同上。已有数据库沿用其实际密码，演示凭据仅限本地开发。

- **初始化数据**：[mysql-all.sql](go-zero-admin/data/db/init/mysql-all.sql) 包含完整表结构与可公开的演示数据。MySQL 空数据卷首次启动自动导入；已有数据卷不会自动覆盖或更新。
- **可选配置**：AI 和邮件不影响基础后台启动，配置项见 [.env.local.example](go-zero-admin/.env.local.example)。本地配置不提交到仓库；完整数据库备份及恢复校验记录见 [20261107](go-zero-admin/data/db/init/20261107)，首次初始化仍使用 `mysql-all.sql`。
- **配置位置**：容器使用 `go-zero-admin/docker/etc/*.yaml`，宿主运行使用各服务的 `etc/*.yaml`。验证码开关为 admin-rpc 的 `Captcha.Enabled`。

详细启动、数据库初始化和排障说明见 [开发手册](ai_document/dev-guide.md)，未覆盖的外部联调和异常场景见 [剩余事项](ai_document/README.md#后端剩余事项)。

## AI 与知识库使用

AI 页面集中在侧栏 **AI 功能**：AI 对话、知识库、常用流程、运行记录、工具与能力、用量与额度。对应路径为 `/ai/chat`、`/ai/knowledge`、`/ai/workflows`、`/ai/tasks`、`/ai/tools`、`/ai/usage`；知识库详情使用独立工作区 `/ai/knowledge/:id`。知识库、会话、流程和任务按创建者隔离；全局用量和策略管理另需管理员权限。页面入口、操作权限与服务端接口授权分别校验。已有实例缺少表结构、菜单或接口权限时，按[开发手册](ai_document/dev-guide.md#3-日常开发与验证流程)精确补齐，保留原有数据与授权。

### 配置模型与检索

- **聊天模型**：默认使用 DeepSeek，在后端 `.env.local` 填写 `AI_DEEPSEEK_KEY`；切换 Qwen 时同步调整 `AI.Provider`、`Model`、`BaseURL` 并填写 `AI_QWEN_KEY`。调用由 Eino 的 OpenAI 兼容模型适配器封装。
- **知识库检索**：Compose 已包含 Qdrant，容器配置默认连接 `qdrant:6334`。Embedding 独立使用 Qwen `text-embedding-v4`，无论聊天选哪个模型，都需要 `AI_QWEN_KEY`。MySQL 保存原文、分块与状态，Qdrant 保存可重建的向量索引。
- **应用配置**：容器使用 [ai-rpc.yaml](go-zero-admin/docker/etc/ai-rpc.yaml)，宿主调试使用 [ai.yaml](go-zero-admin/application/ai/rpc/etc/ai.yaml)。修改 `.env.local` 后需重新创建 `ai-rpc` 容器；宿主进程需自行注入环境变量，详见[配置说明](ai_document/dev-guide.md#ai-与邮件)。真实密钥只放本地配置，不提交。

未配置聊天密钥时，会话管理和本地工具仍可用，发送消息返回降级提示；未配置 Embedding 或 Qdrant 时，知识库与文档管理仍可用，摄取和检索不可用。模型生成与 Embedding 按所选供应商计费。

**用量与额度**从账本启用后记录实际供应商 HTTP 尝试，默认每用户每日 1000 次、全栈并发 8、token 提醒关闭。未知、部分用量和明确未发送分别展示，失败或 0 token 不表示免费；已知 token 提醒不代表金额或未知费用封顶。MySQL 最终校验日额度与全栈活跃预留，Redis 丢失租约不能放宽准入。配置、已有库升级及本地验证边界见[用量治理](ai_document/dev-guide.md#用量治理与持久结构维护)。

### 从资料到有来源的回答

1. 打开 **AI 功能 → 知识库**，新建或进入一个知识库，在独立工作区查看“资料”“测试资料检索”和右侧分析区。
2. 使用“粘贴资料”，或“导入 TXT / Markdown”选择 UTF-8 编码的 `.txt`、`.md`、`.markdown` 文件。每篇正文最多 **64 KiB（65536 UTF-8 字节）**，标题最多 128 字符；每批最多 20 篇，浏览器最多并发提交 2 篇，服务端有 2 个后台处理槽。
3. 接收成功后可关闭导入窗口或离开页面，再从该知识库的“查看导入批次”返回。`queued` 表示等待处理，`running` 表示后台处理中，文档变为“可使用”后再检索。停止批次仅取消尚未开始的项，已开始的作业继续；离开时尚未发出的文件不会自动补传。结果待确认时按原批次和请求标识核对，不自动新建重传。
4. 在“资料”中打开原文预览、复制或编辑；保存后增加版本并提交后台重新处理。失败时重试原文档；版本冲突先核对新版本。使用“测试资料检索”输入问题，查看命中片段、来源标识、相似度与耗时。此操作调用查询 Embedding，**不调用聊天模型**。
5. 在右侧“用这些资料完成工作”填写问题并“开始分析”，无需先创建流程，也会保留正式运行记录。结果包含分析、参考片段与已知聊天用量，可按需“保存为常用流程”。分析仅基于检索到的相关片段，不等于逐篇全文或全库审阅；“打开当前原文”可能看到后续修订，历史摘录保持原样。
6. 对分析结果点击“评价分析”，选择有帮助、回答错误、遗漏或来源不符。在“反馈与复核”中核对历史答案与引用，填写预期答案、事实关键词和原文摘录，采纳后可下载回归样本。本人评价与复核分别需要操作权限，修改评价会重新进入待复核。
7. 打开 **AI 对话**，在输入区选择该知识库，可启用“仅依据资料回答”。该模式的问题最多 2000 UTF-8 字节，无可用片段时不调用聊天模型；回复下方可查看实际提供给模型的来源摘录，旧消息保留当时的来源快照。

相似度表示向量接近程度，不是答案正确率；当前没有最低分阈值，命中片段也可能不足以回答。仓库提供 [11 篇合成演示资料与固定题集](ai_document/ai-rag/README.md)，可按需用于练习与召回评测，不会自动导入。

知识库效果评测使用独立的 [40题题集](ai_document/ai-rag/quality-cases.md)，分为 development24题与 validation16题，分别记录检索、回答规则筛查和串行耗时。以下命令从仓库根目录执行，默认模式与显式 `--plan` 均为 **0 HTTP 请求**：

```powershell
node go-zero-admin/data/scripts/eval-kb-quality.mjs --plan
```

显式 `--run` 会调用真实检索与一次分析，产生查询 Embedding 和聊天模型费用。认证、运行上下文、报告及 baseline/candidate 比较入口见[开发手册](ai_document/dev-guide.md#知识库效果评测)。后续应先根据开发集确定方案，再用新保留集检查回退；当前40题已用于检查或修复，应作为历史回归。回答规则筛查需人工复核，不能称为语义准确率，11篇合成资料的结果也不代表真实业务或生产容量。

回答矛盾与范围问题另有 [12题失败回归](ai_document/ai-rag/answer-regression-cases.md)，覆盖重复、改写和部分资料不足。当前分析按问项展示依据与未知项；[修复与验收记录](ai_document/ai-rag/quality-baseline.md#回答矛盾与问项范围的后续修复)保留已发现的失败与能力边界。

[32题项目业务题集](ai_document/ai-rag/business-cases.md)使用仓库实际文档的8份冻结摘录，题目由AI辅助编写，包含部署、配置、资料维护及权限场景，不是真实用户历史问答。已采纳反馈可通过离线工具并入新的可运行题集；[小规模性能工具](ai_document/ai-rag/performance-baseline.md)分别观测1、3、5个并发工作，保留用量、失败和原始日志。规则通过、小样本耗时及并发工作数均不代表生产准确率或容量。

知识库工作区新增“质量验收”，进入 `/ai/knowledge/:id/quality` 可选择[24道校准项目场景题](ai_document/ai-rag/business-evidence-cases.md)、原场景、历史业务或失败回归题，后台串行执行并保留答案、来源、中文检查项、耗时和已知用量。支持离页返回、原请求核对、停止后续题、单题复测、同条件对照及反馈复核；开始和停止需要 `sys:ai:kb:evaluate` 权限。执行成功与规则通过均须结合实际来源复核，完整CLI基线及已发现的问题见[业务基线](ai_document/ai-rag/business-baseline.md)。

已采纳反馈可直接“生成验收题集”：重新核对当前原文与反馈版本，预览后下载或带入质量验收页，也可导入先前下载的冻结JSON。生成、导入和下载不调用模型；明确开始验收后才执行。原文变化会要求重新复核，未知提交保留原请求标识，避免重复执行。题集和已采纳结论仍需人工判断，不代表质量签署。

当前支持文本资料与浏览器读取 TXT/Markdown 后提交正文，尚未接入 PDF、Office、OCR 或通用附件解析；知识库仅供本人使用，不提供跨用户共享。检索采用向量召回，尚未实现混合检索或重排。完整语义与能力边界见 [API 契约](ai_document/api-contract.md#ai-http-与执行约定)。

### 常用流程、运行记录与能力目录

在 **常用流程** 新建流程，资料分析默认只有一个有效分析步骤，每次运行单独选择本人知识库并填写问题；JSON 模板用于合法 JSON 的格式整理。可以直接试运行当前未保存内容，结果同样保留在 **运行记录**，满意后再保存为常用流程。也可继续添加能力、安排步骤与审批，参数支持固定值、本次输入或前一步结果；已有多步骤和固定参数定义保持兼容。

**运行记录** 统一显示一次性分析与已保存流程的运行，以可读标题、输入摘要、状态和结果为主；实际输入、步骤输出、耗时及原始记录放在技术详情中。带审批的步骤执行前暂停，确认参数后批准或拒绝；也可取消未结束的任务。取消会停止后续步骤，已经发生的模型调用或外部操作不会撤回。结果未知时保留原请求标识、定义与输入，核对原运行，不自动重复执行。

流程修改先核对当前版本，可查看提交前的历史快照并恢复为新版本；恢复不会执行任务。运行记录可对照两条本人终态运行的实际输入、步骤和结果，差异不等于质量提升。版本历史、运行对照、用量页和反馈题集已完成本地主栈验证；异常边界另有隔离回归，生产容量仍需专门测量。

**工具与能力** 按中文用途提供输入输出示例，可在目录创建流程或在编辑器加入当前流程，参数 Schema 默认折叠。目录里的“已配置”“未配置”“配置状态未知”反映服务端客户端配置，**不做供应商实时健康探测**；实际请求仍可能失败。知识库分析 `kb_analyze` 需要检索与聊天模型均已配置，内部通过 Eino 兼容模型封装执行。当前不提供定时调度；PDF/Office/OCR、混合检索、重排和逐篇全文/全库审阅仍未实现。

## 界面预览

### 猫咪登录

Rive 猫咪默认开启互动，支持呼吸、眨眼、鼠标跟随、输入密码时捂眼及登录成功/失败反馈。小屏幕隐藏装饰角色，加载失败时使用静态备用图。

![猫咪登录动态](ai_document/screenshots/login-cat.gif)

### 工作台

![工作台](ai_document/screenshots/dashboard.jpg)

### 部门管理

组织树、部门概览和直属成员分区展示。

![部门管理](ai_document/screenshots/departments.jpg)

### 菜单与角色授权

菜单树维护目录和页面，操作权限按页面单独管理。

![菜单管理](ai_document/screenshots/menus.jpg)

![角色授权](ai_document/screenshots/roles.jpg)

### APP管理

集中维护用户资料、会员、设备和账号状态。

![APP用户](ai_document/screenshots/app-users.jpg)

![APP用户详情](ai_document/screenshots/app-user-details.jpg)

### 租户管理

读取租户状态和详情；全新初始化库的租户列表为空。初始化超级管理员拥有完整菜单，创建、启停等生命周期操作需配置租户运行环境后使用。

![租户管理](ai_document/screenshots/tenants.jpg)

## 项目结构

```text
├── go-zero-admin/          后端：application / data / docker / pkg
├── go-zero-admin-react/    React 管理前端
└── ai_document/            项目文档与截图
```

后端主要使用 go-zero、gRPC、JWT、Casbin、sqlx 与 Eino；前端主要使用 React、Ant Design、TanStack Query、zustand 和 Rive。数据与部署使用 MySQL、Redis、Qdrant、RustFS、Nginx 和 Docker Compose。具体版本以 [go.mod](go-zero-admin/go.mod)、[package.json](go-zero-admin-react/package.json) 和锁文件为准。

## 开发文档

- [开发手册](ai_document/dev-guide.md)：环境、启动、初始化、排障与猫咪动画维护
- [知识库演示与效果评测](ai_document/ai-rag/README.md)：合成资料、召回与回答规则筛查工具
- [API 契约](ai_document/api-contract.md)：当前字段、权限、业务语义和能力边界；未完成事项见[问题清单](ai_document/README.md#后端剩余事项)
- [架构规划](ai_document/refactor-plan.md)：架构与后续方向
- [前端说明](go-zero-admin-react/README.md)：页面、命令与对接约定
- [开发规范](AGENTS.md)：通用规则及前后端规范入口
