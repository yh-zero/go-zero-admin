# go-zero-admin（重构版）

基于 **go-zero + goctl model（sqlx/sqlc 缓存）+ Casbin** 的多租户管理后台。单 API + 多 RPC；每租户独立整套部署、独立数据库，子域名经 Nginx 路由。

完整方案见[架构规划](../ai_document/refactor-plan.md)，当前接口语义见[API 契约](../ai_document/api-contract.md)，启动和排障见[开发手册](../ai_document/dev-guide.md)。

## 工具版本

- Go `1.27.3`（`go 1.27.0` + toolchain `go1.27.3`）
- go-zero `v1.10.3` / goctl `v1.10.2`（分别核对 `go.mod` 与生成脚本/CI 的工具版本，不把声明当作本机安装验证）
- jwt/v5 `v5.3.1` / x/crypto `v0.58.0`
- MySQL `9.7`

## 目录

- `application/<模块>/api|rpc` —— 业务模块（admin / app / file / ai / tenant）
- `data/db/init/mysql-all.sql` —— 完整数据库初始化入口（单库 gozero_admin，含 tenant_registry）
- `data/scripts/gen.ps1` —— 生成命令入口
- `docker/` —— Dockerfile、Nginx 与租户栈模板
- `docker/Dockerfile` —— 多阶段镜像打包（api / 各 rpc）；先经 `data/scripts/build-images.ps1` 编译 Linux 二进制，Dockerfile 本身不编译 Go 源码

## 生成命令

```powershell
.\data\scripts\gen.ps1 api              # API（官方默认模板）
.\data\scripts\gen.ps1 rpc admin        # RPC（admin|app|file|ai|tenant）
.\data\scripts\gen.ps1 model admin sys_user  # 数据层（goctl model，带缓存）
.\data\scripts\gen.ps1 swagger          # Swagger（官方原生）
```

数据层生成前，先在新开发库导入 `data/db/init/mysql-all.sql`；Compose 的 MySQL 空卷首次启动时会自动导入。初始化方式见 [开发手册](../ai_document/dev-guide.md)。

> 当前已有 API/RPC 业务实现、model 扩展和部署配置，不是仅含 desc 的骨架。生成物与可维护骨架须按 [AGENTS.md 的文件分类](AGENTS.md#2-开发与生成规则)分别处理，不能把整个 `internal/`、配置或主入口都视为禁止编辑的生成物。

## AI 协作开发（Codex / DSH / Claude Code / 其他）

- 开发先读[根规范](../AGENTS.md)，再读[后端规范](AGENTS.md)；涉及接口时遵守[共享契约](../ai_document/api-contract.md)。本目录未提供 `CLAUDE.md`，不依赖不存在的指针文件。
- **mcp-zero** 为可选辅助工具：[.mcp.json](.mcp.json) 包含其客户端配置，不代表工具已安装、路径有效或当前客户端已加载。安装/版本变更先获批准并选定版本，不默认安装 `@latest`；未接入也可按 `gen.ps1` 工作流开发，工具不得绕过该生成入口。
- 仓库当前未提供 `.agents/skills/go-zero-admin` / `.claude/skills/go-zero-admin` 两个项目技能目录，不把它们写成已可用能力；如需接入，先确认来源、版本、安装位置及客户端发现方式。
- 可选接入官方指令层/知识库：以下仅为接入示例，须明确批准后执行并记录审查过的提交版本，不自动安装或跟随上游更新；本仓库已由 Git 管理，无需再次 `git init`：
  `git submodule add https://github.com/zeromicro/ai-context.git .github/ai-context`
  `git submodule add https://github.com/zeromicro/zero-skills.git .github/zero-skills`
- 提交前自检：`.\data\scripts\verify.ps1`；静态门槛不代替受影响回归和功能验收，纯文档检查范围按根规范执行。

## 生成物命名约定

- 本仓库 `gen.ps1 rpc` 使用 `--name-from-filename`：`admin.proto` → `admin.go` 与 `etc/admin.yaml`，其他模块同理；这取决于生成参数，不是固定生成 `pb.go`。既有 main 可能被跳过，新 service 仍须检查注册，不为改名丢失已有初始化和业务代码。
- `goctl model` 生成 `<table>_model_gen.go`（勿改）+ `<table>_model.go`（手写扩展）。
