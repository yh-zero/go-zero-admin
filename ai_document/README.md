# 开发文档

| 我要做什么 | 从哪里开始 |
|---|---|
| 第一次运行项目并登录 | [快速开始](../README.md#快速开始) |
| 启动已有环境、配置 AI 或邮件、备份开发库、排查服务故障 | [开发手册](dev-guide.md) |
| 升级已有数据库、查看迁移版本或处理 dirty 状态 | [数据库版本化迁移](dev-guide.md#数据库版本化迁移) |
| 理解 API、RPC、租户栈、部门授权和数据存储的分工 | [架构说明](architecture.md) |
| 确认返回格式、错误码、字段语义和接口限制 | [共享 API 契约](api-contract.md) · [HTTP 定义](../go-zero-admin/application/admin/api/desc/admin.api) |
| 修改后端、生成接口或数据模型 | [根开发规范](../AGENTS.md) · [后端增量规范](../go-zero-admin/AGENTS.md) · [生成脚本](../go-zero-admin/data/scripts/gen.ps1) |
| 开发前端页面或调整 API 代理 | [前端说明](../go-zero-admin-react/README.md) · [前端增量规范](../go-zero-admin-react/AGENTS.md) |
| 导入演示语料或查看人工问题示例 | [知识库演示语料](ai-rag/README.md) |

可复现的性能记录保留测量条件、样本、异常和原始数据位置。业务语料是固定提交的摘录，来源与指纹见[语料清单](ai-rag/business-corpus-provenance.json)。
