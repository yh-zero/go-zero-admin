# ai_document

长期知识库：架构决策 / 契约 / SOP / 事故清单 / 演进规划 / 剩余事项。

## 目录约定

本目录**只放长期知识**。过程性记录（联调日报、验收快照、本轮改动清单）一律放 git 提交信息，不进本目录。文档不得复制命令细节（gen.ps1 等脚本为唯一入口），不得复制规范本体（三层 AGENTS.md 为唯一事实源）。

## 文档索引

| 文档 | 内容 | 角色 |
|---|---|---|
| [refactor-plan.md](refactor-plan.md) | 架构决策（多租户/鉴权模型）、AI 平台三阶段演进、废弃项 | 决策与规划 |
| [api-contract.md](api-contract.md) | 前后端共享 API 契约 + 接口实现状态矩阵 | **唯一契约事实源** |
| [dev-env.md](dev-env.md) | 本地环境拓扑、启动顺序、故障排查 SOP | 联调手册 |
| [pitfalls.md](pitfalls.md) | 15 个历史事故与对策 | 避坑清单 |
| [backend-api-integration-issues.md](backend-api-integration-issues.md) | 后端剩余事项（完成即删行） | 活待办 |
| [rive-login-cat.md](rive-login-cat.md) | 登录猫动画的需求/交互/素材维护 | 功能文档 |

数据库初始化唯一入口为 [mysql-all.sql](../go-zero-admin/data/db/init/mysql-all.sql)：新建库导入完整 SQL，Compose 空卷首启自动导入；模型由 `gen.ps1 model` 从已导入的开发库反射生成。
