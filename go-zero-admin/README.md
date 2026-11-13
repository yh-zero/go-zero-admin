# go-zero-admin（重构版）

基于 **go-zero + goctl model（sqlx/sqlc 缓存）+ Casbin** 的多租户管理后台。单 API + 多 RPC；每租户独立整套部署、独立数据库，子域名经 Nginx 路由。

完整方案见[架构规划](../ai_document/refactor-plan.md)，当前接口语义见[API 契约](../ai_document/api-contract.md)，启动和排障见[开发手册](../ai_document/dev-guide.md)。

## 依赖与工具来源

- Go、go-zero 及运行库版本以 [go.mod](go.mod) / [go.sum](go.sum) 为准。
- 生成工具版本以[生成脚本](data/scripts/gen.ps1)与 [CI](../.github/workflows/ci.yml) 为准；声明不等于本机安装验证。
- MySQL 等服务镜像以 [Compose](docker-compose.yml) 为准；启动步骤见[开发手册](../ai_document/dev-guide.md#2-构建启动与就绪检查)。

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

> 当前已有 API/RPC 业务实现、model 扩展和部署配置，不是仅含 desc 的骨架。生成物与可维护骨架须按 [AGENTS.md 的文件分类](AGENTS.md#2-生成入口与文件边界官方规则之上的项目适配)分别处理，不能把整个 `internal/`、配置或主入口都视为禁止编辑的生成物。

## AI 协作开发（Codex / DSH / Claude Code / 其他）

- 开发先读[根规范](../AGENTS.md)，再读[后端规范](AGENTS.md)；涉及接口时遵守[共享契约](../ai_document/api-contract.md)。本目录未提供 `CLAUDE.md`，不依赖不存在的指针文件。
- **mcp-zero** 为可选辅助工具：[.mcp.json](.mcp.json) 包含其客户端配置，不代表工具已安装、路径有效或当前客户端已加载。安装/版本变更先获批准并选定版本，不默认安装 `@latest`；未接入也可按 `gen.ps1` 工作流开发，工具不得绕过该生成入口。
- 仓库当前未提供 `.agents/skills/go-zero-admin` / `.claude/skills/go-zero-admin` 两个项目技能目录，不把它们写成已可用能力；如需接入，先确认来源、版本、安装位置及客户端发现方式。
- 官方指令层与知识库已通过 `.ai-context/` submodule 接入，入口为 [go-zero 指令](../.ai-context/go-zero/00-instructions.md)与 [zero-skills](../.ai-context/zero-skills/SKILL.md)；初始化与更新纪律见[根规范](../AGENTS.md#1-官方规范层后端-go-zero-任务先读)。
- 提交前自检：`.\data\scripts\verify.ps1`；静态门槛不代替受影响回归和功能验收，纯文档检查范围按根规范执行。

## 生成物命名约定

RPC 文件命名、配置命名及 model 生成物与手写扩展的边界统一见[后端规范](AGENTS.md#2-生成入口与文件边界官方规则之上的项目适配)。
