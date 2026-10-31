# go-zero-admin

基于 **go-zero + React** 的多租户管理后台。单仓库全栈：go-zero 微服务后端 + React 管理前端。

## 仓库结构

```
├── go-zero-admin/          后端（go-zero v1.10.3）
│   ├── application/        admin|app|file|ai|tenant 五模块（api + rpc）
│   ├── data/               migrations（goose）/ seeds / scripts（gen.ps1, verify.ps1）
│   ├── docker/             Dockerfile + nginx + 租户栈模板
│   ├── pkg/                公共基础设施（xerr/jwtx/objectstorage/llm/agenttools/apptoken/casbinauth）
│   └── AGENTS.md           后端开发规范（go-zero 规范优先）
├── go-zero-admin-react/    前端（React 19 + Vite + antd 5）
│   └── AGENTS.md           前端开发规范（React 规范优先）
├── ai_document/
│   ├── refactor-plan.md    重构方案（架构决策、落地顺序、AI 平台演进）
│   └── api-contract.md     前后端共享 API 契约
└── AGENTS.md               仓库通用规范（所有任务必读）
```

## 快速开始

### 后端

```powershell
cd go-zero-admin
docker compose up -d --build --wait   # MySQL 9.7 / Redis / RustFS / 全服务
```

本地开发（依赖：Go 1.27.3（`go 1.27.0` + `toolchain go1.27.3`）/ MySQL 9.7 / Redis 8，端口见各 etc/*.yaml）：

```powershell
go run ./application/admin/rpc -f application/admin/rpc/etc/admin.yaml
go run ./application/admin/api -f application/admin/api/etc/admin.yaml
```

默认账号 `admin / Admin@123456`（需先获取验证码登录）。

### 前端

```powershell
cd go-zero-admin-react
pnpm install
pnpm dev        # http://localhost:5999，/v1 已代理到后端 7001
```

### 提交前自检

```powershell
# 后端（gofmt/vet/build/tidy-diff/api validate/目录白名单）
go-zero-admin: .\data\scripts\verify.ps1
# 前端（类型检查 + 构建）
go-zero-admin-react: npx tsc -b && npx vite build
```

## 架构要点

- **多租户**：每租户独立整套栈 + 独立数据库，子域名经 Nginx 路由；tenant-rpc 负责 provisioning
- **鉴权**：Casbin domains 模型（dom = 部门）+ Redis 策略热更新；接口级 Casbin + 行级 SQL 数据范围
- **AI 平台**：eino 工具调用 Agent（阶段 1）+ 工作流编排与人工审批（阶段 2）；演进规划见 ai_document/refactor-plan.md
- **数据层**：goctl model `--cache true`（go-zero sqlx/sqlc 缓存包），goose 迁移，无 GORM

## 文档

| 文档 | 说明 |
|---|---|
| [AGENTS.md](AGENTS.md) | 仓库通用规范（AI 与开发者共同约束） |
| [ai_document/api-contract.md](ai_document/api-contract.md) | 前后端 API 契约 |
| [ai_document/refactor-plan.md](ai_document/refactor-plan.md) | 架构方案与 AI 平台演进 |
| go-zero-admin/AGENTS.md | 后端专项规范 |
| go-zero-admin-react/AGENTS.md | 前端专项规范 |
