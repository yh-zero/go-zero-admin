# go-zero-admin 架构决策与演进规划

> 本文档只保留**长期知识**：已定型的架构决策、多租户设计、鉴权模型、AI 平台演进。
> 过程性记录（联调日报/验收快照）一律放 git 提交信息，不进本文档。
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
- **平台栈不跑业务**：api（租户管理接口）+ tenant-rpc + nginx + rustfs

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

### 阶段 1 工具调用 Agent（已实现）

eino ChatModel + 工具注册表（`pkg/agenttools`）；单一 SendMessageStream 最多 5 轮 function calling，结果重查复用原 request_id；已知 token 用量落库 `ai_message.tokens`。前端“AI 功能 → AI 对话”提供会话、消息、工具记录与用量展示；真实供应商联调边界见问题清单。

### 阶段 2 工作流编排（已实现）

JSON DSL `{"steps":[{name,tool,input,approval}]}`；审批在对应步骤执行前暂停，包括最后一步；通过任务快照、next_step 和 expected_step 原子比较续跑。任务创建与关键状态更新已有事务/行锁，失败状态为 failed；工具执行不在数据库事务中。工作流管理、任务中心与只读工具目录已接入现有接口，支持顺序步骤编辑、手动运行、本人审批及结果查看；跨人审批、定时调度与中断恢复待开发。

### 阶段 3 多 Agent 协作平台（规划）

Agent 市场/计费/多 Agent 协作；已有 `ai_message.tokens` 作为已知用量基础，缺失用量仍可能未知，不能直接作为完整费用凭证；待明确跨人审批权限、工具副作用的幂等/恢复及业务指标。

### AI 使用体验优化（2026-11-07，用户已批准开发）

目标：普通用户可以选择资料完成有来源的对话，并将本次问题和已完成步骤结果用于后续工作流；资料和长任务可维护、可追踪。

采用现有Eino/工具/页面增量实现。继续使用MySQL事实源、Qdrant派生索引和现有权限；不升级依赖。完整图形编排引擎、外置RAG平台及无幂等保证的自动重放会扩大当前复杂度，本轮采用受限参数引用和有界后台执行。PDF/Office解析留待明确文件解析依赖后扩展，本轮批量TXT/Markdown。

按独立职责并行开发，契约来源和生成统一集成：

1. 聊天：先补指定库/严格资料模式和实际来源；再补反馈、会话整理、受限Markdown和完整导出。修改聊天logic/model、工具scope、聊天页面与对应回归。
2. 工作流：先实现有类型的运行输入/前序结果引用和步骤持久记录，再接request_id、后台执行/取消、审批真实参数预览和复制模板。修改现有workflow/task logic/model与页面，保留旧静态定义。
3. 文档：先补详情/版本替换、元信息投影，再补固定ID状态刷新、原文维护和每批20文件/2并发导入。保持分块代次、删除、重试与权限语义。
4. 集成：共享契约先行，更新.api/.proto/完整SQL，从隔离库生成model，不重置既有开发资料；局部桩及隔离DB回归后构建更新API/AI RPC，浏览器验收受影响页面。

重点回归：跨用户/部门隔离；幂等载荷变化/未知结果；取消与审批/迟到执行竞争；文档更新/删除/摄取代次竞争；不可信HTML/JSON/原型字段；UTF-8字节和输出预算。模型真实联调只在既有授权的有限范围内，不将小样本当作生产性能结论。进度和仍未验收项保留在api-contract.md与README.md，临时资源记录于被忽略tmp/ai-product-20261107。

### 四模块工作区轻重构（2026-11-07，本地实现与验收已完成）

采用用户确认的独立设计草案：资料→直接分析→结果，复用时再保存常用流程。保留Eino、Qdrant、现有顺序执行器、归属/权限/幂等和安全Markdown；不新增依赖。四模块一起交付，不将静态设计当功能完成。

1. 先冻结共享契约，来源`.api/.proto/完整初始化SQL`统一由主任务修改并生成；工作流列表摘要/服务端搜索、任务标题快照/轻量状态、草稿直接运行、KB能力/计数/后台批次接口同步前端types/api。
2. 后台摄取独立实现：MySQL批次+作业事务，2worker、90秒工作/120秒租约、queued恢复及running过期失败；保留版本、chunk代次与删除补偿，不盲目重放外部调用。创建/更新/重试快速返回接收状态，前端可离页后回批次。
3. 前端四模块按已确认图片增量重构：隐藏KB详情工作区、就地分析及原文连通；简化模板/动态运行参数/未保存试运行；结果优先的运行记录和协调刷新；编辑器能力选择与目录/配置状态。
4. 每个包或组件先写关键失败回归再实现，使用隔离MySQL/Redis、本地Embedding/模型桩覆盖异步、幂等、取消、版本/删除竞争和权限边界；记录请求/SQL/外部调用次数，真实主栈只在最终构建版本上验证受影响正常路径与必要失败场景。现有资料与流程保留。
5. 最后更新README/开发手册/契约状态与实际限制，精准清理本轮资源，保留用户要求的项目服务。PDF/DOCX、全文审阅及混合检索/重排按真实需求后续扩展，本轮不以模板提示词冒充实现。

本地主栈已验证两篇文本后台导入与离页返回、一次真实分析及当前原文、未保存与已保存流程执行、审批/失败、筛选和窄屏；隔离回归覆盖异步竞争与异常回执。性能证据与未覆盖边界见开发手册和问题清单。临时验收资料、流程、任务、测试会话及隔离服务已精确清理，原有11篇资料、流程3/4和任务4核对保留，项目服务继续运行。

进度、验证、临时资源及裁决接续已有`tmp/ai-product-20261107`记录；本次用户仅授权开发与启动验证，不将此前一次Git交付授权视为自动持续推送。

## 5. 明确废弃项（防返工）

| 废弃 | 替代 |
|---|---|
| GORM（pkg/orm） | goctl model（sqlx + sqlc 缓存） |
| data/goctl/ 自定义模板 | 官方默认模板（禁 -home） |
| {code,message,result} 自定义信封 | 官方默认 + xerr + SetErrorHandlerCtx |
| swagger 生成后 PS 修补 | goctl api swagger 原生输出 |
| etcd 服务发现 | 直连 Endpoints |
| 权限版本历史/回滚、IP 归属地 | 无（确认废弃） |
