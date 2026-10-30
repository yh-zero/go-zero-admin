# go-zero-admin（重构版）

基于 **go-zero + goctl model（sqlx/sqlc 缓存）+ Casbin** 的多租户管理后台。单 API + 多 RPC；每租户独立整套部署、独立数据库，子域名经 Nginx 路由。

完整方案见 `../ai_document/refactor-plan.md`。

## 工具版本

- Go `1.27.3`（`go 1.27.0` + toolchain `go1.27.3`）
- go-zero `v1.10.3` / goctl `v1.10.2`（2026-10 核实）
- jwt/v5 `v5.3.1` / x/crypto `v0.58.0`
- MySQL `9.7`
- goose（版本待锁定）

## 目录

- `application/<模块>/api|rpc` —— 业务模块（admin / app / file / ai / tenant）
- `data/db/migrations` —— goose 迁移（结构唯一事实源）
- `data/scripts/gen.ps1` —— 生成命令入口
- `docker/` —— Dockerfile、Nginx 与租户栈模板
- `docker/Dockerfile` —— 多阶段构建（api / 各 rpc / migrate）

## 生成命令

```powershell
.\data\scripts\gen.ps1 api              # API（官方默认模板）
.\data\scripts\gen.ps1 rpc admin        # RPC（admin|app|file|ai|tenant）
.\data\scripts\gen.ps1 model admin sys_user  # 数据层（goctl model，带缓存）
.\data\scripts\gen.ps1 db up            # goose 迁移
.\data\scripts\gen.ps1 swagger          # Swagger（官方原生）
```

> 骨架阶段：`internal/`、`etc/`、`pb/`、`client/` 由 goctl 生成，当前仅含 desc 契约与部署骨架。

## AI 协作开发（Codex / DSH / Claude Code / 其他）

- **`AGENTS.md` 是唯一权威规范**，所有 AI 工具与人的共同入口；`CLAUDE.md` 为指针。
- **mcp-zero**（go-zero 官方 MCP）已注册于 `.mcp.json`；安装：`go install github.com/zeromicro/mcp-zero@latest`。
- 项目技能：`.agents/skills/go-zero-admin`（DSH）、`.claude/skills/go-zero-admin`（Claude Code）。
- 可选接入官方指令层（需先 git init）：
  `git submodule add https://github.com/zeromicro/ai-context.git .github/ai-context`
  `git submodule add https://github.com/zeromicro/zero-skills.git .github/zero-skills`
- 提交前自检：`.\data\scripts\verify.ps1`。

## 生成物命名约定

- `goctl rpc` 的 main 文件名与 proto 同名（`admin.proto` → `pb.go`），这是 goctl 固定行为，不要手动改名。
- `goctl model` 生成 `<table>_model_gen.go`（勿改）+ `<table>_model.go`（手写扩展）。
