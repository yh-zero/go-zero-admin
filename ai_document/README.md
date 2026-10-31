# ai_document

项目文档目录。

## 文档索引

| 文档 | 说明 |
|---|---|
| [refactor-plan.md](refactor-plan.md) | go-zero-admin 后端重构方案 v1.0：总体架构（1 API + 5 RPC、双层多租户）、代码生成命令规范（goctl model/sqlx/sqlc）、错误处理规范、数据库初始化与落地顺序 |

数据库初始化唯一入口为 [mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql)。每次新建库导入完整 SQL；Compose 使用空数据卷首次启动时自动导入。模型仍由 `gen.ps1 model` 从已导入的开发库反射生成。
