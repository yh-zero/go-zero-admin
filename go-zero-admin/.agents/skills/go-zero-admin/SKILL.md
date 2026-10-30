---
name: go-zero-admin
description: 在 go-zero-admin 仓库开发时必须遵守的规范：go-zero 官方约束、生成命令入口、目录与依赖纪律、错误处理与数据层规则。任何涉及本仓库代码的创建、修改、重构任务都应先加载。
---

# go-zero-admin 项目开发规范

1. **先读仓库根 `AGENTS.md`**——它是本仓库唯一权威约束（硬性规则、禁止事项、生成命令、自检脚本）。
2. 架构与决策背景读 `../ai_document/refactor-plan.md`（仓库上一级目录）。
3. 生成代码优先调用 **mcp-zero** 的 MCP 工具（创建服务 / 生成 model / 校验 api 与 proto）；无 MCP 时用 `data/scripts/gen.ps1`。
4. 完成前必须运行 `data/scripts/verify.ps1` 并全部通过。
5. 拿不准的 go-zero 用法查官方文档 <https://go-zero.dev> 或官方 zero-skills 知识库，不凭记忆编造。
