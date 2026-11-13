# go-zero-admin 架构决策与演进规划

> 本文档只保留**长期知识**：已定型的架构决策、多租户设计、鉴权模型、AI 平台演进。
> 完成任务和实施流水账不进本文档；历史验收证据集中在验证说明与质量/性能基线。
> 规范本体以三层 AGENTS.md 为准；接口事实源见 [api-contract.md](api-contract.md)。

## 1. 已确认的核心决策

| 项 | 决策 |
|---|---|
| 多租户部署形态 | **每租户独立整套栈**（compose 一套/租户，独立 MySQL/Redis）；隔离在基础设施层，不占 Casbin 维度 |
| 租户访问入口 | **子域名** `<slug>.example.com`，Nginx 泛域名路由；宿主只暴露 80/443 |
| 租户创建入口 | tenant-rpc 在平台后台操作；平台栈 = 第 0 号租户 + tenant-rpc + docker socket / nginx 配置卷 |
| 数据层 | goctl model `--cache true`（go-zero sqlx/sqlc 缓存包），弃 GORM |
| 数据库初始化 | 完整 SQL 导入（[mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql) 唯一入口），不维护增量升级 |
| 功能范围 | 保留 RBAC/字典/部门/文件/审计/AI/App 会员/设备会话；**废弃**：权限版本历史、IP 归属地 |

## 2. 鉴权模型：Casbin（接口级）+ SQL 数据范围（行级）

两类需求分工：

- **角色按部门生效（管理边界）** → Casbin domains 模型，`dom = 部门`
- **部门内资源共享（行级可见性）** → SQL 数据范围过滤，按角色 `data_scope` 生成部门集合条件

Casbin 模型（完整定义见 [pkg/casbinauth](../go-zero-admin/pkg/casbinauth/enforcer.go)）：

```ini
[request_definition]
r = sub, dom, obj, act
[policy_definition]
p = sub, dom, obj, act
[role_definition]
g = _, _, _
[policy_effect]
e = some(where (p.eft == allow))
[matchers]
m = (g(r.sub, p.sub, r.dom) || g(r.sub, p.sub, "*")) && (r.dom == p.dom || p.dom == "*") && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
```

约定：

- 部门域写作 `dept:<id>`；域 `*` 表示全局角色（如 `g, user:1, super_admin, *`）
- `casbin_rule` 表：p 规则 `v0=sub, v1=dom, v2=obj, v3=act`；g 规则 `v0=user, v1=role, v2=dom`
- 租户隔离在数据库层（每租户独立库），`dom` 只承载部门，语义单一

请求部门上下文：JWT 携带主部门；多部门用户用 `X-Dept-Id` 切换；与部门无关的接口（登录/个人中心）只过 JWT。

行级数据范围：角色 `data_scope` = `ALL` / `DEPT_AND_CHILDREN` / `DEPT` / `SELF`；部门树用**物化路径** `path`（如 `/1/7/12/`），"及以下" = `path LIKE '/1/7/%'`，无需递归。Casbin 决定"能不能调接口"，SQL 决定"返回哪些行"。

策略同步：各 API 进程内存 enforcer（CachedEnforcer），admin-rpc 变更后 Redis `casbin:policy:changed` 频道广播 → 全量重拉重建（实现见 admin-api `watchPolicyChanges`）。

## 3. 多租户设计

### 3.1 控制面与业务面分离

- **单库设计**（2026-11-02 合并）：`tenant_registry` 并入 `gozero_admin`，不再独立控制库（原双库是给多机预留，多机已不做）
- **业务库**（每租户独立）：完整业务表，跑在租户栈自己的 MySQL 容器，完全隔离
- **平台栈**：提供完整管理后台并承载 tenant-rpc 控制面；当前服务拓扑见 [Compose](../go-zero-admin/docker-compose.yml) 与[开发手册](dev-guide.md#1-开发环境与配置)。

### 3.2 租户创建流程

1. 填写 slug → 查 `tenant_registry` 校验可用
2. 生成随机密钥：MySQL root / JWT / App JWT
3. 插入登记记录（status=creating）
4. 渲染 `docker/tenant-stack/env.template` → 租户专属 .env
5. `docker compose -p gozero-t-<slug> -f docker/tenant-stack/docker-compose.tenant.yml up -d --build --wait`
6. 渲染 `tenant.conf.template` → `docker/nginx/conf.d/<slug>.conf`，`nginx -s reload`
7. 更新 status=active；失败记 error_msg

### 3.3 生命周期

| 操作 | 实现 |
|---|---|
| 停用/启用 | `docker compose -p gozero-t-<slug> stop/start` + 更新 status |
| 重置管理员密码 | 连租户库 UPDATE `sys_user.password_hash` |
| 删除 | `compose down -v` + 删 nginx conf + 软删 registry 记录 |

### 3.4 默认约定

| 项 | 默认 |
|---|---|
| 租户栈 | api + 4 RPC + MySQL + Redis；共享网络 `gozero-edge`，api 以 `<slug>-api` 别名注册 |
| RustFS | 平台级共享，每租户独立 bucket |
| 本地开发 | `*.localhost`（泛解析 127.0.0.1） |
| 安全 | tenant-rpc 挂 docker socket ≈ 宿主 root，仅平台内网可达；API 仅超管可访问 |

## 4. AI 助手平台演进（三阶段）

### 阶段 1 工具调用 Agent

eino ChatModel + 工具注册表（`pkg/agenttools`）；单一 SendMessageStream 最多 5 轮 function calling，结果重查复用原 request_id；已知 token 用量落库 `ai_message.tokens`。前端“AI 功能 → AI 对话”提供会话、消息、工具记录与用量展示；真实供应商联调边界见问题清单。

### 阶段 2 工作流编排

JSON DSL `{"steps":[{name,tool,input,approval}]}`；审批在对应步骤执行前暂停，包括最后一步；通过任务快照、next_step 和 expected_step 原子比较续跑。任务创建与关键状态更新已有事务/行锁，失败状态为 failed；工具执行不在数据库事务中。工作流管理、任务中心与只读工具目录已接入现有接口，支持顺序步骤编辑、手动运行、本人审批及结果查看；跨人审批、定时调度与中断恢复待开发。

### 阶段 3 业务接入、复用与协作（候选规划）

以下合并 AI Agent 的待选方向、前置依赖与实施边界，不代表已确定全部开发范围，也不代表已实现。已有 AI HTTP 接口、手动工作流、任务快照及步骤审批是基础；定时调度、完整执行追踪、APP 认证域接入和进程中断恢复不能据此视为已完成。当前能力以[共享契约](api-contract.md)及代码为准，真实未完成项继续保留在[问题清单](README.md#后端剩余事项)。

原阶段 3 提出的市场、计费与多 Agent 协作继续作为后续独立需求评估。已有 `ai_message.tokens` 是已知用量基础，缺失用量仍可能未知，不能直接作为完整费用凭证；推进前仍须明确跨人审批权限、工具副作用的幂等/恢复及业务指标。公开 Agent 交易市场不列入基础功能。

#### 候选功能与推进顺序

| 功能 | 首版范围 | 难度 | 建议顺序 |
|---|---|---|---|
| 模型状态与连通性检测 | 展示已配置模型、配置状态，一键测试可用性、响应耗时及失败原因；密钥沿用 env，不回显 | 低 | 第一批 |
| Agent 配置中心 | 管理名称、系统提示词、使用模型、允许调用的工具，便于适配不同业务 | 中 | 第一批 |
| CSV 导入助手 | 识别表头、建议字段映射、提示异常数据，预览确认后导入；校验和写入由业务代码执行 | 中 | 按需 |

优先评估“模型连通性检测 + Agent 配置中心”，再按业务需求评估 CSV 导入。现有配置状态不等于供应商实时健康检测；知识库后续扩展见[问题清单](README.md#后端剩余事项)。不重复规划已有聊天助手、后台业务查询助手的讨论方向及工作流/任务审批页面；业务查询工具尚未接入的事实仍以问题清单为准。

后续候选统一纳入本架构阶段，不再另设产品二期、三期编号：

| 推进批次 | 候选功能 | 批内建议顺序 |
|---|---|---|
| 运行与业务接入 | 业务工具调试和权限、提示词版本、面向业务的结构化输出 | 工具调试和权限 → 提示词版本 |
| 批量与复用 | 批量 AI 处理、Agent 模板导入导出、定时任务、APP 认证域下调用 Agent、可管理的偏好记忆 | 批量处理 + 模板导入导出，再补定时任务 |

#### 后续扩展的历史判断依据

以下沿用截至 2026-11-02 规划中引用的外部资料与设计判断，未在文档合并时重新核实协议版本或适配器兼容性；实施前须再核对，不作为行业未来的保证。

- **业务能力以可复用格式交付。** 原规划参考 Agent Skills 对指令与参考资料的组织方式，以及 MCP 对外部工具与上下文的连接方式；本项目据此将业务方法、工具接入与模型配置分开维护，方便以后替换模型。[Skills 规范](https://agentskills.io/specification)、[MCP 规范（原规划引用版本）](https://modelcontextprotocol.io/specification/2026-07-28)
- **AI 交互呈现表单和业务操作。** 原规划参考 A2UI 用声明式数据描述界面、由客户端可信组件渲染的方案；本项目先复用 Ant Design 实现有限的业务卡片，按需求再评估协议接入。[Google A2UI，2025-12-15](https://developers.googleblog.com/introducing-a2ui-an-open-project-for-agent-driven-interfaces/)
- **长任务保存业务状态、恢复机制和结果验证。** 原规划参考长期 Agent 与评估实践：仅压缩上下文不能解决持续执行的问题，应核对真实结果及多次试验；本项目先保存可验证的执行状态，再增加自动执行范围。[长期任务实践，2025-11-26](https://www.anthropic.com/research/effective-harnesses-for-long-running-agents)、[Agent 评估实践，2026-01-09](https://www.anthropic.com/research/demystifying-evals-for-ai-agents)
- **图片和文档成为业务输入。** 原规划参考图文理解、字段提取及业务校验实践；本项目从一种材料转表单草稿开始。[Google 文档抽取实践，2025-03-03](https://cloud.google.com/blog/products/ai-machine-learning/use-gemini-2-0-to-speed-up-data-processing/)

#### 扩展候选与最小范围

本组依赖前述权限、执行记录和结构化输出；优先级只针对本组，难度为相对估计，不作为工期承诺。

| 功能 | 新增能力 | 首版范围 | 优先级 / 难度 |
|---|---|---|---|
| **业务技能包（Skills）** | 将业务方法作为可供多个 Agent 使用、运行时按需加载的能力；超出完整 Agent 配置的导入导出 | 人工绑定技能；支持说明、步骤、示例和只读参考资料；固定版本并记录本次使用版本 | 优先 / 中 |
| **受控 MCP 连接管理** | 接入外部工具服务，减少每个工具单独编写适配逻辑 | 接入一个可信远程服务；连接测试、工具发现、白名单、启停和 Agent 绑定 | 优先 / 中高 |
| **业务卡片与操作预演** | 让用户直接核对并修改 AI 生成的业务草稿；在已有步骤审批上明确到具体操作和参数 | 先做表单、结果表格、变更确认三种卡片；展示目标及变更前后值，用户确认后调用现有业务接口 | 优先 / 中 |
| **可恢复的长任务** | 在已有任务快照和审批基础上，补齐服务重启后的任务识别与受控续跑 | 保存已完成步骤、进度与执行租约；支持取消和超时；先恢复只读或明确幂等的步骤 | 后续 / 高 |
| **图文材料转业务草稿** | 为结构化输出增加图片、扫描材料输入及原文证据 | 只选一种材料和一种已验证的模型，例如图片表单；提取字段、关联原图位置或原文、人工修正后保存 | 按场景 / 中高 |
| **模型迁移评测与选择** | 将提示词版本比较扩展为模型更换前的效果验收，并记录实际生效的模型配置 | 选 20–50 个脱敏真实样本，对比两个模型的任务成功率、耗时与用量；关键样本重复运行，人工决定切换并保留回退配置 | 优先 / 中 |

#### 扩展能力的安全边界

- **技能包**：首版读取说明和资料，上传脚本不执行；包内工具声明只作为需求，实际权限由服务端校验。版本、来源和许可要可查看。先验证人工绑定的效果，再考虑自动选择技能。[官方格式与按需加载约定](https://agentskills.io/specification)
- **MCP**：管理员限定可连接的服务和网络目的地，凭据沿用服务端配置并按归属使用；工具描述及返回内容不具有授权效力。工具定义变化须重新核对白名单，不能因重新发现工具自动扩大权限。实现前核对双方协议版本及适配器，协议兼容不等于业务权限兼容。[MCP 信任与授权原则（原规划引用版本）](https://modelcontextprotocol.io/specification/2026-07-28)
- **业务卡片**：AI 只返回经过校验的数据，前端映射到允许的 Ant Design 组件；业务动作使用服务端登记的标识。确认绑定目标、参数和数据版本，提交时重新鉴权，重复提交不能重复执行。数据变化后重新预览；预演不产生写入或外发。首版不执行模型生成的 HTML、JavaScript 或任意请求地址。
- **长任务**：租约和重试不能保证外部操作只执行一次；外部写入结果不明时转人工核对，不能直接重试。取消只阻止后续执行，不承诺撤销已完成的外部操作。先明确每个工具的幂等和恢复边界，再开放自动续跑。
- **图文草稿**：保留可核对的来源，识别缺失、模糊或冲突字段交给用户确认；模型自报的置信度不作为自动提交依据。限制文件类型、大小及页数，资料使用范围与现有文件权限一致，调用外部模型前按部署策略控制数据外发。
- **模型评测**：在隔离环境、测试身份及独立数据上执行，每次试验重置测试状态；外部写入与外发使用模拟工具或服务测试环境，禁止向生产工具重放历史任务。校验真实结果和越权拒绝行为，结合规则检查与人工复核；不只让另一个模型打分。用量缺失标记为未知；费用只能按注明日期的价格估算。小样本通过不代表所有场景通过，模型切换不能自动重放已产生业务效果的任务。

#### 选择顺序与采用条件

先做一套贯穿功能的最小场景：**一个业务技能 → 一个受控工具 → 一张结果或确认卡片 → 固定样本验证**。优先使用现有只读业务接口；MCP 在确实需要外部工具时接入，用任务完成率验证新增机制的效果。

长任务在批量、定时任务出现真实恢复需求时推进；多模态从一种材料试点。工具、知识来源和用户记忆都应保留归属、版本及有效期，撤销权限或删除来源后同步处理检索索引、摘要和缓存，不能继续作为可用上下文。

跨平台多 Agent 委派（A2A）按真实集成需求采用：先有一个独立远程 Agent，再考虑任务提交、状态查询和取消。原规划基于项目需求与维护成本暂缓，而非认定协议不成熟；原文引用了稳定版发布说明，实际采用时仍须核对协议与适配器。[A2A 官方发布说明，2026-03-12（历史引用）](https://a2a-protocol.org/latest/blog/2026/03/12/a2a-protocol-ships-v10-production-ready-standard-for-agent-to-agent-communication/)

自动学习反馈可先形成待审核的技能或提示词修改建议，经评测与人工发布后生效；浏览器代操作、任意脚本执行、模型微调及公开 Agent 交易市场留待独立需求论证，不作为基础功能。

#### 实施边界

- 优先复用现有 Eino、认证、权限及数据访问机制。
- 实施前核对接口、表结构、文件清单与依赖需求，再按[开发规范](../AGENTS.md)确认具体范围。
- Eino 和模型适配器版本以 [go.mod](../go-zero-admin/go.mod) 及锁定依赖为准。官方新示例不代表当前适配器直接支持，MCP、多模态等如需新增依赖须先确认，不自动升级。
- 知识库扩展可参考 [Eino Retriever 官方指南](https://www.cloudwego.io/docs/eino/core_modules/components/retriever_guide/)；现有 MySQL/Qdrant 分工保持，附件解析与新的检索方案另行评估。

### 工作区与检索的长期取舍

保留 Eino、Qdrant 和现有顺序执行器，在既有页面与权限体系内提供“资料 → 直接分析 → 结果 → 按需保存为常用流程”的使用路径。流程参数只允许受限的本次输入与前序结果引用，不执行表达式或脚本；MySQL 保存事实及持久状态，Qdrant 保存可重建索引。

后台文档摄取使用持久批次与作业；工作流仍采用有界后台执行。两者的恢复机制分别定义，不将文档队列能力解释为工作流可以自动重放外部调用。完整图形编排、PDF/Office/OCR、全文审阅、混合检索与重排继续按独立需求评估，未完成项见[问题清单](README.md#后端剩余事项)。

### 质量评测的长期取舍

使用冻结资料、固定题集与构建/模型/配置指纹区分召回、回答规则筛查和运行异常；未知结果按原请求核对，不为评测自动追加付费重试或模型裁判。规则筛查不等于语义准确率，历史开发集和已使用的验证集只能作为回归；新方案需要新的独立业务保留集。

接口与执行边界见[共享契约](api-contract.md)，复验入口见[开发手册](dev-guide.md#知识库效果评测)。历史构建、真实失败、分阶段验收和测量范围已合并到[验证证据](validation-evidence.md)、[质量基线](ai-rag/quality-baseline.md)及[性能基线](ai-rag/performance-baseline.md)，不在架构文档保留完成任务清单。

## 5. 明确废弃项（防返工）

| 废弃 | 替代 |
|---|---|
| GORM（pkg/orm） | goctl model（sqlx + sqlc 缓存） |
| data/goctl/ 自定义模板 | 官方默认模板（禁 -home） |
| {code,message,result} 自定义信封 | 官方默认 + xerr + SetErrorHandlerCtx |
| swagger 生成后 PS 修补 | goctl api swagger 原生输出 |
| etcd 服务发现 | 直连 Endpoints |
| 权限版本历史/回滚、IP 归属地 | 无（确认废弃） |
