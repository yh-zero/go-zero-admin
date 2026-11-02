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

- **控制库**（平台栈）：仅 `tenant_registry` 一张表（slug/域名/compose 项目名/随机密钥/状态）
- **业务库**（每租户独立）：完整业务表，跑在租户栈自己的 MySQL 容器，完全隔离
- **平台栈不跑业务**：api（租户管理接口）+ tenant-rpc + 控制库 + nginx + rustfs

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

eino ChatModel + 工具注册表（`pkg/agenttools`）；SendMessage 最多 5 轮 function calling；token 用量落库 `ai_message.tokens`。

### 阶段 2 工作流编排（已实现）

JSON DSL `{"steps":[{name,tool,input,approval}]}`；`approval=true` 中间步骤中断 → 人工 Approve/Reject → 续跑；任务状态机 pending_approval/succeeded/rejected/error。已知缺口：无事务、审批无角色门槛（排入阶段 3 前处理）。

### 阶段 3 多 Agent 协作平台（规划）

Agent 市场/计费/多 Agent 协作；前置依赖：`ai_message.tokens` 计费数据已落库；待补：审批角色门槛、工作流事务、业务指标。

## 5. 明确废弃项（防返工）

| 废弃 | 替代 |
|---|---|
| GORM（pkg/orm） | goctl model（sqlx + sqlc 缓存） |
| data/goctl/ 自定义模板 | 官方默认模板（禁 -home） |
| {code,message,result} 自定义信封 | 官方默认 + xerr + SetErrorHandlerCtx |
| swagger 生成后 PS 修补 | goctl api swagger 原生输出 |
| etcd 服务发现 | 直连 Endpoints |
| 权限版本历史/回滚、IP 归属地 | 无（确认废弃） |
