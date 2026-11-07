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
| 知识库 / RAG | 文本录入、TXT/Markdown 批量导入、原文预览与版本编辑、向量检索和召回测试 |
| 工作流与任务 | 知识库分析/文本处理模板、步骤参数与结果引用、后台执行、人工审批和取消 |
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

AI 页面集中在侧栏 **AI 功能**：AI 对话、知识库、工作流管理、任务中心、工具目录。知识库、会话和任务按创建者隔离；页面入口、操作权限与服务端接口授权分别校验。已有实例缺少菜单或接口权限时，按[开发手册](ai_document/dev-guide.md#3-日常开发与验证流程)补齐，保留原有授权。

### 配置模型与检索

- **聊天模型**：默认使用 DeepSeek，在后端 `.env.local` 填写 `AI_DEEPSEEK_KEY`；切换 Qwen 时同步调整 `AI.Provider`、`Model`、`BaseURL` 并填写 `AI_QWEN_KEY`。调用由 Eino 的 OpenAI 兼容模型适配器封装。
- **知识库检索**：Compose 已包含 Qdrant，容器配置默认连接 `qdrant:6334`。Embedding 独立使用 Qwen `text-embedding-v4`，无论聊天选哪个模型，都需要 `AI_QWEN_KEY`。MySQL 保存原文、分块与状态，Qdrant 保存可重建的向量索引。
- **应用配置**：容器使用 [ai-rpc.yaml](go-zero-admin/docker/etc/ai-rpc.yaml)，宿主调试使用 [ai.yaml](go-zero-admin/application/ai/rpc/etc/ai.yaml)。修改 `.env.local` 后需重新创建 `ai-rpc` 容器；宿主进程需自行注入环境变量，详见[配置说明](ai_document/dev-guide.md#ai-与邮件)。真实密钥只放本地配置，不提交。

未配置聊天密钥时，会话管理和本地工具仍可用，发送消息返回降级提示；未配置 Embedding 或 Qdrant 时，知识库与文档管理仍可用，摄取和检索不可用。模型生成与 Embedding 按所选供应商计费。

### 从资料到有来源的回答

1. 打开 **AI 功能 → 知识库**，新建知识库，点击该行的“文档”。
2. 使用“添加文档”粘贴标题与正文，或“批量导入”选择 UTF-8 编码的 `.txt`、`.md`、`.markdown` 文件。每篇正文最多 **64 KiB（65536 UTF-8 字节）**，标题最多 128 字符；每批最多 20 个文件，同时处理 2 个。
3. 等待处理阶段变为“已就绪”，再用于检索。可打开“原文”预览、复制或编辑；保存后增加正文版本并重新处理。处理失败时重试已有文档；网络中断显示结果待确认时，先刷新核对，避免重复导入。
4. 在知识库列表点击“召回测试”，输入问题，查看实际命中的原文片段、文档/分块 ID、相似度与检索耗时。此操作调用查询 Embedding，**不调用聊天模型**，适合先检查资料是否能被找回。
5. 打开 **AI 对话**，在输入区选择该知识库，可启用“仅依据资料回答”。该模式的问题最多 2000 UTF-8 字节，无可用片段时不调用聊天模型；回复下方可查看实际提供给模型的来源摘录，旧消息保留当时的来源快照。

相似度表示向量接近程度，不是答案正确率；当前没有最低分阈值，命中片段也可能不足以回答。仓库提供 [11 篇合成演示资料与固定题集](ai_document/ai-rag/README.md)，可按需用于练习与召回评测，不会自动导入。

当前支持文本资料与浏览器读取 TXT/Markdown 后提交正文，尚未接入 PDF、Office、OCR 或通用附件解析；知识库仅供本人使用，不提供跨用户共享。检索采用向量召回，尚未实现混合检索或重排。完整语义与能力边界见 [API 契约](ai_document/api-contract.md#ai-http-与执行约定)。

### 工作流与任务

在 **工作流管理** 中从“知识库分析”“文本处理”或空白模板起步，为步骤选择工具，参数可使用固定值、本次输入或前一步结果。知识库分析工具 `kb_analyze` 依据本人知识库的实际检索片段生成分析，需要聊天模型与检索均已配置。

保存后手动运行，在 **任务中心** 查看步骤输入、输出、耗时与状态；带审批的步骤会在执行前暂停，确认参数后批准或拒绝，也可取消未结束的任务。取消会停止后续步骤，已经完成的外部操作不会撤回。当前不提供定时调度；可用工具与参数以 **工具目录** 为准。

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
- [知识库演示与召回评测](ai_document/ai-rag/README.md)：合成资料、固定题集与评测工具
- [API 契约](ai_document/api-contract.md)：当前字段、权限、业务语义和能力边界；未完成事项见[问题清单](ai_document/README.md#后端剩余事项)
- [架构规划](ai_document/refactor-plan.md)：架构与后续方向
- [前端说明](go-zero-admin-react/README.md)：页面、命令与对接约定
- [开发规范](AGENTS.md)：通用规则及前后端规范入口
