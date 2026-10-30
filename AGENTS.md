# AGENTS.md — 仓库通用规范（所有任务必读）

> 本文件是**整个仓库**所有人员与 AI 代理（Codex / DeepSeek Harness / Claude Code / 其他）的共同约束。
> 后端开发另读 `go-zero-admin/AGENTS.md`（go-zero 规范优先）；前端另读 `go-zero-admin-react/AGENTS.md`（React 规范优先）。
> 子目录规则与本文件冲突时，**以子目录为准**；本文件未覆盖的情况，**先问人再动手**。

## 1. 仓库结构

| 目录 | 内容 |
|---|---|
| `go-zero-admin/` | 后端：go-zero v1.10.3 微服务（admin/app/file/ai/tenant 五模块），Go 1.27.3 |
| `go-zero-admin-react/` | 前端：React 19 + Vite 8 + antd 5 |
| `ai_document/api-contract.md` | **前后端共享 API 契约（唯一事实源）** |
| `ai_document/refactor-plan.md` | 架构方案与 AI 平台演进规划 |

## 2. 任务路由（动手前先读对应规范）

| 任务 | 必读 |
|---|---|
| 后端接口/RPC/表结构 | 本文件 → `go-zero-admin/AGENTS.md` |
| 前端页面/组件/对接 | 本文件 → `go-zero-admin-react/AGENTS.md` |
| 接口契约（字段/错误码/路由） | 本文件 → `ai_document/api-contract.md`，并同步两端 |

## 3. 边界纪律（违反一律返工）

1. **改前端不动后端文件，改后端不动前端文件**；跨端改动必须在回复中显式声明两侧改动清单。
2. 契约类改动（响应格式/字段名/错误码/路由）必须先改 `api-contract.md` 再动代码。
3. 禁止修改生成物与锁文件（`*_gen.go`、`routes.go`、`pnpm-lock.yaml`、`go.sum`），除非任务本身就是重新生成。

## 4. 提交规范

- 提交信息用**中文** + conventional 前缀：`feat:` / `fix:` / `refactor:` / `docs:` / `chore:` / `test:`
- 标识符用英文，注释/文档/提交信息用中文
- 提交前必须过对应自检（后端 verify.ps1；前端 tsc -b + vite build），**不过不提交**

## 5. 安全基线

- 密钥/密码只进各服务 yaml 或环境变量，**禁止进仓库**（.env 已 gitignore）
- 发现疑似泄漏的密钥立即报告，不在日志/错误信息中输出
- 不升级/替换依赖版本，除非任务明确批准

## 6. 质量与易读原则（两端共同）

- **安全 > 质量 > 易读 > 简洁**：先保正确与安全，再谈简化
- 代码自解释优先：命名达意 > 注释；注释解释"为什么"而非"是什么"
- 能删不绕：删代码比加代码更值得审查；重复逻辑提取公共函数
- 每个函数/组件单一职责；超过 ~100 行考虑拆分
